package transaction_monitor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"

    "github.com/ethereum-optimism/monitorism/op-monitorism/processor"
)

const (
	MetricsNamespace = "tx_mon"
)

// CheckType represents the type of check to perform on an address
type CheckType string

const (
	// ExactMatchCheck verifies if an address matches exactly with a provided address
	ExactMatchCheck CheckType = "exact_match"
	// DisputeGameCheck verifies if an address is a valid dispute game created by the factory
	DisputeGameCheck CheckType = "dispute_game"
)

// CheckConfig represents a single check configuration
type CheckConfig struct {
	Type   CheckType              `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}

// WatchConfig represents the configuration for watching a specific address
type WatchConfig struct {
	Address common.Address `yaml:"address"`
	Filters []CheckConfig  `yaml:"filters"`
}

type Monitor struct {
	log          log.Logger
	client       *ethclient.Client
	watchConfigs map[common.Address]WatchConfig
	processor    *processor.BlockProcessor
	mu           sync.RWMutex

	// metrics
	transactions        *prometheus.CounterVec
	unauthorizedTx      *prometheus.CounterVec
	ethSpent           *prometheus.CounterVec
	blocksProcessed    *prometheus.CounterVec
	unexpectedRpcErrors *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	client, err := ethclient.Dial(cfg.NodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial node: %w", err)
	}

	mon := &Monitor{
		log:          log,
		client:       client,
		watchConfigs: make(map[common.Address]WatchConfig),

		transactions: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "transactions_total",
				Help:      "Total number of transactions",
			},
			[]string{"from", "to", "status"},
		),

		unauthorizedTx: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "unauthorized_transactions_total",
				Help:      "Number of transactions from unauthorized addresses",
			},
			[]string{"from"},
		),

		ethSpent: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "eth_spent_total",
				Help:      "Cumulative ETH spent by address",
			},
			[]string{"address"},
		),

		blocksProcessed: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "blocks_processed_total",
				Help:      "Number of blocks processed",
			},
			[]string{"number"},
		),

		unexpectedRpcErrors: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "unexpected_rpc_errors_total",
				Help:      "Number of unexpected RPC errors",
			},
			[]string{"section", "name"},
		),
	}

	// Initialize watchConfigs
	for _, config := range cfg.WatchConfigs {
		mon.watchConfigs[config.Address] = config
	}

	// Create the block processor
	processorCfg := &processor.Config{
		StartBlock: big.NewInt(int64(cfg.StartBlock)),
		Interval:   time.Duration(cfg.PollingInterval) * time.Second,
	}

	proc, err := processor.NewBlockProcessor(
		cfg.NodeUrl,
		mon.processTx,
		processorCfg,
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
		m.unexpectedRpcErrors.WithLabelValues("monitor", "processor").Inc()
	}
}

func (m *Monitor) processTx(block *types.Block, tx *types.Transaction, client *ethclient.Client) error {
	from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.unexpectedRpcErrors.WithLabelValues("monitor", "no tx sender").Inc()
		return fmt.Errorf("failed to find tx sender: %w", err)
	}

	if config, exists := m.watchConfigs[from]; exists {
		return m.handleTransaction(context.Background(), tx, config)
	}

	return nil
}

func (m *Monitor) handleTransaction(ctx context.Context, tx *types.Transaction, config WatchConfig) error {
	watchAddr, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.log.Error("failed to get transaction sender", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("monitor", "failed to get sender").Inc()
		return err
	}

	toAddr := *tx.To()

	// Track cumulative ETH spent
	weiValue := new(big.Float).SetInt(tx.Value())
	ethValue := new(big.Float).Quo(weiValue, big.NewFloat(1e18))
	ethFloat, _ := ethValue.Float64()
	m.ethSpent.WithLabelValues(watchAddr.String()).Add(ethFloat)

	// Check if sender is authorized
	if !m.isAddressAllowed(ctx, toAddr) {
		m.unauthorizedTx.WithLabelValues(watchAddr.String()).Inc()
		return nil
	}

	m.transactions.WithLabelValues(
		watchAddr.String(),
		toAddr.String(),
		"processed").Inc()

	m.blocksProcessed.WithLabelValues(fmt.Sprint(tx.Nonce())).Inc()

	return nil
}

func (m *Monitor) isAddressAllowed(ctx context.Context, addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, config := range m.watchConfigs {
		for _, filter := range config.Filters {
			checkFn, ok := AddressChecks[filter.Type]
			if !ok {
				m.log.Error("unknown check type", "type", filter.Type)
				continue
			}

			isValid, err := checkFn(ctx, m.client, addr, filter.Params)
			if err != nil {
				m.log.Error("check failed", "type", filter.Type, "addr", addr, "err", err)
				m.unexpectedRpcErrors.WithLabelValues("monitor", "check_failed").Inc()
				continue
			}

			if isValid {
				return true
			}
		}
	}
	return false
}

func (m *Monitor) Close(ctx context.Context) error {
	m.processor.Stop()
	m.client.Close()
	return nil
}
