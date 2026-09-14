package validator

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// ErrWithdrawalProofDeleted reports that the OptimismPortal deleted the proof record for a
// withdrawal hash and proof submitter. The withdrawal can never finalize with that proof, so the
// monitor skips the event. Without this the monitor would try to read a dispute game at the zero
// address, fail the whole block range, and never advance its L1 cursor.
var ErrWithdrawalProofDeleted = errors.New("withdrawal proof deleted")

// ErrWithdrawalProofMissing reports that the captured block precedes the proof event.
// The monitor retries the block range instead of classifying the empty record.
var ErrWithdrawalProofMissing = errors.New("withdrawal proof record missing without a deletion")

// errProofRecordEmpty reports that the proof record read empty, before the cause is known.
var errProofRecordEmpty = errors.New("withdrawal proof record is empty")

// ValidateProofWithdrawalState represents the state of the proof validation.
type ValidateProofWithdrawalState int8

// Constants representing various states of proof validation.
const (
	INVALID_PROOF_FORGERY_DETECTED ValidateProofWithdrawalState = iota
	INVALID_PROPOSAL_FORGERY_DETECTED
	INVALID_PROPOSAL_INPROGRESS
	INVALID_PROPOSAL_CORRECTLY_RESOLVED
	PROOF_ON_BLACKLISTED_GAME
	VALID_PROOF
)

// EnrichedProvenWithdrawalEvent represents an enriched event for proven withdrawals.
type EnrichedProvenWithdrawalEvent struct {
	Event                         *WithdrawalProvenExtension1Event // The original withdrawal event.
	DisputeGame                   *FaultDisputeGameProxy           // Associated dispute game.
	Blacklisted                   bool                             // Indicates if the game is blacklisted.
	WithdrawalHashPresentOnL2     bool                             // Indicates if the withdrawal hash is present on L2.
	DisputeGameRootClaimIsTrusted bool                             // Indicates if the dispute game root claim is trusted.
	Enriched                      bool                             // Indicates if the event is enriched.
	ProcessedTimeStamp            float64                          // Unix TimeStamp seconds when the event was processed.
	ClientUsed                    string                           // Client used to get the proof
	PreIsthmusUnverifiable        bool                             // Game's L2 block predates Isthmus; cannot be header-verified, flagged for security triage.
}

// ProvenWithdrawalValidator validates proven withdrawal events.
type ProvenWithdrawalValidator struct {
	L1Proxy *L1Proxy        // Helper for interacting with Optimism Portal 2.
	L2Proxy *L2Proxy        // Helper for interacting with L2.
	ctx     context.Context // Context for managing cancellation and timeouts.
	log     log.Logger      // Logger for logging.
}

// String provides a string representation of EnrichedProvenWithdrawalEvent.
func (e *EnrichedProvenWithdrawalEvent) String() string {
	return fmt.Sprintf("Event: %v, DisputeGame: %v, DisputeGameRootClaimIsTrusted: %v, Blacklisted: %v, withdrawalHashPresentOnL2: %v, Enriched: %v, ClientUsed: %v",
		e.Event,
		e.DisputeGame,
		e.DisputeGameRootClaimIsTrusted,
		e.Blacklisted,
		e.WithdrawalHashPresentOnL2,
		e.Enriched,
		e.ClientUsed)
}

// String provides a string representation of ValidateProofWithdrawalState.
func (v ValidateProofWithdrawalState) String() string {
	return [...]string{"INVALID_PROOF_FORGERY_DETECTED", "INVALID_PROPOSAL_FORGERY_DETECTED", "INVALID_PROPOSAL_INPROGRESS", "INVALID_PROPOSAL_CORRECTLY_RESOLVED", "PROOF_ON_BLACKLISTED_GAME", "VALID_PROOF"}[v]
}

func NewWithdrawalValidator(ctx context.Context, log log.Logger, l1GethClientURL string, l2GethClientURL string, l2GethBackupClientsURLs map[string]string, OptimismPortalAddress common.Address) (*ProvenWithdrawalValidator, error) {

	l1Proxy, err := NewL1Proxy(ctx, l1GethClientURL, OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create l1 proxy: %w", err)
	}

	l2Proxy, err := NewL2Proxy(ctx, l2GethClientURL, l2GethBackupClientsURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to create l2 proxy: %w", err)
	}

	return &ProvenWithdrawalValidator{
		L1Proxy: l1Proxy,
		L2Proxy: l2Proxy,
		ctx:     ctx,
		log:     log,
	}, nil
}

type BackupClientResult struct {
	Client *ethclient.Client
	Error  string
}

