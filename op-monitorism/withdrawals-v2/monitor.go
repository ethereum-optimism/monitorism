package withdrawalsv2

import (
	"bytes"
	"context"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/processor"
	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals-v2/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	MetricsNamespace = "withdrawals_v2"
)

type Metrics struct {
	validWithdrawals   *prometheus.CounterVec
	invalidWithdrawals *prometheus.CounterVec
}

type Monitor struct {
	log           log.Logger
	l1Client      *ethclient.Client
	l2Client      *ethclient.Client
	portal        *bindings.OptimismPortal2
	portalAddress common.Address
	processor     *processor.BlockProcessor
	metrics       Metrics
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating withdrawals v2 monitor")

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, err
	}

	l2Client, err := ethclient.Dial(cfg.L2NodeURL)
	if err != nil {
		return nil, err
	}

	portalAddress := common.HexToAddress(cfg.OptimismPortalAddress)
	portal, err := bindings.NewOptimismPortal2(portalAddress, l1Client)
	if err != nil {
		return nil, err
	}

	mon := &Monitor{
		log:           log,
		l1Client:      l1Client,
		l2Client:      l2Client,
		portal:        portal,
		portalAddress: portalAddress,
		metrics: Metrics{
			validWithdrawals: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "valid_withdrawals_total",
					Help:      "Total number of valid withdrawals",
				},
				[]string{},
			),
			invalidWithdrawals: m.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: MetricsNamespace,
					Name:      "invalid_withdrawals_total",
					Help:      "Total number of invalid withdrawals",
				},
				[]string{"reason", "txhash", "wdhash"},
			),
		},
	}

	proc, err := processor.NewBlockProcessor(
		m,
		log,
		cfg.L1NodeURL,
		nil,
		nil,
		mon.processLog,
		&processor.Config{
			StartBlock: big.NewInt(int64(cfg.StartBlock)),
			Interval:   cfg.PollingInterval,
		},
	)
	if err != nil {
		return nil, err
	}

	mon.processor = proc
	return mon, nil
}

func (m *Monitor) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		m.processor.Stop()
	}()

	if err := m.processor.Start(); err != nil {
		m.log.Error("processor error", "err", err)
	}
}

func (m *Monitor) Close(ctx context.Context) error {
	m.processor.Stop()
	m.l1Client.Close()
	m.l2Client.Close()
	return nil
}

// computeWithdrawalStorageKey computes the storage key for a withdrawal hash in the
// L2ToL1MessagePasser contract. The withdrawal mapping is located in the first storage slot (0)
// of the L2ToL1MessagePasser contract.
func computeWithdrawalStorageKey(withdrawalHash [32]byte) common.Hash {
	return crypto.Keccak256Hash(append(withdrawalHash[:], make([]byte, 32)...))
}

// determineFailureReason compares the computed output root with the dispute game's root claim to
// determine the reason for withdrawal validation failure.
func determineFailureReason(computedOutputRoot eth.Bytes32, disputeGameRootClaim [32]byte) string {
	if !bytes.Equal(computedOutputRoot[:], disputeGameRootClaim[:]) {
		// Acceptable case, this can happen normally when dispute game has wrong output root (P3)
		return "bad_output_root"
	}

	// Unacceptable case, dispute game is correct but withdrawal proof is invalid (P0)
	return "bad_withdrawal_proof"
}

// checkWithdrawalExists checks if a withdrawal hash exists in the L2ToL1MessagePasser contract.
// Returns true if the withdrawal exists, false otherwise.
func checkWithdrawalExists(ctx context.Context, l2Client *ethclient.Client, withdrawalHash [32]byte) (bool, error) {
	// Use eth_getStorageAt to check that the withdrawal hash is actually present on the L2.
	// We use the latest block, good enough, if it's present on the L2 we're fine with it executing.
	storageKey := computeWithdrawalStorageKey(withdrawalHash)
	storageVal, err := l2Client.StorageAt(ctx, predeploys.L2ToL1MessagePasserAddr, storageKey, nil)
	if err != nil {
		return false, err
	}

	// Storage value not empty means withdrawal exists
	return len(storageVal) > 0, nil
}

// DisputeGameInfo holds information about a dispute game related to a withdrawal.
type DisputeGameInfo struct {
	RootClaim     [32]byte
	L2BlockNumber *big.Int
}

// getDisputeGameInfo retrieves dispute game information for a proven withdrawal.
// Returns the root claim and L2 block number from the dispute game.
func getDisputeGameInfo(portal *bindings.OptimismPortal2, l1Client *ethclient.Client, withdrawalHash [32]byte, proofSubmitter common.Address) (*DisputeGameInfo, error) {
	// Get the dispute game info from the provenWithdrawals mapping.
	disputeGameInfo, err := portal.ProvenWithdrawals(nil, withdrawalHash, proofSubmitter)
	if err != nil {
		return nil, err
	}

	// Bind to the dispute game contract.
	disputeGame, err := bindings.NewFaultDisputeGame(disputeGameInfo.DisputeGameProxy, l1Client)
	if err != nil {
		return nil, err
	}

	// Get the root claim.
	rootClaim, err := disputeGame.RootClaim(nil)
	if err != nil {
		return nil, err
	}

	// Get the L2 block number from the dispute game.
	l2BlockNumber, err := disputeGame.L2BlockNumber(nil)
	if err != nil {
		return nil, err
	}

	return &DisputeGameInfo{
		RootClaim:     rootClaim,
		L2BlockNumber: l2BlockNumber,
	}, nil
}

