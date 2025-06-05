package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	maxBlockRange uint64

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
	log.Info("Creating withdrawals monitor...")

	log.Debug("Initializing L1 client connection", "url", cfg.L1GethURL)
	l1GethClient, err := ethclient.Dial(cfg.L1GethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	log.Debug("Successfully connected to L1 client")

	mapL2GethBackupURLs := make(map[string]string)
	if len(cfg.L2GethBackupURLs) > 0 {
		log.Debug("Processing L2 backup URLs", "count", len(cfg.L2GethBackupURLs))
		for _, part := range cfg.L2GethBackupURLs {
			parts := strings.Split(part, "=")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid backup URL format, expected name=url, got %s", part)
			}
			name, url := parts[0], parts[1]
			mapL2GethBackupURLs[name] = url
			log.Debug("Added L2 backup URL", "name", name, "url", url)
		}
	}

	log.Debug("Creating withdrawal validator",
		"l1Url", cfg.L1GethURL,
		"l2Url", cfg.L2OpGethURL,
		"backupUrls", len(mapL2GethBackupURLs),
		"portalAddress", cfg.OptimismPortalAddress)
	withdrawalValidator, err := validator.NewWithdrawalValidator(ctx, log, cfg.L1GethURL, cfg.L2OpGethURL, mapL2GethBackupURLs, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal validator: %w", err)
	}
	log.Debug("Successfully created withdrawal validator")

	log.Debug("Querying latest L1 block number")
	latestL1Height, err := l1GethClient.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block number: %w", err)
	}
	log.Debug("Retrieved latest L1 block number", "height", latestL1Height)

	metrics := NewMetrics(m)
	log.Debug("Initialized metrics")

	ret := &Monitor{
		log: log,

		ctx:                 ctx,
		withdrawalValidator: *withdrawalValidator,

		maxBlockRange: cfg.EventBlockRange,

		state:   State{},
		metrics: *metrics,
	}

	// is starting block is set it takes precedence
	var startingL1BlockHeight uint64
	hoursInThePastToStartFrom := cfg.HoursInThePastToStartFrom

	// In this case StartingL1BlockHeight is not set
	if cfg.StartingL1BlockHeight == -1 {
		log.Debug("Starting block height not set, calculating from hours in past",
			"hoursInPast", hoursInThePastToStartFrom,
			"defaultHours", DefaultHoursInThePastToStartFrom)

		// in this case is not set how many hours in the past to start from, we use default value that is 14 days.
		if hoursInThePastToStartFrom == 0 {
			hoursInThePastToStartFrom = DefaultHoursInThePastToStartFrom
			log.Debug("Using default hours in past", "hours", DefaultHoursInThePastToStartFrom)
		}

		// get the block number closest to the timestamp from two weeks ago
		latestL1HeightBigInt := new(big.Int).SetUint64(latestL1Height)
		log.Debug("Searching for block at approximate time",
			"latestHeight", latestL1HeightBigInt.String(),
			"hoursInPast", hoursInThePastToStartFrom)
		startingL1BlockHeightBigInt, err := ret.getBlockAtApproximateTimeBinarySearch(ctx, l1GethClient, latestL1HeightBigInt, big.NewInt(int64(hoursInThePastToStartFrom)))
		if err != nil {
			return nil, fmt.Errorf("failed to get block at approximate time: %w", err)
		}
		startingL1BlockHeight = startingL1BlockHeightBigInt.Uint64()
		log.Debug("Found starting block height", "height", startingL1BlockHeight)
	} else {
		startingL1BlockHeight = uint64(cfg.StartingL1BlockHeight)
		log.Debug("Using provided starting block height", "height", startingL1BlockHeight)
	}

	log.Debug("Querying latest L2 block number")
	latestL2Height, err := ret.withdrawalValidator.GetL2BlockNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest L2 height: %w", err)
	}
	log.Debug("Retrieved latest L2 block number", "height", latestL2Height)

	if startingL1BlockHeight > latestL1Height {
		log.Info("Next L1 height is greater than latest L1 height, starting from latest",
			"nextL1Height", startingL1BlockHeight,
			"latestL1Height", latestL1Height)
		startingL1BlockHeight = latestL1Height
	}

	log.Debug("Creating initial state")
	state, err := NewState(log, withdrawalValidator)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	state.nextL1Height = startingL1BlockHeight
	state.latestL1Height = latestL1Height
	state.latestL2Height = latestL2Height

	ret.state = *state

	// log state and metrics
	ret.state.LogState()
	ret.metrics.UpdateMetricsFromState(&ret.state)

	log.Info("Successfully initialized monitor",
		"startingL1Height", startingL1BlockHeight,
		"latestL1Height", latestL1Height,
		"latestL2Height", latestL2Height,
		"maxBlockRange", cfg.EventBlockRange)

	return ret, nil
}

