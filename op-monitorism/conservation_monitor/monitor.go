package conservation_monitor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/monitorism/op-monitorism/processor"
)

const (
	MetricsNamespace = "conservation_mon"
)

type Metrics struct {
	invariantHeld       *prometheus.CounterVec
	invariantViolations *prometheus.CounterVec
}

type Monitor struct {
	log       log.Logger
	client    *ethclient.Client
	processor *processor.BlockProcessor
	metrics   Metrics
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	client, err := ethclient.Dial(cfg.NodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial node: %w", err)
	}

	mon := &Monitor{
		log:    log,
		client: client,
		metrics: Metrics{
			invariantHeld: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "invariant_held",
					Help:      "Total blocks that hold the ETH conservation invariant",
				},
				[]string{"held"},
			),
			invariantViolations: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "invariant_violations",
					Help:      "Total violations of the ETH conservation invariant",
				},
				[]string{"violations"},
			),
		},
	}

	// Create the block processor
	proc, err := processor.NewBlockProcessor(
		m,
		log,
		cfg.NodeUrl,
		nil,
		mon.processBlock,
		nil,
		&processor.Config{
			StartBlock: big.NewInt(int64(cfg.StartBlock)),
			Interval:   cfg.PollingInterval,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create block processor: %w", err)
	}

	mon.processor = proc
	return mon, nil
}

func (m *Monitor) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.processor.Stop()
	}()

	if err := m.processor.Start(); err != nil {
		m.log.Error("processor error", "err", err)
	}
}

func (m *Monitor) processBlock(block *types.Block, client *ethclient.Client) error {
	// Call-trace the block to get every touched address.
	var trace []txTraceResult
	traceKind := "callTracer"
	err := client.Client().Call(&trace, "debug_traceBlockByHash", block.Hash(), tracers.TraceConfig{Tracer: &traceKind})
	if err != nil {
		return fmt.Errorf("failed to trace block %s: %w", block.Hash().Hex(), err)
	}

	held, err := m.checkInvariantHeld(block, trace)
	if err != nil {
		return fmt.Errorf("failed to check invariant: %w", err)
	}

	if held {
		m.metrics.invariantHeld.WithLabelValues("held").Inc()
	} else {
		m.metrics.invariantViolations.WithLabelValues("violations").Inc()
	}
	return nil
}

func (m *Monitor) Close(ctx context.Context) error {
	m.processor.Stop()
	m.client.Close()
	return nil
}

func (m *Monitor) checkInvariantHeld(block *types.Block, trace []txTraceResult) (bool, error) {
	// Compute the total amount of ETH minted in the block
	totalMinted := big.NewInt(0)
	for _, tx := range block.Transactions() {
		// Check if the transaction is a deposit
		if tx.IsDepositTx() && tx.Mint() != nil {
			totalMinted = totalMinted.Add(totalMinted, tx.Mint())
		}
	}

	// Extract all addresses touched by the block's execution
	addresses := extractAllTouchedAddresses(trace)

	// Extend `addresses` to include fee vaults. These are accessed outside of execution, so they're not
	// picked up in the call trace.
	addresses = append(addresses, []common.Address{
		predeploys.L1FeeVaultAddr,
		predeploys.SequencerFeeVaultAddr,
		predeploys.BaseFeeVaultAddr,
		predeploys.OperatorFeeVaultAddr,
	}...)

	ctx := context.Background()
	balancesParent, err := batchGetBalance(ctx, addresses, block.Number().Uint64()-1, m.client)
	if err != nil {
		return false, fmt.Errorf("failed to get parent block balances: %w", err)
	}
	balancesCurrent, err := batchGetBalance(ctx, addresses, block.Number().Uint64(), m.client)
	if err != nil {
		return false, fmt.Errorf("failed to get current block balances: %w", err)
	}

	totalBalancesParent := big.NewInt(0)
	totalBalancesCurrent := big.NewInt(0)
	for _, balance := range balancesParent {
		totalBalancesParent = totalBalancesParent.Add(totalBalancesParent, balance.ToInt())
	}
	for _, balance := range balancesCurrent {
		totalBalancesCurrent = totalBalancesCurrent.Add(totalBalancesCurrent, balance.ToInt())
	}

	// Check that the total ETH balance of all addresses in the block is conserved, relative to their balances
	// at the end of the parent block.
	invariantHeld := totalBalancesParent.Cmp(totalBalancesCurrent.Sub(totalBalancesCurrent, totalMinted)) >= 0
	if !invariantHeld {
		m.log.Warn(
			fmt.Sprintf("ETH conservation invariant violated. %d != %d - %d", totalBalancesParent, totalBalancesCurrent, totalMinted),
			"block", block.Number().Uint64(),
		)
	} else {
		m.log.Info(
			"ETH conservation invariant held",
			"block", block.Number().Uint64(),
			"num_touched_accounts", len(addresses),
			"deposit_mint_amount", totalMinted,
		)
	}

	return invariantHeld, nil
}

