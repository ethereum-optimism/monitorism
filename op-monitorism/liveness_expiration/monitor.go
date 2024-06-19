package liveness_expiration

import (
	"context"
	"fmt"
	// "math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/liveness_expiration/bindings"
	// "github.com/ethereum/go-ethereum/accounts/abi/bind"
	// "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	// "github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	// "github.com/ethereum/go-ethereum"
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
	// optimismPortal        *bindings.OptimismPortalCaller
	// l2ToL1MP              *bindings.L2ToL1MessagePasserCaller

	maxBlockRange uint64
	nextL1Height  uint64
	GnosisSafe    *bindings.GnosisSafe

	// metrics
	highestBlockNumber     *prometheus.GaugeVec
	isDetectingForgeries   prometheus.Gauge
	withdrawalsValidated   prometheus.Counter
	nodeConnectionFailures *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("Starting the liveness expiration monitoring...")
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	fmt.Println("Safe Address: ", cfg.SafeAddress)
	if cfg.SafeAddress.Cmp(common.Address{}) == 0 {
		return nil, fmt.Errorf("Incorrect SafeAddress specified is set to -> %s", cfg.SafeAddress)
	}
	GnosisSafe, err := bindings.NewGnosisSafe(cfg.SafeAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}
	owners, _ := GnosisSafe.GetOwners(nil)
	fmt.Println(owners)
	// l2Client, err := ethclient.Dial(cfg.L2NodeURL)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to dial l2: %w", err)
	// }
	//
	// optimismPortal, err := bindings.NewOptimismPortalCaller(cfg.OptimismPortalAddress, l1Client)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	// }
	// l2ToL1MP, err := bindings.NewL2ToL1MessagePasserCaller(predeploys.L2ToL1MessagePasserAddr, l2Client)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	// }

	return &Monitor{
		log: log,

		l1Client:   l1Client,
		GnosisSafe: GnosisSafe,
		// optimismPortal: optimismPortal,
		// l2ToL1MP:       l2ToL1MP,

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
	latestL1Height, _ := m.l1Client.BlockNumber(ctx)

	m.log.Info("", "BlockNumber", latestL1Height)

	// callOpts := bind.CallOpts{
	// 	BlockNumber: big.NewInt(int64(latestL1Height)),
	// }

	GnosisSafe, err := m.GnosisSafe.GetOwners(nil)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
	}

	// 1. call the safe.owners()
	// 2. livenessGuard.lastLive(owner)
	// 3. save the livenessInterval()
	// 4. Need to understand this => (lock.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true

	m.log.Info("", "Owners", GnosisSafe)
	// Update markers
	// m.nextL1Height = toBlockNumber + 1
	// m.isDetectingForgeries.Set(0)
	// m.highestBlockNumber.WithLabelValues("checked").Set(float64(toBlockNumber))
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