// getBlockAtApproximateTimeBinarySearch finds the block number corresponding to the timestamp from two weeks ago using a binary search approach.
func (m *Monitor) getBlockAtApproximateTimeBinarySearch(ctx context.Context, client *ethclient.Client, latestBlockNumber *big.Int, hoursInThePast *big.Int) (*big.Int, error) {

	secondsInThePast := hoursInThePast.Mul(hoursInThePast, big.NewInt(60*60))
	m.log.Info("Looking for a block at approximate time of hours back",
		"secondsInThePast", fmt.Sprintf("%v", secondsInThePast),
		"time", fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05 MST")),
		"latestBlockNumber", fmt.Sprintf("%v", latestBlockNumber))
	// Calculate the total seconds in two weeks
	targetTime := big.NewInt(time.Now().Unix())
	targetTime.Sub(targetTime, secondsInThePast)

	// Initialize the search range
	left := big.NewInt(0)
	right := new(big.Int).Set(latestBlockNumber)

	var mid *big.Int
	acceptablediff := big.NewInt(60 * 60) //60 minutes

	m.log.Debug("Starting binary search",
		"targetTime", time.Unix(targetTime.Int64(), 0).Format("2006-01-02 15:04:05 MST"),
		"leftBound", left.String(),
		"rightBound", right.String(),
		"acceptableDiffSeconds", acceptablediff.String())

	// Perform binary search
	for left.Cmp(right) <= 0 {
		//interrupt in case of context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		// Calculate the midpoint
		mid = new(big.Int).Add(left, right)
		mid.Div(mid, big.NewInt(2))

		m.log.Debug("Binary search iteration",
			"left", left.String(),
			"right", right.String(),
			"mid", mid.String())

		// Get the block at mid
		block, err := client.BlockByNumber(context.Background(), mid)
		if err != nil {
			m.log.Error("Failed to get block", "blockNumber", mid.String(), "error", err)
			return nil, err
		}

		// Check the block's timestamp
		blockTime := big.NewInt(int64(block.Time()))
		blockTimeFormatted := time.Unix(blockTime.Int64(), 0).Format("2006-01-02 15:04:05 MST")

		//calculate the difference between the block time and the target time
		diff := new(big.Int).Sub(blockTime, targetTime)
		diffHours := new(big.Int).Div(diff, big.NewInt(3600))

		m.log.Debug("Block time comparison",
			"blockNumber", mid.String(),
			"blockTime", blockTimeFormatted,
			"timeDiffHours", diffHours.String(),
			"timeDiffSeconds", diff.String())

		// If block time is less than or equal to target time, check if we need to search to the right
		if blockTime.Cmp(targetTime) <= 0 {
			left.Set(mid) // Move left boundary up to mid
			m.log.Debug("Moving left boundary up", "newLeft", left.String())
		} else {
			right.Sub(mid, big.NewInt(1)) // Move right boundary down
			m.log.Debug("Moving right boundary down", "newRight", right.String())
		}
		if new(big.Int).Abs(diff).Cmp(acceptablediff) <= 0 {
			//if the difference is less than or equal to 1 hour, we can consider this block as the block closest to the target time
			m.log.Debug("Found acceptable block",
				"blockNumber", mid.String(),
				"blockTime", blockTimeFormatted,
				"timeDiffHours", diffHours.String())
			break
		}
	}

	// log the block number closest to the target time and the time
	m.log.Info("Found block closest to target time",
		"block", left.String(),
		"time", time.Unix(targetTime.Int64(), 0).Format("2006-01-02 15:04:05 MST"),
		"blockTime", time.Unix(targetTime.Int64(), 0).Format("2006-01-02 15:04:05 MST"))
	// After exiting the loop, left should be the block number closest to the target time
	return left, nil
}

