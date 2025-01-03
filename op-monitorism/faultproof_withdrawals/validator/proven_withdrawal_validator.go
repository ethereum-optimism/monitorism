package validator

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	Event                     *WithdrawalProvenExtension1Event // The original withdrawal event.
	DisputeGame               *FaultDisputeGameProxy           // Associated dispute game.
	ExpectedRootClaim         [32]byte                         // Expected root claim for validation.
	Blacklisted               bool                             // Indicates if the game is blacklisted.
	WithdrawalHashPresentOnL2 bool                             // Indicates if the withdrawal hash is present on L2.
	Enriched                  bool                             // Indicates if the event is enriched.
	ProcessedTimeStamp        float64                          // Unix TimeStamp seconds when the event was processed.
}

// ProvenWithdrawalValidator validates proven withdrawal events.
type ProvenWithdrawalValidator struct {
	OptimismPortal2Helper     *OptimismPortal2Helper     // Helper for interacting with Optimism Portal 2.
	L2NodeHelper              *OpNodeHelper              // Helper for L2 node interactions.
	L2ToL1MessagePasserHelper *L2ToL1MessagePasserHelper // Helper for L2 to L1 message passing.
	FaultDisputeGameHelper    *FaultDisputeGameHelper    // Helper for dispute game interactions.
	ctx                       context.Context            // Context for managing cancellation and timeouts.
}

// String provides a string representation of EnrichedProvenWithdrawalEvent.
func (e *EnrichedProvenWithdrawalEvent) String() string {
	return fmt.Sprintf("Event: %v, DisputeGame: %v, ExpectedRootClaim: %s, Blacklisted: %v, withdrawalHashPresentOnL2: %v, Enriched: %v",
		e.Event,
		e.DisputeGame,
		common.Bytes2Hex(e.ExpectedRootClaim[:]),
		e.Blacklisted,
		e.WithdrawalHashPresentOnL2,
		e.Enriched)
}

// String provides a string representation of ValidateProofWithdrawalState.
func (v ValidateProofWithdrawalState) String() string {
	return [...]string{"INVALID_PROOF_FORGERY_DETECTED", "INVALID_PROPOSAL_FORGERY_DETECTED", "INVALID_PROPOSAL_INPROGRESS", "INVALID_PROPOSAL_CORRECTLY_RESOLVED", "PROOF_ON_BLACKLISTED_GAME", "VALID_PROOF"}[v]
}

// NewWithdrawalValidator initializes a new ProvenWithdrawalValidator.
// It binds necessary helpers and returns the validator instance.
func NewWithdrawalValidator(ctx context.Context, l1GethURL string, l2OpGethURL string, l2OpNodeURL string, OptimismPortalAddress common.Address) (*ProvenWithdrawalValidator, error) {

	l1GethClient, err := ethclient.Dial(l1GethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}
	l2OpGethClient, err := ethclient.Dial(l2OpGethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}
	l2OpNodeClient, err := ethclient.Dial(l2OpNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	optimismPortal2Helper, err := NewOptimismPortal2Helper(ctx, l1GethClient, OptimismPortalAddress)
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

	l2NodeHelper, err := NewOpNodeHelper(ctx, l2OpNodeClient, l2OpGethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create l2 node helper: %w", err)
	}

	return &ProvenWithdrawalValidator{
		OptimismPortal2Helper:     optimismPortal2Helper,
		L2NodeHelper:              l2NodeHelper,
		L2ToL1MessagePasserHelper: l2ToL1MessagePasserHelper,
		FaultDisputeGameHelper:    faultDisputeGameHelper,
		ctx:                       ctx,
	}, nil
}

// GetEnrichedWithdrawalEvent retrieves an enriched withdrawal event based on the given withdrawal event.
// It returns the enriched event along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalEvent(withdrawalEvent *WithdrawalProvenExtension1Event) (*EnrichedProvenWithdrawalEvent, error) {
	disputeGameProxy, err := wv.getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalEvent.WithdrawalHash, withdrawalEvent.ProofSubmitter)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute games: %w", err)
	}

	enrichedWithdrawalEvent := EnrichedProvenWithdrawalEvent{
		Event:             withdrawalEvent,
		DisputeGame:       &disputeGameProxy,
		ExpectedRootClaim: [32]byte{},
		Blacklisted:       false,
		Enriched:          false,
	}

	return &enrichedWithdrawalEvent, nil
}

// getDisputeGamesFromWithdrawalhashAndProofSubmitter retrieves a DisputeGame object
// based on the provided withdrawal hash and proof submitter address.
func (wv *ProvenWithdrawalValidator) getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalHash [32]byte, proofSubmitter common.Address) (FaultDisputeGameProxy, error) {
	submittedProofData, err := wv.OptimismPortal2Helper.GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitter)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games addresses: %w", err)
	}
	disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
	disputeGame, err := wv.FaultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if err != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games: %w", err)
	}

	return disputeGame, nil
}

// GetProvenWithdrawalsExtension1Events retrieves proven withdrawal extension 1 events
// within the specified block range. It returns a slice of WithdrawalProvenExtension1Event along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtension1Event, error) {
	return wv.OptimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, end)
}

// GetEnrichedWithdrawalsEvents retrieves enriched withdrawal events within the specified block range.
// It returns a slice of EnrichedProvenWithdrawalEvent along with any error encountered.
func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEvents(start uint64, end *uint64) ([]EnrichedProvenWithdrawalEvent, error) {
	events, err := wv.OptimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, end)
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
	iterator, err := wv.OptimismPortal2Helper.GetProvenWithdrawalsExtension1EventsIterator(start, end)
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
