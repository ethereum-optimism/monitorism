package transaction_monitor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
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
	startBlock   uint64
	mu           sync.RWMutex

	// metrics
	transactions        *prometheus.CounterVec
	unauthorizedTx      *prometheus.CounterVec
	ethSpent            *prometheus.CounterVec
	blocksProcessed     *prometheus.CounterVec
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
		startBlock:   cfg.StartBlock,

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

	return mon, nil
}

func (m *Monitor) Run(ctx context.Context) {
	// Initialize starting block
	currentBlock := m.startBlock
	if currentBlock == 0 {
		var err error
		currentBlock, err = m.client.BlockNumber(ctx)
		if err != nil {
			m.log.Error("failed to get initial block number", "err", err)
			m.unexpectedRpcErrors.WithLabelValues("monitor", "blockNumber").Inc()
			return
		}
		m.log.Info("starting from latest block", "number", currentBlock)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitor stopping")
			return
		case <-ticker.C:
			// Get latest block number
			latestBlock, err := m.client.BlockNumber(ctx)
			if err != nil {
				m.unexpectedRpcErrors.WithLabelValues("monitor", "blockNumber").Inc()
				continue
			}

			// Process blocks until we catch up to head
			for currentBlock <= latestBlock {
				if err := m.processBlock(ctx, currentBlock); err != nil {
					m.log.Error("failed to process block", "number", currentBlock, "err", err)
					if !errors.Is(err, ethereum.NotFound) {
						m.unexpectedRpcErrors.WithLabelValues("monitor", "getBlock").Inc()
					}
					break
				}
				currentBlock++
			}
		}
	}
}

func (m *Monitor) processBlock(ctx context.Context, blockNum uint64) error {
	block, err := m.client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	for _, tx := range block.Transactions() {
		from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
		if err != nil {
			m.unexpectedRpcErrors.WithLabelValues("monitor", "no tx sender").Inc()
			return fmt.Errorf("failed to find tx sender: %w", err)
		}

		if config, exists := m.watchConfigs[from]; exists {
			m.processTx(ctx, tx, config)
		}
	}

	m.blocksProcessed.WithLabelValues(fmt.Sprint(blockNum)).Inc()

	return nil
}

func (m *Monitor) isAddressAllowed(ctx context.Context, addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Iterate through all watch configs and their filters
	for _, config := range m.watchConfigs {
		for _, filter := range config.Filters {
			// Get the check function for this filter type
			checkFn, ok := AddressChecks[filter.Type]
			if !ok {
				m.log.Error("unknown check type", "type", filter.Type)
				continue
			}

			// Run the check
			isValid, err := checkFn(ctx, m.client, addr, filter.Params)
			if err != nil {
				m.log.Error("check failed", "type", filter.Type, "addr", addr, "err", err)
				m.unexpectedRpcErrors.WithLabelValues("monitor", "check_failed").Inc()
				continue
			}

			// If any check passes, the address is allowed
			if isValid {
				return true
			}
		}
	}
	return false
}

func (m *Monitor) processTx(ctx context.Context, tx *types.Transaction, config WatchConfig) {
	watchAddr, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.log.Error("failed to get transaction sender", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("monitor", "failed to get sender").Inc()
		return
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
		return
	}

	m.transactions.WithLabelValues(
		watchAddr.String(),
		toAddr.String(),
		"processed").Inc()
}

func (m *Monitor) Close(ctx context.Context) error {
	m.client.Close()
	return nil
}
