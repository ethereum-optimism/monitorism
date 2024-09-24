package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"math"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace = "faultproof_two_step_monitor"
)

type State struct {
	nextL1Height           uint64
	latestL1Height         uint64
	lastNotFaultProofBlock uint64

	processedProvenWithdrawalsEvents uint64
	processedGames                   uint64

	isDetectingForgeries     uint64
	withdrawalsValidated     uint64
	withdrawalsNotFaultProof uint64

	nodeConnectionFailures uint64

	forgeriesWithdrawalsEvents       []ProposalWithdrawalsEvent
	invalidProposalWithdrawalsEvents []ProposalWithdrawalsEvent
}

type ProposalWithdrawalsEvent struct {
	DisputeGame *DisputeGame
	Event       *l1.OptimismPortal2WithdrawalProven
}

type Metrics struct {
	NextL1HeightGauge                       prometheus.Gauge
	LatestL1HeightGauge                     prometheus.Gauge
	LastNotFaultProofBlockGauge             prometheus.Gauge
	ProcessedProvenWithdrawalsEventsCounter prometheus.Counter
	ProcessedGamesCounter                   prometheus.Counter
	IsDetectingForgeriesGauge               prometheus.Gauge
	WithdrawalsValidatedCounter             prometheus.Counter
	WithdrawalsNotFaultProofCounter         prometheus.Counter
	NodeConnectionFailuresCounter           prometheus.Counter
	ForgeriesWithdrawalsEventsGauge         prometheus.Gauge
	InvalidProposalWithdrawalsEventsGauge   prometheus.Gauge
}

type Monitor struct {
	log log.Logger

	ctx            context.Context
	l1GethClient   *ethclient.Client
	l2OpGethClient *ethclient.Client
	l2OpNodeClient *ethclient.Client
	l1ChainID      *big.Int
	l2ChainID      *big.Int

	maxBlockRange            uint64
	disputeGameHelper        DisputeGameHelper
	disputeFactoryGameHelper DisputeFactoryGameHelper
	withdrawalHelper         WithdrawalHelper

	state   State
	metrics Metrics

	// Previous values for counters
	previousProcessedWithdrawals     uint64
	previousProcessedGames           uint64
	previousWithdrawalsValidated     uint64
	previousWithdrawalsNotFaultProof uint64
	previousNodeConnectionFailures   uint64
}

func NewMetrics(m metrics.Factory) *Metrics {
	ret := &Metrics{
		NextL1HeightGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "next_l1_height",
			Help:      "Next L1 Height",
		}),
		LatestL1HeightGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "latest_l1_height",
			Help:      "Latest L1 Height",
		}),
		LastNotFaultProofBlockGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "last_not_fault_proof_block",
			Help:      "Last Not Fault Proof Block",
		}),
		ProcessedProvenWithdrawalsEventsCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "processed_provenwithdrawals_events_total",
			Help:      "Total number of processed provebwithdrawals events",
		}),
		ProcessedGamesCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "processed_games_total",
			Help:      "Total number of processed games",
		}),
		IsDetectingForgeriesGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "is_detecting_forgeries",
			Help:      "Is Detecting Forgeries (0 or 1)",
		}),
		WithdrawalsValidatedCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "withdrawals_validated_total",
			Help:      "Total number of withdrawals validated",
		}),
		WithdrawalsNotFaultProofCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "withdrawals_not_fault_proof_total",
			Help:      "Total number of withdrawals not fault-proof",
		}),
		NodeConnectionFailuresCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "node_connection_failures_total",
			Help:      "Total number of node connection failures",
		}),
		ForgeriesWithdrawalsEventsGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "forgeries_withdrawals_events_count",
			Help:      "Number of forgeries withdrawals events",
		}),
		InvalidProposalWithdrawalsEventsGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "invalid_proposal_withdrawals_events_count",
			Help:      "Number of invalid proposal withdrawals events",
		}),
	}

	return ret
}

