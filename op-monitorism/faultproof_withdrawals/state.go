package faultproof_withdrawals

import (
	"fmt"
	"math"
	"time"

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
	logger  log.Logger
	metrics *Metrics

	nextL1Height    uint64
	latestL1Height  uint64
	initialL1Height uint64
	latestL2Height  uint64

	eventsProcessed      uint64 // This counts the events that we have taken care of, and we are aware of.
	withdrawalsProcessed uint64 // This counts the withdrawals that have being completed and processed and we are not tracking anymore. eventProcessed >= withdrawalsProcessed. withdrawalsProcessed does not includes potential attacks with games in progress.

	l1NodeConnection         uint64
	l1NodeConnectionFailures uint64

	l2NodeConnection         uint64
	l2NodeConnectionFailures uint64

	latestEventsProcessed          *validator.WithdrawalValidationRef
	latestEventsProcessedTimestamp float64

	// possible attacks detected
	// Forgeries detected on games that are already resolved
	potentialAttackOnDefenderWinsGames          map[common.Hash]*validator.WithdrawalValidationRef
	numberOfPotentialAttacksOnDefenderWinsGames uint64

	// Forgeries detected on games that are still in progress
	// Faultproof system should make them invalid
	potentialAttackOnInProgressGames         map[common.Hash]*validator.WithdrawalValidationRef
	numberOfPotentialAttackOnInProgressGames uint64

	// Suspicious events
	// It is unlikely that someone is going to use a withdrawal hash on a games that resolved with ChallengerWins. If this happens, maybe there is a bug somewhere in the UI used by the users or it is a malicious attack that failed
	suspiciousEventsOnChallengerWinsGames         *lru.Cache
	numberOfSuspiciousEventsOnChallengerWinsGames uint64
}

type Metrics struct {
	InitialL1HeightGauge prometheus.Gauge
	NextL1HeightGauge    prometheus.Gauge
	LatestL1HeightGauge  prometheus.Gauge
	LatestL2HeightGauge  prometheus.Gauge

	EventsProcessedCounter      prometheus.Counter
	WithdrawalsProcessedCounter prometheus.Counter

	L1NodeConnectionFailuresCounter prometheus.Counter
	L2NodeConnectionFailuresCounter prometheus.Counter

	PotentialAttackOnDefenderWinsGamesGauge    prometheus.Gauge
	PotentialAttackOnInProgressGamesGauge      prometheus.Gauge
	SuspiciousEventsOnChallengerWinsGamesGauge prometheus.Gauge

	PotentialAttackOnDefenderWinsGamesGaugeVec    *prometheus.GaugeVec
	PotentialAttackOnInProgressGamesGaugeVec      *prometheus.GaugeVec
	SuspiciousEventsOnChallengerWinsGamesGaugeVec *prometheus.GaugeVec

	// Previous values for counters
	previousEventsProcessed          uint64
	previousWithdrawalsProcessed     uint64
	previousl1NodeConnectionFailures uint64
	previousl2NodeConnectionFailures uint64
}

func NewState(logger log.Logger, nextL1Height uint64, latestL1Height uint64, latestL2Height uint64, m metrics.Factory) (*State, error) {

	if nextL1Height > latestL1Height {
		logger.Info("nextL1Height is greater than latestL1Height, starting from latest", "nextL1Height", nextL1Height, "latestL1Height", latestL1Height)
		nextL1Height = latestL1Height
	}

	ret := State{
		potentialAttackOnDefenderWinsGames:          make(map[common.Hash]*validator.WithdrawalValidationRef),
		numberOfPotentialAttacksOnDefenderWinsGames: 0,
		suspiciousEventsOnChallengerWinsGames: func() *lru.Cache {
			cache, err := lru.New(suspiciousEventsOnChallengerWinsGamesCacheSize)
			if err != nil {
				logger.Error("Failed to create LRU cache", "error", err)
				return nil
			}
			return cache
		}(),
		numberOfSuspiciousEventsOnChallengerWinsGames: 0,

		potentialAttackOnInProgressGames:         make(map[common.Hash]*validator.WithdrawalValidationRef),
		numberOfPotentialAttackOnInProgressGames: 0,

		eventsProcessed: 0,

		withdrawalsProcessed:     0,
		l1NodeConnectionFailures: 0,
		l2NodeConnectionFailures: 0,

		nextL1Height:    nextL1Height,
		latestL1Height:  latestL1Height,
		initialL1Height: nextL1Height,
		latestL2Height:  latestL2Height,
		logger:          logger,
		metrics:         NewMetrics(m),
	}

	return &ret, nil
}

