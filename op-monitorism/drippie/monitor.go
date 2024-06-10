package drippie

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/drippie/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "drippie_mon"
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client

	drippieAddress common.Address
	drippie        *bindings.Drippie
	created        []string

	// Metrics
	dripCount              *prometheus.GaugeVec
	dripLastTimestamp      *prometheus.GaugeVec
	dripExecutableState    *prometheus.GaugeVec
	highestBlockNumber     *prometheus.GaugeVec
	nodeConnectionFailures *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating drippie monitor...")

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}

	drippie, err := bindings.NewDrippie(cfg.DrippieAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to Drippie: %w", err)
	}

	return &Monitor{
		log: log,

		l1Client: l1Client,

		drippieAddress: cfg.DrippieAddress,
		drippie:        drippie,

		// Metrics
		dripCount: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "dripCounts",
			Help:      "number of drips created",
		}, []string{"name"}),
		dripLastTimestamp: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "dripLatestTimestamp",
			Help:      "latest timestamp of drips",
		}, []string{"name"}),
		dripExecutableState: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "dripExecutableState",
			Help:      "drip executable state",
		}, []string{"name"}),
		highestBlockNumber: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestBlockNumber",
			Help:      "observed l1 heights (checked and known)",
		}, []string{"type"}),
		nodeConnectionFailures: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "nodeConnectionFailures",
			Help:      "number of times node connection has failed",
		}, []string{"layer", "section"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	// Determine current L1 block height.
	latestL1Height, err := m.l1Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "blockNumber").Inc()
		return
	}

	// Update metrics.
	m.highestBlockNumber.WithLabelValues("known").Set(float64(latestL1Height))

	// Set up the call options once.
	callOpts := bind.CallOpts{
		BlockNumber: big.NewInt(int64(latestL1Height)),
	}

	// Grab the number of created drips at the current block height.
	numCreated, err := m.drippie.GetDripCount(&callOpts)
	if err != nil {
		m.log.Error("failed to query Drippie for number of created drips", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "dripCount").Inc()
		return
	}

	// Add new drip names if the number of created drips has increased.
	if numCreated.Cmp(big.NewInt(int64(len(m.created)))) >= 0 {
		// Iterate through the new drip indices and add their names to the stored list.
		// TODO: You can optimize this with a multicall. Current code is good enough for now since we
		// don't expect a large number of drips to be created. If this changes, consider multicall to
		// batch the requests into a single call.
		for i := len(m.created); i < int(numCreated.Int64()); i++ {
			// Grab the name of the drip at the current index.
			m.log.Info("pulling name for new drip index", "index", i)
			name, err := m.drippie.Created(&callOpts, big.NewInt(int64(i)))
			if err != nil {
				m.log.Error("failed to query Drippie for Drip name", "index", i, "err", err)
				m.nodeConnectionFailures.WithLabelValues("l1", "dripName").Inc()
				return
			}

			// Add the name to the list of created drips.
			m.log.Info("got drip name", "index", i, "name", name)
			m.created = append(m.created, name)
		}
	} else {
		// Should not happen, log an error and reset the created drips.
		m.log.Error("number of created drips decreased", "old", len(m.created), "new", numCreated)
		m.created = nil
		return
	}

	// Iterate through all created drips and update their metrics.
	for _, name := range m.created {
		// Grab the drip state.
		m.log.Info("querying metrics for drip", "name", name)
		drip, err := m.drippie.Drips(&callOpts, name)
		if err != nil {
			m.log.Error("failed to query Drippie for Drip", "name", name, "err", err)
			m.nodeConnectionFailures.WithLabelValues("l1", "drips").Inc()
			return
		}

		// Update metrics.
		m.dripCount.WithLabelValues(name).Set(float64(drip.Count.Int64()))
		m.dripLastTimestamp.WithLabelValues(name).Set(float64(drip.Last.Int64()))

		// Check if this drip is executable.
		executable, err := m.drippie.Executable(&callOpts, name)
		if err != nil || !executable {
			m.dripExecutableState.WithLabelValues(name).Set(0)
		} else {
			m.dripExecutableState.WithLabelValues(name).Set(1)
		}

		// Log so we know what's happening.
		m.log.Info("updated metrics for drip", "name", name, "count", drip.Count, "last", drip.Last, "executable", executable)
	}
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