func (m *Monitor) UpdateMetricsFromState() {

	// Update Gauges
	m.metrics.NextL1HeightGauge.Set(float64(m.state.nextL1Height))
	m.metrics.LatestL1HeightGauge.Set(float64(m.state.latestL1Height))
	m.metrics.LastNotFaultProofBlockGauge.Set(float64(m.state.lastNotFaultProofBlock))

	m.metrics.IsDetectingForgeriesGauge.Set(float64(m.state.isDetectingForgeries))

	m.metrics.ForgeriesWithdrawalsEventsGauge.Set(float64(len(m.state.forgeriesWithdrawalsEvents)))
	m.metrics.InvalidProposalWithdrawalsEventsGauge.Set(float64(len(m.state.invalidProposalWithdrawalsEvents)))

	// Update Counters by calculating deltas
	// Processed Withdrawals
	processedWithdrawalsDelta := m.state.processedProvenWithdrawalsEvents - m.previousProcessedWithdrawals
	if processedWithdrawalsDelta > 0 {
		m.metrics.ProcessedProvenWithdrawalsEventsCounter.Add(float64(processedWithdrawalsDelta))
	}
	m.previousProcessedWithdrawals = m.state.processedProvenWithdrawalsEvents

	// Processed Games
	processedGamesDelta := m.state.processedGames - m.previousProcessedGames
	if processedGamesDelta > 0 {
		m.metrics.ProcessedGamesCounter.Add(float64(processedGamesDelta))
	}
	m.previousProcessedGames = m.state.processedGames

	// Withdrawals Validated
	withdrawalsValidatedDelta := m.state.withdrawalsValidated - m.previousWithdrawalsValidated
	if withdrawalsValidatedDelta > 0 {
		m.metrics.WithdrawalsValidatedCounter.Add(float64(withdrawalsValidatedDelta))
	}
	m.previousWithdrawalsValidated = m.state.withdrawalsValidated

	// Withdrawals Not Fault-Proof
	withdrawalsNotFaultProofDelta := m.state.withdrawalsNotFaultProof - m.previousWithdrawalsNotFaultProof
	if withdrawalsNotFaultProofDelta > 0 {
		m.metrics.WithdrawalsNotFaultProofCounter.Add(float64(withdrawalsNotFaultProofDelta))
	}
	m.previousWithdrawalsNotFaultProof = m.state.withdrawalsNotFaultProof

	// Node Connection Failures
	nodeConnectionFailuresDelta := m.state.nodeConnectionFailures - m.previousNodeConnectionFailures
	if nodeConnectionFailuresDelta > 0 {
		m.metrics.NodeConnectionFailuresCounter.Add(float64(nodeConnectionFailuresDelta))
	}
	m.previousNodeConnectionFailures = m.state.nodeConnectionFailures
}

func NewState(log log.Logger, nextL1Height uint64, latestL1Height uint64) (*State, error) {

	if nextL1Height > latestL1Height {
		log.Info("nextL1Height is greater than latestL1Height, starting from latest", "nextL1Height", nextL1Height, "latestL1Height", latestL1Height)
		nextL1Height = latestL1Height
	}

	ret := State{
		processedProvenWithdrawalsEvents: 0,
		nextL1Height:                     nextL1Height,
		latestL1Height:                   latestL1Height,
		isDetectingForgeries:             0,
		withdrawalsValidated:             0,
		nodeConnectionFailures:           0,
		lastNotFaultProofBlock:           0,
		processedGames:                   0,
	}

	return &ret, nil
}