func GethBackupClientsDictionary(ctx context.Context, L2GethBackupURLs map[string]string, l2ChainID *big.Int) (map[string]*ethclient.Client, map[string]string, error) {
	goodClients := make(map[string]*ethclient.Client)
	badClients := make(map[string]string)

	for name, url := range L2GethBackupURLs {
		backupClient, err := ethclient.Dial(url)
		if err != nil {
			badClients[name] = fmt.Sprintf("failed to dial: %v", err)
			continue
		}

		backupChainID, err := backupClient.ChainID(ctx)
		if err != nil {
			badClients[name] = fmt.Sprintf("failed to get chain ID: %v", err)
			continue
		}

		if backupChainID.Cmp(l2ChainID) != 0 {
			badClients[name] = fmt.Sprintf("chain ID mismatch, expected: %d, got: %d", l2ChainID, backupChainID)
			continue
		}

		goodClients[name] = backupClient
	}

	if len(goodClients) == 0 {
		return nil, badClients, fmt.Errorf("no valid backup clients found")
	}

	return goodClients, badClients, nil
}

// UpdateEnrichedWithdrawalEvent updates the enriched withdrawal event with relevant data.
// It checks for blacklisting, validates root claims, and verifies withdrawal presence on L2.
func (wv *ProvenWithdrawalValidator) UpdateEnrichedWithdrawalEvent(event *EnrichedProvenWithdrawalEvent) error {
	if event.DisputeGame.DisputeGameData.Status == IN_PROGRESS {
		if event.DisputeGame == nil {
			return fmt.Errorf("dispute game is nil")
		}
		err := event.DisputeGame.RefreshState()
		if err != nil {
			return fmt.Errorf("failed to refresh game state: %w", err)
		}
	}

	// Check if the game is blacklisted only if not confirmed already that it is blacklisted
	if event.Blacklisted || !event.Enriched {
		blacklisted, err := wv.L1Proxy.IsGameBlacklisted(event.DisputeGame)
		if err != nil {
			return fmt.Errorf("failed to check if game is blacklisted: %w", err)
		}
		event.Blacklisted = blacklisted
	}

	// Check if the game root claim is valid on L2 only if not confirmed already that it is on L2
	if !event.Enriched {
		latest_known_l2_block, err := wv.L2Proxy.BlockNumber()
		if err != nil {
			return fmt.Errorf("failed to get latest known L2 block number: %w", err)
		}
		if latest_known_l2_block >= event.DisputeGame.DisputeGameData.L2blockNumber.Uint64() {
			trustedRootClaim, preIsthmus, err := wv.L2Proxy.VerifyRootClaimFromHeader(event.DisputeGame.DisputeGameData.L2blockNumber, event.DisputeGame.DisputeGameData.RootClaim)
			if err != nil {
				return fmt.Errorf("failed to verify root claim from header: %w", err)
			}
			if preIsthmus {
				// Cannot header-verify a pre-Isthmus block. Not treated as valid nor as a
				// forgery: flagged for security triage (see monitor.ConsumeEvent).
				event.PreIsthmusUnverifiable = true
				event.DisputeGameRootClaimIsTrusted = false
				event.WithdrawalHashPresentOnL2 = false
			} else {
				event.DisputeGameRootClaimIsTrusted = trustedRootClaim
				// Independently re-check that the withdrawal is genuinely present in the
				// L2ToL1MessagePasser (defense-in-depth against an OptimismPortal
				// inclusion-proof bug). sentMessages is append-only, so we query it at
				// head — no archive state, no eth_getProof, and it can never stall.
				present, err := wv.L2Proxy.IsWithdrawalPresentAtHead(event.Event.WithdrawalHash)
				if err != nil {
					return fmt.Errorf("failed to check withdrawal presence at head: %w", err)
				}
				event.WithdrawalHashPresentOnL2 = present
			}
		} else {
			event.DisputeGameRootClaimIsTrusted = false
		}

	}

	event.Enriched = true
	return nil
}

// GetEnrichedWithdrawalEvent retrieves one enriched withdrawal event at the captured L1 block.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalEvent(
	withdrawalEvent *WithdrawalProvenExtension1Event,
	headerNumber uint64,
	headerHash common.Hash,
) (*EnrichedProvenWithdrawalEvent, error) {
	if withdrawalEvent.Raw.BlockNumber > headerNumber {
		return nil, fmt.Errorf(
			"%w: event block:%d captured block:%d withdrawal hash:%x proof submitter:%x",
			ErrWithdrawalProofMissing,
			withdrawalEvent.Raw.BlockNumber,
			headerNumber,
			withdrawalEvent.WithdrawalHash,
			withdrawalEvent.ProofSubmitter,
		)
	}

	disputeGameProxy, err := wv.getDisputeGame(
		withdrawalEvent.WithdrawalHash,
		withdrawalEvent.ProofSubmitter,
		headerHash,
	)
	if errors.Is(err, errProofRecordEmpty) {
		return nil, ErrWithdrawalProofDeleted
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute games: %w", err)
	}

	enrichedWithdrawalEvent := EnrichedProvenWithdrawalEvent{
		Event:       withdrawalEvent,
		DisputeGame: &disputeGameProxy,

		Blacklisted: false,
		Enriched:    false,
	}

	return &enrichedWithdrawalEvent, nil
}

