package withdrawalsv2

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/monitorism/op-monitorism/processor"
	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals-v2/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	MetricsNamespace = "withdrawals_v2"

	// proveMethodName is the portal method whose arguments carry the withdrawal proof.
	proveMethodName = "proveWithdrawalTransaction"

	// Failure reasons (values of the "reason" metric label).
	//
	// Both P0 reasons mean: the portal accepted a proof that a correct verifier
	// rejects. They are game-independent by design — we deliberately do NOT gate
	// on dispute-game validity, so a proof-system bug is caught even when the
	// game is invalid.
	reasonBadOutputRootBinding = "bad_output_root_binding" // keccak(outputRootProof) != game.rootClaim (P0)
	reasonBadWithdrawalProof   = "bad_withdrawal_proof"    // storage proof does not prove inclusion (P0)

	// Unverifiable reasons — we could not recover/re-verify the proof. These are
	// NOT P0; they mean the monitor is blind for this withdrawal and needs an
	// operator to look (e.g. archive/trace RPC unavailable).
	reasonTraceUnavailable = "trace_unavailable"
	reasonNoProveCallFound = "no_prove_call_found"
	reasonDecodeError      = "decode_error"
	reasonGameReadError    = "game_read_error"

	// traceAttempts bounds the immediate in-line trace retries for a single log.
	// Events that still don't resolve are parked in the pending set and retried
	// asynchronously (see retryPending), so the processor is never blocked.
	traceAttempts = 3

	// pendingRetryInterval is how often the async loop re-evaluates events that
	// have not yet reached a terminal valid/invalid verdict.
	pendingRetryInterval = 60 * time.Second
)

// Metrics holds the Prometheus counters and gauges for the monitor.
type Metrics struct {
	validWithdrawals   *prometheus.CounterVec
	invalidWithdrawals *prometheus.CounterVec
	unverifiable       *prometheus.CounterVec
	// pending is the current number of prove events awaiting a terminal verdict.
	pending prometheus.Gauge
	// oldestPendingSeconds is the age of the oldest such event — the alert signal:
	// a value that keeps growing means an event never reached valid/invalid.
	oldestPendingSeconds prometheus.Gauge
}

// Monitor re-verifies, from L1 only, that every withdrawal the OptimismPortal2
// accepted carries a proof a correct verifier also accepts.
type Monitor struct {
	log           log.Logger
	l1Client      *ethclient.Client
	l1Raw         *rpc.Client
	portal        *bindings.OptimismPortal2
	dgf           *bindings.DisputeGameFactory
	systemConfig  *bindings.SystemConfig
	portalAddress common.Address
	portalABI     *abi.ABI
	proveSelector [4]byte
	wdTxArgs      abi.Arguments
	processor     *processor.BlockProcessor
	metrics       Metrics

	// l2ChainID is read lazily (only for super-game root claims) and cached. A
	// dedicated mutex lets the processor and the async retry loop resolve it
	// concurrently without racing on the first read.
	l2ChainIDMu sync.Mutex
	l2ChainID   *big.Int

	// pending holds prove events that have not yet reached a terminal verdict,
	// keyed by "<txHash>:<logIndex>". The async retryPending loop drives each to
	// valid/invalid so no event is ever silently dropped after a transient RPC
	// failure. Guarded by pendingMu (processor and retry goroutines both touch it).
	pendingMu sync.Mutex
	pending   map[string]*pendingEvent
}

// pendingEvent is a prove event awaiting a terminal verdict.
type pendingEvent struct {
	log       types.Log
	wdHash    [32]byte
	firstSeen time.Time
	attempts  int
}

// pendingKey uniquely identifies a prove event by its log position.
func pendingKey(lg types.Log) string {
	return lg.TxHash.Hex() + ":" + strconv.FormatUint(uint64(lg.Index), 10)
}

// newWithdrawalTxArgs builds the ABI argument list used to recompute a withdrawal
// hash: keccak256(abi.encode(nonce, sender, target, value, gasLimit, data)).
func newWithdrawalTxArgs() (abi.Arguments, error) {
	uint256T, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	addrT, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	bytesT, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	return abi.Arguments{
		{Type: uint256T}, // nonce
		{Type: addrT},    // sender
		{Type: addrT},    // target
		{Type: uint256T}, // value
		{Type: uint256T}, // gasLimit
		{Type: bytesT},   // data
	}, nil
}