// getOutputRoot retrieves the output root for a given L2 block number.
func getOutputRoot(ctx context.Context, l2Client *ethclient.Client, l2BlockNumber *big.Int) (eth.Bytes32, error) {
	// Get the L2 block from the client.
	l2Block, err := l2Client.BlockByNumber(ctx, l2BlockNumber)
	if err != nil {
		return eth.Bytes32{}, err
	}

	// Get the storage root of the L2ToL1MessagePasser contract at the L2 block.
	proof := struct{ StorageHash common.Hash }{}
	if err := l2Client.Client().CallContext(ctx, &proof, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(l2Block.Number())); err != nil {
		return eth.Bytes32{}, err
	}

	// Generate the output root.
	outputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(l2Block.Root()),
		MessagePasserStorageRoot: eth.Bytes32(proof.StorageHash),
		BlockHash:                l2Block.Hash(),
	})

	return outputRoot, nil
}

func (m *Monitor) isValidWithdrawal(ctx context.Context, txHash common.Hash, wdHash common.Hash, proofSubmitter common.Address) (bool, string, error) {
	// Check if the withdrawal exists on the L2.
	exists, err := checkWithdrawalExists(ctx, m.l2Client, wdHash)
	if err != nil {
		return false, "", err
	}

	// Withdrawal exists on the L2, we can move on.
	if exists {
		return true, "", nil
	}

	// At this point we're looking at an invalid withdrawal. We need to figure out if this withdrawal
	// is invalid because it's being proven against an invalid dispute game (allowed to happen) or if
	// it's invalid because it actually managed to trick the withdrawal proving code. Note that we
	// don't need to bother checking how the games resolve, we assume that other monitoring exists to
	// check if invalid dispute games are resolving incorrectly.
	m.log.Info("withdrawal is invalid, performing additional checks", "txHash", txHash.String(), "wdHash", wdHash.String())

	// Get the dispute game info from the provenWithdrawals mapping.
	disputeGameInfo, err := getDisputeGameInfo(m.portal, m.l1Client, wdHash, proofSubmitter)
	if err != nil {
		return false, "", err
	}

	// Grab the output root from the L2 block number.
	outputRoot, err := getOutputRoot(ctx, m.l2Client, disputeGameInfo.L2BlockNumber)
	if err != nil {
		return false, "", err
	}

	// Determine failure reason by comparing output roots.
	reason := determineFailureReason(outputRoot, disputeGameInfo.RootClaim)

	// Log and increment metrics.
	m.log.Info("withdrawal is invalid", "txHash", txHash.String(), "wdHash", wdHash.String(), "reason", reason)
	m.metrics.invalidWithdrawals.WithLabelValues(reason, txHash.String(), wdHash.String()).Inc()
	return false, reason, nil
}

// parseWithdrawalEvent parses a WithdrawalProvenExtension1 event from a log.
// Returns the withdrawal proven event if it's a WithdrawalProvenExtension1 event, nil otherwise.
func (m *Monitor) parseWithdrawalEvent(lg types.Log) (*bindings.OptimismPortal2WithdrawalProvenExtension1, error) {
	// We only care about events logged by the OptimismPortal2 contract.
	if lg.Address != m.portalAddress {
		return nil, nil
	}

	// Need the ABI so we can get the event ID.
	abi, err := bindings.OptimismPortal2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	// We only care about WithdrawalProvenExtension1 events.
	if lg.Topics[0] != abi.Events["WithdrawalProvenExtension1"].ID {
		return nil, nil
	}

	// Parse the withdrawal proven event.
	provenWithdrawal, err := m.portal.ParseWithdrawalProvenExtension1(lg)
	if err != nil {
		return nil, err
	}

	return provenWithdrawal, nil
}

func (m *Monitor) processLog(block *types.Block, lg types.Log, client *ethclient.Client) error {
	ctx := context.Background()

	// Parse the withdrawal proven event.
	provenWithdrawal, err := m.parseWithdrawalEvent(lg)
	if err != nil {
		return err
	}

	// If this isn't a WithdrawalProvenExtension1 event to the portal, we can move on.
	if provenWithdrawal == nil {
		return nil
	}

	// We actually got a withdrawal proven event, let's process it.
	txHash := lg.TxHash.String()
	wdHash := common.BytesToHash(provenWithdrawal.WithdrawalHash[:]).String()
	m.log.Info("processing withdrawal proven event", "txHash", txHash, "wdHash", wdHash)

	// Check if the withdrawal is valid.
	isValid, reason, err := m.isValidWithdrawal(ctx, lg.TxHash, common.BytesToHash(provenWithdrawal.WithdrawalHash[:]), provenWithdrawal.ProofSubmitter)
	if err != nil {
		return err
	}

	// Log and increment metrics.
	if isValid {
		m.log.Info("withdrawal is valid", "txHash", txHash, "wdHash", wdHash)
		m.metrics.validWithdrawals.WithLabelValues().Inc()
	} else {
		m.log.Info("withdrawal is invalid", "txHash", txHash, "wdHash", wdHash, "reason", reason)
		m.metrics.invalidWithdrawals.WithLabelValues(reason, txHash, wdHash).Inc()
	}

	return nil
}
