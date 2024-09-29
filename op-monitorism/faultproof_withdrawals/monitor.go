package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace = "faultproof_two_step_monitor"
)

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
	faultDisputeGameHelper    FaultDisputeGameHelper
	optimismPortal2Helper     OptimismPortal2Helper
	l2ToL1MessagePasserHelper L2ToL1MessagePasserHelper
	l2NodeHelper              L2NodeHelper
	withdrawalValidator       WithdrawalValidator

	// state
	state   State
	metrics Metrics
}

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

	optimismPortal2Helper, err := NewOptimismPortal2Helper(ctx, l1GethClient, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	faultDisputeGameHelper, err := NewFaultDisputeGameHelper(ctx, l1GethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game helper: %w", err)
	}

	l2ToL1MessagePasserHelper, err := NewL2ToL1MessagePasserHelper(ctx, l2OpGethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create l2 to l1 message passer helper: %w", err)
	}

	l2NodeHelper, err := NewL2NodeHelper(ctx, l2OpNodeClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create l2 node helper: %w", err)
	}

	withdrawalValidator := NewWithdrawalValidator(ctx, optimismPortal2Helper, l2NodeHelper, l2ToL1MessagePasserHelper, faultDisputeGameHelper)

	latestL1Height, err := l1GethClient.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block number: %w", err)
	}

	state, err := NewState(log, cfg.StartingL1BlockHeight, latestL1Height)
	// state, err := NewState(log, cfg.StartingL1BlockHeight, latestL1Height)
	if err != nil {
		return nil, fmt.Errorf("failed to create state: %w", err)
	}

	metrics := NewMetrics(m)

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

		faultDisputeGameHelper:    *faultDisputeGameHelper,
		optimismPortal2Helper:     *optimismPortal2Helper,
		l2ToL1MessagePasserHelper: *l2ToL1MessagePasserHelper,
		l2NodeHelper:              *l2NodeHelper,
		withdrawalValidator:       *withdrawalValidator,

		maxBlockRange: cfg.EventBlockRange,

		state:   *state,
		metrics: *metrics,
	}

	return ret, nil
}

func (m *Monitor) GetLatestBlock() (uint64, error) {
	latestL1Height, err := m.l1GethClient.BlockNumber(m.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block number: %w", err)
	}
	m.state.latestL1Height = latestL1Height
	return latestL1Height, nil
}

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

func (m *Monitor) Run(ctx context.Context) {

	start := m.state.nextL1Height

	stop, err := m.GetMaxBlock()
	if err != nil {
		m.state.nodeConnectionFailures++
		m.log.Error("failed to get max block", "error", err)
		return
	}

	var newInvalidProposalWithdrawalsEvents []EnrichedWithdrawalEvent = make([]EnrichedWithdrawalEvent, 0)

	for _, enrichedWithdrawalEvent := range m.state.invalidProposalWithdrawalsEvents {

		err := m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(&enrichedWithdrawalEvent)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return
		}
		addToInvalidProposalWithdrawalsEvents, err := m.UpdateWithdrawalState(&enrichedWithdrawalEvent)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to update withdrawal state", "error", err)
			return
		}
		if addToInvalidProposalWithdrawalsEvents != nil {
			newInvalidProposalWithdrawalsEvents = append(newInvalidProposalWithdrawalsEvents, *addToInvalidProposalWithdrawalsEvents)
		}
	}

	m.state.invalidProposalWithdrawalsEvents = newInvalidProposalWithdrawalsEvents

	events, err := m.optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, &stop)
	if err != nil {
		m.state.nodeConnectionFailures++
		return
	}

	for event := range events {
		enrichedWithdrawalEvent, err := m.withdrawalValidator.GetEnrichedWithdrawalEvent(&events[event])
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to get enriched withdrawal event", "error", err)
			return
		}
		err = m.withdrawalValidator.UpdateEnrichedWithdrawalEvent(enrichedWithdrawalEvent)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to update enriched withdrawal event", "error", err)
			return
		}
		addToInvalidProposalWithdrawalsEvents, err := m.UpdateWithdrawalState(enrichedWithdrawalEvent)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to update withdrawal state", "error", err)
			return
		}
		if addToInvalidProposalWithdrawalsEvents != nil {
			m.state.invalidProposalWithdrawalsEvents = append(m.state.invalidProposalWithdrawalsEvents, *addToInvalidProposalWithdrawalsEvents)
		}
	}

	// update state
	m.state.nextL1Height = stop

	// log state and metrics
	m.state.LogState(m.log)
	m.metrics.UpdateMetricsFromState(&m.state)
}

func (m *Monitor) UpdateWithdrawalState(enrichedWithdrawalEvent *EnrichedWithdrawalEvent) (*EnrichedWithdrawalEvent, error) {

	result, err := m.withdrawalValidator.ValidateWithdrawal(enrichedWithdrawalEvent)
	if err != nil {
		m.log.Error("failed to validate withdrawal", "error", err)
		return nil, err
	}

	if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID.Cmp(m.l2ChainID) != 0 {
		m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.L2ChainID))
	}

	switch result {
	case PROOF_ON_BLACKLISTED_GAME:
		m.log.Info("game is blacklisted,removing from invalidProposalWithdrawalsEvents list", "game", enrichedWithdrawalEvent.DisputeGame.DisputeGameData.ProxyAddress.Hex())
		// m.state.blacklistedGames++
		return nil, nil
	case INVALID_PROOF_FORGERY_DETECTED:
		m.log.Error("withdrawal is NOT valid, forgery detected",
			"enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		m.state.isDetectingForgeries++
		m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, *enrichedWithdrawalEvent)
	case INVALID_PROPOSAL_FORGERY_DETECTED:
		m.log.Error("withdrawal is NOT valid, forgery detected",
			"enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		m.state.isDetectingForgeries++
		m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, *enrichedWithdrawalEvent)
	case INVALID_PROPOSAL_INPROGRESS:
		m.log.Warn("game is still in progress",
			"enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		return enrichedWithdrawalEvent, nil
	case INVALID_PROPOSAL_CORRECTLY_RESOLVED:
		m.log.Info("withdrawal was correctly resolved",
			"enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		m.state.withdrawalsValidated++
	case VALID_PROOF:
		m.log.Info("withdrawal is valid", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		m.state.withdrawalsValidated++
	}
	return nil, nil
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