// logStartupConfig reports which env vars are set vs. unset and the full resolved
// configuration values, so debugging is easy: every effective value — the node URL,
// portal, start block, poll interval — is printed at startup.
// redactURL masks the middle of an RPC URL so startup logs reveal only enough to
// identify which endpoint is configured (for debugging) without exposing embedded
// credentials or tokens — RPC URLs from the ExternalSecret commonly include both.
// Only the first and last few characters are shown; short strings are fully masked.
func redactURL(u string) string {
	const keepStart, keepEnd = 12, 6
	if len(u) <= keepStart+keepEnd {
		return "***"
	}
	return u[:keepStart] + "..." + u[len(u)-keepEnd:]
}

func logStartupConfig(log log.Logger, cfg CLIConfig) {
	envVars := []string{
		"WITHDRAWALS_V2_MON_L1_NODE_URL",
		"WITHDRAWALS_V2_MON_START_BLOCK",
		"WITHDRAWALS_V2_MON_LOOKBACK_BLOCKS",
		"WITHDRAWALS_V2_MON_POLL_INTERVAL",
		"WITHDRAWALS_V2_MON_OPTIMISM_PORTAL",
	}
	var set, unset []string
	for _, name := range envVars {
		if _, ok := os.LookupEnv(name); ok {
			set = append(set, name)
		} else {
			unset = append(unset, name)
		}
	}
	log.Info("environment variables", "set", strings.Join(set, ","), "unset", strings.Join(unset, ","))
	log.Info("resolved config",
		"l1_node_url", redactURL(cfg.L1NodeURL),
		"optimism_portal", cfg.OptimismPortalAddress,
		"start_block", cfg.StartBlock,
		"start_block_note", "inclusive (this block is scanned); 0 means start near finalized and re-scan lookback_blocks",
		"lookback_blocks", cfg.LookbackBlocks,
		"poll_interval", cfg.PollingInterval,
	)
}

// logNodeHeights queries and logs the node's latest/finalized heights and the tag
// the monitor will scan against, warning loudly if the start block is ahead of the
// chain (the monitor would then sit idle with no per-tick output — which looks dead
// but isn't). Best-effort: never fails construction.
func logNodeHeights(ctx context.Context, log log.Logger, l1 *ethclient.Client, cfg CLIConfig) {
	latest, err := l1.BlockNumber(ctx)
	if err != nil {
		log.Warn("could not fetch latest block height at startup", "err", err)
		return
	}

	var finalizedHdr *types.Header
	_ = l1.Client().CallContext(ctx, &finalizedHdr, "eth_getBlockByNumber", "finalized", false)
	finalizedStr := "<null>"
	if finalizedHdr != nil {
		finalizedStr = finalizedHdr.Number.String()
	}

	tag := "finalized"
	head := uint64(0)
	if finalizedHdr != nil {
		head = finalizedHdr.Number.Uint64()
	}
	if cfg.UseLatest {
		tag, head = "latest", latest
	}
	log.Info("node heights", "latest", latest, "finalized", finalizedStr, "scanning_tag", tag)

	switch {
	case cfg.StartBlock == 0:
		log.Info("start block 0: scanning will begin from the latest " + tag + " block")
	case !cfg.UseLatest && finalizedHdr == nil:
		log.Warn("node reports no finalized block; monitor will idle. For local/dev (anvil) nodes pass --use.latest")
	case cfg.StartBlock > head:
		log.Warn("START BLOCK IS AHEAD OF CHAIN HEAD — monitor will idle (no blocks to scan) until the chain reaches it",
			"start_block", cfg.StartBlock, "scanning_tag", tag, "tag_height", head)
	}
}

