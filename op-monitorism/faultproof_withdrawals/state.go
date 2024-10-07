package faultproof_withdrawals

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const (
	suspiciousEventsOnChallengerWinsGamesCacheSize = 1000
)

type State struct {
	logger          log.Logger
	nextL1Height    uint64
	latestL1Height  uint64
	initialL1Height uint64

	processedProvenWithdrawalsExtension1Events uint64

	withdrawalsValidated uint64

	nodeConnectionFailures uint64

	// possible attacks detected

	// Forgeries detected on games that are already resolved
	potentialAttackOnDefenderWinsGames         map[common.Hash]validator.EnrichedProvenWithdrawalEvent
	numberOfPotentialAttackOnDefenderWinsGames uint64

	// Forgeries detected on games that are still in progress
	// Because games are still in progress and Faulproof system should make them invalid
	potentialAttackOnInProgressGames         map[common.Hash]validator.EnrichedProvenWithdrawalEvent
	numberOfPotentialAttackOnInProgressGames uint64

	// Suspicious events
	// It is unlikely that someone is going to use a withdrawal hash on a games that resolved with ChallengerWins. If this happens, maybe there is a bug somewhere in the UI used by the users or it is a malicious attack that failed
	suspiciousEventsOnChallengerWinsGames         *lru.Cache
	numberOFSuspiciousEventsOnChallengerWinsGames uint64
}

func NewState(logger log.Logger, nextL1Height uint64, latestL1Height uint64) (*State, error) {

	if nextL1Height > latestL1Height {
		logger.Info("nextL1Height is greater than latestL1Height, starting from latest", "nextL1Height", nextL1Height, "latestL1Height", latestL1Height)
		nextL1Height = latestL1Height
	}

	ret := State{
		potentialAttackOnDefenderWinsGames:         make(map[common.Hash]validator.EnrichedProvenWithdrawalEvent),
		numberOfPotentialAttackOnDefenderWinsGames: 0,
		suspiciousEventsOnChallengerWinsGames: func() *lru.Cache {
			cache, err := lru.New(suspiciousEventsOnChallengerWinsGamesCacheSize)
			if err != nil {
				logger.Error("Failed to create LRU cache", "error", err)
				return nil
			}
			return cache
		}(),
		numberOFSuspiciousEventsOnChallengerWinsGames: 0,

		potentialAttackOnInProgressGames:         make(map[common.Hash]validator.EnrichedProvenWithdrawalEvent),
		numberOfPotentialAttackOnInProgressGames: 0,

		processedProvenWithdrawalsExtension1Events: 0,

		withdrawalsValidated:   0,
		nodeConnectionFailures: 0,

		nextL1Height:    nextL1Height,
		latestL1Height:  latestL1Height,
		initialL1Height: nextL1Height,
		logger:          logger,
	}

	return &ret, nil
}

func (s *State) LogState() {
	blockToProcess, syncPercentage := s.GetPercentages()

	s.logger.Info("STATE:",
		"withdrawalsValidated", fmt.Sprintf("%d", s.withdrawalsValidated),

		"initialL1Height", fmt.Sprintf("%d", s.initialL1Height),
		"nextL1Height", fmt.Sprintf("%d", s.nextL1Height),
		"latestL1Height", fmt.Sprintf("%d", s.latestL1Height),
		"blockToProcess", fmt.Sprintf("%d", blockToProcess),
		"syncPercentage", fmt.Sprintf("%d%%", syncPercentage),

		"processedProvenWithdrawalsExtension1Events", fmt.Sprintf("%d", s.processedProvenWithdrawalsExtension1Events),
		"numberOfDetectedForgery", fmt.Sprintf("%d", s.numberOfPotentialAttackOnDefenderWinsGames),
		"numberOfInvalidWithdrawals", fmt.Sprintf("%d", s.numberOfPotentialAttackOnInProgressGames),
		"nodeConnectionFailures", fmt.Sprintf("%d", s.nodeConnectionFailures),

		"potentialAttackOnDefenderWinsGames", fmt.Sprintf("%d", s.numberOFSuspiciousEventsOnChallengerWinsGames),
		"potentialAttackOnInProgressGames", fmt.Sprintf("%d", s.numberOfPotentialAttackOnInProgressGames),
		"suspiciousEventsOnChallengerWinsGames", fmt.Sprintf("%d", s.numberOFSuspiciousEventsOnChallengerWinsGames),
	)
}

