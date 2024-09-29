package faultproof_withdrawals

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

type State struct {
	nextL1Height   uint64
	latestL1Height uint64

	processedProvenWithdrawalsExtension1Events uint64
	processedGames                             uint64

	isDetectingForgeries uint64
	withdrawalsValidated uint64

	nodeConnectionFailures uint64

	forgeriesWithdrawalsEvents       []EnrichedWithdrawalEvent
	invalidProposalWithdrawalsEvents []EnrichedWithdrawalEvent
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

func (m *Metrics) UpdateMetricsFromState(state *State) {

	// Update Gauges
	m.NextL1HeightGauge.Set(float64(state.nextL1Height))
	m.LatestL1HeightGauge.Set(float64(state.latestL1Height))

	m.IsDetectingForgeriesGauge.Set(float64(state.isDetectingForgeries))

	m.ForgeriesWithdrawalsEventsGauge.Set(float64(len(state.forgeriesWithdrawalsEvents)))
	m.InvalidProposalWithdrawalsEventsGauge.Set(float64(len(state.invalidProposalWithdrawalsEvents)))

	// Update Counters by calculating deltas
	// Processed Withdrawals
	processedWithdrawalsDelta := state.processedProvenWithdrawalsExtension1Events - m.previousProcessedProvenWithdrawalsExtension1Events
	if processedWithdrawalsDelta > 0 {
		m.ProcessedProvenWithdrawalsEventsExtensions1Counter.Add(float64(processedWithdrawalsDelta))
	}
	m.previousProcessedProvenWithdrawalsExtension1Events = state.processedProvenWithdrawalsExtension1Events

	// Processed Games
	processedGamesDelta := state.processedGames - m.previousProcessedGames
	if processedGamesDelta > 0 {
		m.ProcessedGamesCounter.Add(float64(processedGamesDelta))
	}
	m.previousProcessedGames = state.processedGames

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
}
