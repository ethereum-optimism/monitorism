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
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "liveness_expiration_mon"
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client

	// optimismPortal        *bindings.OptimismPortalCaller
	// l2ToL1MP              *bindings.L2ToL1MessagePasserCaller

	maxBlockRange        uint64
	nextL1Height         uint64
	GnosisSafe           *bindings.GnosisSafe
	GnosisSafeAddress    common.Address
	LivenessGuard        *bindings.GnosisSafe
	LivenessGuardAddress *bindings.GnosisSafe
	LivenessModule       common.Address
	highestBlockNumber   *prometheus.GaugeVec
	unexpectedRpcErrors  *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("Starting the liveness expiration monitoring...")
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	if cfg.SafeAddress.Cmp(common.Address{}) == 0 {
		return nil, fmt.Errorf("The `SafeAddress` specified is set to -> %s", cfg.SafeAddress)
	}
	GnosisSafe, err := bindings.NewGnosisSafe(cfg.SafeAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}

	LivenessGuard, err := bindings.NewGnosisSafe(cfg.SafeAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}

	LivenessModule, err := bindings.NewGnosisSafe(cfg.SafeAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}

	log.Info("----------------------- Liveness Expiration Monitoring (Infos) -----------------------------")
	log.Info("", "Safe Address", cfg.SafeAddress)
	log.Info("", "Interval", cfg.LoopIntervalMsec)
	log.Info("", "L1RpcUrl", cfg.L1NodeURL)
	log.Info("--------------------------- End of Infos -------------------------------------------------------")

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

		l1Client:          l1Client,
		GnosisSafe:        GnosisSafe,
		GnosisSafeAddress: cfg.SafeAddress,

		LivenessGuard:         GnosisSafe,
		LivenessGuardAddress:  cfg.LivenessGuardAddress,
		LivenessModule:        GnosisSafe,
		LivenessModuleAddress: cfg.LivenessModuleAddress,
		/** Metrics **/
		highestBlockNumber: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestBlockNumber",
			Help:      "observed l1 heights (checked and known)",
		}, []string{"type"}),
		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpectedRpcErrors",
			Help:      "number of unexpcted rpc errors",
		}, []string{"section", "name"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	latestL1Height, err := m.l1Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("l1", "blockNumber").Inc()
	}

	m.log.Info("", "BlockNumber", latestL1Height)

	// callOpts := bind.CallOpts{
	// 	BlockNumber: big.NewInt(int64(latestL1Height)),
	// }

	GnosisSafe, err := m.GnosisSafe.GetOwners(nil)
	if err != nil {
		m.log.Error("failed to query the method `GetOwners`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "GetOwners").Inc()
	}

	// Liveness module mainnet  -> https://etherscan.io/address/0x0454092516c9A4d636d3CAfA1e82161376C8a748
	// Liveness guard mainnet  ->  https://etherscan.io/address/0x24424336F04440b1c28685a38303aC33C9D14a25
	// 1. call the safe.owners()
	// 2. livenessGuard.lastLive(owner)
	// 3. save the livenessInterval()
	// 4. Need to understand this => (lock.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true

	m.log.Info("", "Current Owners", GnosisSafe, "SafeAddress", m.GnosisSafeAddress, "highestBlockNumber", latestL1Height)
	// Update markers
	// m.nextL1Height = toBlockNumber + 1
	// m.isDetectingForgeries.Set(0)
	// m.highestBlockNumber.WithLabelValues("checked").Set(float64(toBlockNumber))
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
