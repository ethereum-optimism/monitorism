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

	maxBlockRange         uint64
	nextL1Height          uint64
	GnosisSafe            *bindings.GnosisSafe
	GnosisSafeAddress     common.Address
	LivenessGuard         *bindings.LivenessGuard
	LivenessGuardAddress  common.Address
	LivenessModule        *bindings.LivenessModule
	LivenessModuleAddress common.Address
	/** Metrics **/
	highestBlockNumber  *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
	intervalLiveness    *prometheus.GaugeVec
	lastLiveOfAOwner    *prometheus.GaugeVec
	blockTimestamp      *prometheus.GaugeVec
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
	if cfg.LivenessGuardAddress.Cmp(common.Address{}) == 0 {
		return nil, fmt.Errorf("The `LivenessGuardAddress` specified is set to -> %s", cfg.LivenessGuardAddress)
	}
	if cfg.LivenessModuleAddress.Cmp(common.Address{}) == 0 {
		return nil, fmt.Errorf("The `LivenessModuleAddress` specified is set to -> %s", cfg.LivenessModuleAddress)
	}

	GnosisSafe, err := bindings.NewGnosisSafe(cfg.SafeAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}

	LivenessGuard, err := bindings.NewLivenessGuard(cfg.LivenessGuardAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the LivenessGuard: %w", err)
	}

	LivenessModule, err := bindings.NewLivenessModule(cfg.LivenessModuleAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the LivenessModule: %w", err)
	}

	log.Info("----------------------- Liveness Expiration Monitoring (Infos) -----------------------------")
	log.Info("", "Safe Address", cfg.SafeAddress)
	log.Info("", "LivenessModuleAddress", cfg.LivenessModuleAddress)
	log.Info("", "LivenessGuardAddress", cfg.LivenessGuardAddress)
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

		l1Client: l1Client,

		GnosisSafe:            GnosisSafe,
		GnosisSafeAddress:     cfg.SafeAddress,
		LivenessGuard:         LivenessGuard,
		LivenessGuardAddress:  cfg.LivenessGuardAddress,
		LivenessModule:        LivenessModule,
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
			Help:      "number of unexpected rpc errors",
		}, []string{"section", "name"}),
		intervalLiveness: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "intervalLiveness",
			Help:      "Interval in (second) of the liveness from the liveness module",
		}, []string{"interval"}),
		lastLiveOfAOwner: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "lastLiveOfAOwner",
			Help:      "Last Live of an owner from the liveness guard, means the last time an owner make an action.",
		}, []string{"address"}),
		blockTimestamp: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "BlockTimestamp",
			Help:      "Block Timestamp of the last block.",
		}, []string{"blocktimestamp"}),
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

	listOwners, err := m.GnosisSafe.GetOwners(nil) // 1. Get the owners from the safe.
	if err != nil {
		m.log.Error("failed to query the method `GetOwners`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "GetOwners").Inc()
	}

	for _, owner := range listOwners {
		lastLive, err := m.LivenessGuard.LastLive(nil, owner) // 2. Get the last live from the liveness guard.
		if err != nil {
			m.log.Error("failed to query the method `LastLive`", "err", err, "blockNumber", latestL1Height)
			m.unexpectedRpcErrors.WithLabelValues("l1", "LastLive").Inc()
		}
		m.lastLiveOfAOwner.WithLabelValues(owner.String()).Set(float64(lastLive.Uint64()))
		m.log.Info("", "lastLive", lastLive, "owner", owner)
	}

	interval, err := m.LivenessModule.LivenessInterval(nil) // 3. Get the interval from the liveness module.
	if err != nil {
		m.log.Error("failed to query the method `LivenessInterval`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "LivenessInterval").Inc()
	}
	m.intervalLiveness.WithLabelValues("interval").Set(float64(interval.Uint64()))
	// (block.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true

	m.log.Info("", "interval", interval, "Owners", listOwners, "SafeAddress", m.GnosisSafeAddress, "highestBlockNumber", latestL1Height)

	// Liveness module mainnet  -> https://etherscan.io/address/0x0454092516c9A4d636d3CAfA1e82161376C8a748
	// Liveness guard mainnet  ->  https://etherscan.io/address/0x24424336F04440b1c28685a38303aC33C9D14a25
	// 1. call the safe.owners()
	// 2. livenessGuard.lastLive(owner)
	// 3. save the livenessInterval()
	// 4. Need to understand this => (lock.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true

	// Update markers
	// m.nextL1Height = toBlockNumber + 1
	// m.isDetectingForgeries.Set(0)
	// m.highestBlockNumber.WithLabelValues("checked").Set(float64(toBlockNumber))
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
