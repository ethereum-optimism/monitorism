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
	l1GethClient    *ethclient.Client
	l2OpGethClient  *ethclient.Client
	l2BackupClients map[string]*ethclient.Client
	l1ChainID       *big.Int
	l2ChainID       *big.Int
	maxBlockRange   uint64

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

	l1ChainID, err := l1GethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 chain id: %w", err)
	}

	l2OpGethClient, err := ethclient.Dial(cfg.L2OpGethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}
	l2ChainID, err := l2OpGethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	// if backup urls are provided, create a backup client for each
	var l2OpGethBackupClients map[string]*ethclient.Client
	if len(cfg.L2GethBackupURLs) > 0 {
		l2OpGethBackupClients, err = GethBackupClientsDictionary(ctx, cfg.L2GethBackupURLs, l2ChainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup clients: %w", err)
		}

	}

	withdrawalValidator, err := validator.NewWithdrawalValidator(ctx, log, l1GethClient, l2OpGethClient, l2OpGethBackupClients, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal validator: %w", err)
	}

	latestL1Height, err := l1GethClient.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block number: %w", err)
	}

	metrics := NewMetrics(m)

	ret := &Monitor{
		log: log,

		ctx:             ctx,
		l1GethClient:    l1GethClient,
		l2OpGethClient:  l2OpGethClient,
		l2BackupClients: l2OpGethBackupClients,

		l1ChainID: l1ChainID,
		l2ChainID: l2ChainID,

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
		// in this case is not set how many hours in the past to start from, we use default value that is 14 days.
		if hoursInThePastToStartFrom == 0 {
			hoursInThePastToStartFrom = DefaultHoursInThePastToStartFrom
		}

		// get the block number closest to the timestamp from two weeks ago
		latestL1HeightBigInt := new(big.Int).SetUint64(latestL1Height)
		startingL1BlockHeightBigInt, err := ret.getBlockAtApproximateTimeBinarySearch(ctx, l1GethClient, latestL1HeightBigInt, big.NewInt(int64(hoursInThePastToStartFrom)))
		if err != nil {
			return nil, fmt.Errorf("failed to get block at approximate time: %w", err)
		}
		startingL1BlockHeight = startingL1BlockHeightBigInt.Uint64()

	} else {
		startingL1BlockHeight = uint64(cfg.StartingL1BlockHeight)
	}

	latestL2Height, err := ret.withdrawalValidator.L2NodeHelper.BlockNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest L2 height: %w", err)
	}
	state, err := NewState(log, startingL1BlockHeight, latestL1Height, latestL2Height)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}
	ret.state = *state

	// log state and metrics
	ret.state.LogState()
	ret.metrics.UpdateMetricsFromState(&ret.state)

	return ret, nil
}

func GethBackupClientsDictionary(ctx context.Context, L2GethBackupURLs []string, l2ChainID *big.Int) (map[string]*ethclient.Client, error) {
	dictionary := make(map[string]*ethclient.Client)
	for _, rawURL := range L2GethBackupURLs {
		parts := strings.Split(rawURL, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid backup URL format, expected name=url, got %s", rawURL)
		}
		name, url := parts[0], parts[1]
		backupClient, err := ethclient.Dial(url)
		if err != nil {
			return nil, fmt.Errorf("failed to dial l2 backup, error: %w", err)
		}
		backupChainID, err := backupClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup L2 chain ID, error: %w", err)
		}
		if backupChainID.Cmp(l2ChainID) != 0 {
			return nil, fmt.Errorf("backup L2 client chain ID mismatch, expected: %d, got: %d", l2ChainID, backupChainID)
		}
		dictionary[name] = backupClient
	}
	return dictionary, nil
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

		// Get the block at mid
		block, err := client.BlockByNumber(context.Background(), mid)
		if err != nil {
			return nil, err
		}

		// Check the block's timestamp
		blockTime := big.NewInt(int64(block.Time()))

		//calculate the difference between the block time and the target time
		diff := new(big.Int).Sub(blockTime, targetTime)

		// If block time is less than or equal to target time, check if we need to search to the right
		if blockTime.Cmp(targetTime) <= 0 {
			left.Set(mid) // Move left boundary up to mid
		} else {
			right.Sub(mid, big.NewInt(1)) // Move right boundary down
		}
		if new(big.Int).Abs(diff).Cmp(acceptablediff) <= 0 {
			//if the difference is less than or equal to 1 hour, we can consider this block as the block closest to the target time
			break
		}

	}

	// log the block number closest to the target time and the time
	m.log.Info("block number closest to target time", "block", fmt.Sprintf("%v", left), "time", time.Unix(targetTime.Int64(), 0))
	// After exiting the loop, left should be the block number closest to the target time
	return left, nil
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
	// Defer the update function
	defer m.metrics.UpdateMetricsFromState(&m.state)
	defer m.state.LogState()

	start := m.state.nextL1Height

	stop, err := m.GetMaxBlock()
	m.state.nodeConnections++
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to get max block", "error", err)
		return
	}

	// review previous invalidProposalWithdrawalsEvents
	err = m.ConsumeEvents(m.state.potentialAttackOnInProgressGames)
	m.state.nodeConnections++
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to consume events", "error", err)
		return
	}

	// get new events
	m.log.Info("getting enriched withdrawal events", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))
	newEvents, err := m.withdrawalValidator.GetEnrichedWithdrawalsEventsMap(start, &stop)
	m.state.nodeConnections++
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
	m.state.nodeConnections++
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
		err := m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return err
		}
		//upgrade state to the latest L2 height	after the event is processed
		m.state.latestL2Height, err = m.withdrawalValidator.L2NodeHelper.BlockNumber()
		if err != nil {
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return err
		}

		err = m.ConsumeEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.log.Error("failed to consume event", "error", err, "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
			return err
		}
	}

	return nil
}

// ConsumeEvent processes a single enriched withdrawal event.
// It logs the event details and checks for any forgery detection.
func (m *Monitor) ConsumeEvent(enrichedWithdrawalEvent *validator.EnrichedProvenWithdrawalEvent) error {
	if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID.Cmp(m.l2ChainID) != 0 {
		m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID))
	}
	valid, err := m.withdrawalValidator.IsWithdrawalEventValid(enrichedWithdrawalEvent)
	if err != nil {
		m.log.Error("failed to check if forgery detected", "error", err, "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
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
	m.metrics.UpdateMetricsFromState(&m.state)
	return nil
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
