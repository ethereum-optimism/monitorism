package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace                 = "faultproof_withdrawals"
	DefaultHoursInThePastToStartFrom = 14 * 24 //14 days
)

// Monitor monitors the state and events related to withdrawal forgery.
type Monitor struct {
	// context
	logger log.Logger
	ctx    context.Context

	// user arguments
	maxBlockRange uint64

	// dependencies
	L1Proxy   validator.L1ProxyInterface
	L2Proxy   validator.L2ProxyInterface
	validator *Validator

	// state
	l1ChainID *big.Int
	l2ChainID *big.Int

	startingL1Height uint64
	latestL1Height   uint64
	latestL2Height   uint64
	nextL1Height     uint64

	// currentWithdrawalsQueue map[common.Hash]*validator.WithdrawalProvenExtensionEvent
	currentWithdrawalsQueue map[common.Hash]*validator.WithdrawalProvenExtensionEvent

	// state
	state   State
	metrics Metrics
}

// NewMonitor creates a new Monitor instance with the provided configuration.
// It establishes connections to the specified L1 and L2 Geth clients, initializes
// the withdrawal validator, and sets up the initial state and metrics.
func NewMonitor(ctx context.Context, logger log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	logger.Info("creating withdrawals monitor...")

	l1Proxy, err := validator.NewL1Proxy(&ctx, cfg.L1GethURL, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 proxy: %w", err)
	}

	l2Proxy, err := validator.NewL2Proxy(&ctx, cfg.L2OpGethURL, cfg.L2OpNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 proxy: %w", err)
	}

	validator, err := NewValidator(&ctx, l1Proxy, l2Proxy)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	latestL1Height, err := l1Proxy.LatestHeight()
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block number: %w", err)
	}

	startingL1Height, err := GetStartingBlock(ctx, cfg, latestL1Height, l1Proxy, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting block: %w", err)
	}

	latestL2Height, err := l2Proxy.LatestHeight()
	if err != nil {
		logger.Error("failed to get latest known L2 block number", "error", err)
		return nil, err

	}

	l1ChainID, err := l1Proxy.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 chain id: %w", err)
	}
	l2ChainID, err := l2Proxy.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	metrics := NewMetrics(m)

	return &Monitor{
		logger: logger,

		ctx: ctx,

		L1Proxy:   l1Proxy,
		L2Proxy:   l2Proxy,
		validator: validator,

		l1ChainID:        l1ChainID,
		l2ChainID:        l2ChainID,
		latestL1Height:   latestL1Height,
		latestL2Height:   latestL2Height,
		startingL1Height: startingL1Height,
		nextL1Height:     startingL1Height,

		maxBlockRange: cfg.EventBlockRange,

		state:   State{},
		metrics: *metrics,
	}, nil
}

// It retrieves new events, processes them, and updates the state accordingly.
func (m *Monitor) Run(ctx context.Context) {
	// Defer the update function
	// defer m.state.LogState()

	start := m.nextL1Height

	latestL1Height, err := m.L1Proxy.LatestHeight()
	if err != nil {
		m.logger.Error("failed to query latest block number", "error", err)
		return
	}
	m.latestL1Height = latestL1Height

	stop := m.nextL1Height + m.maxBlockRange
	if stop > latestL1Height {
		stop = latestL1Height
	}

	latestKnownL2BlockNumber, err := m.L2Proxy.LatestHeight()
	if err != nil {
		m.logger.Error("failed to get latest known L2 block number", "error", err)
		return
	}

	m.state.latestL2Height = latestKnownL2BlockNumber

	extractedEvents, err := m.validator.GetRange(start, stop)
	if err != nil {
		m.logger.Error("failed to extract withdrawals", "error", err)
		return
	}

	// In here we review the extracted events and add to currentWithdrawalsQueue the one that needs to be reviewed again later
	for key, withdrawal := range extractedEvents {
		m.logger.Info("Withdrawals Extracted", "count", len(extractedEvents), "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))
		m.logger.Info("Reviewing newly extracted event", "key", key, "withdrawal", &withdrawal)

		// if !withdrawal.IsWithdrawalValid {
		// 	// We need to keep track only of games that are not valid
		// 	m.currentWithdrawalsQueue[withdrawal.] = withdrawal
		// }
	}

	// m.logger.Info("Withdrawals Extracted", "count", len(extractedEvents), "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))

	// for key, withdrawal := range extractedEvents {
	// 	m.state.currentWithdrawalsQueue[key] = withdrawal
	// }

	// m.logger.Info("Withdrawals Queue", "count", len(m.state.currentWithdrawalsQueue), "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))

	// for key, withdrawal := range m.state.currentWithdrawalsQueue {
	// 	select {
	// 	case <-ctx.Done():
	// 		return
	// 	default:
	// 	}

	// 	err = m.enrichWithL2Data(withdrawal)
	// 	if err != nil {
	// 		m.state.l2NodeConnectionFailures++
	// 		m.logger.Error("failed to consume event", "error", err)
	// 		return
	// 	}

	// 	m.logger.Info("Withdrawal enriched with L2 data", "key", key)
	// 	err = m.FaultDisputeGameHelper.RefreshState(withdrawal.DisputeGame)
	// 	if err != nil {
	// 		m.state.l1NodeConnectionFailures++
	// 		m.logger.Error("failed to refresh game state", "error", err)
	// 		return
	// 	}
	// 	// m.log.Info("PROCESSING WITHDRAWAL: L2 data present", "WithdrawalHash", withdrawal.WithdrawalProvenExtension1Event.WithdrawalHash, "ProofSubmitter", withdrawal.WithdrawalProvenExtension1Event.ProofSubmitter, "Status", withdrawal.WithdrawalProcessingStatus, "DisputeGameProxyAddress", withdrawal.DisputeGame.ProxyAddress)
	// }

	// update state
	m.nextL1Height = stop
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.L1Proxy.Close()
	m.L2Proxy.Close()
	return nil
}
