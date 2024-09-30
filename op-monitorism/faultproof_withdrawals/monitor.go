package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace = "faultproof_two_step_monitor"
)

// Monitor monitors the state and events related to withdrawal forgery.
type Monitor struct {
	// context
	log log.Logger
	ctx context.Context

	// user arguments
	l1GethClient   *ethclient.Client
	l2OpGethClient *ethclient.Client
	l2OpNodeClient *ethclient.Client
	l1ChainID      *big.Int
	l2ChainID      *big.Int
	maxBlockRange  uint64

	// helpers
	withdrawalValidator validator.ProvenWithdrawalValidator

	// state
	state   State
	metrics Metrics
}

// NewMonitor creates a new Monitor instance with the provided configuration.
// It establishes connections to the specified L1 and L2 Geth clients, initializes
// the withdrawal validator, and sets up the initial state and metrics.
func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating withdrawals monitor...")

	l1GethClient, err := ethclient.Dial(cfg.L1GethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	l2OpGethClient, err := ethclient.Dial(cfg.L2OpGethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}
	l2OpNodeClient, err := ethclient.Dial(cfg.L2OpNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	withdrawalValidator, err := validator.NewWithdrawalValidator(ctx, l1GethClient, l2OpGethClient, l2OpNodeClient, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal validator: %w", err)
	}

	latestL1Height, err := l1GethClient.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block number: %w", err)
	}

	l1ChainID, err := l1GethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 chain id: %w", err)
	}
	l2ChainID, err := l2OpGethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	state, err := NewState(log, cfg.StartingL1BlockHeight, latestL1Height)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	metrics := NewMetrics(m)

	ret := &Monitor{
		log: log,

		ctx:            ctx,
		l1GethClient:   l1GethClient,
		l2OpGethClient: l2OpGethClient,
		l2OpNodeClient: l2OpNodeClient,

		l1ChainID: l1ChainID,
		l2ChainID: l2ChainID,

		withdrawalValidator: *withdrawalValidator,

		maxBlockRange: cfg.EventBlockRange,

		state:   *state,
		metrics: *metrics,
	}

	// log state and metrics
	ret.state.LogState(ret.log)
	ret.metrics.UpdateMetricsFromState(&ret.state)

	return ret, nil
}

// GetLatestBlock retrieves the latest block number from the L1 Geth client.
// It updates the state with the latest L1 height.
func (m *Monitor) GetLatestBlock() (uint64, error) {
	latestL1Height, err := m.l1GethClient.BlockNumber(m.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block number: %w", err)
	}
	m.state.latestL1Height = latestL1Height
	return latestL1Height, nil
}

// GetMaxBlock calculates the maximum block number to be processed.
// It considers the next L1 height and the defined max block range.
func (m *Monitor) GetMaxBlock() (uint64, error) {
	latestL1Height, err := m.GetLatestBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block number: %w", err)
	}

	stop := m.state.nextL1Height + m.maxBlockRange
	if stop > latestL1Height {
		stop = latestL1Height
	}
	return stop, nil
}

// Run executes the main monitoring loop.
// It retrieves new events, processes them, and updates the state accordingly.
func (m *Monitor) Run(ctx context.Context) {
	start := m.state.nextL1Height

	stop, err := m.GetMaxBlock()
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to get max block", "error", err)
		return
	}

	// review previous invalidProposalWithdrawalsEvents
	invalidProposalWithdrawalsEvents, err := m.ConsumeEvents(m.state.invalidProposalWithdrawalsEvents)
	if err != nil {
		m.log.Error("failed to consume events", "error", err)
		return
	}

	// update state
	m.state.invalidProposalWithdrawalsEvents = *invalidProposalWithdrawalsEvents

	// get new events
	newEvents, err := m.withdrawalValidator.GetEnrichedWithdrawalsEvents(start, &stop)
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to get enriched withdrawal events", "error", err)
		return
	}
	newInvalidProposalWithdrawalsEvents, err := m.ConsumeEvents(newEvents)
	if err != nil {
		m.log.Error("failed to consume events", "error", err)
		return
	}

	// update state
	if len(*newInvalidProposalWithdrawalsEvents) > 0 && newInvalidProposalWithdrawalsEvents != nil {
		m.state.invalidProposalWithdrawalsEvents = append(m.state.invalidProposalWithdrawalsEvents, *newInvalidProposalWithdrawalsEvents...)
	}

	// update state
	m.state.nextL1Height = stop

	// log state and metrics
	m.state.LogState(m.log)
	m.metrics.UpdateMetricsFromState(&m.state)
}

// ConsumeEvents processes a slice of enriched withdrawal events and updates their states.
// It returns any events detected during the consumption that requires to be re-analysed again at a later stage (when the event referenced DisputeGame completes).
func (m *Monitor) ConsumeEvents(enrichedWithdrawalEvent []validator.EnrichedProvenWithdrawalEvent) (*[]validator.EnrichedProvenWithdrawalEvent, error) {
	var newForgeriesGameInProgressEvent []validator.EnrichedProvenWithdrawalEvent = make([]validator.EnrichedProvenWithdrawalEvent, 0)
	for _, enrichedWithdrawalEvent := range enrichedWithdrawalEvent {
		err := m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(&enrichedWithdrawalEvent)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return nil, err
		}

		consumedEvent, err := m.ConsumeEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("failed to consume event", "error", err)
			return nil, err
		} else if !consumedEvent {
			newForgeriesGameInProgressEvent = append(newForgeriesGameInProgressEvent, enrichedWithdrawalEvent)
		}
	}

	return &newForgeriesGameInProgressEvent, nil
}

// ConsumeEvent processes a single enriched withdrawal event.
// It logs the event details and checks for any forgery detection.
func (m *Monitor) ConsumeEvent(enrichedWithdrawalEvent validator.EnrichedProvenWithdrawalEvent) (bool, error) {
	m.log.Info("processing withdrawal event", "event", enrichedWithdrawalEvent.Event)
	if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID.Cmp(m.l2ChainID) != 0 {
		m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID))
	}
	valid, err := m.withdrawalValidator.IsWithdrawalEventValid(&enrichedWithdrawalEvent)
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to check if forgery detected", "error", err)
		return false, err
	}
	eventConsumed := false

	if !valid {
		if !enrichedWithdrawalEvent.Blacklisted {
			if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.CHALLENGER_WINS {
				m.log.Error("withdrawal is NOT valid, but the game is correctly resolved", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
				m.state.withdrawalsValidated++
				eventConsumed = true
			} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.DEFENDER_WINS {
				m.log.Error("withdrawal is NOT valid, forgery detected", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
				m.state.isDetectingForgeries++
				// add to forgeries
				m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, enrichedWithdrawalEvent)
				eventConsumed = true
			} else {
				m.log.Error("withdrawal is NOT valid, game is still in progress", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
				// add to events to be re-processed
				eventConsumed = false
			}
		} else {
			m.log.Warn("withdrawal is NOT valid, but game is blacklisted", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
			m.state.withdrawalsValidated++
			eventConsumed = true
		}
	} else {
		m.log.Info("Valid withdrawal", "valid", valid, "blacklisted", enrichedWithdrawalEvent.Blacklisted, "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		m.state.withdrawalsValidated++
		eventConsumed = true
	}
	m.state.processedProvenWithdrawalsExtension1Events++
	m.metrics.UpdateMetricsFromState(&m.state)
	return eventConsumed, nil
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