func (wv *ProvenWithdrawalValidator) getDisputeGame(
	withdrawalHash [32]byte,
	proofSubmitter common.Address,
	blockHash common.Hash,
) (FaultDisputeGameProxy, error) {
	submittedProofData, err := wv.L1Proxy.GetSubmittedProofsDataAtBlockHash(
		withdrawalHash,
		proofSubmitter,
		blockHash,
	)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games addresses: %w", err)
	}
	if submittedProofData.IsEmpty() {
		return FaultDisputeGameProxy{}, errProofRecordEmpty
	}

	disputeGameProxy, err := wv.L1Proxy.GetDisputeGameProxyFromAddress(
		submittedProofData.disputeGameProxyAddress,
	)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games: %w", err)
	}

	return disputeGameProxy, nil
}

// GetEnrichedWithdrawalsEvents retrieves enriched events at the captured L1 block.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEvents(
	start uint64,
	end *uint64,
	headerNumber uint64,
	headerHash common.Hash,
) ([]EnrichedProvenWithdrawalEvent, error) {
	events, err := wv.L1Proxy.GetProvenWithdrawalsExtension1Events(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events: %w", err)
	}

	enrichedEvents := make([]EnrichedProvenWithdrawalEvent, 0)
	for _, event := range events {
		enrichedEvent, err := wv.GetEnrichedWithdrawalEvent(&event, headerNumber, headerHash)
		if errors.Is(err, ErrWithdrawalProofDeleted) {
			wv.logSkippedDeletedProof(&event)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get enriched withdrawal event: %w", err)
		}
		enrichedEvents = append(enrichedEvents, *enrichedEvent)
	}
	return enrichedEvents, nil
}

// GetEnrichedWithdrawalsEventsMap retrieves enriched events at the captured L1 block.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEventsMap(
	start uint64,
	end *uint64,
	headerNumber uint64,
	headerHash common.Hash,
) (map[common.Hash]*EnrichedProvenWithdrawalEvent, error) {
	iterator, err := wv.L1Proxy.GetProvenWithdrawalsExtension1EventsIterator(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 iterator error:%w", err)
	}

	enrichedEvents := make(map[common.Hash]*EnrichedProvenWithdrawalEvent)
	for iterator.Next() {
		event := iterator.Event
		withdrawalEvent := WithdrawalProvenExtension1Event{
			WithdrawalHash: event.WithdrawalHash,
			ProofSubmitter: event.ProofSubmitter,
			Raw: Raw{
				BlockNumber: event.Raw.BlockNumber,
				TxHash:      event.Raw.TxHash,
			},
		}

		enrichedEvent, err := wv.GetEnrichedWithdrawalEvent(
			&withdrawalEvent,
			headerNumber,
			headerHash,
		)
		if errors.Is(err, ErrWithdrawalProofDeleted) {
			wv.logSkippedDeletedProof(&withdrawalEvent)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get enriched withdrawal event: %w", err)
		}
		enrichedEvents[enrichedEvent.Event.Raw.TxHash] = enrichedEvent
	}
	return enrichedEvents, nil
}

// logSkippedDeletedProof records that an event was skipped because its proof record is gone.
func (wv *ProvenWithdrawalValidator) logSkippedDeletedProof(event *WithdrawalProvenExtension1Event) {
	wv.log.Info("WITHDRAWAL: proof deleted, skipping event",
		"withdrawalHash", common.BytesToHash(event.WithdrawalHash[:]),
		"proofSubmitter", event.ProofSubmitter,
		"TxHash", event.Raw.TxHash)
}

// IsWithdrawalEventValid checks if the enriched withdrawal event is valid.
// It returns true if the event is valid, otherwise returns false.
func (wv *ProvenWithdrawalValidator) IsWithdrawalEventValid(enrichedWithdrawalEvent *EnrichedProvenWithdrawalEvent) (bool, error) {
	if !enrichedWithdrawalEvent.Enriched {
		return false, fmt.Errorf("game not enriched")
	}

	// Valid iff (a) the game's root claim is canonical (recomputed from the L2
	// header) AND (b) the withdrawal is genuinely present in the L2ToL1MessagePasser
	// (checked at head — see IsWithdrawalPresentAtHead). (a) catches a dishonest
	// game; (b) is an independent re-check against an OptimismPortal inclusion-proof
	// bug. Pre-Isthmus events are routed for triage before this and never reach it.
	return enrichedWithdrawalEvent.DisputeGameRootClaimIsTrusted && enrichedWithdrawalEvent.WithdrawalHashPresentOnL2, nil
}

func (wv *ProvenWithdrawalValidator) GetL2BlockNumber() (uint64, error) {
	return wv.L2Proxy.BlockNumber()
}

func (wv *ProvenWithdrawalValidator) GetL1BlockNumber() (uint64, error) {
	return wv.L1Proxy.BlockNumber()
}

// GetLatestL1Header returns the current unsafe L1 header.
func (wv *ProvenWithdrawalValidator) GetLatestL1Header() (*types.Header, error) {
	return wv.L1Proxy.LatestHeader()
}
