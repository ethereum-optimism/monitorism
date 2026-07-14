package withdrawalsv2

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

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

	// traceAttempts bounds transient-error retries before we give up on a log and
	// advance. We never block the processor indefinitely on a single withdrawal.
	traceAttempts = 3

	// logSeparator visually delimits each withdrawal verdict in the logs.
	logSeparator = "===================================================================="
)

// Metrics holds the Prometheus counters for the monitor.
type Metrics struct {
	validWithdrawals   *prometheus.CounterVec
	invalidWithdrawals *prometheus.CounterVec
	unverifiable       *prometheus.CounterVec
}

// Monitor re-verifies, from L1 only, that every withdrawal the OptimismPortal2
// accepted carries a proof a correct verifier also accepts.
type Monitor struct {
	log           log.Logger
	l1Client      *ethclient.Client
	l1Raw         *rpc.Client
	portal        *bindings.OptimismPortal2
	portalAddress common.Address
	portalABI     *abi.ABI
	proveSelector [4]byte
	wdTxArgs      abi.Arguments
	processor     *processor.BlockProcessor
	metrics       Metrics
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
func logStartupConfig(log log.Logger, cfg CLIConfig) {
	envVars := []string{
		"WITHDRAWALS_V2_MON_L1_NODE_URL",
		"WITHDRAWALS_V2_MON_START_BLOCK",
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
		"l1_node_url", cfg.L1NodeURL,
		"optimism_portal", cfg.OptimismPortalAddress,
		"start_block", cfg.StartBlock,
		"start_block_note", "inclusive (this block is scanned); 0 means start from latest finalized block",
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
					Help:      "Withdrawals the monitor could not re-verify (blind spot, needs operator attention)",
				},
				[]string{"reason", "txhash", "wdhash"},
			),
		},
	}

	// The processor treats Config.StartBlock as already-processed and scans from
	// StartBlock+1. Decrement by one so --start.block is INCLUSIVE: the block the
	// user names is the first block scanned. StartBlock == 0 is left untouched so
	// the processor's "start from latest finalized block" behavior is preserved.
	procStartBlock := big.NewInt(int64(cfg.StartBlock))
	if cfg.StartBlock > 0 {
		procStartBlock = big.NewInt(int64(cfg.StartBlock) - 1)
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

// Run starts the block processor until the context is cancelled.
func (m *Monitor) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.processor.Stop()
	}()

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
func (m *Monitor) collectProveInputs(frame *callFrame, out *[][]byte) {
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

// gameInfo reads the dispute game the withdrawal was proven against: its proxy
// address and root claim. This is a plain data read of current L1 state — NOT a
// validity gate. The game address is returned even when the root-claim read fails,
// so it can still be logged.
func (m *Monitor) gameInfo(wdHash [32]byte, proofSubmitter common.Address) (common.Address, [32]byte, error) {
	proven, err := m.portal.ProvenWithdrawals(nil, wdHash, proofSubmitter)
	if err != nil {
		return common.Address{}, [32]byte{}, err
	}
	game, err := bindings.NewFaultDisputeGame(proven.DisputeGameProxy, m.l1Client)
	if err != nil {
		return proven.DisputeGameProxy, [32]byte{}, err
	}
	rootClaim, err := game.RootClaim(nil)
	return proven.DisputeGameProxy, rootClaim, err
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

// processLog handles one L1 log. It NEVER returns a non-nil error: the shared
// processor retries forever on error (processor.processLogWithRetry), so a single
// un-traceable withdrawal must not stall the whole monitor. Blind spots are
// surfaced via the unverifiable metric instead.
func (m *Monitor) processLog(block *types.Block, lg types.Log, client *ethclient.Client) error {
	ctx := context.Background()

	provenWithdrawal, err := m.parseWithdrawalEvent(lg)
	if err != nil {
		// A malformed event with the right topic is unexpected; treat it as an
		// unverifiable blind spot rather than returning an error (which the shared
		// processor would retry forever, stalling the monitor).
		m.log.Error("⚠️  could not parse WithdrawalProvenExtension1 event (UNVERIFIABLE)", "txHash", lg.TxHash.String(), "err", err)
		m.metrics.unverifiable.WithLabelValues(reasonDecodeError, lg.TxHash.String(), "").Inc()
		return nil
	}
	if provenWithdrawal == nil {
		return nil
	}

	wdHash := provenWithdrawal.WithdrawalHash
	txHash := lg.TxHash
	txHashStr := txHash.String()
	wdHashStr := common.BytesToHash(wdHash[:]).String()
	m.log.Info("processing withdrawal proven event", "txHash", txHashStr, "wdHash", wdHashStr)
	// Every withdrawal we start processing ends with a visual separator.
	defer m.log.Info(logSeparator)

	dp := m.recoverProof(ctx, txHash, wdHash, txHashStr, wdHashStr)
	if dp == nil {
		return nil
	}

	gameProxy, rootClaim, err := m.gameInfo(wdHash, provenWithdrawal.ProofSubmitter)
	if err != nil {
		m.log.Error("⚠️  could not read dispute game (UNVERIFIABLE)",
			"txHash", txHashStr, "wdHash", wdHashStr,
			"disputeGame", gameProxy.Hex(), "factoryGameIndex", dp.disputeGameIndex, "err", err)
		m.metrics.unverifiable.WithLabelValues(reasonGameReadError, txHashStr, wdHashStr).Inc()
		return nil
	}

	if reason := verifyProof(wdHash, rootClaim, dp); reason != "" {
		// The portal accepted a proof a correct verifier rejects. This is P0 and
		// holds regardless of whether the dispute game is valid.
		m.log.Error("❌ INVALID WITHDRAWAL PROOF ACCEPTED BY PORTAL (P0)",
			"txHash", txHashStr, "wdHash", wdHashStr, "reason", reason,
			"disputeGame", gameProxy.Hex(), "factoryGameIndex", dp.disputeGameIndex)
		m.metrics.invalidWithdrawals.WithLabelValues(reason, txHashStr, wdHashStr).Inc()
		return nil
	}

	m.log.Info("✅ withdrawal proof re-verified",
		"txHash", txHashStr, "wdHash", wdHashStr,
		"disputeGame", gameProxy.Hex(), "factoryGameIndex", dp.disputeGameIndex)
	m.metrics.validWithdrawals.WithLabelValues().Inc()
	return nil
}

// recoverProof traces the transaction, then finds and decodes the prove call whose
// withdrawal hash matches the event's. It emits an unverifiable metric and returns
// nil if it cannot (never propagates an error to the processor).
func (m *Monitor) recoverProof(ctx context.Context, txHash common.Hash, wdHash [32]byte, txHashStr, wdHashStr string) *decodedProof {
	var inputs [][]byte
	var err error
	for attempt := 0; attempt < traceAttempts; attempt++ {
		inputs, err = m.traceProveInputs(ctx, txHash)
		if err == nil {
			break
		}
	}
	if err != nil {
		reason := reasonTraceUnavailable
		if strings.Contains(err.Error(), proveMethodName) {
			reason = reasonNoProveCallFound
		}
		m.log.Error("⚠️  could not recover proof from trace (UNVERIFIABLE)", "txHash", txHashStr, "wdHash", wdHashStr, "reason", reason, "err", err)
		m.metrics.unverifiable.WithLabelValues(reason, txHashStr, wdHashStr).Inc()
		return nil
	}

	// A batch tx contains one prove call per withdrawal; select the one whose
	// recomputed withdrawal hash matches this event.
	for _, input := range inputs {
		dp, derr := m.decodeProveInput(input)
		if derr != nil {
			m.log.Error("⚠️  could not decode proof input (UNVERIFIABLE)", "txHash", txHashStr, "wdHash", wdHashStr, "err", derr)
			m.metrics.unverifiable.WithLabelValues(reasonDecodeError, txHashStr, wdHashStr).Inc()
			return nil
		}
		candidate, herr := m.withdrawalHash(dp.withdrawalTx)
		if herr != nil {
			m.log.Error("⚠️  could not recompute withdrawal hash (UNVERIFIABLE)", "txHash", txHashStr, "wdHash", wdHashStr, "err", herr)
			m.metrics.unverifiable.WithLabelValues(reasonDecodeError, txHashStr, wdHashStr).Inc()
			return nil
		}
		if candidate == wdHash {
			return dp
		}
	}

	m.log.Error("⚠️  no prove call in tx matched the withdrawal hash (UNVERIFIABLE)", "txHash", txHashStr, "wdHash", wdHashStr)
	m.metrics.unverifiable.WithLabelValues(reasonNoProveCallFound, txHashStr, wdHashStr).Inc()
	return nil
}
