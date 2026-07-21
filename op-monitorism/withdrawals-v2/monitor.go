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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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

	// replaySafetyMarginSeconds extends startup replay beyond the portal's proof
	// maturity window. This covers finality/polling lag and makes a steady-state
	// restart rediscover every withdrawal that could still be awaiting finalization.
	replaySafetyMarginSeconds = uint64((24 * time.Hour) / time.Second)

	// assessTimeout bounds the RPC work (trace + game reads + receipt) for a single
	// event. ethclient.Dial sets no HTTP timeout, so without this a stuck
	// debug_traceTransaction could wedge the processor or the retry loop
	// indefinitely. Derived from the cancellable run context, so shutdown also
	// aborts an in-flight trace.
	assessTimeout = 30 * time.Second
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
	systemConfig  *bindings.SystemConfig
	portalAddress common.Address
	portalABI     *abi.ABI
	proveSelector [4]byte
	wdTxArgs      abi.Arguments
	processor     *processor.BlockProcessor
	metrics       Metrics

	// baseCtx is the cancellable run context; per-event work derives a timeout from
	// it so shutdown aborts in-flight RPCs. Set in Run; defaults to Background so
	// the monitor is usable (e.g. in tests) before Run is called.
	baseCtx context.Context

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
		"start_block_note", "inclusive (this block is scanned); 0 means replay proof maturity + safety margin at startup",
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

	// Bind SystemConfig so we can read l2ChainId when a super game is encountered
	// (mirrors the portal's super-game predicate). The chain id itself is read
	// lazily, so a SystemConfig without l2ChainId() never breaks legacy chains.
	systemConfigAddr, err := portal.SystemConfig(&bind.CallOpts{Context: ctx})
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
		systemConfig:  systemConfig,
		baseCtx:       context.Background(),
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
					Help:      "Withdrawals the portal accepted but a correct verifier rejects (P0). Offending tx/withdrawal hashes are in the logs.",
				},
				[]string{"reason"},
			),
			unverifiable: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "unverifiable_withdrawals_total",
					Help:      "Withdrawals the monitor could not re-verify on first sight (parked for async retry). Hashes are in the logs and the pending store.",
				},
				[]string{"reason"},
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
	var procStartBlock *big.Int
	switch {
	case cfg.StartBlock > 0:
		procStartBlock = big.NewInt(int64(cfg.StartBlock) - 1)
	default:
		// No explicit start block: replay at least the full on-chain withdrawal
		// maturity window plus a safety margin. This makes in-memory pending state
		// recoverable across restarts even when an event has been unresolved for far
		// longer than the optional block-count lookback.
		maturity, err := portal.ProofMaturityDelaySeconds(&bind.CallOpts{Context: ctx})
		if err != nil {
			return nil, fmt.Errorf("read proofMaturityDelaySeconds: %w", err)
		}
		if !maturity.IsUint64() || maturity.Uint64() > ^uint64(0)-replaySafetyMarginSeconds {
			return nil, fmt.Errorf("proof maturity delay does not fit uint64 seconds: %s", maturity)
		}
		replayWindowSeconds := maturity.Uint64() + replaySafetyMarginSeconds
		start, err := replayStartBlock(ctx, l1Client, cfg.UseLatest, cfg.LookbackBlocks, replayWindowSeconds)
		if err != nil {
			return nil, err
		}
		procStartBlock = start
		log.Info("replaying withdrawal maturity window on startup",
			"processorStartBlock", start,
			"proofMaturitySeconds", maturity,
			"safetyMarginSeconds", replaySafetyMarginSeconds,
			"configuredLookbackBlocks", cfg.LookbackBlocks,
		)
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
	// Adopt the run context as the base for per-event timeouts BEFORE starting any
	// goroutine, so shutdown cancels in-flight traces.
	m.baseCtx = ctx

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

type headerReader interface {
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

// replayStartBlock returns the already-processed cursor for steady-state startup.
// The next block scanned is the earlier of:
//   - the configured block-count lookback; and
//   - the first block inside proof-maturity + safety-margin seconds.
//
// The time-based floor is what makes pending-event recovery safe across restarts:
// it cannot silently shrink below the portal's withdrawal maturity window when L1
// block cadence changes. A cursor of -1 is intentional when replay begins at
// genesis; BlockProcessor then scans block 0 next.
func replayStartBlock(
	ctx context.Context,
	l1Client *ethclient.Client,
	useLatest bool,
	lookback uint64,
	replayWindowSeconds uint64,
) (*big.Int, error) {
	tag := "finalized"
	if useLatest {
		tag = "latest"
	}
	var header *types.Header
	if err := l1Client.Client().CallContext(ctx, &header, "eth_getBlockByNumber", tag, false); err != nil {
		return nil, fmt.Errorf("read %s header for replay: %w", tag, err)
	}
	if header == nil {
		return nil, fmt.Errorf("%s block is null (node may not support the %q tag)", tag, tag)
	}
	return replayStartBlockFromHead(ctx, l1Client, header, lookback, replayWindowSeconds)
}

func replayStartBlockFromHead(
	ctx context.Context,
	reader headerReader,
	head *types.Header,
	lookback uint64,
	replayWindowSeconds uint64,
) (*big.Int, error) {
	if head == nil || head.Number == nil || !head.Number.IsUint64() {
		return nil, fmt.Errorf("invalid replay head")
	}

	configuredFirst := new(big.Int).Sub(head.Number, new(big.Int).SetUint64(lookback))
	if configuredFirst.Sign() < 0 {
		configuredFirst.SetUint64(0)
	}
	configuredCursor := new(big.Int).Sub(configuredFirst, common.Big1)

	targetTimestamp := uint64(0)
	if head.Time > replayWindowSeconds {
		targetTimestamp = head.Time - replayWindowSeconds
	}
	timeFirst, err := firstBlockAtOrAfter(ctx, reader, head.Number.Uint64(), targetTimestamp)
	if err != nil {
		return nil, err
	}
	timeCursor := new(big.Int).Sub(new(big.Int).SetUint64(timeFirst), common.Big1)
	if timeCursor.Cmp(configuredCursor) < 0 {
		return timeCursor, nil
	}
	return configuredCursor, nil
}

// firstBlockAtOrAfter binary-searches monotonically increasing L1 block
// timestamps and returns the first block whose timestamp is at least target.
func firstBlockAtOrAfter(ctx context.Context, reader headerReader, headNumber, target uint64) (uint64, error) {
	lo, hi := uint64(0), headNumber
	for lo < hi {
		mid := lo + (hi-lo)/2
		header, err := reader.HeaderByNumber(ctx, new(big.Int).SetUint64(mid))
		if err != nil {
			return 0, fmt.Errorf("read block %d while locating replay window: %w", mid, err)
		}
		if header == nil {
			return 0, fmt.Errorf("block %d is null while locating replay window", mid)
		}
		if header.Time >= target {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo, nil
}

// computeWithdrawalStorageKey computes the storage slot of a withdrawal hash in the
// L2ToL1MessagePasser sentMessages mapping (slot 0):
// slot = keccak256(abi.encode(withdrawalHash, uint256(0))).
func computeWithdrawalStorageKey(withdrawalHash [32]byte) common.Hash {
	return crypto.Keccak256Hash(append(withdrawalHash[:], make([]byte, 32)...))
}

// callFrame is the shape returned by geth's callTracer (recursive).
type callFrame struct {
	Type   string      `json:"type"`
	To     string      `json:"to"`
	Input  string      `json:"input"`
	Output string      `json:"output"`
	Error  string      `json:"error"`
	Calls  []callFrame `json:"calls"`
}

type tracedProveCall struct {
	input []byte
	frame *callFrame
}

// collectProveCalls walks the call tree (depth-first, including the root frame)
// and returns every frame that calls the portal with the
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
// event), so its calldata proves nothing. Candidates must also be CALL frames:
// a DELEGATECALL to the portal runs in the caller's context and cannot emit the
// portal event this monitor is pairing with. Without these checks, a relayer
// could prepend a decoy carrying the prove selector to poison the candidate set
// before making the real successful prove call.
func (m *Monitor) collectProveCalls(frame *callFrame, out *[]tracedProveCall) {
	if frame.Error != "" {
		return
	}
	if frame.Type == "CALL" && strings.EqualFold(frame.To, m.portalAddress.Hex()) {
		input := common.FromHex(frame.Input)
		if len(input) >= 4 && bytes.Equal(input[:4], m.proveSelector[:]) {
			*out = append(*out, tracedProveCall{input: input, frame: frame})
		}
	}
	for i := range frame.Calls {
		m.collectProveCalls(&frame.Calls[i], out)
	}
}

// collectProveInputs is retained as a small test/helper surface for callers that
// only need calldata. Proof assessment uses collectProveCalls so it also retains
// the exact successful frame and can bind the historical factory/game lookup.
func (m *Monitor) collectProveInputs(frame *callFrame, out *[][]byte) {
	var calls []tracedProveCall
	m.collectProveCalls(frame, &calls)
	for _, call := range calls {
		*out = append(*out, call.input)
	}
}

// decodedProof holds the arguments recovered from a proveWithdrawalTransaction call.
type decodedProof struct {
	withdrawalTx     bindings.TypesWithdrawalTransaction
	disputeGameIndex *big.Int
	outputRootProof  bindings.TypesOutputRootProof
	withdrawalProof  [][]byte
}

type recoveredProof struct {
	proof     *decodedProof
	factory   common.Address
	gameType  uint32
	gameProxy common.Address
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

// gameRootClaim reads the output-root claim from the exact game returned by the
// historical prove call's gameAtIndex trace. The claim is selected exactly as the
// portal does: super game types expose the per-chain output root via
// rootClaimByChainId(l2ChainId); legacy games use rootClaim. This is a plain data
// read — NOT a validity gate.
func (m *Monitor) gameRootClaim(ctx context.Context, gameType uint32, gameProxy common.Address) ([32]byte, error) {
	opts := &bind.CallOpts{Context: ctx}
	if isSuperGame(gameType) {
		return m.superGameRootClaim(ctx, gameProxy)
	}
	fdg, err := bindings.NewFaultDisputeGame(gameProxy, m.l1Client)
	if err != nil {
		return [32]byte{}, err
	}
	return fdg.RootClaim(opts)
}

var gameAtIndexSelector = func() (selector [4]byte) {
	copy(selector[:], crypto.Keccak256([]byte("gameAtIndex(uint256)"))[:4])
	return selector
}()

// gameFromProveTrace recovers the exact factory identity and immutable game tuple
// used by this accepted prove call. Both values come from the successful
// gameAtIndex call frame, so a later AnchorStateRegistry/factory migration cannot
// make a running monitor or restarted backfill resolve the same index in the
// wrong factory.
func gameFromProveTrace(proveFrame *callFrame, index *big.Int) (common.Address, uint32, common.Address, error) {
	if proveFrame == nil {
		return common.Address{}, 0, common.Address{}, fmt.Errorf("missing prove call frame")
	}
	for i := range proveFrame.Calls {
		factory, gameType, proxy, found, err := findGameAtIndexCall(&proveFrame.Calls[i], index)
		if err != nil {
			return common.Address{}, 0, common.Address{}, err
		}
		if found {
			return factory, gameType, proxy, nil
		}
	}
	return common.Address{}, 0, common.Address{}, fmt.Errorf("successful gameAtIndex(%s) call not found in prove trace", index)
}

func findGameAtIndexCall(frame *callFrame, index *big.Int) (common.Address, uint32, common.Address, bool, error) {
	if frame.Error != "" {
		return common.Address{}, 0, common.Address{}, false, nil
	}
	input := common.FromHex(frame.Input)
	if len(input) == 36 && bytes.Equal(input[:4], gameAtIndexSelector[:]) && new(big.Int).SetBytes(input[4:]).Cmp(index) == 0 {
		output := common.FromHex(frame.Output)
		if len(output) < 96 {
			return common.Address{}, 0, common.Address{}, false, fmt.Errorf("gameAtIndex trace output is %d bytes, want at least 96", len(output))
		}
		gameTypeWord := new(big.Int).SetBytes(output[:32])
		if gameTypeWord.BitLen() > 32 {
			return common.Address{}, 0, common.Address{}, false, fmt.Errorf("gameAtIndex game type does not fit uint32: %s", gameTypeWord)
		}
		factory := common.HexToAddress(frame.To)
		proxy := common.BytesToAddress(output[64:96])
		if factory == (common.Address{}) || proxy == (common.Address{}) {
			return common.Address{}, 0, common.Address{}, false, fmt.Errorf("gameAtIndex trace returned zero factory or game proxy")
		}
		return factory, uint32(gameTypeWord.Uint64()), proxy, true, nil
	}
	for i := range frame.Calls {
		factory, gameType, proxy, found, err := findGameAtIndexCall(&frame.Calls[i], index)
		if err != nil || found {
			return factory, gameType, proxy, found, err
		}
	}
	return common.Address{}, 0, common.Address{}, false, nil
}

// superGameRootClaim reads a super game's output root for this chain, matching
// the portal's rootClaimByChainId(systemConfig.l2ChainId()) branch.
func (m *Monitor) superGameRootClaim(ctx context.Context, proxy common.Address) ([32]byte, error) {
	chainID, err := m.getL2ChainID(ctx)
	if err != nil {
		return [32]byte{}, err
	}
	sg, err := bindings.NewSuperFaultDisputeGame(proxy, m.l1Client)
	if err != nil {
		return [32]byte{}, err
	}
	return sg.RootClaimByChainId(&bind.CallOpts{Context: ctx}, chainID)
}

// getL2ChainID reads and caches the L2 chain id from SystemConfig. It is only
// consulted for super games, and the cached value never changes for a chain.
func (m *Monitor) getL2ChainID(ctx context.Context) (*big.Int, error) {
	m.l2ChainIDMu.Lock()
	defer m.l2ChainIDMu.Unlock()
	if m.l2ChainID != nil {
		return m.l2ChainID, nil
	}
	id, err := m.systemConfig.L2ChainId(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("read l2ChainId: %w", err)
	}
	m.l2ChainID = id
	return id, nil
}

// traceProveCalls traces the L1 transaction and returns every successful
// proveWithdrawalTransaction frame reaching the portal (direct or via wrappers).
// Retaining each frame is required to recover the historical factory/game lookup.
func (m *Monitor) traceProveCalls(ctx context.Context, txHash common.Hash) ([]tracedProveCall, error) {
	var frame callFrame
	err := m.l1Raw.CallContext(ctx, &frame, "debug_traceTransaction", txHash, map[string]interface{}{"tracer": "callTracer"})
	if err != nil {
		return nil, err
	}
	var calls []tracedProveCall
	m.collectProveCalls(&frame, &calls)
	if len(calls) == 0 {
		return nil, fmt.Errorf("no %s call to portal found in trace", proveMethodName)
	}
	return calls, nil
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
	factory   common.Address
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
		m.metrics.unverifiable.WithLabelValues(reasonDecodeError).Inc()
		return nil
	}
	if provenWithdrawal == nil {
		return nil
	}
	firstSeen := time.Now()
	if block != nil {
		firstSeen = time.Unix(int64(block.Time()), 0)
	}
	m.evaluateAndApply(lg, provenWithdrawal.WithdrawalHash, firstSeen)
	return nil
}

// evaluateAndApply assesses one event and records the outcome. Shared by the
// processor (first sighting) and the async retry loop (subsequent attempts).
func (m *Monitor) evaluateAndApply(lg types.Log, wdHash [32]byte, firstSeen time.Time) {
	m.applyAssessment(m.assess(lg, wdHash), lg, wdHash, firstSeen)
}

// assess re-verifies one prove event from L1 only and returns its verdict. It has
// NO side effects (no metrics, no pending mutation), so it is safe to call from
// both the processor and the retry loop. All RPC work runs under a per-event
// timeout derived from the run context, so a stuck trace cannot wedge the caller.
func (m *Monitor) assess(lg types.Log, wdHash [32]byte) assessment {
	ctx, cancel := context.WithTimeout(m.baseCtx, assessTimeout)
	defer cancel()

	recovered, reason := m.recoverProof(ctx, lg, wdHash)
	if recovered == nil {
		return assessment{v: verdictUnresolved, reason: reason}
	}
	rootClaim, err := m.gameRootClaim(ctx, recovered.gameType, recovered.gameProxy)
	if err != nil {
		m.log.Warn("could not read dispute game", "txHash", lg.TxHash.Hex(), "factory", recovered.factory,
			"factoryGameIndex", recovered.proof.disputeGameIndex, "gameProxy", recovered.gameProxy, "err", err)
		return assessment{v: verdictUnresolved, reason: reasonGameReadError, factory: recovered.factory,
			gameProxy: recovered.gameProxy, gameIndex: recovered.proof.disputeGameIndex}
	}
	if r := verifyProof(wdHash, rootClaim, recovered.proof); r != "" {
		return assessment{v: verdictInvalid, reason: r, factory: recovered.factory,
			gameProxy: recovered.gameProxy, gameIndex: recovered.proof.disputeGameIndex}
	}
	return assessment{v: verdictValid, factory: recovered.factory,
		gameProxy: recovered.gameProxy, gameIndex: recovered.proof.disputeGameIndex}
}

// applyAssessment records an assessment: emits metrics, logs, and updates the
// pending set — parking unresolved events, releasing terminal ones.
func (m *Monitor) applyAssessment(a assessment, lg types.Log, wdHash [32]byte, firstSeen time.Time) {
	txHashStr := lg.TxHash.Hex()
	wdHashStr := common.BytesToHash(wdHash[:]).Hex()
	switch a.v {
	case verdictValid:
		m.log.Info("✅ withdrawal proof re-verified", "txHash", txHashStr, "wdHash", wdHashStr,
			"factory", a.factory.Hex(), "disputeGame", a.gameProxy.Hex(), "factoryGameIndex", a.gameIndex)
		m.metrics.validWithdrawals.WithLabelValues().Inc()
		m.resolvePending(lg)
	case verdictInvalid:
		// The portal accepted a proof a correct verifier rejects. P0, regardless of
		// whether the dispute game is valid.
		m.log.Error("❌ INVALID WITHDRAWAL PROOF ACCEPTED BY PORTAL (P0)", "txHash", txHashStr, "wdHash", wdHashStr,
			"reason", a.reason, "factory", a.factory.Hex(), "disputeGame", a.gameProxy.Hex(), "factoryGameIndex", a.gameIndex)
		m.metrics.invalidWithdrawals.WithLabelValues(a.reason).Inc()
		m.resolvePending(lg)
	case verdictUnresolved:
		if m.enqueuePending(lg, wdHash, firstSeen) {
			// Count each event once, at first sighting, to keep the counter bounded;
			// ongoing backlog is tracked by the pending/oldest gauges.
			m.log.Warn("⚠️  withdrawal not yet verifiable — parked for retry", "txHash", txHashStr, "wdHash", wdHashStr, "reason", a.reason)
			m.metrics.unverifiable.WithLabelValues(a.reason).Inc()
		} else {
			m.log.Debug("withdrawal still pending", "txHash", txHashStr, "wdHash", wdHashStr, "reason", a.reason)
		}
	}
}

// enqueuePending parks an unresolved event for async retry. Returns true if the
// event was newly added (not already pending).
func (m *Monitor) enqueuePending(lg types.Log, wdHash [32]byte, firstSeen time.Time) bool {
	key := pendingKey(lg)
	m.pendingMu.Lock()
	_, exists := m.pending[key]
	if !exists {
		m.pending[key] = &pendingEvent{log: lg, wdHash: wdHash, firstSeen: firstSeen}
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

// retryPendingOnce re-evaluates every currently-pending event once, stopping early
// if the run context is cancelled.
func (m *Monitor) retryPendingOnce(ctx context.Context) {
	m.pendingMu.Lock()
	snapshot := make([]*pendingEvent, 0, len(m.pending))
	for _, e := range m.pending {
		snapshot = append(snapshot, e)
	}
	m.pendingMu.Unlock()

	for _, e := range snapshot {
		if ctx.Err() != nil {
			return
		}
		m.evaluateAndApply(e.log, e.wdHash, e.firstSeen)
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
		age := time.Since(oldest).Seconds()
		if age < 0 {
			age = 0
		}
		m.metrics.oldestPendingSeconds.Set(age)
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
func (m *Monitor) recoverProof(ctx context.Context, lg types.Log, wdHash [32]byte) (*recoveredProof, string) {
	txHashStr := lg.TxHash.Hex()
	var calls []tracedProveCall
	var err error
	for attempt := 0; attempt < traceAttempts; attempt++ {
		calls, err = m.traceProveCalls(ctx, lg.TxHash)
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
	type candidate struct {
		proof *decodedProof
		frame *callFrame
	}
	var candidates []candidate
	for _, call := range calls {
		dp, derr := m.decodeProveInput(call.input)
		if derr != nil {
			m.log.Warn("skipping undecodable prove candidate", "txHash", txHashStr, "err", derr)
			continue
		}
		candidateHash, herr := m.withdrawalHash(dp.withdrawalTx)
		if herr != nil {
			m.log.Warn("skipping prove candidate with unhashable withdrawal tx", "txHash", txHashStr, "err", herr)
			continue
		}
		if candidateHash == wdHash {
			candidates = append(candidates, candidate{proof: dp, frame: call.frame})
		}
	}

	var selected candidate
	switch len(candidates) {
	case 0:
		return nil, reasonNoProveCallFound
	case 1:
		selected = candidates[0]
	default:
		// The same hash is proven multiple times in this tx; disambiguate by the
		// event's position among the matching logs.
		ordinal, oerr := m.eventOrdinal(ctx, lg, wdHash)
		if oerr != nil || ordinal < 0 || ordinal >= len(candidates) {
			m.log.Warn("could not position-match duplicate prove call",
				"txHash", txHashStr, "ordinal", ordinal, "candidates", len(candidates), "err", oerr)
			return nil, reasonNoProveCallFound
		}
		selected = candidates[ordinal]
	}

	factory, gameType, gameProxy, err := gameFromProveTrace(selected.frame, selected.proof.disputeGameIndex)
	if err != nil {
		m.log.Warn("could not recover historical factory/game from prove trace",
			"txHash", txHashStr, "factoryGameIndex", selected.proof.disputeGameIndex, "err", err)
		return nil, reasonGameReadError
	}
	return &recoveredProof{
		proof: selected.proof, factory: factory, gameType: gameType, gameProxy: gameProxy,
	}, ""
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
