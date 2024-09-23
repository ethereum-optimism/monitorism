package liveness_expiration

import (
	"context"
	"fmt"
	"math/big"
	"math/bits"
	"time"

	"github.com/ethereum-optimism/monitorism/op-monitorism/liveness_expiration/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "liveness_expiration_mon"
)

type Monitor struct {
	log      log.Logger
	l1Client *ethclient.Client

	/** Contracts **/
	GnosisSafe            *bindings.GnosisSafe
	GnosisSafeAddress     common.Address
	LivenessGuard         *bindings.LivenessGuard
	LivenessGuardAddress  common.Address
	LivenessModule        *bindings.LivenessModule
	LivenessModuleAddress common.Address
	/** Metrics **/
	highestBlockNumber      *prometheus.GaugeVec
	unexpectedRpcErrors     *prometheus.CounterVec
	intervalLiveness        *prometheus.GaugeVec
	lastLiveOfAOwner        *prometheus.GaugeVec
	blockTimestamp          *prometheus.GaugeVec
	ownerStalePeriod        *prometheus.GaugeVec
	ownerDaysBeforeDeadline *prometheus.GaugeVec
}

// NewMonitor creates a new monitor.
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
	log.Info("", "L1RpcUrl", cfg.L1NodeURL)
	log.Info("--------------------------- End of Infos -------------------------------------------------------")

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
		}, []string{"blockNumber"}),
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
		ownerDaysBeforeDeadline: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "ownerDaysBeforeDeadline",
			Help:      "Number of days before the deadline is reached for a specific owner.",
		}, []string{"safeOwnerAddress"}),
		ownerStalePeriod: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "ownerStalePeriod",
			Help:      "Safe Owner Stale Period, the time that a safe owner address is not active anymore, should always be 0. The values can be 0 (normal), 1 (1 day - HIGH 1 day left), 7 (7 days - MEDIUM 7 days left), 14 (14 days - LOW 14 days left).",
		}, []string{"safeOwnerAddress"}),
		blockTimestamp: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "BlockTimestamp",
			Help:      "Block Timestamp of the last block.",
		}, []string{"blocktimestamp"}),
	}, nil
}

// Run is the main loop of the monitor.
// This loop will update the metrics `blockTimestamp`, `highestBlockNumber`, `lastLiveOfAOwner`, `intervalLiveness`.
// Thanks to these metrics we can monitor the liveness expiration through  (block.timestamp + BUFFER > lastLive(owner) + livenessInterval).
// NOTE: 	// Liveness module mainnet  -> https://etherscan.io/address/0x0454092516c9A4d636d3CAfA1e82161376C8a748
// Liveness guard mainnet  ->  https://etherscan.io/address/0x24424336F04440b1c28685a38303aC33C9D14a25
// 1. call the safe.owners()
// 2. livenessGuard.lastLive(owner)
// 3. save the livenessInterval()
// 4. Ensure that the invariant is not broken -> (block.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true
func (m *Monitor) Run(ctx context.Context) {
	day := uint64(86400) // 1 day in seconds
	blocknumber := new(big.Int)

	latestL1Height, err := m.l1Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("l1", "blockNumber").Inc()
		return
	}

	blocknumber.SetUint64(uint64(latestL1Height))
	blockTimestamp, err := m.l1Client.BlockByNumber(ctx, blocknumber)
	if err != nil {
		m.log.Error("failed to query the method `BlockByNumber`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "BlockByNumber").Inc()
		return
	}
	now := blockTimestamp.Time()

	listOwners, err := m.GnosisSafe.GetOwners(nil) // 1. Get the list of owner from the safe.
	if err != nil {
		m.log.Error("failed to query the method `GetOwners`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "GetOwners").Inc()
		return
	}

	interval, err := m.LivenessModule.LivenessInterval(nil) // 2. Get the interval from the liveness module.
	if err != nil {
		m.log.Error("failed to query the method `LivenessInterval`", "err", err, "blockNumber", latestL1Height)
		m.unexpectedRpcErrors.WithLabelValues("l1", "LivenessInterval").Inc()
		return
	}
	m.intervalLiveness.WithLabelValues("interval").Set(float64(interval.Uint64()))

	for _, owner := range listOwners {
		lastLive, err := m.LivenessGuard.LastLive(nil, owner) // 3. Get the last live from the liveness guard for each owner
		big_deadline := big.NewInt(0)
		if err != nil {
			m.log.Error("failed to query the method `LastLive`", "err", err, "blockNumber", latestL1Height)
			m.unexpectedRpcErrors.WithLabelValues("l1", "LastLive").Inc()
			return
		}

		m.lastLiveOfAOwner.WithLabelValues(owner.String()).Set(float64(lastLive.Uint64()))

		big_deadline.Add(lastLive, interval)
		deadline := big_deadline.Uint64()

		deadline_date := time.Unix(int64(deadline), 0)
		formattedDate := deadline_date.Format("Monday, January 2, 2006")
		// 4. Ensure that the invariant is not broken -> (block.timestamp + BUFFER > lastLive(owner) + livenessInterval) == true
		result, borrow := bits.Sub64(deadline, now, 0)
		if borrow != 0 {
			m.log.Warn("`deadline - now` is negative means that the `owner` is not active anymore at all and should be removed fast! This is not suppose to happen because we will be intervening before ensure that is not happening", "deadline", deadline, "now", now, "owner", owner)
		}

		days_left_before_deadline := result / day

		m.log.Info("", "owner", owner, "now", now, "deadline", deadline, "lastlive", lastLive, "interval", interval, "deadline_date", formattedDate, "days_left_before_deadline", days_left_before_deadline)
		m.ownerDaysBeforeDeadline.WithLabelValues(owner.String()).Set(float64(days_left_before_deadline))

		if result <= 1*day {
			m.log.Info("deadline is less than 1 day we need to ensure that the owner is doing something in the last 24h otherwise we need to remove it!", "lastLive", lastLive, "owner", owner)
			m.ownerStalePeriod.WithLabelValues(owner.String()).Set(float64(1))
		} else if result <= 7*day {
			m.log.Info("deadline is less than 7 days we need to ensure that the owner is doing something in the last 7 days otherwise we need to remove it!", "lastLive", lastLive, "owner", owner)
			m.ownerStalePeriod.WithLabelValues(owner.String()).Set(float64(7))

		} else if result <= 14*day {
			m.log.Info("deadline is less than 14 days we need to ensure that the owner is doing something in the last 14 days otherwise we need to remove it!", "lastLive", lastLive, "owner", owner)
			m.ownerStalePeriod.WithLabelValues(owner.String()).Set(float64(14))

		} else { //If Owner is not stalling (most of the time) we set the metric to 0 for the owner because he is not stalling.
			m.ownerStalePeriod.WithLabelValues(owner.String()).Set(float64(0))
		}
	}

	m.log.Info("", "interval", interval, "Owners", listOwners, "SafeAddress", m.GnosisSafeAddress, "highestBlockNumber", latestL1Height)

	m.highestBlockNumber.WithLabelValues("blockNumber").Set(float64(latestL1Height))
}

// Close closes the monitor.
func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