func (s *State) LogState(log log.Logger) {
	blockToProcess, syncPercentage := s.GetPercentages()

	log.Info("State",
		"withdrawalsValidated", fmt.Sprintf("%d", s.withdrawalsValidated),
		"withdrawalsNotFaultProof", fmt.Sprintf("%d", s.withdrawalsNotFaultProof),
		"nextL1Height", fmt.Sprintf("%d", s.nextL1Height),
		"latestL1Height", fmt.Sprintf("%d", s.latestL1Height),
		"blockToProcess", fmt.Sprintf("%d", blockToProcess),
		"syncPercentage", fmt.Sprintf("%d%%", syncPercentage),
		"lastNotFaultProofBlock", fmt.Sprintf("%d", s.lastNotFaultProofBlock),
		"processedProvenWithdrawalsEvents", fmt.Sprintf("%d", s.processedProvenWithdrawalsEvents),
		"processedGames", fmt.Sprintf("%d", s.processedGames),
		"isDetectingForgeries", fmt.Sprintf("%d", s.isDetectingForgeries),
		"nodeConnectionFailures", fmt.Sprintf("%d", s.nodeConnectionFailures),
		"forgeriesWithdrawalsEvents", fmt.Sprintf("%d", len(s.forgeriesWithdrawalsEvents)),
		"invalidProposalWithdrawalsEvents", fmt.Sprintf("%d", len(s.invalidProposalWithdrawalsEvents)),
	)
}

func (s *State) GetPercentages() (uint64, uint64) {
	blockToProcess := s.latestL1Height - s.nextL1Height
	syncPercentage := uint64(math.Floor(100 - (float64(blockToProcess) / float64(s.latestL1Height) * 100)))
	return blockToProcess, syncPercentage
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

	withdrawalHelper, err := NewWithdrawalHelper(ctx, l1GethClient, l2OpGethClient, cfg.OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	disputeGameHelper, err := NewDisputeGameHelper(ctx, l1GethClient, l2OpNodeClient, withdrawalHelper.GetOptimismPortal2())
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game helper: %w", err)
	}

	disputeFactoryGameHelper, err := NewDisputeGameFactoryHelper(ctx, l1GethClient, withdrawalHelper.GetOptimismPortal2())
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game factory helper: %w", err)
	}

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

		disputeGameHelper:        *disputeGameHelper,
		disputeFactoryGameHelper: *disputeFactoryGameHelper,
		withdrawalHelper:         *withdrawalHelper,

		maxBlockRange: cfg.EventBlockRange,

		state:   *state,
		metrics: *metrics,
	}
	ret.Init()

	return ret, nil
}

func (m *Monitor) Init() {
	m.state.LogState(m.log)
	m.UpdateMetricsFromState()
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

	m.ProcessInvalidWithdrawals()

	events, err := m.withdrawalHelper.GetProvenWithdrawalsEventsIterartor(start, &stop)
	if err != nil {
		m.state.nodeConnectionFailures++
		return
	}

	for events.Next() {
		event := events.Event

		err := m.ProcessEvent(event)
		if err != nil {
			m.log.Error("failed to process event", "error", err)
			return
		}
	}

	// update state
	m.state.nextL1Height = stop

	// log state and metrics
	m.state.LogState(m.log)
	m.UpdateMetricsFromState()

}