// GetLatestBlock retrieves the latest block number from the L1 Geth client.
// It updates the state with the latest L1 height.
func (m *Monitor) GetLatestBlock() (uint64, error) {
	m.log.Debug("Getting latest L1 block number")
	latestL1Height, err := m.withdrawalValidator.GetL1BlockNumber()
	if err != nil {
		m.log.Error("Failed to query latest block number", "error", err)
		return 0, fmt.Errorf("failed to query latest block number: %w", err)
	}
	m.state.latestL1Height = latestL1Height
	m.log.Debug("Updated latest L1 block number", "height", latestL1Height)
	return latestL1Height, nil
}

// GetMaxBlock calculates the maximum block number to be processed.
// It considers the next L1 height and the defined max block range.
func (m *Monitor) GetMaxBlock() (uint64, error) {
	m.log.Debug("Calculating max block number",
		"nextL1Height", m.state.nextL1Height,
		"maxBlockRange", m.maxBlockRange)

	latestL1Height, err := m.GetLatestBlock()
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block number: %w", err)
	}

	stop := m.state.nextL1Height + m.maxBlockRange
	if stop > latestL1Height {
		stop = latestL1Height
		m.log.Debug("Max block adjusted to latest L1 height",
			"originalStop", m.state.nextL1Height+m.maxBlockRange,
			"adjustedStop", stop)
	}
	m.log.Debug("Calculated max block number", "stop", stop)
	return stop, nil
}

// Run executes the main monitoring loop.
// It retrieves new events, processes them, and updates the state accordingly.
func (m *Monitor) Run(ctx context.Context) {
	// Defer the update function
	defer m.metrics.UpdateMetricsFromState(&m.state)
	defer m.state.LogState()

	start := m.state.nextL1Height
	m.log.Debug("Starting monitoring run", "startBlock", start)

	stop, err := m.GetMaxBlock()
	if err != nil {
		m.log.Error("Failed to get max block", "error", err)
		return
	}
	m.log.Debug("Got max block for processing", "stopBlock", stop)

	// review previous invalidProposalWithdrawalsEvents
	m.log.Debug("Processing previous potential attack events",
		"count", len(m.state.potentialAttackOnInProgressGames))
	err = m.ConsumeEvents(m.state.potentialAttackOnInProgressGames)
	if err != nil {
		m.log.Error("Failed to consume previous events", "error", err)
		return
	}

	// get new events
	m.log.Info("Getting enriched withdrawal events",
		"start", start,
		"stop", stop,
		"range", stop-start)
	newEvents, err := m.withdrawalValidator.GetEnrichedWithdrawalsEventsMap(start, &stop)

	if err != nil {
		if start >= stop {
			m.log.Info("No new events to process", "start", start, "stop", stop)
		} else if stop-start <= 1 {
			m.log.Info("Failed to get enriched withdrawal events, range too small",
				"error", err,
				"start", start,
				"stop", stop)
		} else {
			m.log.Error("Failed to get enriched withdrawal events",
				"error", err,
				"start", start,
				"stop", stop)
		}
		return
	}

	m.log.Debug("Retrieved new withdrawal events", "count", len(newEvents))
	err = m.ConsumeEvents(newEvents)
	if err != nil {
		m.log.Error("Failed to consume new events", "error", err)
		return
	}

	// update state
	m.log.Debug("Updating next L1 height",
		"oldHeight", m.state.nextL1Height,
		"newHeight", stop)
	m.state.nextL1Height = stop
}

