package transaction_monitor

import (
	"context"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "tx_mon"
)

type CheckType string

const (
	ExactMatchCheck  CheckType = "exact_match"
	DisputeGameCheck CheckType = "dispute_game"
)

type CheckConfig struct {
	Type   CheckType              `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}

type WatchConfig struct {
	Address    common.Address      `yaml:"address"`
	Filters    []CheckConfig       `yaml:"filters"`
	Thresholds map[string]*big.Int `yaml:"thresholds"`
}

type DisputeGameWatcher struct {
	factory      common.Address
	allowedGames map[common.Address]bool
	mu           sync.RWMutex
	lastBlock    uint64
}

type Monitor struct {
	log             log.Logger
	client          *ethclient.Client
	watchConfigs    map[common.Address]WatchConfig
	allowedAddrs    map[common.Address]bool
	factoryWatchers map[common.Address]*DisputeGameWatcher
	startBlock      uint64
	mu              sync.RWMutex

	// metrics
	transactions        *prometheus.CounterVec
	unauthorizedTx      *prometheus.CounterVec
	thresholdExceededTx *prometheus.CounterVec
	ethSpent            *prometheus.CounterVec
	unexpectedRpcErrors *prometheus.CounterVec
}

var disputeGameFactoryABI = `[{
	"anonymous": false,
	"inputs": [
		{
			"indexed": true,
			"name": "disputeProxy",
			"type": "address"
		},
		{
			"indexed": true,
			"name": "gameType",
			"type": "uint8"
		},
		{
			"indexed": true,
			"name": "rootClaim",
			"type": "bytes32"
		}
	],
	"name": "DisputeGameCreated",
	"type": "event"
}]`

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	client, err := ethclient.Dial(cfg.L1NodeUrl)
	if err != nil {
		return nil, err
	}

	mon := &Monitor{
		log:             log,
		client:          client,
		watchConfigs:    make(map[common.Address]WatchConfig),
		allowedAddrs:    make(map[common.Address]bool),
		factoryWatchers: make(map[common.Address]*DisputeGameWatcher),
		startBlock:      cfg.StartBlock,

		transactions: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "transactions_total",
				Help:      "Total number of transactions",
			},
			[]string{"watch_address", "from", "to", "status"},
		),

		unauthorizedTx: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "unauthorized_transactions_total",
				Help:      "Number of transactions from unauthorized addresses",
			},
			[]string{"watch_address", "from"},
		),

		thresholdExceededTx: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "threshold_exceeded_transactions_total",
				Help:      "Number of transactions exceeding allowed threshold",
			},
			[]string{"watch_address", "from", "threshold"},
		),

		ethSpent: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "eth_spent_total",
				Help:      "Cumulative ETH spent by address",
			},
			[]string{"address"},
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

	// Initialize filters and watchers
	for _, config := range cfg.WatchConfigs {
		mon.watchConfigs[config.Address] = config

		for _, filter := range config.Filters {
			switch filter.Type {
			case ExactMatchCheck:
				if match, ok := filter.Params["match"].(string); ok {
					mon.allowedAddrs[common.HexToAddress(match)] = true
				}
			case DisputeGameCheck:
				if factory, ok := filter.Params["disputeGameFactory"].(string); ok {
					factoryAddr := common.HexToAddress(factory)
					watcher := &DisputeGameWatcher{
						factory:      factoryAddr,
						allowedGames: make(map[common.Address]bool),
					}
					mon.factoryWatchers[factoryAddr] = watcher
				}
			}
		}
	}

	return mon, nil
}

func (m *Monitor) Run(ctx context.Context) {
	currentBlock := m.startBlock
	var err error

	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitor stopping")
			return
		default:
			// Get current block if not specified
			if currentBlock == 0 {
				currentBlock, err = m.client.BlockNumber(ctx)
				if err != nil {
					m.log.Error("failed to get block number", "err", err)
					continue
				}
			}

			// Update dispute game watchers
			for _, watcher := range m.factoryWatchers {
				if err := m.updateDisputeGames(ctx, watcher, currentBlock); err != nil {
					m.log.Error("failed to update dispute games", "err", err)
					continue
				}
			}

			block, err := m.client.BlockByNumber(ctx, big.NewInt(int64(currentBlock)))
			if err != nil {
				m.log.Error("failed to get block", "number", currentBlock, "err", err)
				m.unexpectedRpcErrors.WithLabelValues("monitor", "getBlock").Inc()
				continue
			}

			for _, tx := range block.Transactions() {
				if tx.To() == nil {
					continue
				}

				if config, exists := m.watchConfigs[*tx.To()]; exists {
					m.processTx(tx, config)
				}
			}

			currentBlock++
		}
	}
}

func (m *Monitor) updateDisputeGames(ctx context.Context, watcher *DisputeGameWatcher, currentBlock uint64) error {
	contractABI, err := abi.JSON(strings.NewReader(disputeGameFactoryABI))
	if err != nil {
		return err
	}

	// Query logs since last check
	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(watcher.lastBlock),
		ToBlock:   new(big.Int).SetUint64(currentBlock),
		Addresses: []common.Address{watcher.factory},
	}

	logs, err := m.client.FilterLogs(ctx, query)
	if err != nil {
		return err
	}

	for _, vLog := range logs {
		event := struct {
			DisputeProxy common.Address
		}{}
		err := contractABI.UnpackIntoInterface(&event, "DisputeGameCreated", vLog.Data)
		if err != nil {
			m.log.Error("failed to unpack event", "err", err)
			continue
		}

		watcher.mu.Lock()
		watcher.allowedGames[event.DisputeProxy] = true
		watcher.mu.Unlock()
	}

	watcher.lastBlock = currentBlock
	return nil
}

func (m *Monitor) processTx(tx *types.Transaction, config WatchConfig) {
	from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.log.Error("failed to get transaction sender", "err", err)
		return
	}

	watchAddr := *tx.To()

	// Track cumulative ETH spent
	weiValue := new(big.Float).SetInt(tx.Value())
	ethValue := new(big.Float).Quo(weiValue, big.NewFloat(1e18))
	ethFloat, _ := ethValue.Float64()
	m.ethSpent.WithLabelValues(from.String()).Add(ethFloat)

	// Check if sender is authorized
	if !m.isAddressAllowed(from) {
		m.unauthorizedTx.WithLabelValues(
			watchAddr.String(),
			from.String()).Inc()
		return
	}

	m.transactions.WithLabelValues(
		watchAddr.String(),
		from.String(),
		tx.To().String(),
		"processed").Inc()

	// Check threshold
	if threshold, ok := config.Thresholds[from.Hex()]; ok && tx.Value().Cmp(threshold) > 0 {
		m.thresholdExceededTx.WithLabelValues(
			watchAddr.String(),
			from.String(),
			threshold.String()).Inc()
	}
}

func (m *Monitor) isAddressAllowed(addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.allowedAddrs[addr] {
		return true
	}

	// Check if address is an allowed dispute game
	for _, watcher := range m.factoryWatchers {
		if watcher.isGameAllowed(addr) {
			return true
		}
	}

	return false
}

func (w *DisputeGameWatcher) isGameAllowed(addr common.Address) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.allowedGames[addr]
}

func (m *Monitor) Close(ctx context.Context) error {
	m.client.Close()
	return nil
}