func (s *State) LogState() {
	blockToProcess, syncPercentage := s.GetPercentages()

	s.logger.Info("STATE:",
		"withdrawalsProcessed", fmt.Sprintf("%d", s.withdrawalsProcessed),

		"initialL1Height", fmt.Sprintf("%d", s.initialL1Height),
		"nextL1Height", fmt.Sprintf("%d", s.nextL1Height),
		"latestL1Height", fmt.Sprintf("%d", s.latestL1Height),
		"latestL2Height", fmt.Sprintf("%d", s.latestL2Height),
		"blockToProcess", fmt.Sprintf("%d", blockToProcess),
		"syncPercentage", fmt.Sprintf("%d%%", syncPercentage),

		"eventsProcessed", fmt.Sprintf("%d", s.eventsProcessed),

		"l1NodeConnectionFailures", fmt.Sprintf("%d", s.l1NodeConnectionFailures),
		"l1NodeConnection", fmt.Sprintf("%d", s.l1NodeConnection),
		"l2NodeConnectionFailures", fmt.Sprintf("%d", s.l2NodeConnectionFailures),
		"l2NodeConnection", fmt.Sprintf("%d", s.l2NodeConnection),

		"latestEventsProcessed", fmt.Sprintf("%v", s.latestEventsProcessed),
		"latestEventsProcessedTimestamp", fmt.Sprintf("%f", s.latestEventsProcessedTimestamp),

		"potentialAttackOnDefenderWinsGames", fmt.Sprintf("%d", s.numberOfPotentialAttacksOnDefenderWinsGames),
		"potentialAttackOnInProgressGames", fmt.Sprintf("%d", s.numberOfPotentialAttackOnInProgressGames),
		"suspiciousEventsOnChallengerWinsGames", fmt.Sprintf("%d", s.numberOfSuspiciousEventsOnChallengerWinsGames),
	)

	s.metrics.UpdateMetricsFromState(s)
}

func (s *State) IncrementWithdrawalsValidated(withdrawalValidationRef *validator.WithdrawalValidationRef) {
	s.withdrawalsProcessed++
	s.latestEventsProcessed = withdrawalValidationRef
	s.latestEventsProcessedTimestamp = float64(time.Now().Unix())
}

func (s *State) IncrementPotentialAttackOnDefenderWinsGames(withdrawalValidationRef *validator.WithdrawalValidationRef) {
	key := withdrawalValidationRef.DisputeGameEvent.EventRef.TxHash

	s.potentialAttackOnDefenderWinsGames[key] = withdrawalValidationRef
	s.numberOfPotentialAttacksOnDefenderWinsGames++

	if _, ok := s.potentialAttackOnInProgressGames[key]; ok {
		delete(s.potentialAttackOnInProgressGames, key)
		s.numberOfPotentialAttackOnInProgressGames--
	}

	s.latestEventsProcessed = withdrawalValidationRef
	s.latestEventsProcessedTimestamp = float64(time.Now().Unix())
	s.withdrawalsProcessed++

}

func (s *State) IncrementPotentialAttackOnInProgressGames(withdrawalValidationRef *validator.WithdrawalValidationRef) {
	key := withdrawalValidationRef.DisputeGameEvent.EventRef.TxHash
	// check if key already exists
	if _, ok := s.potentialAttackOnInProgressGames[key]; !ok {
		s.numberOfPotentialAttackOnInProgressGames++
		s.latestEventsProcessed = withdrawalValidationRef
		s.latestEventsProcessedTimestamp = float64(time.Now().Unix())
	}

	// eventually update the map with the new withdrawalValidationRef
	s.potentialAttackOnInProgressGames[key] = withdrawalValidationRef
}