// ConsumeEvents processes a slice of enriched withdrawal events and updates their states.
// It returns any events detected during the consumption that requires to be re-analysed again at a later stage (when the event referenced DisputeGame completes).
func (m *Monitor) ConsumeEvents(enrichedWithdrawalEvents map[common.Hash]*validator.EnrichedProvenWithdrawalEvent) error {
	m.log.Debug("Starting to consume events", "count", len(enrichedWithdrawalEvents))

	for hash, enrichedWithdrawalEvent := range enrichedWithdrawalEvents {
		if enrichedWithdrawalEvent == nil {
			m.log.Error("WITHDRAWAL: enrichedWithdrawalEvent is nil in ConsumeEvents", "hash", hash)
			panic("WITHDRAWAL: enrichedWithdrawalEvent is nil in ConsumeEvents")
		}
		m.log.Debug("Processing withdrawal event",
			"hash", hash,
			"event", enrichedWithdrawalEvent)

		err := m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("Failed to update enriched withdrawal event",
				"error", err,
				"hash", hash)
			return err
		}

		//upgrade state to the latest L2 height after the event is processed
		m.log.Debug("Updating L2 block number after event processing")
		m.state.latestL2Height, err = m.withdrawalValidator.GetL2BlockNumber()
		if err != nil {
			m.log.Error("Failed to get L2 block number", "error", err)
			return err
		}

		err = m.ConsumeEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("Failed to consume event",
				"error", err,
				"hash", hash,
				"event", enrichedWithdrawalEvent)
			return err
		}
	}

	m.log.Debug("Finished consuming events", "processedCount", len(enrichedWithdrawalEvents))
	return nil
}

// ConsumeEvent processes a single enriched withdrawal event.
// It logs the event details and checks for any forgery detection.
func (m *Monitor) ConsumeEvent(enrichedWithdrawalEvent *validator.EnrichedProvenWithdrawalEvent) error {
	m.log.Debug("Validating withdrawal event",
		"gameStatus", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status)

	valid, err := m.withdrawalValidator.IsWithdrawalEventValid(enrichedWithdrawalEvent)
	if err != nil {
		m.log.Error("Failed to check if forgery detected",
			"error", err)
		return err
	}

	if !valid {
		if !enrichedWithdrawalEvent.Blacklisted {
			if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.CHALLENGER_WINS {
				m.log.Debug("Incrementing suspicious events on challenger wins games")
				m.state.IncrementSuspiciousEventsOnChallengerWinsGames(enrichedWithdrawalEvent)
			} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.DEFENDER_WINS {
				m.log.Debug("Incrementing potential attack on defender wins games")
				m.state.IncrementPotentialAttackOnDefenderWinsGames(enrichedWithdrawalEvent)
			} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == validator.IN_PROGRESS {
				m.log.Debug("Incrementing potential attack on in-progress games")
				m.state.IncrementPotentialAttackOnInProgressGames(enrichedWithdrawalEvent)
			} else {
				m.log.Error("WITHDRAWAL: is NOT valid, game status is unknown",
					"status", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status)
			}
		} else {
			m.log.Debug("Incrementing suspicious events on blacklisted event")
			m.state.IncrementSuspiciousEventsOnChallengerWinsGames(enrichedWithdrawalEvent)
		}
	} else {
		m.log.Debug("Incrementing validated withdrawals")
		m.state.IncrementWithdrawalsValidated(enrichedWithdrawalEvent)
	}

	m.state.eventsProcessed++
	m.metrics.UpdateMetricsFromState(&m.state)

	m.log.Debug("Finished processing event",
		"valid", valid,
		"totalProcessed", m.state.eventsProcessed)
	return nil
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.log.Debug("Closing monitor")
	return nil
}