// NewMonitor constructs the withdrawals-v2 monitor.
func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating withdrawals v2 monitor")
	logStartupConfig(log, cfg)

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, err
	}
	logNodeHeights(ctx, log, l1Client, cfg)

	portalAddress := common.HexToAddress(cfg.OptimismPortalAddress)
	portal, err := bindings.NewOptimismPortal2(portalAddress, l1Client)
	if err != nil {
		return nil, err
	}

	// Resolve the DisputeGameFactory once. Games are looked up by their factory
	// index (immutable), not via provenWithdrawals (mutable — a re-prove
	// overwrites it), so the monitor always re-verifies against the exact game a
	// given prove call targeted.
	factoryAddr, err := portal.DisputeGameFactory(nil)
	if err != nil {
		return nil, fmt.Errorf("read disputeGameFactory: %w", err)
	}
	dgf, err := bindings.NewDisputeGameFactory(factoryAddr, l1Client)
	if err != nil {
		return nil, err
	}

	// Bind SystemConfig so we can read l2ChainId when a super game is encountered
	// (mirrors the portal's super-game predicate). The chain id itself is read
	// lazily, so a SystemConfig without l2ChainId() never breaks legacy chains.
	systemConfigAddr, err := portal.SystemConfig(nil)
	if err != nil {
		return nil, fmt.Errorf("read systemConfig: %w", err)
	}
	systemConfig, err := bindings.NewSystemConfig(systemConfigAddr, l1Client)
	if err != nil {
		return nil, err
	}

	portalABI, err := bindings.OptimismPortal2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	method, ok := portalABI.Methods[proveMethodName]
	if !ok {
		return nil, fmt.Errorf("portal ABI missing %s", proveMethodName)
	}
	var proveSelector [4]byte
	copy(proveSelector[:], method.ID)

	wdTxArgs, err := newWithdrawalTxArgs()
	if err != nil {
		return nil, err
	}

	mon := &Monitor{
		log:           log,
		l1Client:      l1Client,
		l1Raw:         l1Client.Client(),
		portal:        portal,
		dgf:           dgf,
		systemConfig:  systemConfig,
		pending:       make(map[string]*pendingEvent),
		portalAddress: portalAddress,
		portalABI:     portalABI,
		proveSelector: proveSelector,
		wdTxArgs:      wdTxArgs,
		metrics: Metrics{
			validWithdrawals: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "valid_withdrawals_total",
					Help:      "Total number of withdrawals whose proof re-verified correctly",
				},
				[]string{},
			),
			invalidWithdrawals: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "invalid_withdrawals_total",
					Help:      "Withdrawals the portal accepted but a correct verifier rejects (P0)",
				},
				[]string{"reason", "txhash", "wdhash"},
			),
			unverifiable: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "unverifiable_withdrawals_total",
					Help:      "Withdrawals the monitor could not re-verify on first sight (parked for async retry)",
				},
				[]string{"reason", "txhash", "wdhash"},
			),
			pending: m.NewGauge(
				prometheus.GaugeOpts{
					Namespace: MetricsNamespace,
					Name:      "pending_withdrawals",
					Help:      "Prove events awaiting a terminal valid/invalid verdict (retried asynchronously)",
				},
			),
			oldestPendingSeconds: m.NewGauge(
				prometheus.GaugeOpts{
					Namespace: MetricsNamespace,
					Name:      "oldest_pending_seconds",
					Help:      "Age of the oldest prove event still awaiting a verdict; alert if it keeps growing",
				},
			),
		},
	}

	// The processor treats Config.StartBlock as already-processed and scans from
	// StartBlock+1. Decrement by one so --start.block is INCLUSIVE: the block the
	// user names is the first block scanned. StartBlock == 0 is left untouched so
	// the processor's "start from latest finalized block" behavior is preserved.
	procStartBlock := big.NewInt(int64(cfg.StartBlock))
	switch {
	case cfg.StartBlock > 0:
		procStartBlock = big.NewInt(int64(cfg.StartBlock) - 1)
	case !cfg.UseLatest && cfg.LookbackBlocks > 0:
		// No explicit start block: re-scan a lookback window below the finalized
		// head so events proven while the monitor was down are re-evaluated and
		// re-parked as pending if still unresolved. This is what makes the
		// in-memory pending set durable across restarts without external storage.
		start, err := lookbackStartBlock(ctx, l1Client, cfg.LookbackBlocks)
		if err != nil {
			return nil, err
		}
		procStartBlock = start
		log.Info("re-scanning lookback window on startup", "startBlock", start, "lookbackBlocks", cfg.LookbackBlocks)
	}

	proc, err := processor.NewBlockProcessor(
		m,
		log,
		cfg.L1NodeURL,
		nil,
		nil,
		mon.processLog,
		&processor.Config{
			StartBlock: procStartBlock,
			Interval:   cfg.PollingInterval,
			UseLatest:  cfg.UseLatest,
			// Scan logs via eth_getLogs filtered to the portal's
			// WithdrawalProvenExtension1 events — far cheaper than pulling every
			// block receipt, and works against nodes that don't serve batch
			// eth_getBlockReceipts for a block (e.g. anvil fork-base blocks).
			LogFilterAddresses: []common.Address{portalAddress},
			LogFilterTopics:    [][]common.Hash{{portalABI.Events["WithdrawalProvenExtension1"].ID}},
		},
	)
	if err != nil {
		return nil, err
	}

	mon.processor = proc
	return mon, nil
}

// Run starts the block processor and the async pending-retry loop until the
// context is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.processor.Stop()
	}()

	// Drive parked events to a terminal verdict independently of block processing.
	go m.retryPending(ctx)

	if err := m.processor.Start(); err != nil {
		m.log.Error("processor error", "err", err)
	}
}

