package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"math"

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
	nextL1Height   uint64
	latestL1Height uint64

	processedProvenWithdrawalsExtension1Events uint64
	processedGames                             uint64

	isDetectingForgeries uint64
	withdrawalsValidated uint64

	nodeConnectionFailures uint64

	forgeriesWithdrawalsEvents       []ProposalWithdrawalsEvent
	invalidProposalWithdrawalsEvents []ProposalWithdrawalsEvent
}

type ProposalWithdrawalsEvent struct {
	DisputeGame *FaultDisputeGameProxy
	Event       *WithdrawalProvenExtension1Event
}

type Metrics struct {
	NextL1HeightGauge                                  prometheus.Gauge
	LatestL1HeightGauge                                prometheus.Gauge
	ProcessedProvenWithdrawalsEventsExtensions1Counter prometheus.Counter
	ProcessedGamesCounter                              prometheus.Counter
	IsDetectingForgeriesGauge                          prometheus.Gauge
	WithdrawalsValidatedCounter                        prometheus.Counter
	NodeConnectionFailuresCounter                      prometheus.Counter
	ForgeriesWithdrawalsEventsGauge                    prometheus.Gauge
	InvalidProposalWithdrawalsEventsGauge              prometheus.Gauge
}

type Monitor struct {
	log log.Logger

	ctx            context.Context
	l1GethClient   *ethclient.Client
	l2OpGethClient *ethclient.Client
	l2OpNodeClient *ethclient.Client
	l1ChainID      *big.Int
	l2ChainID      *big.Int

	maxBlockRange             uint64
	faultDisputeGameHelper    FaultDisputeGameHelper
	optimismPortal2Helper     OptimismPortal2Helper
	l2ToL1MessagePasserHelper L2ToL1MessagePasserHelper
	l2NodeHelper              L2NodeHelper

	state   State
	metrics Metrics

	// Previous values for counters
	previousProcessedProvenWithdrawalsExtension1Events uint64
	previousProcessedGames                             uint64
	previousWithdrawalsValidated                       uint64
	previousNodeConnectionFailures                     uint64
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
		ProcessedProvenWithdrawalsEventsExtensions1Counter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "processed_provenwithdrawalsextension1_events_total",
			Help:      "Total number of processed provenwithdrawalsextension1 events",
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

	m.metrics.IsDetectingForgeriesGauge.Set(float64(m.state.isDetectingForgeries))

	m.metrics.ForgeriesWithdrawalsEventsGauge.Set(float64(len(m.state.forgeriesWithdrawalsEvents)))
	m.metrics.InvalidProposalWithdrawalsEventsGauge.Set(float64(len(m.state.invalidProposalWithdrawalsEvents)))

	// Update Counters by calculating deltas
	// Processed Withdrawals
	processedWithdrawalsDelta := m.state.processedProvenWithdrawalsExtension1Events - m.previousProcessedProvenWithdrawalsExtension1Events
	if processedWithdrawalsDelta > 0 {
		m.metrics.ProcessedProvenWithdrawalsEventsExtensions1Counter.Add(float64(processedWithdrawalsDelta))
	}
	m.previousProcessedProvenWithdrawalsExtension1Events = m.state.processedProvenWithdrawalsExtension1Events

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
		processedProvenWithdrawalsExtension1Events: 0,
		nextL1Height:           nextL1Height,
		latestL1Height:         latestL1Height,
		isDetectingForgeries:   0,
		withdrawalsValidated:   0,
		nodeConnectionFailures: 0,
		processedGames:         0,
	}

	return &ret, nil
}

