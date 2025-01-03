package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace                 = "faultproof_withdrawals"
	DefaultHoursInThePastToStartFrom = 14 * 24 //14 days
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
	state State
}

// NewMonitor creates a new Monitor instance with the provided configuration.
// It establishes connections to the specified L1 and L2 Geth clients, initializes
// the withdrawal validator, and sets up the initial state and metrics.
func NewMonitor(ctx context.Context, log log.Logger, metricsFactory metrics.Factory, cfg CLIConfig) (*Monitor, error) {
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

	withdrawalValidator, err := validator.NewWithdrawalValidator(ctx, cfg.L1GethURL, cfg.L2OpGethURL, cfg.L2OpNodeURL, cfg.OptimismPortalAddress)
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

		state: State{},
	}

	// is starting block is set it takes precedence

	var startingL1BlockHeight uint64
	hoursInThePastToStartFrom := cfg.HoursInThePastToStartFrom

	// In this case StartingL1BlockHeight is not set
	if cfg.StartingL1BlockHeight == -1 {
		// in this case is not set how many hours in the past to start from, we use default value that is 14 days.
		if hoursInThePastToStartFrom == 0 {
			hoursInThePastToStartFrom = DefaultHoursInThePastToStartFrom
		}

		// get the block number closest to the timestamp from two weeks ago
		latestL1HeightBigInt := new(big.Int).SetUint64(latestL1Height)
		startingL1BlockHeightBigInt, err := GetBlockAtApproximateTimeBinarySearch(ctx, l1GethClient, latestL1HeightBigInt, big.NewInt(int64(hoursInThePastToStartFrom)), log)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at approximate time: %w", err)
		}
		startingL1BlockHeight = startingL1BlockHeightBigInt.Uint64()

	} else {
		startingL1BlockHeight = uint64(cfg.StartingL1BlockHeight)
	}

	latestKnownL2BlockNumber, err := ret.l2OpGethClient.BlockNumber(ret.ctx)
	if err != nil {
		ret.log.Error("failed to get latest known L2 block number", "error", err)
		return nil, err

	}

	state, err := NewState(log, startingL1BlockHeight, latestL1Height, latestKnownL2BlockNumber, metricsFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}
	ret.state = *state

	// log state and metrics
	ret.state.LogState()

	return ret, nil
}

// Run executes the main monitoring loop.
// It retrieves new events, processes them, and updates the state accordingly.
func (m *Monitor) Run(ctx context.Context) {
	// Defer the update function
	defer m.state.LogState()

	start := m.state.nextL1Height

	latestL1Height, err := m.l1GethClient.BlockNumber(m.ctx)
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to query latest block number", "error", err)

		return
	}
	m.state.latestL1Height = latestL1Height

	stop := m.state.nextL1Height + m.maxBlockRange
	if stop > latestL1Height {
		stop = latestL1Height
	}

	// review previous invalidProposalWithdrawalsEvents
	err = m.ConsumeEvents(m.state.potentialAttackOnInProgressGames)
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to consume events", "error", err)
		return
	}

	// get new events
	m.log.Info("getting enriched withdrawal events", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))
	newEvents, err := m.withdrawalValidator.GetEnrichedWithdrawalsEventsMap(start, &stop)
	if err != nil {
		if start >= stop {
			m.log.Info("no new events to process", "start", start, "stop", stop)
		} else if stop-start <= 1 {
			//in this case it happens when the range is too small, we can ignore the error as it is normal for the Iterator to not be ready yet
			m.log.Info("failed to get enriched withdrawal events, should not be an issue as start and stop blocks are too close", "error", err)
		} else {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to get enriched withdrawal events", "error", err)
		}
		return
	}

	err = m.ConsumeEvents(newEvents)
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to consume events", "error", err)
		return
	}

	// update state
	m.state.nextL1Height = stop

}

// ConsumeEvents processes a slice of enriched withdrawal events and updates their states.
// It returns any events detected during the consumption that requires to be re-analysed again at a later stage (when the event referenced DisputeGame completes).
func (m *Monitor) ConsumeEvents(enrichedWithdrawalEvents map[common.Hash]*validator.EnrichedProvenWithdrawalEvent) error {
	for _, enrichedWithdrawalEvent := range enrichedWithdrawalEvents {
		if enrichedWithdrawalEvent == nil {
			m.log.Error("WITHDRAWAL: enrichedWithdrawalEvent is nil in ConsumeEvents")
			panic("WITHDRAWAL: enrichedWithdrawalEvent is nil in ConsumeEvents")
		}
		m.log.Info("processing withdrawal event", "event", enrichedWithdrawalEvent)

		latestKnownL2BlockNumber, err := m.l2OpGethClient.BlockNumber(m.ctx)
		if err != nil {
			m.log.Error("failed to get latest known L2 block number", "error", err)
			return err

		}
		m.state.latestL2Height = latestKnownL2BlockNumber

		err = m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(enrichedWithdrawalEvent)
		//upgrade state to the latest L2 height	after the event is processed

		if err != nil {
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return err
		}

		err = m.ConsumeEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("failed to consume event", "error", err)
			return err
		}
	}

	return nil
}

// ConsumeEvent processes a single enriched withdrawal event.
// It logs the event details and checks for any forgery detection.
func (m *Monitor) ConsumeEvent(enrichedWithdrawalEvent *validator.EnrichedProvenWithdrawalEvent) error {
	defer m.state.LogState()

	if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID.Cmp(m.l2ChainID) != 0 {
		m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID))
	}
	valid, err := m.withdrawalValidator.IsWithdrawalEventValid(enrichedWithdrawalEvent)
	if err != nil {
		m.log.Error("failed to check if forgery detected", "error", err)
		return err
	}

	if !valid {
		if !enrichedWithdrawalEvent.Blacklisted {
			if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.CHALLENGER_WINS {
				m.state.IncrementSuspiciousEventsOnChallengerWinsGames(enrichedWithdrawalEvent)
			} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.DEFENDER_WINS {
				m.state.IncrementPotentialAttackOnDefenderWinsGames(enrichedWithdrawalEvent)
			} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.IN_PROGRESS {
				m.state.IncrementPotentialAttackOnInProgressGames(enrichedWithdrawalEvent)
				// add to events to be re-processed
			} else {
				m.log.Error("WITHDRAWAL: is NOT valid, game status is unknown. UNKNOWN STATE SHOULD NEVER HAPPEN", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
			}

		} else {
			m.state.IncrementSuspiciousEventsOnChallengerWinsGames(enrichedWithdrawalEvent)
		}
	} else {
		m.state.IncrementWithdrawalsValidated(enrichedWithdrawalEvent)
	}
	m.state.eventsProcessed++
	return nil
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