// Close stops the processor and releases the L1 client.
func (m *Monitor) Close(ctx context.Context) error {
	m.processor.Stop()
	m.l1Client.Close()
	return nil
}

// lookbackStartBlock returns the processor start block for a startup re-scan:
// `lookback` blocks below the current finalized head (never negative). The extra
// -1 accounts for the processor scanning from StartBlock+1, so the first block
// actually scanned is exactly finalized-lookback.
func lookbackStartBlock(ctx context.Context, l1Client *ethclient.Client, lookback uint64) (*big.Int, error) {
	var header *types.Header
	if err := l1Client.Client().CallContext(ctx, &header, "eth_getBlockByNumber", "finalized", false); err != nil {
		return nil, fmt.Errorf("read finalized header for lookback: %w", err)
	}
	if header == nil {
		return nil, fmt.Errorf("finalized block is null (node may not support the \"finalized\" tag)")
	}
	start := new(big.Int).Sub(header.Number, new(big.Int).SetUint64(lookback+1))
	if start.Sign() < 0 {
		start = big.NewInt(0)
	}
	return start, nil
}

// computeWithdrawalStorageKey computes the storage slot of a withdrawal hash in the
// L2ToL1MessagePasser sentMessages mapping (slot 0):
// slot = keccak256(abi.encode(withdrawalHash, uint256(0))).
func computeWithdrawalStorageKey(withdrawalHash [32]byte) common.Hash {
	return crypto.Keccak256Hash(append(withdrawalHash[:], make([]byte, 32)...))
}

// callFrame is the shape returned by geth's callTracer (recursive).
type callFrame struct {
	Type  string      `json:"type"`
	To    string      `json:"to"`
	Input string      `json:"input"`
	Error string      `json:"error"`
	Calls []callFrame `json:"calls"`
}

// collectProveInputs walks the call tree (depth-first, including the root frame)
// and returns the input of every frame that calls the portal with the
// proveWithdrawalTransaction selector. A single transaction can prove many
// withdrawals (a batch relayer emits one internal call each), so we collect all
// of them and let the caller match by withdrawal hash.
//
// The trace is the ONLY reliable source of the proof arguments: the proof is
// ephemeral (never stored on-chain) and a wrapper contract can construct it
// internally so it never appears in the top-level tx calldata — but the internal
// CALL to the portal always carries it.
//
// Reverted frames are skipped entirely, together with their subtree: a frame
// that carries a callTracer `error` did not take effect (no state change, no
// event), so its calldata proves nothing. Without this, a relayer could prepend
// a deliberately reverting portal call carrying the prove selector to poison the
// candidate set before making the real successful prove call.
func (m *Monitor) collectProveInputs(frame *callFrame, out *[][]byte) {
	if frame.Error != "" {
		return
	}
	if strings.EqualFold(frame.To, m.portalAddress.Hex()) {
		input := common.FromHex(frame.Input)
		if len(input) >= 4 && bytes.Equal(input[:4], m.proveSelector[:]) {
			*out = append(*out, input)
		}
	}
	for i := range frame.Calls {
		m.collectProveInputs(&frame.Calls[i], out)
	}
}

// decodedProof holds the arguments recovered from a proveWithdrawalTransaction call.
type decodedProof struct {
	withdrawalTx     bindings.TypesWithdrawalTransaction
	disputeGameIndex *big.Int
	outputRootProof  bindings.TypesOutputRootProof
	withdrawalProof  [][]byte
}

// decodeProveInput ABI-decodes a recovered call input into the proof arguments.
func (m *Monitor) decodeProveInput(input []byte) (*decodedProof, error) {
	method := m.portalABI.Methods[proveMethodName]
	vals, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return nil, fmt.Errorf("unpack %s: %w", proveMethodName, err)
	}
	// proveWithdrawalTransaction(_tx, _disputeGameIndex, _outputRootProof, _withdrawalProof)
	if len(vals) != 4 {
		return nil, fmt.Errorf("expected 4 args, got %d", len(vals))
	}
	wdTx := *abi.ConvertType(vals[0], new(bindings.TypesWithdrawalTransaction)).(*bindings.TypesWithdrawalTransaction)
	gameIndex, ok := vals[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("disputeGameIndex has unexpected type %T", vals[1])
	}
	orp := *abi.ConvertType(vals[2], new(bindings.TypesOutputRootProof)).(*bindings.TypesOutputRootProof)
	proof, ok := vals[3].([][]byte)
	if !ok {
		return nil, fmt.Errorf("withdrawalProof has unexpected type %T", vals[3])
	}
	return &decodedProof{withdrawalTx: wdTx, disputeGameIndex: gameIndex, outputRootProof: orp, withdrawalProof: proof}, nil
}

