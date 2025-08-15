package transaction_monitor

import (
	"context"
	"fmt"
	"math/big"

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

type Metrics struct {
	transactions   *prometheus.CounterVec
	unauthorizedTx *prometheus.CounterVec
	ethSpent       *prometheus.CounterVec
}

type Monitor struct {
	log          log.Logger
	client       *ethclient.Client
	watchConfigs map[common.Address]WatchConfig
	processor    *processor.BlockProcessor
	metrics      Metrics
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
		metrics: Metrics{
			transactions: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "transactions_total",
					Help:      "Total number of transactions",
				},
				[]string{"from"},
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
		},
	}

	// Initialize and validate watchConfigs
	for _, config := range cfg.WatchConfigs {
		for _, filter := range config.Filters {
			err := ParamValidations[filter.Type](filter.Params)
			if err != nil {
				return nil, fmt.Errorf("invalid parameters for check type %s: %w", filter.Type, err)
			}
		}
		mon.watchConfigs[config.Address] = config
	}

	// Create the block processor
	proc, err := processor.NewBlockProcessor(
		m,
		log,
		cfg.NodeUrl,
		mon.processTx,
		nil,
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

func (m *Monitor) processTx(block *types.Block, tx *types.Transaction, client *ethclient.Client) error {
	ctx := context.Background()

	// Grab the sender of the transaction.
	from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		return fmt.Errorf("failed to find tx sender: %w", err)
	}

	// Return if we're not watching this address.
	if _, exists := m.watchConfigs[from]; !exists {
		return nil
	}

	// If to is nil, use the created address.
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	} else {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			return fmt.Errorf("failed to get transaction receipt: %w", err)
		}
		to = receipt.ContractAddress
	}

	// Check if the recipient is authorized.
	allowed, err := m.isAddressAllowed(ctx, from, to)
	if err != nil {
		return fmt.Errorf("error checking address: %w", err)
	}

	// Track metrics.
	weiValue := new(big.Float).SetInt(tx.Value())
	ethValue := new(big.Float).Quo(weiValue, big.NewFloat(1e18))
	ethFloat, _ := ethValue.Float64()
	m.metrics.ethSpent.WithLabelValues(from.String()).Add(ethFloat)
	m.metrics.transactions.WithLabelValues(from.String()).Inc()
	if !allowed {
		m.metrics.unauthorizedTx.WithLabelValues(from.String()).Inc()
	}

	return nil
}

func (m *Monitor) isAddressAllowed(ctx context.Context, from common.Address, addr common.Address) (bool, error) {
	// Make sure there's a watch config for this address.
	watchConfig, ok := m.watchConfigs[from]
	if !ok {
		return false, fmt.Errorf("no watch config found for address %s", from.String())
	}

	// Check each filter.
	for _, filter := range watchConfig.Filters {
		checkFn, ok := AddressChecks[filter.Type]
		if !ok {
			return false, fmt.Errorf("unknown check type: %s", filter.Type)
		}

		isValid, err := checkFn(ctx, m.client, addr, filter.Params)
		if err != nil {
			return false, fmt.Errorf("error running check: %w", err)
		}

		if isValid {
			return true, nil
		}
	}

	return false, nil
}

func (m *Monitor) Close(ctx context.Context) error {
	m.processor.Stop()
	m.client.Close()
	return nil
}