// batchGetBalance retrieves the balance of multiple addresses at a specific block number in a single batch call.
func batchGetBalance(
	ctx context.Context,
	addresses []common.Address,
	blockNum uint64,
	client *ethclient.Client,
) (map[common.Address]*hexutil.Big, error) {
	calls := make([]rpc.BatchElem, 0)
	balances := make(map[common.Address]*hexutil.Big)

	for _, addr := range addresses {
		result := new(hexutil.Big)

		calls = append(calls, rpc.BatchElem{
			Method: "eth_getBalance",
			Args:   []interface{}{addr, hexutil.Uint64(blockNum)},
			Result: &result,
		})

		balances[addr] = result
	}

	err := client.Client().BatchCallContext(ctx, calls)
	if err != nil {
		return nil, fmt.Errorf("failed to batch call getBalance: %w", err)
	}

	return balances, nil
}

// extractAllTouchedAddresses recursively extracts all addresses touched by a list of transaction traces. The returned
// list is deduplicated if multiple calls touch the same address.
func extractAllTouchedAddresses(trace []txTraceResult) []common.Address {
	addresses := make([]common.Address, 0)

	// Collect all addresses.
	for _, trace := range trace {
		touched := extractAllTouchedAddressesRecursive(trace.Result)
		addresses = append(addresses, touched...)
	}

	return deduplicate(addresses)
}

// extractAllTouchedAddressesRecursive recursively extracts all addresses touched by a single call trace and its
// internal calls.
func extractAllTouchedAddressesRecursive(txTrace callTrace) []common.Address {
	addresses := make([]common.Address, 0)

	// Add the from and to addresses of the call
	addresses = append(addresses, txTrace.From)
	if txTrace.To != nil {
		addresses = append(addresses, *txTrace.To)
	}

	for _, call := range txTrace.Calls {
		internalTouched := extractAllTouchedAddressesRecursive(call)
		addresses = append(addresses, internalTouched...)
	}

	return addresses
}

// deduplicate removes duplicate elements from a slice.
func deduplicate[T comparable](arr []T) []T {
	present := make(map[T]bool)
	list := []T{}
	for _, item := range arr {
		if _, value := present[item]; !value {
			present[item] = true
			list = append(list, item)
		}
	}
	return list
}

// ========================================================
// Copied from op-geth. Why not expose RPC return types? 😡
// ========================================================

// txTraceResult is the result of a single transaction trace.
type txTraceResult struct {
	TxHash common.Hash `json:"txHash"`           // transaction hash
	Result callTrace   `json:"result,omitempty"` // Trace results produced by the tracer
	Error  string      `json:"error,omitempty"`  // Trace failure produced by the tracer
}

// callLog is the result of LOG opCode
type callLog struct {
	Address  common.Address `json:"address"`
	Topics   []common.Hash  `json:"topics"`
	Data     hexutil.Bytes  `json:"data"`
	Position hexutil.Uint   `json:"position"`
}

// callTrace is the result of a callTracer run.
type callTrace struct {
	From         common.Address  `json:"from"`
	Gas          *hexutil.Uint64 `json:"gas"`
	GasUsed      *hexutil.Uint64 `json:"gasUsed"`
	To           *common.Address `json:"to,omitempty"`
	Input        hexutil.Bytes   `json:"input"`
	Output       hexutil.Bytes   `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
	RevertReason string          `json:"revertReason,omitempty"`
	Calls        []callTrace     `json:"calls,omitempty"`
	Logs         []callLog       `json:"logs,omitempty"`
	Value        *hexutil.Big    `json:"value,omitempty"`
	// Gencodec adds overridden fields at the end
	Type string `json:"type"`
}