// withdrawalHash recomputes hashWithdrawal(_tx) so a prove call can be matched to
// the WithdrawalProvenExtension1 event it produced.
func (m *Monitor) withdrawalHash(tx bindings.TypesWithdrawalTransaction) ([32]byte, error) {
	enc, err := m.wdTxArgs.Pack(tx.Nonce, tx.Sender, tx.Target, tx.Value, tx.GasLimit, tx.Data)
	if err != nil {
		return [32]byte{}, err
	}
	return crypto.Keccak256Hash(enc), nil
}

// verifyProof re-runs, byte-exact, the two checks the portal performs:
//
//	(a) keccak256(abi.encode(outputRootProof)) == game.rootClaim
//	(b) the withdrawal hash is included in the L2ToL1MessagePasser storage trie
//	    rooted at outputRootProof.messagePasserStorageRoot, with the sentinel value.
//
// It returns "" when the proof is valid, or a P0 reason when a correct verifier
// disagrees with the portal's acceptance.
func verifyProof(wdHash [32]byte, rootClaim [32]byte, dp *decodedProof) string {
	orp := dp.outputRootProof

	// (a) The submitted output-root proof must hash to the game's root claim.
	// abi.encode of four bytes32 is their concatenation.
	computed := crypto.Keccak256Hash(
		orp.Version[:],
		orp.StateRoot[:],
		orp.MessagePasserStorageRoot[:],
		orp.LatestBlockhash[:],
	)
	if computed != common.Hash(rootClaim) {
		return reasonBadOutputRootBinding
	}

	// (b) Independently verify storage inclusion against the message-passer storage
	// root. The storage trie is a secure trie, so its key is keccak256(slot).
	slot := computeWithdrawalStorageKey(wdHash)
	secureKey := crypto.Keccak256(slot[:])

	proofDB := memorydb.New()
	for _, node := range dp.withdrawalProof {
		if err := proofDB.Put(crypto.Keccak256(node), node); err != nil {
			return reasonBadWithdrawalProof
		}
	}

	value, err := trie.VerifyProof(common.Hash(orp.MessagePasserStorageRoot), secureKey, proofDB)
	// The portal proves inclusion of the sentinel value RLP(1) == 0x01.
	if err != nil || !bytes.Equal(value, []byte{0x01}) {
		return reasonBadWithdrawalProof
	}
	return ""
}

// Super-root game types. These commit to a super root spanning many chains, so
// the portal reads this chain's output root via rootClaimByChainId rather than
// rootClaim. Kept in sync with GameTypes.isSuperGame in the monorepo
// (src/dispute/lib/Types.sol).
const (
	gameTypeSuperCannon             uint32 = 4
	gameTypeSuperPermissionedCannon uint32 = 5
	gameTypeSuperAsteriscKona       uint32 = 7
	gameTypeSuperCannonKona         uint32 = 9
)

// isSuperGame mirrors OptimismPortal2's GameTypes.isSuperGame.
func isSuperGame(gameType uint32) bool {
	switch gameType {
	case gameTypeSuperCannon, gameTypeSuperPermissionedCannon, gameTypeSuperAsteriscKona, gameTypeSuperCannonKona:
		return true
	default:
		return false
	}
}

// gameInfo reads the dispute game a prove call targeted, by its factory index:
// the game proxy and the output-root claim to verify against. The index is
// decoded from the prove call itself (the _disputeGameIndex argument), so we
// resolve the exact game that call proved against — not whatever provenWithdrawals
// currently points to, which a later re-prove could have overwritten. gameAtIndex
// is immutable, so reading it at latest L1 state is correct.
//
// The claim is selected exactly as the portal does: super game types expose the
// per-chain output root via rootClaimByChainId(l2ChainId); legacy games use
// rootClaim. This is a plain data read — NOT a validity gate. The game address is
// returned even when the root-claim read fails, so it can still be logged.
func (m *Monitor) gameInfo(index *big.Int) (common.Address, [32]byte, error) {
	game, err := m.dgf.GameAtIndex(nil, index)
	if err != nil {
		return common.Address{}, [32]byte{}, err
	}
	if isSuperGame(game.GameType) {
		rootClaim, err := m.superGameRootClaim(game.Proxy)
		return game.Proxy, rootClaim, err
	}
	fdg, err := bindings.NewFaultDisputeGame(game.Proxy, m.l1Client)
	if err != nil {
		return game.Proxy, [32]byte{}, err
	}
	rootClaim, err := fdg.RootClaim(nil)
	return game.Proxy, rootClaim, err
}

