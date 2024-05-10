package withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "two_step_monitor"

	// event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);
	WithdrawalProvenEventABI = "WithdrawalProven(bytes32,address,address)"
)

var (
	WithdrawalProvenEventABIHash = crypto.Keccak256Hash([]byte(WithdrawalProvenEventABI))
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client
	l2Client *ethclient.Client

	optimismPortalAddress common.Address
	optimismPortal        *bindings.OptimismPortalCaller
	l2ToL1MP              *bindings.L2ToL1MessagePasserCaller

	maxBlockRange uint64
	nextL1Height  uint64

	// metrics
	highestBlockNumber     *prometheus.GaugeVec
	isDetectingForgeries   prometheus.Gauge
	withdrawalsValidated   prometheus.Counter
	nodeConnectionFailures *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating withdrawals monitor...")

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	l2Client, err := ethclient.Dial(cfg.L2NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	optimismPortal, err := bindings.NewOptimismPortalCaller(cfg.OptimismPortalAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}
	l2ToL1MP, err := bindings.NewL2ToL1MessagePasserCaller(predeploys.L2ToL1MessagePasserAddr, l2Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	return &Monitor{
		log: log,

		l1Client: l1Client,
		l2Client: l2Client,

		optimismPortalAddress: cfg.OptimismPortalAddress,
		optimismPortal:        optimismPortal,
		l2ToL1MP:              l2ToL1MP,

		maxBlockRange: cfg.EventBlockRange,
		nextL1Height:  cfg.StartingL1BlockHeight,

		/** Metrics **/
		isDetectingForgeries: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "isDetectingForgeries",
			Help:      "0 if state is ok. 1 if forged withdrawals are detected",
		}),
		withdrawalsValidated: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "withdrawalsValidated",
			Help:      "number of withdrawals succesfully validated",
		}),
		highestBlockNumber: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestBlockNumber",
			Help:      "observed l1 heights (checked and known)",
		}, []string{"type"}),
		nodeConnectionFailures: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "nodeConnectionFailures",
			Help:      "number of times node connection has failed",
		}, []string{"layer", "section"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	latestL1Height, err := m.l1Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "blockNumber").Inc()
		return
	}

	m.highestBlockNumber.WithLabelValues("known").Set(float64(latestL1Height))

	fromBlockNumber := m.nextL1Height
	if fromBlockNumber > latestL1Height {
		m.log.Info("no new blocks", "next_height", fromBlockNumber, "latest_height", latestL1Height)
		return
	}

	toBlockNumber := latestL1Height
	if toBlockNumber-fromBlockNumber > m.maxBlockRange {
		toBlockNumber = fromBlockNumber + m.maxBlockRange
	}

	m.log.Info("querying block range", "from_height", fromBlockNumber, "to_height", toBlockNumber)
	filterQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(fromBlockNumber)),
		ToBlock:   big.NewInt(int64(toBlockNumber)),
		Addresses: []common.Address{m.optimismPortalAddress},
		Topics:    [][]common.Hash{{WithdrawalProvenEventABIHash}},
	}
	provenWithdrawalLogs, err := m.l1Client.FilterLogs(ctx, filterQuery)
	if err != nil {
		m.log.Error("failed to query withdrawal proven event logs", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "filterLogs").Inc()
		return
	}

	// Check the withdrawals against the L2toL1MP contract

	if len(provenWithdrawalLogs) == 0 {
		m.log.Info("no proven withdrawals found", "from_height", fromBlockNumber, "to_height", toBlockNumber)
	} else {
		m.log.Info("detected proven withdrawals", "num", len(provenWithdrawalLogs), "from_height", fromBlockNumber, "to_height", toBlockNumber)
	}

	for _, provenWithdrawalLog := range provenWithdrawalLogs {
		withdrawalHash := provenWithdrawalLog.Topics[1]
		m.log.Info("checking withdrawal", "withdrawal_hash", withdrawalHash.String(),
			"block_height", provenWithdrawalLog.BlockNumber, "tx_hash", provenWithdrawalLog.TxHash.String())

		seen, err := m.l2ToL1MP.SentMessages(nil, withdrawalHash)
		if err != nil {
			// Return early and loop back into the same block range
			log.Error("failed to query L2ToL1MP sentMessages mapping", "withdrawal_hash", withdrawalHash.String(), "err", err)
			m.nodeConnectionFailures.WithLabelValues("l2", "sentMessages").Inc()
			return
		}

		// If forgery is detected, update alerted metrics and return early to enter
		// into a loop at this block range. May want to update this logic such that future
		// forgeries can be detected -- the existence of one implies many others likely exist.
		if !seen {
			m.log.Warn("forgery detected!!!!", "withdrawal_hash", withdrawalHash.String())
			m.isDetectingForgeries.Set(1)
			return
		}

		m.withdrawalsValidated.Inc()
	}

	m.log.Info("validated withdrawals", "height", toBlockNumber)

	// Update markers
	m.nextL1Height = toBlockNumber + 1
	m.isDetectingForgeries.Set(0)
	m.highestBlockNumber.WithLabelValues("checked").Set(float64(toBlockNumber))
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	m.l2Client.Close()
	return nil
}
