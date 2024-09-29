package validator

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ValidateProofWithdrawalState int8

const (
	INVALID_PROOF_FORGERY_DETECTED ValidateProofWithdrawalState = iota
	INVALID_PROPOSAL_FORGERY_DETECTED
	INVALID_PROPOSAL_INPROGRESS
	INVALID_PROPOSAL_CORRECTLY_RESOLVED
	PROOF_ON_BLACKLISTED_GAME
	VALID_PROOF
)

type EnrichedProvenWithdrawalEvent struct {
	Event                     *WithdrawalProvenExtension1Event
	DisputeGame               *FaultDisputeGameProxy
	ExpectedRootClaim         [32]byte
	Blacklisted               bool
	WithdrawalHashPresentOnL2 bool
	Enriched                  bool
}

type ProvenWithdrawalValidator struct {
	optimismPortal2Helper     *OptimismPortal2Helper
	l2NodeHelper              *OpNodeHelper
	l2ToL1MessagePasserHelper *L2ToL1MessagePasserHelper
	faultDisputeGameHelper    *FaultDisputeGameHelper
	ctx                       context.Context
}

func (e *EnrichedProvenWithdrawalEvent) String() string {
	return fmt.Sprintf("Event: %v, DisputeGame: %v, ExpectedRootClaim: %s, Blacklisted: %v, withdrawalHashPresentOnL2: %v, Enriched: %v",
		e.Event,
		e.DisputeGame,
		common.BytesToHash(e.ExpectedRootClaim[:]),
		e.Blacklisted,
		e.WithdrawalHashPresentOnL2,
		e.Enriched)
}

func (v ValidateProofWithdrawalState) String() string {
	return [...]string{"INVALID_PROOF_FORGERY_DETECTED", "INVALID_PROPOSAL_FORGERY_DETECTED", "INVALID_PROPOSAL_INPROGRESS", "INVALID_PROPOSAL_CORRECTLY_RESOLVED", "PROOF_ON_BLACKLISTED_GAME", "VALID_PROOF"}[v]
}

func NewWithdrawalValidator(ctx context.Context, l1GethClient *ethclient.Client, l2OpGethClient *ethclient.Client, l2OpNodeClient *ethclient.Client, OptimismPortalAddress common.Address) (*ProvenWithdrawalValidator, error) {

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

	l2NodeHelper, err := NewOpNodeHelper(ctx, l2OpNodeClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create l2 node helper: %w", err)
	}

	return &ProvenWithdrawalValidator{
		optimismPortal2Helper:     optimismPortal2Helper,
		l2NodeHelper:              l2NodeHelper,
		l2ToL1MessagePasserHelper: l2ToL1MessagePasserHelper,
		faultDisputeGameHelper:    faultDisputeGameHelper,
		ctx:                       ctx,
	}, nil
}

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

	// Check if the game is blacklisted only if not confirmed already that is blaclisted
	if event.Blacklisted || !event.Enriched {
		blacklisted, err := wv.optimismPortal2Helper.IsGameBlacklisted(event.DisputeGame)
		if err != nil {
			return fmt.Errorf("failed to check if game is blacklisted: %w", err)
		}
		event.Blacklisted = blacklisted
	}

	//check if the game root claim is valid on L2 only if not confirmed already that is on L2
	if !event.Enriched {
		trustedRootClaim, err := wv.l2NodeHelper.GetOutputRootFromTrustedL2Node(event.DisputeGame.DisputeGameData.L2blockNumber)
		if err != nil {
			return fmt.Errorf("failed to get trustedRootClaim from Op-node: %w", err)
		}
		event.ExpectedRootClaim = trustedRootClaim
	}

	//check if the withdrawal exists on L2 only if not confirmed already that is on L2
	if !event.WithdrawalHashPresentOnL2 || !event.Enriched {
		withdrawalHashPresentOnL2, err := wv.l2ToL1MessagePasserHelper.WithdrawalExistsOnL2(event.Event.WithdrawalHash)
		if err != nil {
			return fmt.Errorf("failed to check withdrawal existence on L2: %w", err)
		}
		event.WithdrawalHashPresentOnL2 = withdrawalHashPresentOnL2
	}

	event.Enriched = true
	return nil
}

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
//
// This function performs the following steps:
//  1. Retrieves the proof data related to the withdrawal hash and proof submitter
//     using the `GetSumittedProofsDataFromWithdrawalhashAndProofSubmitterAddress` method.
//  2. Extracts the dispute game proxy address from the retrieved proof data.
//  3. Retrieves the corresponding DisputeGame object using the
//     `GetDisputeGameFromAddress` method with the proxy address.
//
// If any errors occur during these steps, the connection failure counter
// (`nodeConnectionFailures`) is incremented, and the error is returned.
//
// Parameters:
// - withdrawalHash: A 32-byte array representing the withdrawal hash.
// - proofSubmitter: The address of the submitter of the proof.
//
// Returns:
// - A pointer to a DisputeGame object if successful.
// - An error if any issue occurs while fetching the dispute game or proof data.
func (wv *ProvenWithdrawalValidator) getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalHash [32]byte, proofSubmitter common.Address) (FaultDisputeGameProxy, error) {

	submittedProofData, error := wv.optimismPortal2Helper.GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitter)
	if error != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games addresses: %w", error)
	}
	disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
	disputeGame, error := wv.faultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if error != nil {
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get games: %w", error)
	}

	return disputeGame, nil
}

func (wv *ProvenWithdrawalValidator) GetProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtension1Event, error) {
	return wv.optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, end)
}

func (wv *ProvenWithdrawalValidator) GetEnrichedWithdrawalsEvents(start uint64, end *uint64) ([]EnrichedProvenWithdrawalEvent, error) {
	events, err := wv.optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events: %w", err)
	}
	enrichedProvenWithdrawalEvents := make([]EnrichedProvenWithdrawalEvent, 0)

	for event := range events {
		enrichedWithdrawalEvent, err := wv.GetEnrichedWithdrawalEvent(&events[event])
		if err != nil {
			return nil, fmt.Errorf("failed to get enriched withdrawal event: %w", err)
		}
		enrichedProvenWithdrawalEvents = append(enrichedProvenWithdrawalEvents, *enrichedWithdrawalEvent)
	}

	return enrichedProvenWithdrawalEvents, nil
}

func (wv *ProvenWithdrawalValidator) IsWithdrawalEventValid(enrichedWithdrawalEvent *EnrichedProvenWithdrawalEvent) (bool, error) {
	if enrichedWithdrawalEvent.ExpectedRootClaim == [32]byte{} {
		return false, fmt.Errorf("trustedRootClaim is nil, game not enriched")
	}
	validGameRootClaim := enrichedWithdrawalEvent.DisputeGame.DisputeGameData.RootClaim == enrichedWithdrawalEvent.ExpectedRootClaim

	fmt.Println("enrichedWithdrawalEvent.WithdrawalHashPresentOnL2 ", enrichedWithdrawalEvent.WithdrawalHashPresentOnL2)
	if validGameRootClaim && enrichedWithdrawalEvent.WithdrawalHashPresentOnL2 {
		return true, nil
	} else {
		return false, nil
	}
}