// superGameRootClaim reads a super game's output root for this chain, matching
// the portal's rootClaimByChainId(systemConfig.l2ChainId()) branch.
func (m *Monitor) superGameRootClaim(proxy common.Address) ([32]byte, error) {
	chainID, err := m.getL2ChainID()
	if err != nil {
		return [32]byte{}, err
	}
	sg, err := bindings.NewSuperFaultDisputeGame(proxy, m.l1Client)
	if err != nil {
		return [32]byte{}, err
	}
	return sg.RootClaimByChainId(nil, chainID)
}

// getL2ChainID reads and caches the L2 chain id from SystemConfig. It is only
// consulted for super games, and the cached value never changes for a chain.
func (m *Monitor) getL2ChainID() (*big.Int, error) {
	m.l2ChainIDMu.Lock()
	defer m.l2ChainIDMu.Unlock()
	if m.l2ChainID != nil {
		return m.l2ChainID, nil
	}
	id, err := m.systemConfig.L2ChainId(nil)
	if err != nil {
		return nil, fmt.Errorf("read l2ChainId: %w", err)
	}
	m.l2ChainID = id
	return id, nil
}

// traceProveInputs traces the L1 transaction and returns every
// proveWithdrawalTransaction input reaching the portal (direct or via wrappers).
func (m *Monitor) traceProveInputs(ctx context.Context, txHash common.Hash) ([][]byte, error) {
	var frame callFrame
	err := m.l1Raw.CallContext(ctx, &frame, "debug_traceTransaction", txHash, map[string]interface{}{"tracer": "callTracer"})
	if err != nil {
		return nil, err
	}
	var inputs [][]byte
	m.collectProveInputs(&frame, &inputs)
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no %s call to portal found in trace", proveMethodName)
	}
	return inputs, nil
}

// parseWithdrawalEvent parses a WithdrawalProvenExtension1 event from a log.
func (m *Monitor) parseWithdrawalEvent(lg types.Log) (*bindings.OptimismPortal2WithdrawalProvenExtension1, error) {
	if lg.Address != m.portalAddress {
		return nil, nil
	}
	if len(lg.Topics) == 0 || lg.Topics[0] != m.portalABI.Events["WithdrawalProvenExtension1"].ID {
		return nil, nil
	}
	return m.portal.ParseWithdrawalProvenExtension1(lg)
}

// verdict classifies the outcome of evaluating a prove event.
type verdict int

const (
	verdictValid      verdict = iota // proof re-verified correctly
	verdictInvalid                   // portal accepted a proof a correct verifier rejects (P0)
	verdictUnresolved                // could not re-verify yet — parked for async retry
)

// assessment is the result of evaluating one prove event.
type assessment struct {
	v         verdict
	reason    string // set for invalid and unresolved
	gameProxy common.Address
	gameIndex *big.Int
}

// processLog handles one L1 log from the block processor. It NEVER returns a
// non-nil error: the shared processor retries forever on error, which would stall
// the whole monitor. Instead, an event that does not reach a terminal verdict is
// parked in the pending set and retried asynchronously (retryPending), so no prove
// event is ever silently dropped after a transient RPC failure.
func (m *Monitor) processLog(block *types.Block, lg types.Log, client *ethclient.Client) error {
	provenWithdrawal, err := m.parseWithdrawalEvent(lg)
	if err != nil {
		// A malformed event with the right topic is unexpected; surface it rather
		// than returning an error (which the shared processor would retry forever).
		m.log.Error("⚠️  could not parse WithdrawalProvenExtension1 event", "txHash", lg.TxHash.String(), "err", err)
		m.metrics.unverifiable.WithLabelValues(reasonDecodeError, lg.TxHash.String(), "").Inc()
		return nil
	}
	if provenWithdrawal == nil {
		return nil
	}
	m.evaluateAndApply(context.Background(), lg, provenWithdrawal.WithdrawalHash)
	return nil
}

// evaluateAndApply assesses one event and records the outcome. Shared by the
// processor (first sighting) and the async retry loop (subsequent attempts).
func (m *Monitor) evaluateAndApply(ctx context.Context, lg types.Log, wdHash [32]byte) {
	m.applyAssessment(m.assess(ctx, lg, wdHash), lg, wdHash)
}