func (s *State) IncrementSuspiciousEventsOnChallengerWinsGames(withdrawalValidationRef *validator.WithdrawalValidationRef) {
	key := withdrawalValidationRef.DisputeGameEvent.EventRef.TxHash

	s.suspiciousEventsOnChallengerWinsGames.Add(key, withdrawalValidationRef)
	s.numberOfSuspiciousEventsOnChallengerWinsGames++

	if _, ok := s.potentialAttackOnInProgressGames[key]; ok {
		delete(s.potentialAttackOnInProgressGames, key)
		s.numberOfPotentialAttackOnInProgressGames--
	}

	s.withdrawalsProcessed++
	s.latestEventsProcessed = withdrawalValidationRef
	s.latestEventsProcessedTimestamp = float64(time.Now().Unix())
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

func (m *Metrics) String() string {
	initialL1HeightGaugeValue, _ := GetGaugeValue(m.InitialL1HeightGauge)
	nextL1HeightGaugeValue, _ := GetGaugeValue(m.NextL1HeightGauge)
	latestL1HeightGaugeValue, _ := GetGaugeValue(m.LatestL1HeightGauge)
	latestL2HeightGaugeValue, _ := GetGaugeValue(m.LatestL2HeightGauge)

	withdrawalsProcessedCounterValue, _ := GetCounterValue(m.WithdrawalsProcessedCounter)
	eventsProcessedCounterValue, _ := GetCounterValue(m.EventsProcessedCounter)

	l1NodeConnectionFailuresCounterValue, _ := GetCounterValue(m.L1NodeConnectionFailuresCounter)
	l2NodeConnectionFailuresCounterValue, _ := GetCounterValue(m.L2NodeConnectionFailuresCounter)

	potentialAttackOnDefenderWinsGamesGaugeValue, _ := GetGaugeValue(m.PotentialAttackOnDefenderWinsGamesGauge)
	potentialAttackOnInProgressGamesGaugeValue, _ := GetGaugeValue(m.PotentialAttackOnInProgressGamesGauge)

	forgeriesWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.PotentialAttackOnDefenderWinsGamesGaugeVec, prometheus.Labels{})
	invalidProposalWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.PotentialAttackOnInProgressGamesGaugeVec, prometheus.Labels{})

	return fmt.Sprintf(
		"InitialL1HeightGauge: %d\nNextL1HeightGauge: %d\nLatestL1HeightGauge: %d\n latestL2HeightGaugeValue: %d\n eventsProcessedCounterValue: %d\nwithdrawalsProcessedCounterValue: %d\nl1NodeConnectionFailuresCounterValue: %d\nl2NodeConnectionFailuresCounterValue: %d \n potentialAttackOnDefenderWinsGamesGaugeValue: %d\n potentialAttackOnInProgressGamesGaugeValue: %d\n  forgeriesWithdrawalsEventsGaugeVecValue: %d\n invalidProposalWithdrawalsEventsGaugeVecValue: %d\n previousEventsProcessed: %d\n previousWithdrawalsProcessed: %d\n previousl1NodeConnectionFailures: %d\n previousl2NodeConnectionFailures: %d",
		uint64(initialL1HeightGaugeValue),
		uint64(nextL1HeightGaugeValue),
		uint64(latestL1HeightGaugeValue),
		uint64(latestL2HeightGaugeValue),
		uint64(eventsProcessedCounterValue),
		uint64(withdrawalsProcessedCounterValue),
		uint64(l1NodeConnectionFailuresCounterValue),
		uint64(l2NodeConnectionFailuresCounterValue),
		uint64(potentialAttackOnDefenderWinsGamesGaugeValue),
		uint64(potentialAttackOnInProgressGamesGaugeValue),
		uint64(forgeriesWithdrawalsEventsGaugeVecValue),
		uint64(invalidProposalWithdrawalsEventsGaugeVecValue),
		m.previousEventsProcessed,
		m.previousWithdrawalsProcessed,
		m.previousl1NodeConnectionFailures,
		m.previousl2NodeConnectionFailures,
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
		LatestL2HeightGauge: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "latest_l2_height",
			Help:      "Latest L2 Height",
		}),
		EventsProcessedCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "events_processed_total",
			Help:      "Total number of events processed",
		}),
		WithdrawalsProcessedCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "withdrawals_processed_total",
			Help:      "Total number of withdrawals processed",
		}),
		L1NodeConnectionFailuresCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "l1_node_connection_failures_total",
			Help:      "Total number of L1 node connection failures",
		}),
		L2NodeConnectionFailuresCounter: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "l2_node_connection_failures_total",
			Help:      "Total number of L2 node connection failures",
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
	m.LatestL2HeightGauge.Set(float64(state.latestL2Height))

	m.PotentialAttackOnDefenderWinsGamesGauge.Set(float64(state.numberOfPotentialAttacksOnDefenderWinsGames))
	m.PotentialAttackOnInProgressGamesGauge.Set(float64(state.numberOfPotentialAttackOnInProgressGames))
	m.SuspiciousEventsOnChallengerWinsGamesGauge.Set(float64(state.numberOfSuspiciousEventsOnChallengerWinsGames))

	// Update Counters by calculating deltas
	// Processed Withdrawals
	eventsProcessedDelta := state.eventsProcessed - m.previousEventsProcessed
	if eventsProcessedDelta > 0 {
		m.EventsProcessedCounter.Add(float64(eventsProcessedDelta))
	}
	m.previousEventsProcessed = state.eventsProcessed

	// Withdrawals Validated
	withdrawalsProcessedDelta := state.withdrawalsProcessed - m.previousWithdrawalsProcessed
	if withdrawalsProcessedDelta > 0 {
		m.WithdrawalsProcessedCounter.Add(float64(withdrawalsProcessedDelta))
	}
	m.previousWithdrawalsProcessed = state.withdrawalsProcessed

	// Node Connection Failures
	l1NodeConnectionFailuresDelta := state.l1NodeConnectionFailures - m.previousl1NodeConnectionFailures
	if l1NodeConnectionFailuresDelta > 0 {
		m.L1NodeConnectionFailuresCounter.Add(float64(l1NodeConnectionFailuresDelta))
	}
	m.previousl1NodeConnectionFailures = state.l1NodeConnectionFailures

	l2NodeConnectionFailuresDelta := state.l2NodeConnectionFailures - m.previousl2NodeConnectionFailures
	if l2NodeConnectionFailuresDelta > 0 {
		m.L2NodeConnectionFailuresCounter.Add(float64(l2NodeConnectionFailuresDelta))
	}
	m.previousl2NodeConnectionFailures = state.l2NodeConnectionFailures

	// Clear the previous values
	m.PotentialAttackOnDefenderWinsGamesGaugeVec.Reset()

	// Update metrics for forgeries withdrawals events
	for _, event := range state.potentialAttackOnDefenderWinsGames {
		withdrawalHash := common.BytesToHash(event.DisputeGameEvent.EventRef.WithdrawalHash[:]).Hex()
		proofSubmitter := event.DisputeGameEvent.EventRef.ProofSubmitter.String()
		status := event.DisputeGameEvent.DisputeGame.GameStatus.String()

		m.PotentialAttackOnDefenderWinsGamesGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.DisputeGameEvent.DisputeGame.IsGameBlacklisted),
			fmt.Sprintf("%v", event.WithdrawalPresentOnL2ToL1MessagePasser),
			fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockNumber),
			fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockTime),
			event.String(),
		).Set(1)
	}

	// Clear the previous values
	m.PotentialAttackOnInProgressGamesGaugeVec.Reset()

	// Update metrics for invalid proposal withdrawals events
	for _, event := range state.potentialAttackOnInProgressGames {
		withdrawalHash := common.BytesToHash(event.DisputeGameEvent.EventRef.WithdrawalHash[:]).Hex()
		proofSubmitter := event.DisputeGameEvent.EventRef.ProofSubmitter.String()
		status := event.DisputeGameEvent.DisputeGame.GameStatus.String()

		m.PotentialAttackOnInProgressGamesGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.DisputeGameEvent.DisputeGame.IsGameBlacklisted),
			fmt.Sprintf("%v", event.WithdrawalPresentOnL2ToL1MessagePasser),
			fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockNumber),
			fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockTime),
			event.String(),
		).Set(1)
	}

	// Clear the previous values
	m.SuspiciousEventsOnChallengerWinsGamesGaugeVec.Reset()
	// Update metrics for invalid proposal withdrawals events
	for _, key := range state.suspiciousEventsOnChallengerWinsGames.Keys() {
		enrichedEvent, ok := state.suspiciousEventsOnChallengerWinsGames.Get(key)
		if ok {
			event := enrichedEvent.(*validator.WithdrawalValidationRef)
			withdrawalHash := common.BytesToHash(event.DisputeGameEvent.EventRef.WithdrawalHash[:]).Hex()
			proofSubmitter := event.DisputeGameEvent.EventRef.ProofSubmitter.String()
			status := event.DisputeGameEvent.DisputeGame.GameStatus.String()

			m.PotentialAttackOnInProgressGamesGaugeVec.WithLabelValues(
				withdrawalHash,
				proofSubmitter,
				status,
				fmt.Sprintf("%v", event.DisputeGameEvent.DisputeGame.IsGameBlacklisted),
				fmt.Sprintf("%v", event.WithdrawalPresentOnL2ToL1MessagePasser),
				fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockNumber),
				fmt.Sprintf("%v", event.DisputeGameEvent.EventRef.BlockInfo.BlockTime),
				event.String(),
			).Set(1)

		}
	}
}