func (s *State) IncrementWithdrawalsValidated(enrichedWithdrawalEvent validator.EnrichedProvenWithdrawalEvent) {
	s.logger.Info("STATE WITHDRAWAL: valid", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
	s.withdrawalsValidated++
}

func (s *State) IncrementPotentialAttackOnDefenderWinsGames(enrichedWithdrawalEvent validator.EnrichedProvenWithdrawalEvent) {
	key := enrichedWithdrawalEvent.Event.Raw.TxHash

	s.logger.Error("STATE WITHDRAWAL: is NOT valid, forgery detected", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
	s.potentialAttackOnDefenderWinsGames[key] = enrichedWithdrawalEvent
	s.numberOfPotentialAttackOnDefenderWinsGames++

	if _, ok := s.potentialAttackOnInProgressGames[key]; ok {
		s.logger.Error("STATE WITHDRAWAL: added to potential attacks. Removing from inProgress", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		delete(s.potentialAttackOnInProgressGames, key)
	}
}

func (s *State) IncrementPotentialAttackOnInProgressGames(enrichedWithdrawalEvent validator.EnrichedProvenWithdrawalEvent) {
	key := enrichedWithdrawalEvent.Event.Raw.TxHash
	// check if key already exists
	if _, ok := s.potentialAttackOnInProgressGames[key]; ok {
		s.logger.Error("STATE WITHDRAWAL:is NOT valid, game is still in progress", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
	} else {
		s.logger.Error("STATE WITHDRAWAL:is NOT valid, game is still in progress. New game found In Progress", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		s.numberOfPotentialAttackOnInProgressGames++
	}

	// eventually update the map with the new enrichedWithdrawalEvent
	s.potentialAttackOnInProgressGames[key] = enrichedWithdrawalEvent
}

func (s *State) IncrementSuspiciousEventsOnChallengerWinsGames(enrichedWithdrawalEvent validator.EnrichedProvenWithdrawalEvent) {
	key := enrichedWithdrawalEvent.Event.Raw.TxHash

	s.logger.Error("STATE WITHDRAWAL:is NOT valid, is NOT valid, but the game is correctly resolved", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
	s.suspiciousEventsOnChallengerWinsGames.Add(key, enrichedWithdrawalEvent)
	s.numberOFSuspiciousEventsOnChallengerWinsGames++

	if _, ok := s.potentialAttackOnInProgressGames[key]; ok {
		s.logger.Error("STATE WITHDRAWAL: added to suspicious attacks. Removing from inProgress", "enrichedWithdrawalEvent", enrichedWithdrawalEvent)
		delete(s.potentialAttackOnInProgressGames, key)
	}
}

func (s *State) GetPercentages() (uint64, uint64) {
	blockToProcess := s.latestL1Height - s.nextL1Height
	divisor := float64(s.latestL1Height) * 100
	//checking to avoid division by 0
	if divisor == 0 {
		return 0, 0
	}
	syncPercentage := uint64(math.Floor(100 - (float64(blockToProcess) / divisor)))
	return blockToProcess, syncPercentage
}

type Metrics struct {
	InitialL1HeightGauge                               prometheus.Gauge
	NextL1HeightGauge                                  prometheus.Gauge
	LatestL1HeightGauge                                prometheus.Gauge
	ProcessedProvenWithdrawalsEventsExtensions1Counter prometheus.Counter
	NumberOfDetectedForgeryGauge                       prometheus.Gauge
	NumberOfInvalidWithdrawalsGauge                    prometheus.Gauge
	WithdrawalsValidatedCounter                        prometheus.Counter
	NodeConnectionFailuresCounter                      prometheus.Counter
	PotentialAttackOnDefenderWinsGamesGauge            prometheus.Gauge
	PotentialAttackOnInProgressGamesGauge              prometheus.Gauge
	SuspiciousEventsOnChallengerWinsGamesGauge         prometheus.Gauge
	PotentialAttackOnDefenderWinsGamesGaugeVec         *prometheus.GaugeVec
	PotentialAttackOnInProgressGamesGaugeVec           *prometheus.GaugeVec
	SuspiciousEventsOnChallengerWinsGamesGaugeVec      *prometheus.GaugeVec

	// Previous values for counters
	previousProcessedProvenWithdrawalsExtension1Events uint64
	previousWithdrawalsValidated                       uint64
	previousNodeConnectionFailures                     uint64
}

func (m *Metrics) String() string {
	initialL1HeightGaugeValue, _ := GetGaugeValue(m.InitialL1HeightGauge)
	nextL1HeightGaugeValue, _ := GetGaugeValue(m.NextL1HeightGauge)
	latestL1HeightGaugeValue, _ := GetGaugeValue(m.LatestL1HeightGauge)
	processedProvenWithdrawalsEventsExtensions1CounterValue, _ := GetCounterValue(m.ProcessedProvenWithdrawalsEventsExtensions1Counter)
	numberOfDetectedForgeryGaugeValue, _ := GetGaugeValue(m.NumberOfDetectedForgeryGauge)
	numberOfInvalidWithdrawalsGaugeValue, _ := GetGaugeValue(m.NumberOfInvalidWithdrawalsGauge)
	withdrawalsValidatedCounterValue, _ := GetCounterValue(m.WithdrawalsValidatedCounter)
	nodeConnectionFailuresCounterValue, _ := GetCounterValue(m.NodeConnectionFailuresCounter)
	forgeriesWithdrawalsEventsGaugeValue, _ := GetGaugeValue(m.PotentialAttackOnDefenderWinsGamesGauge)
	invalidProposalWithdrawalsEventsGaugeValue, _ := GetGaugeValue(m.PotentialAttackOnInProgressGamesGauge)

	forgeriesWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.PotentialAttackOnDefenderWinsGamesGaugeVec, prometheus.Labels{})
	invalidProposalWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.PotentialAttackOnInProgressGamesGaugeVec, prometheus.Labels{})

	return fmt.Sprintf(
		"InitialL1HeightGauge: %d\nNextL1HeightGauge: %d\nLatestL1HeightGauge: %d\nProcessedProvenWithdrawalsEventsExtensions1Counter: %d\nNumberOfDetectedForgeryGauge: %d\nNumberOfInvalidWithdrawalsGauge: %d\nWithdrawalsValidatedCounter: %d\nNodeConnectionFailuresCounter: %d\nForgeriesWithdrawalsEventsGauge: %d\nInvalidProposalWithdrawalsEventsGauge: %d\nForgeriesWithdrawalsEventsGaugeVec: %d\nInvalidProposalWithdrawalsEventsGaugeVec: %d\npreviousProcessedProvenWithdrawalsExtension1Events: %d\npreviousWithdrawalsValidated: %d\npreviousNodeConnectionFailures: %d",
		uint64(initialL1HeightGaugeValue),
		uint64(nextL1HeightGaugeValue),
		uint64(latestL1HeightGaugeValue),
		uint64(processedProvenWithdrawalsEventsExtensions1CounterValue),
		uint64(numberOfDetectedForgeryGaugeValue),
		uint64(numberOfInvalidWithdrawalsGaugeValue),
		uint64(withdrawalsValidatedCounterValue),
		uint64(nodeConnectionFailuresCounterValue),
		uint64(forgeriesWithdrawalsEventsGaugeValue),
		uint64(invalidProposalWithdrawalsEventsGaugeValue),
		uint64(forgeriesWithdrawalsEventsGaugeVecValue),
		uint64(invalidProposalWithdrawalsEventsGaugeVecValue),
		m.previousProcessedProvenWithdrawalsExtension1Events,
		m.previousWithdrawalsValidated,
		m.previousNodeConnectionFailures,
	)
}

// Generic function to get the value of any prometheus.Counter
func GetCounterValue(counter prometheus.Counter) (float64, error) {
	metric := &dto.Metric{}
	err := counter.Write(metric)
	if err != nil {
		return 0, err
	}
	return metric.GetCounter().GetValue(), nil
}

// Generic function to get the value of any prometheus.Gauge
func GetGaugeValue(gauge prometheus.Gauge) (float64, error) {
	metric := &dto.Metric{}
	err := gauge.Write(metric)
	if err != nil {
		return 0, err
	}
	return metric.GetGauge().GetValue(), nil
}

// Function to get the value of a specific Gauge within a GaugeVec
func GetGaugeVecValue(gaugeVec *prometheus.GaugeVec, labels prometheus.Labels) (float64, error) {
	gauge, err := gaugeVec.GetMetricWith(labels)
	if err != nil {
		return 0, err
	}

	metric := &dto.Metric{}
	err = gauge.Write(metric)
	if err != nil {
		return 0, err
	}
	return metric.GetGauge().GetValue(), nil
}

func NewMetrics(m metrics.Factory) *Metrics {
	ret := &Metrics{
		InitialL1HeightGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "initial_l1_height",
			Help:      "Initial L1 Height",
		}),
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
		NumberOfDetectedForgeryGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "number_of_detected_forgeries",
			Help:      "Number of detected forgeries",
		}),
		NumberOfInvalidWithdrawalsGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "number_of_invalid_withdrawals",
			Help:      "Number of invalid withdrawals",
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
		PotentialAttackOnDefenderWinsGamesGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "potential_attack_on_defender_wins_games_count",
			Help:      "Number of potential attacks on defender wins games",
		}),
		PotentialAttackOnInProgressGamesGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "potential_attack_on_in_progress_games_count",
			Help:      "Number of potential attacks on in progress games",
		}),
		SuspiciousEventsOnChallengerWinsGamesGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "suspicious_events_on_challenger_wins_games_count",
			Help:      "Number of suspicious events on challenger wins games",
		}),
		PotentialAttackOnDefenderWinsGamesGaugeVec: m.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricsNamespace,
				Name:      "potential_attack_on_defender_wins_games_gauge_vec",
				Help:      "Information about potential attacks on defender wins games.",
			},
			[]string{"withdrawal_hash", "proof_submitter", "status", "blacklisted", "withdrawal_hash_present", "enriched", "event_block_number", "event_tx_hash"},
		),
		PotentialAttackOnInProgressGamesGaugeVec: m.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricsNamespace,
				Name:      "potential_attack_on_in_progress_games_gauge_vec",
				Help:      "Information about potential attacks on in progress games.",
			},
			[]string{"withdrawal_hash", "proof_submitter", "status", "blacklisted", "withdrawal_hash_present", "enriched", "event_block_number", "event_tx_hash"},
		),
		SuspiciousEventsOnChallengerWinsGamesGaugeVec: m.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricsNamespace,
				Name:      "suspicious_events_on_challenger_wins_games_info",
				Help:      "Information about suspicious events on challenger wins games.",
			},
			[]string{"withdrawal_hash", "proof_submitter", "status", "blacklisted", "withdrawal_hash_present", "enriched", "event_block_number", "event_tx_hash"},
		),
	}

	return ret
}

