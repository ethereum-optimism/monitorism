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
	L1Proxy             validator.L1ProxyInterface
	L2Proxy             validator.L2ProxyInterface
	WithdrawalValidator *WithdrawalValidator

	// state
	l1ChainID *big.Int
	l2ChainID *big.Int

	startingL1Height uint64
	nextL1Height     uint64

	// currentWithdrawalsQueue map[common.Hash]*validator.WithdrawalValidationRef
	currentWithdrawalsQueue map[common.Hash]validator.WithdrawalValidationRef

	// SECURITY NODE: for now this is a map, but in the future to avoid exausting memory we should consider a cache. For now is an aceptable solution.
	// Using a database to store the data would be a better solution also for future scalability and traceability.
	currentWithdrawalsQueueAttacks map[common.Hash]validator.WithdrawalValidationRef

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

	withdrawalValidator, err := NewWithdrawalValidator(&ctx, l1Proxy, l2Proxy)
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

	l1ChainID, err := l1Proxy.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 chain id: %w", err)
	}
	l2ChainID, err := l2Proxy.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	currentWithdrawalsQueue := make(map[common.Hash]validator.WithdrawalValidationRef)
	currentWithdrawalsQueueAttacks := make(map[common.Hash]validator.WithdrawalValidationRef)

	metrics := NewMetrics(m)

	return &Monitor{
		logger: logger,

		ctx: ctx,

		L1Proxy:             l1Proxy,
		L2Proxy:             l2Proxy,
		WithdrawalValidator: withdrawalValidator,

		l1ChainID:        l1ChainID,
		l2ChainID:        l2ChainID,
		startingL1Height: startingL1Height.BlockNumber,
		nextL1Height:     startingL1Height.BlockNumber,

		currentWithdrawalsQueue:        currentWithdrawalsQueue,
		currentWithdrawalsQueueAttacks: currentWithdrawalsQueueAttacks,

		maxBlockRange: cfg.EventBlockRange,

		state:   State{},
		metrics: *metrics,
	}, nil
}