func (s *State) LogState(log log.Logger) {
	blockToProcess, syncPercentage := s.GetPercentages()

	log.Info("State",
		"withdrawalsValidated", fmt.Sprintf("%d", s.withdrawalsValidated),
		"nextL1Height", fmt.Sprintf("%d", s.nextL1Height),
		"latestL1Height", fmt.Sprintf("%d", s.latestL1Height),
		"blockToProcess", fmt.Sprintf("%d", blockToProcess),
		"syncPercentage", fmt.Sprintf("%d%%", syncPercentage),
		"processedProvenWithdrawalsExtension1Events", fmt.Sprintf("%d", s.processedProvenWithdrawalsExtension1Events),
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

	events, err := m.optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, &stop)
	if err != nil {
		m.state.nodeConnectionFailures++
		return
	}

	for event := range events {
		err := m.ProcessEvent(&events[event])
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
func (m *Monitor) ProcessEvent(withdrawalEvent *WithdrawalProvenExtension1Event) error {

	m.log.Info("processing event:",
		"WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber))

	m.state.processedProvenWithdrawalsExtension1Events++
	game, err := m.getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalEvent.WithdrawalHash, withdrawalEvent.ProofSubmitter)
	if err != nil {
		return fmt.Errorf("failed to get dispute games: %w", err)
	}
	m.state.processedGames++

	//extra safety check
	if game.DisputeGameData.L2ChainID.Cmp(m.l2ChainID) != 0 {
		m.log.Error("l2ChainID mismatch", "expected", fmt.Sprintf("%d", m.l2ChainID), "got", fmt.Sprintf("%d", game.DisputeGameData.L2ChainID))
	}
	validRoot, err := m.l2NodeHelper.IsValidOutputRoot(game.DisputeGameData.RootClaim, game.DisputeGameData.L2blockNumber)
	if err != nil {
		return fmt.Errorf("failed to validate output root: %w", err)
	}
	if validRoot {
		m.log.Info("output root is valid for", "game", game.DisputeGameData.ProxyAddress.Hex(), "rootClaim", common.BytesToHash(game.DisputeGameData.RootClaim[:]).Hex(), "l2blockNumber", fmt.Sprintf("%d", game.DisputeGameData.L2blockNumber))

		withdrawalExists, err := m.l2ToL1MessagePasserHelper.WithdrawalExistsOnL2(withdrawalEvent.WithdrawalHash)
		if err != nil {
			return fmt.Errorf("failed to check withdrawal existence on L2: %w", err)
		}
		if withdrawalExists {
			m.log.Info("withdrawal is valid", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", common.BytesToHash(withdrawalEvent.Raw.TxHash[:]).Hex(), "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", game.DisputeGameData.ProxyAddress.Hex())
			m.state.withdrawalsValidated++
		} else {
			m.log.Error("withdrawal is NOT valid, forgery detected", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", common.BytesToHash(withdrawalEvent.Raw.TxHash[:]).Hex(), "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", game.DisputeGameData.ProxyAddress.Hex())
			m.state.isDetectingForgeries++
			m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, ProposalWithdrawalsEvent{
				DisputeGame: game,
				Event:       withdrawalEvent,
			})
		}
	} else {
		m.log.Warn("output root is invalid for", "game", game.DisputeGameData.ProxyAddress.Hex(), "rootClaim", common.BytesToHash(game.DisputeGameData.RootClaim[:]).Hex(), "l2blockNumber", fmt.Sprintf("%d", game.DisputeGameData.L2blockNumber))
		m.state.invalidProposalWithdrawalsEvents = append(m.state.invalidProposalWithdrawalsEvents, ProposalWithdrawalsEvent{
			DisputeGame: game,
			Event:       withdrawalEvent,
		})
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

		disputeGameProxy := proposalWithdrawalsEvent.DisputeGame
		blacklisted, err := m.optimismPortal2Helper.IsGameBlacklisted(disputeGameProxy)
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to check if game is blacklisted", "error", err)
			return err
		}

		if blacklisted {
			m.log.Info("game is blacklisted,removing from invalidProposalWithdrawalsEvents list", "game", disputeGameProxy.DisputeGameData.ProxyAddress.Hex())
			continue
		}
		//refresh state to make sure is the latest one we have on the game
		err = disputeGameProxy.RefreshState()
		if err != nil {
			m.state.nodeConnectionFailures++
			m.log.Error("failed to refresh game state", "error", err)
			return err
		}

		if disputeGameProxy.DisputeGameData.Status == IN_PROGRESS {
			m.log.Warn("game is still in progress", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGameProxy.DisputeGameData.ProxyAddress.Hex())
			newInvalidProposalWithdrawalsEvents = append(newInvalidProposalWithdrawalsEvents, proposalWithdrawalsEvent)
		} else {

			if disputeGameProxy.DisputeGameData.Status == CHALLENGER_WINS {
				m.log.Info("withdrawal was correctly resolved", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGameProxy.DisputeGameData.ProxyAddress.Hex())
				m.state.withdrawalsValidated++
				continue
			} else {
				if disputeGameProxy.DisputeGameData.Status == DEFENDER_WINS {
					m.log.Error("withdrawal is NOT valid, forgery detected", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGameProxy.DisputeGameData.ProxyAddress.Hex())
					m.state.isDetectingForgeries++
					m.state.forgeriesWithdrawalsEvents = append(m.state.forgeriesWithdrawalsEvents, proposalWithdrawalsEvent)
					continue
				} else {
					m.log.Error("THIS CASE SHOULD NEVER HAPPEN, WE DO NOT KNOW THE STATE OF THE GAME", "WithdrawalHash", common.BytesToHash(withdrawalEvent.WithdrawalHash[:]).Hex(), "eventTx", withdrawalEvent.Raw.TxHash, "eventBlock", fmt.Sprintf("%d", withdrawalEvent.Raw.BlockNumber), "game", disputeGameProxy.DisputeGameData.ProxyAddress.Hex())
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
func (m *Monitor) getDisputeGamesFromWithdrawalhash(withdrawalHash [32]byte) ([]FaultDisputeGameProxy, error) {

	submittedProofsData, error := m.optimismPortal2Helper.GetSumittedProofsDataFromWithdrawalhash(withdrawalHash)
	if error != nil {
		m.state.nodeConnectionFailures++
		return nil, fmt.Errorf("failed to get games addresses: %w", error)
	}
	gameList := make([]FaultDisputeGameProxy, 0)
	for _, submittedProofData := range submittedProofsData {
		disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
		game, error := m.faultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
		if error != nil {
			m.state.nodeConnectionFailures++
			return nil, fmt.Errorf("failed to get games: %w", error)
		}

		gameList = append(gameList, *game)
	}

	return gameList, nil
}

// getDisputeGamesFromWithdrawalhashAndProofSubmitter retrieves a DisputeGame object
// based on the provided withdrawal hash and proof submitter address.
//
// This function performs the following steps:
//  1. Retrieves the proof data related to the withdrawal hash and proof submitter
//     using the `GetSumittedProofsDataFromWithdrawalhashAndProofSubmitterAddress` method.
//  2. Extracts the dispute game proxy address from the retrieved proof data.
//  3. Retrieves the corresponding DisputeGame object using the
//     `GetDisputeGameFromAddress` method with the proxy address.
//
// If any errors occur during these steps, the connection failure counter
// (`nodeConnectionFailures`) is incremented, and the error is returned.
//
// Parameters:
// - withdrawalHash: A 32-byte array representing the withdrawal hash.
// - proofSubmitter: The address of the submitter of the proof.
//
// Returns:
// - A pointer to a DisputeGame object if successful.
// - An error if any issue occurs while fetching the dispute game or proof data.
func (m *Monitor) getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalHash [32]byte, proofSubmitter common.Address) (*FaultDisputeGameProxy, error) {

	submittedProofData, error := m.optimismPortal2Helper.GetSumittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitter)
	if error != nil {
		m.state.nodeConnectionFailures++
		return nil, fmt.Errorf("failed to get games addresses: %w", error)
	}
	disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
	disputeGame, error := m.faultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if error != nil {
		m.state.nodeConnectionFailures++
		return nil, fmt.Errorf("failed to get games: %w", error)
	}

	return disputeGame, nil
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1GethClient.Close()
	m.l2OpGethClient.Close()
	return nil
}
