package fault

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace = "fault_detector"
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client
	l2Client *ethclient.Client

	currOutputIndex  uint64
	faultProofWindow uint64

	l2OO *bindings.L2OutputOracleCaller

	// metrics
	highestOutputIndex     *prometheus.GaugeVec
	isCurrentlyMismatched  prometheus.Gauge
	nodeConnectionFailures *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating fault monitor...")

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	l2Client, err := ethclient.Dial(cfg.L2NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	optimismPortal, err := bindings.NewOptimismPortalCaller(cfg.OptimismPortalAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	l2OOAddress, err := optimismPortal.L2ORACLE(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to query L2OO address: %w", err)
	}
	log.Info("configured L2OutputOracle", "address", l2OOAddress.String())

	l2OO, err := bindings.NewL2OutputOracleCaller(l2OOAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the L2OutputOracle: %w", err)
	}
	faultProofWindow, err := l2OO.FinalizationPeriodSeconds(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to query for finalization window: %w", err)
	}

	monitor := &Monitor{
		log: log,

		l1Client: l1Client,
		l2Client: l2Client,

		l2OO:             l2OO,
		faultProofWindow: faultProofWindow.Uint64(),

		highestOutputIndex: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestOutputIndex",
			Help:      "Highest output indicies (checked and known)",
		}, []string{"type"}),
		isCurrentlyMismatched: m.NewGauge(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "isCurrentlyMismatched",
			Help:      "0 if state is ok, 1 if state is mismatched",
		}),
		nodeConnectionFailures: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "nodeConnectionFailures",
			Help:      "number of times node connection has failed",
		}, []string{"layer", "section"}),
	}

	startingOutputIndex := cfg.StartOutputIndex
	if startingOutputIndex < 0 {
		firstUnfinalizedIndex, err := monitor.findFirstUnfinalizedOutputIndex(ctx, monitor.faultProofWindow)
		if err != nil {
			monitor.nodeConnectionFailures.WithLabelValues("l1", "firstUnfinalizedIndex").Inc()
			return nil, fmt.Errorf("failed to find first unfinalized output index: %w", err)
		}
		startingOutputIndex = int64(firstUnfinalizedIndex)
	}

	log.Info("configured starting index", "index", startingOutputIndex)
	monitor.currOutputIndex = uint64(startingOutputIndex)
	return monitor, nil
}

func (m *Monitor) Run(ctx context.Context) {
	callOpts := &bind.CallOpts{Context: ctx}

	// Check for available outputs to validate

	nextOutputIndex, err := m.l2OO.NextOutputIndex(callOpts)
	if err != nil {
		m.log.Error("failed to query next output index", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "nextOutputIndex").Inc()
		return
	}
	if m.currOutputIndex >= nextOutputIndex.Uint64() {
		m.log.Info("waiting for next output", "index", m.currOutputIndex, "next_index", nextOutputIndex)
		return
	}

	m.highestOutputIndex.WithLabelValues("known").Set(float64(nextOutputIndex.Int64()))
	m.log.Info("checking output", "index", m.currOutputIndex)

	// Fetch Output

	output, err := m.l2OO.GetL2Output(callOpts, big.NewInt(int64(m.currOutputIndex)))
	if err != nil {
		m.log.Error("failed to query output", "index", m.currOutputIndex, "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "getL2Output").Inc()
		return
	}
	l2Height, err := m.l2Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest l2 height", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l2", "blockNumber").Inc()
		return
	}
	if l2Height < output.L2BlockNumber.Uint64() {
		m.log.Warn("l2 node is behind, waiting for sync...")
		return
	}

	// Fetch pre-image information for the output root from L2 to reconstruct

	block, err := m.l2Client.BlockByNumber(ctx, output.L2BlockNumber)
	if err != nil {
		m.log.Error("failed to query l2 block", "height", output.L2BlockNumber, "err", err)
		m.nodeConnectionFailures.WithLabelValues("l2", "blockByNumber").Inc()
		return
	}
	proof := struct{ StorageHash common.Hash }{}
	if err := m.l2Client.Client().CallContext(ctx, &proof, "eth_getProof",
		predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(block.Number())); err != nil {
		m.log.Error("failed to query for proof response of l2ToL1MP contract", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l2", "getProof").Inc()
		return
	}

	// Reconstruct & verify

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: eth.Bytes32(block.Root()), MessagePasserStorageRoot: eth.Bytes32(proof.StorageHash), BlockHash: block.Hash()})
	if outputRoot != eth.Bytes32(output.OutputRoot) {
		m.log.Error("output root mismatch!!!",
			"index", m.currOutputIndex,
			"expected_output_root", outputRoot.String(),
			"actual_output_root", common.Hash(output.OutputRoot).String(),
			"finalization_time", time.Unix(int64(block.Time()+m.faultProofWindow), 0).String(),
		)

		m.isCurrentlyMismatched.Set(1)
		return
	}

	// Continue

	m.log.Info("validated ouput", "index", m.currOutputIndex, "output_root", outputRoot.String(), "finalization_time", time.Unix(int64(block.Time()+m.faultProofWindow), 0).String())
	m.highestOutputIndex.WithLabelValues("checked").Set(float64(m.currOutputIndex))

	m.currOutputIndex++
	m.isCurrentlyMismatched.Set(0)
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	m.l2Client.Close()
	return nil
}

func (m *Monitor) findFirstUnfinalizedOutputIndex(ctx context.Context, finalizationWindow uint64) (uint64, error) {
	m.log.Info("searching for first unfinalized output")
	callOpts := &bind.CallOpts{Context: ctx}

	latestBlock, err := m.l2Client.BlockByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block: %w", err)
	}
	totalOutputsBig, err := m.l2OO.NextOutputIndex(callOpts)
	if err != nil {
		return 0, fmt.Errorf("failed to query next output index: %w", err)
	}

	// Binary search the list of posted outputs

	totalOutputs := totalOutputsBig.Uint64()
	low, high := uint64(0), totalOutputs
	for low < high {
		mid := (low + high) / 2
		output, err := m.l2OO.GetL2Output(callOpts, big.NewInt(int64(mid)))
		if err != nil {
			return 0, fmt.Errorf("failed to query output index %d: %w", mid, err)
		}

		if output.Timestamp.Uint64()+finalizationWindow < latestBlock.Time() {
			low = mid + 1
		} else {
			high = mid
		}
	}

	// If no outputs have been posted for an entire finalization window,
	// `low == totalOutputs`, which is also the next expected output index
	m.log.Info("first unfinalized output index", "index", low)
	return low, nil
}