func (m *Metrics) UpdateMetricsFromState(state *State) {

	// Update Gauges
	m.InitialL1HeightGauge.Set(float64(state.initialL1Height))
	m.NextL1HeightGauge.Set(float64(state.nextL1Height))
	m.LatestL1HeightGauge.Set(float64(state.latestL1Height))

	m.NumberOfDetectedForgeryGauge.Set(float64(state.numberOfPotentialAttackOnDefenderWinsGames))
	m.NumberOfInvalidWithdrawalsGauge.Set(float64(state.numberOfPotentialAttackOnInProgressGames))
	m.PotentialAttackOnDefenderWinsGamesGauge.Set(float64(state.numberOfPotentialAttackOnDefenderWinsGames))
	m.PotentialAttackOnInProgressGamesGauge.Set(float64(state.numberOfPotentialAttackOnInProgressGames))
	m.SuspiciousEventsOnChallengerWinsGamesGauge.Set(float64(state.numberOFSuspiciousEventsOnChallengerWinsGames))

	// Update Counters by calculating deltas
	// Processed Withdrawals
	processedWithdrawalsDelta := state.processedProvenWithdrawalsExtension1Events - m.previousProcessedProvenWithdrawalsExtension1Events
	if processedWithdrawalsDelta > 0 {
		m.ProcessedProvenWithdrawalsEventsExtensions1Counter.Add(float64(processedWithdrawalsDelta))
	}
	m.previousProcessedProvenWithdrawalsExtension1Events = state.processedProvenWithdrawalsExtension1Events

	// Withdrawals Validated
	withdrawalsValidatedDelta := state.withdrawalsValidated - m.previousWithdrawalsValidated
	if withdrawalsValidatedDelta > 0 {
		m.WithdrawalsValidatedCounter.Add(float64(withdrawalsValidatedDelta))
	}
	m.previousWithdrawalsValidated = state.withdrawalsValidated

	// Node Connection Failures
	nodeConnectionFailuresDelta := state.nodeConnectionFailures - m.previousNodeConnectionFailures
	if nodeConnectionFailuresDelta > 0 {
		m.NodeConnectionFailuresCounter.Add(float64(nodeConnectionFailuresDelta))
	}
	m.previousNodeConnectionFailures = state.nodeConnectionFailures

	// Update metrics for forgeries withdrawals events
	for _, event := range state.potentialAttackOnDefenderWinsGames {
		withdrawalHash := common.BytesToHash(event.Event.WithdrawalHash[:]).Hex()
		proofSubmitter := event.Event.ProofSubmitter.String()
		status := event.DisputeGame.DisputeGameData.Status.String()

		m.PotentialAttackOnDefenderWinsGamesGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.Blacklisted),
			fmt.Sprintf("%v", event.WithdrawalHashPresentOnL2),
			fmt.Sprintf("%v", event.Enriched),
			fmt.Sprintf("%v", event.Event.Raw.BlockNumber),
			event.Event.Raw.TxHash.String(),
		).Set(1) // Set a value  for existence
	}

	// Clear the previous values
	m.PotentialAttackOnInProgressGamesGaugeVec.Reset()

	// Update metrics for invalid proposal withdrawals events
	for _, event := range state.potentialAttackOnInProgressGames {
		withdrawalHash := common.BytesToHash(event.Event.WithdrawalHash[:]).Hex()
		proofSubmitter := event.Event.ProofSubmitter.String()
		status := event.DisputeGame.DisputeGameData.Status.String()

		m.PotentialAttackOnInProgressGamesGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.Blacklisted),
			fmt.Sprintf("%v", event.WithdrawalHashPresentOnL2),
			fmt.Sprintf("%v", event.Enriched),
			fmt.Sprintf("%v", event.Event.Raw.BlockNumber),
			event.Event.Raw.TxHash.String(),
		).Set(1) // Set a value  for existence
	}

	// Clear the previous values
	// m.SuspiciousEventsOnChallengerWinsGamesGaugeVec.Reset()
	// Update metrics for invalid proposal withdrawals events
	for key := range state.suspiciousEventsOnChallengerWinsGames.Keys() {
		enrichedEvent, ok := state.suspiciousEventsOnChallengerWinsGames.Get(key)
		if ok {
			event := enrichedEvent.(validator.EnrichedProvenWithdrawalEvent)

			withdrawalHash := common.BytesToHash(event.Event.WithdrawalHash[:]).Hex()
			proofSubmitter := event.Event.ProofSubmitter.String()
			status := event.DisputeGame.DisputeGameData.Status.String()

			m.PotentialAttackOnInProgressGamesGaugeVec.WithLabelValues(
				withdrawalHash,
				proofSubmitter,
				status,
				fmt.Sprintf("%v", event.Blacklisted),
				fmt.Sprintf("%v", event.WithdrawalHashPresentOnL2),
				fmt.Sprintf("%v", event.Enriched),
				fmt.Sprintf("%v", event.Event.Raw.BlockNumber),
				event.Event.Raw.TxHash.String(),
			).Set(1) // Set a value  for existence
		}
	}

}