// ProcessEvent
// For a given event:
// Retrieve Dispute Games associated with this withdrawal event:
//
//	Calls getDisputeGamesFromWithdrawalhash(event.withdrawalhash) to obtain all dispute games associated with the withdrawal hash.
//
// Process Each Dispute Game:
//
//	  For each dispute game:
//			Validate Game Output Root:
//				Calls isValidOutputRoot(rootClaim, l2BlockNumber) to check if the output root is valid.
//			If Output Root is Valid:
//				Check Withdrawal Existence on L2:
//					Checks if the withdrawal hash exists in the sentMessages mapping of the L2ToL1MessagePasser contract on L2.
//				If Withdrawal Exists:
//					Logs that the withdrawal is valid.
//					Increments the withdrawalsValidated metric.
//				If Withdrawal Does Not Exist:
//					Adds the withdrawal to invalidProofWithdrawals.
//					Logs an error indicating a forgery was detected.
//					Sets forgeryDetected to true.
//			If Output Root is Invalid:
//				Adds the withdrawal to invalidProposalWithdrawalsEvents.
//				Logs a warning indicating an invalid proposal.
//
// Parameters:
// - withdrawalEvent: A pointer to an OptimismPortal2WithdrawalProven event.
//
// Returns:
// - error: An error if any step in the process fails, otherwise nil.
func (m *Monitor) ProcessEvent(withdrawalEvent *l1.OptimismPortal2WithdrawalProven) error {

	m.log.Info("processing event:",
		"WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber))

	games, err := m.getDisputeGamesFromWithdrawalhash(withdrawalEvent.WithdrawalHash)
	if err != nil {
		return fmt.Errorf("failed to get dispute games: %w", err)
	}
	found_at_least_a_game := false
	m.state.processedProvenWithdrawalsEvents++
	for _, game := range games {
		found_at_least_a_game = true
		m.state.processedGames++
		//extra safety check
		if game.l2ChainID.Cmp(m.l2ChainID) != 0 {
			m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", game.l2ChainID))
		}
		validRoot, err := m.disputeGameHelper.IsValidOutputRoot(game.rootClaim, game.l2blockNumber)
		if err != nil {
			return fmt.Errorf("failed to validate output root: %w", err)
		}
		if validRoot {
			m.log.Info("output root is valid for", "game", game.disputeGameProxyAddress.Hex(), "rootClaim", common.BytesToHash(game.rootClaim[:]).Hex(), "l2blockNumber", fmt.Sprintf("%d", game.l2blockNumber))

			withdrawalExists, err := m.withdrawalHelper.WithdrawalExistsOnL2(withdrawalEvent.WithdrawalHash)
			if err != nil {
				return fmt.Errorf("failed to check withdrawal existence on L2: %w", err)
			}
			if withdrawalExists {
				m.log.Info("withdrawal is valid", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", game.disputeGameProxyAddress.Hex())
				m.state.withdrawalsValidated++
			} else {
				m.log.Error("withdrawal is NOT valid, forgery detected", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", game.disputeGameProxyAddress.Hex())
				m.state.isDetectingForgeries++
				m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, ProposalWithdrawalsEvent{
					DisputeGame: &game,
					Event:       withdrawalEvent,
				})
			}
		} else {
			m.log.Warn("output root is invalid for", "game", game.disputeGameProxyAddress.Hex(), "rootClaim", common.BytesToHash(game.rootClaim[:]).Hex(), "l2blockNumber", fmt.Sprintf("%d", game.l2blockNumber))
			m.state.invalidProposalWithdrawalsEvents = append(m.state.invalidProposalWithdrawalsEvents, ProposalWithdrawalsEvent{
				DisputeGame: &game,
				Event:       withdrawalEvent,
			})
		}
	}

	if !found_at_least_a_game {
		m.log.Warn("Faultprood probably not implemented yet at this block as no games have being found for withdrawal", "withdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex())
		m.state.lastNotFaultProofBlock = withdrawalEvent.Raw.BlockNumber
		m.state.withdrawalsNotFaultProof++
	}

	return nil
}

// ProcessInvalidWithdrawals:
// Iterates over invalidProposalWithdrawals to check the status of associated dispute games.
//
//	If Game is Blacklisted:
//		Removes the withdrawal from invalidProposalWithdrawals.
//	If Challenger Wins:
//		Removes the withdrawal from invalidProposalWithdrawals.
//		Logs that the withdrawal was correctly resolved.
//	If Defender Wins:
//		Logs an error indicating a forgery was detected.
//		Sets forgeryDetected to true.
//	If Game In Progress:
//		Logs a warning indicating the dispute game is still ongoing.
//
// Returns an error if any of the checks fail due to node connection issues.
func (m *Monitor) ProcessInvalidWithdrawals() error {

	var newInvalidProposalWithdrawalsEvents []ProposalWithdrawalsEvent = make([]ProposalWithdrawalsEvent, 0)

	for _, proposalWithdrawalsEvent := range m.state.invalidProposalWithdrawalsEvents {
		withdrawalEvent := proposalWithdrawalsEvent.Event
		m.log.Info("processing invalid proposal event:",
			"WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber))

		disputeGame := proposalWithdrawalsEvent.DisputeGame
		blacklisted, err := m.disputeGameHelper.IsGameBlacklisted(disputeGame)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to check if game is blacklisted", "error", err)
			return err
		}

		if blacklisted {
			m.log.Info("game is blacklisted,removing from invalidProposalWithdrawalsEvents list", "game", disputeGame.disputeGameProxyAddress.Hex())
			continue
		}
		inProgress, err := m.disputeGameHelper.IsGameStateINPROGRESS(disputeGame)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to check if game is in progress", "error", err)
			return err
		}
		if inProgress {
			m.log.Warn("game is still in progress", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGame.disputeGameProxyAddress.Hex())
			newInvalidProposalWithdrawalsEvents = append(newInvalidProposalWithdrawalsEvents, proposalWithdrawalsEvent)
		} else {

			challengerWins, err := m.disputeGameHelper.IsGameStateCHALLENGER_WINS(disputeGame)
			if err != nil {
				m.state.nodeConnectionFailures++
				m.log.Error("failed to check if challenger wins", "error", err)
				return err
			}
			if challengerWins {
				m.log.Info("withdrawal was correctly resolved", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGame.disputeGameProxyAddress.Hex())
				m.state.withdrawalsValidated++
				continue
			} else {

				defenderWins, err := m.disputeGameHelper.IsGameStateDEFENDER_WINS(disputeGame)
				if err != nil {
					m.state.nodeConnectionFailures++
					m.log.Error("failed to check if defender wins", "error", err)
					return err
				}
				if defenderWins {
					m.log.Error("withdrawal is NOT valid, forgery detected", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGame.disputeGameProxyAddress.Hex())
					m.state.isDetectingForgeries++
					m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, proposalWithdrawalsEvent)
					continue
				} else {
					m.log.Error("THIS CASE SHOULD NEVER HAPPEN, WE DO NOT KNOW THE STATE OF THE GAME", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGame.disputeGameProxyAddress.Hex())
				}
			}

		}

	}

	m.state.invalidProposalWithdrawalsEvents = newInvalidProposalWithdrawalsEvents

	return nil
}

// getDisputeGamesFromWithdrawalhash retrieves a list of DisputeGame instances associated with a given withdrawal hash.
// It first fetches the submitted proofs data using the provided withdrawal hash, and then iterates over the data to
// obtain the corresponding DisputeGame instances from their proxy addresses.
//
// Parameters:
//   - withdrawalHash: A 32-byte array representing the hash of the withdrawal.
//
// Returns:
//   - A slice of DisputeGame instances associated with the provided withdrawal hash.
//   - An error if there is a failure in fetching the submitted proofs data or the dispute games.
//
// The function increments the nodeConnectionFailures counter in the state if there are any errors during the process.
func (m *Monitor) getDisputeGamesFromWithdrawalhash(withdrawalHash [32]byte) ([]DisputeGame, error) {

	submittedProofsData, error := m.withdrawalHelper.GetSumittedProofsDataFromWithdrawalhash(withdrawalHash)
	if error != nil {
		m.state.nodeConnectionFailures++
		return nil, fmt.Errorf("failed to get games addresses: %w", error)
	}
	gameList := make([]DisputeGame, 0)
	for _, submittedProofData := range submittedProofsData {
		disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
		game, error := m.disputeGameHelper.GetDisputeGameFromAddress(disputeGameProxyAddress)
		if error != nil {
			m.state.nodeConnectionFailures++
			return nil, fmt.Errorf("failed to get games: %w", error)
		}

		gameList = append(gameList, *game)
	}

	return gameList, nil
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