// It retrieves new events, processes them, and updates the state accordingly.
func (m *Monitor) Run(ctx context.Context) {

	// -- Section where we set the new block range to be reviewed --
	start := m.nextL1Height

	latestL1HeightBlock, err := m.L1Proxy.LatestHeight()
	if err != nil {
		m.logger.Error("RUN PANIC", "error", err, "start", start)
		return
	}
	latestL1Height := latestL1HeightBlock.BlockNumber
	stop := m.nextL1Height + m.maxBlockRange
	if stop > latestL1Height {
		stop = latestL1Height
	}

	l2TrustedNodeHeight, err := m.L2Proxy.LatestHeight()
	// -- Section where we review events in  currentWithdrawalsQueue--
	for _, withdrawalRef := range m.currentWithdrawalsQueue {
		m.logger.Info("QUEUE: REVIEW", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)

		if withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus == validator.IN_PROGRESS {
			// update the game if it is still in progress, so that we get new fresh data for this game
			m.L1Proxy.GetDisputeGameProxyUpdates(withdrawalRef.DisputeGameEvent.DisputeGame)
			// if the game is still in progress, we do nothing and keep it in the queue for the next iteration
			if withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus == validator.IN_PROGRESS {
				continue
			} else {
				// if the game is not in progress anymore, we need to get the withdrawal validation again
				withdrawalValidationRef, err := m.L2Proxy.GetWithdrawalValidation(withdrawalRef.DisputeGameEvent)
				if err != nil {
					m.logger.Error("RUN PANIC", "error", err, "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), &withdrawalRef, "BlockPresentOnL2")
					return
				}
				if withdrawalValidationRef.IsWithdrawalValid {
					m.logger.Info("QUEUE: REMOVE", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
					// if the withdrawal is valid, we remove it from the queue
					delete(m.currentWithdrawalsQueue, withdrawalRef.DisputeGameEvent.EventRef.TxHash)
				} else {
					m.logger.Warn("QUEUE: RE-ADD", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
					// if the withdrawal is not valid, we keep it in the queue for the next iteration
					m.currentWithdrawalsQueue[withdrawalRef.DisputeGameEvent.EventRef.TxHash] = *withdrawalValidationRef
				}
			}
		} else if !withdrawalRef.BlockPresentOnL2 {
			/** this is the second case in which we need to get the withdrawal validation again. In this case, the game is not in progress anymore, but the block is not present on L2.This means that the game was finalized, but the block was not submitted to L2 or we are missing the block because the Trusted Node is not yet in sync with the L2 chain.
			 */

			// if the event timestamp is before the current l2TrustedNodeHeight, we delete the event from the queue and add to the attacks queue
			if uint64(withdrawalRef.DisputeGameEvent.EventRef.BlockInfo.BlockTime) < uint64(l2TrustedNodeHeight.BlockTime) {
				m.logger.Warn("QUEUE: REMOVE", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
				// if the withdrawal is valid, we remove it from the queue
				delete(m.currentWithdrawalsQueue, withdrawalRef.DisputeGameEvent.EventRef.TxHash)
				m.currentWithdrawalsQueueAttacks[withdrawalRef.DisputeGameEvent.EventRef.TxHash] = withdrawalRef

			} else {
				// in this case, we need to get the withdrawal validation again
				withdrawalValidationRef, err := m.L2Proxy.GetWithdrawalValidation(withdrawalRef.DisputeGameEvent)
				if err != nil {
					m.logger.Error("RUN PANIC", "error", err, "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), &withdrawalRef, "BlockPresentOnL2")
					return
				}
				if withdrawalValidationRef.IsWithdrawalValid {
					m.logger.Info("QUEUE: REMOVE", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
					// if the withdrawal is valid, we remove it from the queue
					delete(m.currentWithdrawalsQueue, withdrawalRef.DisputeGameEvent.EventRef.TxHash)
				} else {
					m.logger.Warn("QUEUE: RE-ADD", "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop), "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
					// if the withdrawal is not valid, we keep it in the queue for the next iteration
					m.currentWithdrawalsQueue[withdrawalRef.DisputeGameEvent.EventRef.TxHash] = *withdrawalValidationRef
				}
			}
		}
		// Nothing to do in this case, the event is in the queue and should stay there for
	}

	// -- Section where we review new events --
	// In case of exception we want to stop and not update the state
	extractedEvents, err := m.WithdrawalValidator.GetRange(start, stop)
	if err != nil {
		m.logger.Error("RUN PANIC", "error", err, "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))
		return
	}

	m.logger.Info("WITHDRAWALS: EXTRACT", "count", len(extractedEvents), "start", fmt.Sprintf("%d", start), "stop", fmt.Sprintf("%d", stop))
	// In here we review the extracted events and add to currentWithdrawalsQueue the one that needs to be reviewed again later
	for key, withdrawalRef := range extractedEvents {
		m.logger.Info("WITHDRAWAL: REVIEW", "key", key, "withdrawalRef", &withdrawalRef)

		if !withdrawalRef.IsWithdrawalValid {
			// We need to keep track only of games that are not valid
			m.logger.Warn("QUEUE: ADD", "withdrawalRef", &withdrawalRef, "BlockPresentOnL2", withdrawalRef.BlockPresentOnL2, "WithdrawalPresentOnL2ToL1MessagePasser", withdrawalRef.WithdrawalPresentOnL2ToL1MessagePasser, "GameStatus", withdrawalRef.DisputeGameEvent.DisputeGame.GameStatus)
			m.currentWithdrawalsQueue[withdrawalRef.DisputeGameEvent.EventRef.TxHash] = withdrawalRef
		}
	}

	// update state
	m.nextL1Height = stop
}

// Close gracefully shuts down the Monitor by closing the Geth clients.
func (m *Monitor) Close(_ context.Context) error {
	m.L1Proxy.Close()
	m.L2Proxy.Close()
	return nil
}