// assess re-verifies one prove event from L1 only and returns its verdict. It has
// NO side effects (no metrics, no pending mutation), so it is safe to call from
// both the processor and the retry loop.
func (m *Monitor) assess(ctx context.Context, lg types.Log, wdHash [32]byte) assessment {
	dp, reason := m.recoverProof(ctx, lg, wdHash)
	if dp == nil {
		return assessment{v: verdictUnresolved, reason: reason}
	}
	gameProxy, rootClaim, err := m.gameInfo(dp.disputeGameIndex)
	if err != nil {
		m.log.Warn("could not read dispute game", "txHash", lg.TxHash.Hex(), "factoryGameIndex", dp.disputeGameIndex, "err", err)
		return assessment{v: verdictUnresolved, reason: reasonGameReadError, gameProxy: gameProxy, gameIndex: dp.disputeGameIndex}
	}
	if r := verifyProof(wdHash, rootClaim, dp); r != "" {
		return assessment{v: verdictInvalid, reason: r, gameProxy: gameProxy, gameIndex: dp.disputeGameIndex}
	}
	return assessment{v: verdictValid, gameProxy: gameProxy, gameIndex: dp.disputeGameIndex}
}

// applyAssessment records an assessment: emits metrics, logs, and updates the
// pending set — parking unresolved events, releasing terminal ones.
func (m *Monitor) applyAssessment(a assessment, lg types.Log, wdHash [32]byte) {
	txHashStr := lg.TxHash.Hex()
	wdHashStr := common.BytesToHash(wdHash[:]).Hex()
	switch a.v {
	case verdictValid:
		m.log.Info("✅ withdrawal proof re-verified", "txHash", txHashStr, "wdHash", wdHashStr,
			"disputeGame", a.gameProxy.Hex(), "factoryGameIndex", a.gameIndex)
		m.metrics.validWithdrawals.WithLabelValues().Inc()
		m.resolvePending(lg)
	case verdictInvalid:
		// The portal accepted a proof a correct verifier rejects. P0, regardless of
		// whether the dispute game is valid.
		m.log.Error("❌ INVALID WITHDRAWAL PROOF ACCEPTED BY PORTAL (P0)", "txHash", txHashStr, "wdHash", wdHashStr,
			"reason", a.reason, "disputeGame", a.gameProxy.Hex(), "factoryGameIndex", a.gameIndex)
		m.metrics.invalidWithdrawals.WithLabelValues(a.reason, txHashStr, wdHashStr).Inc()
		m.resolvePending(lg)
	case verdictUnresolved:
		if m.enqueuePending(lg, wdHash) {
			// Count each event once, at first sighting, to keep the counter bounded;
			// ongoing backlog is tracked by the pending/oldest gauges.
			m.log.Warn("⚠️  withdrawal not yet verifiable — parked for retry", "txHash", txHashStr, "wdHash", wdHashStr, "reason", a.reason)
			m.metrics.unverifiable.WithLabelValues(a.reason, txHashStr, wdHashStr).Inc()
		} else {
			m.log.Debug("withdrawal still pending", "txHash", txHashStr, "wdHash", wdHashStr, "reason", a.reason)
		}
	}
}

// enqueuePending parks an unresolved event for async retry. Returns true if the
// event was newly added (not already pending).
func (m *Monitor) enqueuePending(lg types.Log, wdHash [32]byte) bool {
	key := pendingKey(lg)
	m.pendingMu.Lock()
	_, exists := m.pending[key]
	if !exists {
		m.pending[key] = &pendingEvent{log: lg, wdHash: wdHash, firstSeen: time.Now()}
	}
	n := len(m.pending)
	m.pendingMu.Unlock()
	m.metrics.pending.Set(float64(n))
	return !exists
}

// resolvePending removes an event that has reached a terminal verdict.
func (m *Monitor) resolvePending(lg types.Log) {
	key := pendingKey(lg)
	m.pendingMu.Lock()
	delete(m.pending, key)
	n := len(m.pending)
	m.pendingMu.Unlock()
	m.metrics.pending.Set(float64(n))
}

// retryPending re-evaluates parked events on a fixed cadence until each reaches a
// terminal valid/invalid verdict. Runs until ctx is cancelled.
func (m *Monitor) retryPending(ctx context.Context) {
	ticker := time.NewTicker(pendingRetryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.retryPendingOnce(ctx)
		}
	}
}

// retryPendingOnce re-evaluates every currently-pending event once.
func (m *Monitor) retryPendingOnce(ctx context.Context) {
	m.pendingMu.Lock()
	snapshot := make([]*pendingEvent, 0, len(m.pending))
	for _, e := range m.pending {
		snapshot = append(snapshot, e)
	}
	m.pendingMu.Unlock()

	for _, e := range snapshot {
		e.attempts++
		m.evaluateAndApply(ctx, e.log, e.wdHash)
	}
	m.updatePendingGauges()
}

