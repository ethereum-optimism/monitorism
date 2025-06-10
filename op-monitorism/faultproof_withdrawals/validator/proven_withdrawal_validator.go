package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

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

func NewWithdrawalValidator(ctx context.Context, log log.Logger, l1GethClientURL string, l2GethClientURL string, l2GethBackupClientsURLs map[string]string, OptimismPortalAddress common.Address, maxRetries int) (*ProvenWithdrawalValidator, error) {

	l1Proxy, err := NewL1Proxy(ctx, l1GethClientURL, OptimismPortalAddress, log, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to create l1 proxy: %w", err)
	}

	l2Proxy, err := NewL2Proxy(ctx, l2GethClientURL, l2GethBackupClientsURLs, log, maxRetries)
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

func GethBackupClientsDictionary(ctx context.Context, L2GethBackupURLs map[string]string, l2ChainID *big.Int) (map[string]*ethclient.Client, error) {
	dictionary := make(map[string]*ethclient.Client)
	for name, url := range L2GethBackupURLs {
		backupClient, err := ethclient.Dial(url)
		if err != nil {
			return nil, fmt.Errorf("failed to dial l2 backup, error: %w", err)
		}
		backupChainID, err := backupClient.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get backup L2 chain ID, error: %w", err)
		}
		if backupChainID.Cmp(l2ChainID) != 0 {
			return nil, fmt.Errorf("backup L2 client chain ID mismatch, expected: %d, got: %d", l2ChainID, backupChainID)
		}
		dictionary[name] = backupClient
	}
	return dictionary, nil
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
			trustedRootClaim, withdrawalHashPresentOnL2, clientUsed, err := wv.L2Proxy.VerifyRootClaimAndWithdrawalHash(event.DisputeGame.DisputeGameData.L2blockNumber, event.DisputeGame.DisputeGameData.RootClaim, event.Event.WithdrawalHash)
			if err != nil {
				return fmt.Errorf("failed to get trustedRootClaim from Op-node %s: %w", clientUsed, err)
			}
			event.DisputeGameRootClaimIsTrusted = trustedRootClaim
			event.WithdrawalHashPresentOnL2 = withdrawalHashPresentOnL2
			event.ClientUsed = clientUsed
		} else {
			event.DisputeGameRootClaimIsTrusted = false
		}

	}

	event.Enriched = true
	return nil
}

// GetEnrichedWithdrawalEvent retrieves an enriched withdrawal event based on the given withdrawal event.
// It returns the enriched event along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalEvent(withdrawalEvent *WithdrawalProvenExtension1Event) (*EnrichedProvenWithdrawalEvent, error) {
	disputeGameProxy, err := wv.getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalEvent.WithdrawalHash, withdrawalEvent.ProofSubmitter)
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

// getDisputeGamesFromWithdrawalhashAndProofSubmitter retrieves a DisputeGame object
// based on the provided withdrawal hash and proof submitter address.
func (wv *ProvenWithdrawalValidator) getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalHash [32]byte, proofSubmitter common.Address) (FaultDisputeGameProxy, error) {
	submittedProofData, err := wv.L1Proxy.GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitter)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games addresses: %w", err)
	}
	disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
	disputeGame, err := wv.L1Proxy.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games: %w", err)
	}

	return disputeGame, nil
}

// GetEnrichedWithdrawalsEvents retrieves enriched withdrawal events within the specified block range.
// It returns a slice of EnrichedProvenWithdrawalEvent along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEvents(start uint64, end *uint64) ([]EnrichedProvenWithdrawalEvent, error) {
	events, err := wv.L1Proxy.GetProvenWithdrawalsExtension1Events(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events: %w", err)
	}

	enrichedProvenWithdrawalEvents := make([]EnrichedProvenWithdrawalEvent, 0)

	for _, event := range events {
		enrichedWithdrawalEvent, err := wv.GetEnrichedWithdrawalEvent(&event)
		if err != nil {
			return nil, fmt.Errorf("failed to get enriched withdrawal event: %w", err)
		}
		enrichedProvenWithdrawalEvents = append(enrichedProvenWithdrawalEvents, *enrichedWithdrawalEvent)
	}

	return enrichedProvenWithdrawalEvents, nil
}

// GetEnrichedWithdrawalsEvents retrieves enriched withdrawal events within the specified block range.
// It returns a slice of EnrichedProvenWithdrawalEvent along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEventsMap(start uint64, end *uint64) (map[common.Hash]*EnrichedProvenWithdrawalEvent, error) {
	iterator, err := wv.L1Proxy.GetProvenWithdrawalsExtension1EventsIterator(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 iterator error:%w", err)
	}

	enrichedProvenWithdrawalEvents := make(map[common.Hash]*EnrichedProvenWithdrawalEvent)

	for iterator.Next() {
		event := iterator.Event

		enrichedWithdrawalEvent, err := wv.GetEnrichedWithdrawalEvent(&WithdrawalProvenExtension1Event{
			WithdrawalHash: event.WithdrawalHash,
			ProofSubmitter: event.ProofSubmitter,
			Raw: Raw{
				BlockNumber: event.Raw.BlockNumber,
				TxHash:      event.Raw.TxHash,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get enriched withdrawal event: %w", err)
		}

		key := enrichedWithdrawalEvent.Event.Raw.TxHash
		enrichedProvenWithdrawalEvents[key] = enrichedWithdrawalEvent
	}

	return enrichedProvenWithdrawalEvents, nil
}

// IsWithdrawalEventValid checks if the enriched withdrawal event is valid.
// It returns true if the event is valid, otherwise returns false.
func (wv *ProvenWithdrawalValidator) IsWithdrawalEventValid(enrichedWithdrawalEvent *EnrichedProvenWithdrawalEvent) (bool, error) {
	if !enrichedWithdrawalEvent.Enriched {
		return false, fmt.Errorf("game not enriched")
	}

	return enrichedWithdrawalEvent.DisputeGameRootClaimIsTrusted && enrichedWithdrawalEvent.WithdrawalHashPresentOnL2, nil
}

func (wv *ProvenWithdrawalValidator) GetL2BlockNumber() (uint64, error) {
	return wv.L2Proxy.BlockNumber()
}

func (wv *ProvenWithdrawalValidator) GetL1BlockNumber() (uint64, error) {
	return wv.L1Proxy.BlockNumber()
}
