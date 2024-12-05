package txmonitor

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "tx_mon"
)

type Monitor struct {
	log           log.Logger
	client        *ethclient.Client
	watchConfigs  map[common.Address]WatchConfig // map of watch address to its config

	// metrics
	transactions         *prometheus.CounterVec
	unauthorizedTx      *prometheus.CounterVec
	thresholdExceededTx *prometheus.CounterVec
	unexpectedRpcErrors *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating transaction monitor")
	client, err := ethclient.Dial(cfg.L1NodeUrl)
	if err != nil {
		return nil, err
	}

	// Create map for quick lookups
	watchConfigs := make(map[common.Address]WatchConfig)
	for _, config := range cfg.WatchConfigs {
		watchConfigs[config.Address] = config
		log.Info("monitoring address", 
			"address", config.Address, 
			"allowlist_size", len(config.AllowList))
	}

	return &Monitor{
		log:          log,
		client:       client,
		watchConfigs: watchConfigs,

		transactions: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "transactions_total",
			Help:      "total number of transactions",
		}, []string{"watch_address", "from", "to", "status"}),

		unauthorizedTx: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unauthorized_transactions_total",
			Help:      "number of transactions from unauthorized addresses",
		}, []string{"watch_address", "from"}),

		thresholdExceededTx: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "threshold_exceeded_transactions_total",
			Help:      "number of transactions exceeding allowed threshold",
		}, []string{"watch_address", "from", "threshold"}),

		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpected_rpc_errors_total",
			Help:      "number of unexpected rpc errors",
		}, []string{"section", "name"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	blockNumber, err := m.client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to get latest block", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("monitor", "getBlockNumber").Inc()
		return
	}

	block, err := m.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
	if err != nil {
		m.log.Error("failed to get block", "number", blockNumber, "err", err)
		m.unexpectedRpcErrors.WithLabelValues("monitor", "getBlock").Inc()
		return
	}

	for _, tx := range block.Transactions() {
		if tx.To() == nil {
			continue
		}
		
		// Check if this transaction is to any of our watch addresses
		if config, exists := m.watchConfigs[*tx.To()]; exists {
			m.processTx(tx, config)
		}
	}
}

func (m *Monitor) processTx(tx *types.Transaction, config WatchConfig) {
	from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.log.Error("failed to get transaction sender", "err", err)
		return
	}

	watchAddr := *tx.To()

	// Log all transactions
	m.transactions.WithLabelValues(
		watchAddr.String(),
		from.String(),
		tx.To().String(),
		"processed").Inc()

	// Check if sender is in allowlist
	isAuthorized := false
	for _, addr := range config.AllowList {
		if addr == from {
			isAuthorized = true
			break
		}
	}

	if !isAuthorized {
		m.log.Warn("unauthorized transaction detected", 
			"watch_address", watchAddr,
			"from", from, 
			"value", tx.Value())
		m.unauthorizedTx.WithLabelValues(
			watchAddr.String(),
			from.String()).Inc()
		return
	}

	// Check threshold
	threshold := config.Thresholds[from.Hex()]
	if tx.Value().Cmp(threshold) > 0 {
		m.log.Warn("transaction exceeds threshold",
			"watch_address", watchAddr,
			"from", from,
			"threshold", threshold,
			"value", tx.Value())
		m.thresholdExceededTx.WithLabelValues(
			watchAddr.String(),
			from.String(),
			threshold.String()).Inc()
	}
}

func (m *Monitor) Close(_ context.Context) error {
	m.client.Close()
	return nil
}