// updatePendingGauges refreshes the pending-count and oldest-age gauges. The
// oldest-age gauge is the alert signal: a value that keeps climbing means an event
// never reached a terminal verdict.
func (m *Monitor) updatePendingGauges() {
	m.pendingMu.Lock()
	n := len(m.pending)
	var oldest time.Time
	for _, e := range m.pending {
		if oldest.IsZero() || e.firstSeen.Before(oldest) {
			oldest = e.firstSeen
		}
	}
	m.pendingMu.Unlock()
	m.metrics.pending.Set(float64(n))
	if n == 0 {
		m.metrics.oldestPendingSeconds.Set(0)
	} else {
		m.metrics.oldestPendingSeconds.Set(time.Since(oldest).Seconds())
	}
}

// recoverProof traces the transaction and returns the decoded proof for THIS
// event. A single tx may prove the same withdrawal hash more than once (e.g. a
// batch that re-proves against a different game). Trace order equals on-chain
// call order equals event-emission order, so we collect every prove call whose
// recomputed hash matches this event and, when there is more than one, pair the
// event to its call by position (the k-th matching log is the k-th matching
// call). On success it returns the decoded proof and an empty reason; otherwise
// it returns (nil, reason) so the caller can park the event as pending. It has no
// metric side effects, so it is safe to call repeatedly from the retry loop.
func (m *Monitor) recoverProof(ctx context.Context, lg types.Log, wdHash [32]byte) (*decodedProof, string) {
	txHashStr := lg.TxHash.Hex()
	var inputs [][]byte
	var err error
	for attempt := 0; attempt < traceAttempts; attempt++ {
		inputs, err = m.traceProveInputs(ctx, lg.TxHash)
		if err == nil {
			break
		}
	}
	if err != nil {
		reason := reasonTraceUnavailable
		if strings.Contains(err.Error(), proveMethodName) {
			reason = reasonNoProveCallFound
		}
		m.log.Warn("could not recover proof from trace", "txHash", txHashStr, "reason", reason, "err", err)
		return nil, reason
	}

	// Decode every prove call and keep those matching this withdrawal hash, in
	// call (== event) order. A malformed candidate is skipped, not fatal: a
	// relayer could otherwise include a well-formed-selector-but-garbage-args
	// portal call to abort processing before the real prove call is reached. We
	// only give up if NO candidate matches this hash.
	var candidates []*decodedProof
	for _, input := range inputs {
		dp, derr := m.decodeProveInput(input)
		if derr != nil {
			m.log.Warn("skipping undecodable prove candidate", "txHash", txHashStr, "err", derr)
			continue
		}
		candidate, herr := m.withdrawalHash(dp.withdrawalTx)
		if herr != nil {
			m.log.Warn("skipping prove candidate with unhashable withdrawal tx", "txHash", txHashStr, "err", herr)
			continue
		}
		if candidate == wdHash {
			candidates = append(candidates, dp)
		}
	}

	switch len(candidates) {
	case 0:
		return nil, reasonNoProveCallFound
	case 1:
		return candidates[0], ""
	default:
		// The same hash is proven multiple times in this tx; disambiguate by the
		// event's position among the matching logs.
		ordinal, oerr := m.eventOrdinal(ctx, lg, wdHash)
		if oerr != nil || ordinal < 0 || ordinal >= len(candidates) {
			m.log.Warn("could not position-match duplicate prove call",
				"txHash", txHashStr, "ordinal", ordinal, "candidates", len(candidates), "err", oerr)
			return nil, reasonNoProveCallFound
		}
		return candidates[ordinal], ""
	}
}

// eventOrdinal returns the zero-based position of log lg among all
// WithdrawalProvenExtension1 logs for the same withdrawal hash in its
// transaction. It is only consulted when a single tx proves one hash more than
// once, to pair each event with the correct prove call by position.
func (m *Monitor) eventOrdinal(ctx context.Context, lg types.Log, wdHash [32]byte) (int, error) {
	receipt, err := m.l1Client.TransactionReceipt(ctx, lg.TxHash)
	if err != nil {
		return -1, err
	}
	ext1 := m.portalABI.Events["WithdrawalProvenExtension1"].ID
	target := common.BytesToHash(wdHash[:])
	ordinal := 0
	for _, rl := range receipt.Logs {
		if rl.Address != m.portalAddress || len(rl.Topics) < 2 || rl.Topics[0] != ext1 || rl.Topics[1] != target {
			continue
		}
		if rl.Index == lg.Index {
			return ordinal, nil
		}
		ordinal++
	}
	return -1, fmt.Errorf("event log %d not found in receipt for tx %s", lg.Index, lg.TxHash.Hex())
}
