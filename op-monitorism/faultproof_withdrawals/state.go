package faultproof_withdrawals

import (
	"fmt"
	"math"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type State struct {
	nextL1Height    uint64
	latestL1Height  uint64
	initialL1Height uint64

	processedProvenWithdrawalsExtension1Events uint64

	numberOfDetectedForgery    uint64
	numberOfInvalidWithdrawals uint64
	withdrawalsValidated       uint64

	nodeConnectionFailures uint64

	forgeriesWithdrawalsEvents       []validator.EnrichedProvenWithdrawalEvent
	invalidProposalWithdrawalsEvents []validator.EnrichedProvenWithdrawalEvent
}

func NewState(log log.Logger, nextL1Height uint64, latestL1Height uint64) (*State, error) {

	if nextL1Height > latestL1Height {
		log.Info("nextL1Height is greater than latestL1Height, starting from latest", "nextL1Height", nextL1Height, "latestL1Height", latestL1Height)
		nextL1Height = latestL1Height
	}

	ret := State{
		processedProvenWithdrawalsExtension1Events: 0,
		nextL1Height:               nextL1Height,
		latestL1Height:             latestL1Height,
		numberOfDetectedForgery:    0,
		withdrawalsValidated:       0,
		nodeConnectionFailures:     0,
		numberOfInvalidWithdrawals: 0,
		initialL1Height:            nextL1Height,
	}

	return &ret, nil
}

func (s *State) LogState(log log.Logger) {
	blockToProcess, syncPercentage := s.GetPercentages()

	log.Info("STATE:",
		"withdrawalsValidated", fmt.Sprintf("%d", s.withdrawalsValidated),
		"initialL1Height", fmt.Sprintf("%d", s.initialL1Height),
		"nextL1Height", fmt.Sprintf("%d", s.nextL1Height),
		"latestL1Height", fmt.Sprintf("%d", s.latestL1Height),
		"blockToProcess", fmt.Sprintf("%d", blockToProcess),
		"syncPercentage", fmt.Sprintf("%d%%", syncPercentage),
		"processedProvenWithdrawalsExtension1Events", fmt.Sprintf("%d", s.processedProvenWithdrawalsExtension1Events),
		"numberOfDetectedForgery", fmt.Sprintf("%d", s.numberOfDetectedForgery),
		"numberOfInvalidWithdrawals", fmt.Sprintf("%d", s.numberOfInvalidWithdrawals),
		"nodeConnectionFailures", fmt.Sprintf("%d", s.nodeConnectionFailures),
		"forgeriesWithdrawalsEvents", fmt.Sprintf("%d", len(s.forgeriesWithdrawalsEvents)),
		"invalidProposalWithdrawalsEvents", fmt.Sprintf("%d", len(s.invalidProposalWithdrawalsEvents)),
	)
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
	ForgeriesWithdrawalsEventsGauge                    prometheus.Gauge
	InvalidProposalWithdrawalsEventsGauge              prometheus.Gauge
	ForgeriesWithdrawalsEventsGaugeVec                 *prometheus.GaugeVec
	InvalidProposalWithdrawalsEventsGaugeVec           *prometheus.GaugeVec

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
	forgeriesWithdrawalsEventsGaugeValue, _ := GetGaugeValue(m.ForgeriesWithdrawalsEventsGauge)
	invalidProposalWithdrawalsEventsGaugeValue, _ := GetGaugeValue(m.InvalidProposalWithdrawalsEventsGauge)

	forgeriesWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.ForgeriesWithdrawalsEventsGaugeVec, prometheus.Labels{})
	invalidProposalWithdrawalsEventsGaugeVecValue, _ := GetGaugeVecValue(m.InvalidProposalWithdrawalsEventsGaugeVec, prometheus.Labels{})

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
		ForgeriesWithdrawalsEventsGaugeVec: m.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricsNamespace,
				Name:      "forgeries_withdrawals_events_info",
				Help:      "Information about forgeries withdrawals events.",
			},
			[]string{"withdrawal_hash", "proof_submitter", "status", "blacklisted", "withdrawal_hash_present", "enriched", "event_block_number", "event_tx_hash", "event_index"},
		),
		InvalidProposalWithdrawalsEventsGaugeVec: m.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: MetricsNamespace,
				Name:      "invalid_proposal_withdrawals_events_info",
				Help:      "Information about invalid proposal withdrawals events.",
			},
			[]string{"withdrawal_hash", "proof_submitter", "status", "blacklisted", "withdrawal_hash_present", "enriched", "event_block_number", "event_tx_hash", "event_index"},
		),
	}

	return ret
}

func (m *Metrics) UpdateMetricsFromState(state *State) {

	// Update Gauges
	m.InitialL1HeightGauge.Set(float64(state.initialL1Height))
	m.NextL1HeightGauge.Set(float64(state.nextL1Height))
	m.LatestL1HeightGauge.Set(float64(state.latestL1Height))

	m.NumberOfDetectedForgeryGauge.Set(float64(state.numberOfDetectedForgery))
	m.NumberOfInvalidWithdrawalsGauge.Set(float64(state.numberOfInvalidWithdrawals))
	m.ForgeriesWithdrawalsEventsGauge.Set(float64(len(state.forgeriesWithdrawalsEvents)))
	m.InvalidProposalWithdrawalsEventsGauge.Set(float64(len(state.invalidProposalWithdrawalsEvents)))

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
	for index, event := range state.forgeriesWithdrawalsEvents {
		withdrawalHash := common.BytesToHash(event.Event.WithdrawalHash[:]).Hex()
		proofSubmitter := event.Event.ProofSubmitter.String()
		status := event.DisputeGame.DisputeGameData.Status.String()

		m.ForgeriesWithdrawalsEventsGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.Blacklisted),
			fmt.Sprintf("%v", event.WithdrawalHashPresentOnL2),
			fmt.Sprintf("%v", event.Enriched),
			fmt.Sprintf("%v", event.Event.Raw.BlockNumber),
			event.Event.Raw.TxHash.String(),
			fmt.Sprintf("%v", index),
		).Set(1) // Set a value  for existence
	}

	// Clear the previous values
	m.InvalidProposalWithdrawalsEventsGaugeVec.Reset()

	// Update metrics for invalid proposal withdrawals events
	for index, event := range state.invalidProposalWithdrawalsEvents {
		withdrawalHash := common.BytesToHash(event.Event.WithdrawalHash[:]).Hex()
		proofSubmitter := event.Event.ProofSubmitter.String()
		status := event.DisputeGame.DisputeGameData.Status.String()

		m.InvalidProposalWithdrawalsEventsGaugeVec.WithLabelValues(
			withdrawalHash,
			proofSubmitter,
			status,
			fmt.Sprintf("%v", event.Blacklisted),
			fmt.Sprintf("%v", event.WithdrawalHashPresentOnL2),
			fmt.Sprintf("%v", event.Enriched),
			fmt.Sprintf("%v", event.Event.Raw.BlockNumber),
			event.Event.Raw.TxHash.String(),
			fmt.Sprintf("%v", index),
		).Set(1) // Set a value  for existence
	}
}
