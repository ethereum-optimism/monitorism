package faultproof_withdrawals

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type ValidateProofWithdrawalState uint8

const (
	INVALID_PROOF_FORGERY_DETECTED ValidateProofWithdrawalState = iota
	INVALID_PROPOSAL_FORGERY_DETECTED
	INVALID_PROPOSAL_INPROGRESS
	INVALID_PROPOSAL_CORRECTLY_RESOLVED
	PROOF_ON_BLACKLISTED_GAME
	VALID_PROOF
)

type EnrichedWithdrawalEvent struct {
	Event                     *WithdrawalProvenExtension1Event
	DisputeGame               *FaultDisputeGameProxy
	validGameRootClaim        bool
	Blacklisted               bool
	withdrawalHashPresentOnL2 bool
	Enriched                  bool
}

type WithdrawalValidator struct {
	optimismPortal2Helper     *OptimismPortal2Helper
	l2NodeHelper              *L2NodeHelper
	l2ToL1MessagePasserHelper *L2ToL1MessagePasserHelper
	faultDisputeGameHelper    *FaultDisputeGameHelper
	ctx                       context.Context
}

func (e *EnrichedWithdrawalEvent) String() string {
	return fmt.Sprintf("Event: %v, DisputeGame: %v, validGameRootClaim: %v, Blacklisted: %v, withdrawalHashPresentOnL2: %v, Enriched: %v", e.Event, e.DisputeGame, e.validGameRootClaim, e.Blacklisted, e.withdrawalHashPresentOnL2, e.Enriched)
}

func (v ValidateProofWithdrawalState) String() string {
	return [...]string{"INVALID_PROOF_FORGERY_DETECTED", "INVALID_PROPOSAL_FORGERY_DETECTED", "INVALID_PROPOSAL_INPROGRESS", "INVALID_PROPOSAL_CORRECTLY_RESOLVED", "PROOF_ON_BLACKLISTED_GAME", "VALID_PROOF"}[v]
}

func NewWithdrawalValidator(ctx context.Context, optimismPortal2Helper *OptimismPortal2Helper, l2NodeHelper *L2NodeHelper, l2ToL1MessagePasserHelper *L2ToL1MessagePasserHelper, faultDisputeGameHelper *FaultDisputeGameHelper) *WithdrawalValidator {
	return &WithdrawalValidator{
		optimismPortal2Helper:     optimismPortal2Helper,
		l2NodeHelper:              l2NodeHelper,
		l2ToL1MessagePasserHelper: l2ToL1MessagePasserHelper,
		faultDisputeGameHelper:    faultDisputeGameHelper,
		ctx:                       ctx,
	}
}

func (m *WithdrawalValidator) UpdateEnrichedWithdrawalEvent(event *EnrichedWithdrawalEvent) error {

	err := event.DisputeGame.RefreshState()
	if err != nil {
		return fmt.Errorf("failed to refresh game state: %w", err)
	}

	// Check if the game is blacklisted only if not confirmed already that is blaclisted
	if event.Blacklisted || !event.Enriched {
		blacklisted, err := m.optimismPortal2Helper.IsGameBlacklisted(event.DisputeGame)
		if err != nil {
			return fmt.Errorf("failed to check if game is blacklisted: %w", err)
		}
		event.Blacklisted = blacklisted
	}

	//check if the game root claim is valid on L2 only if not confirmed already that is on L2
	if !event.validGameRootClaim || !event.Enriched {
		validGameRootClaim, err := m.l2NodeHelper.IsValidOutputRoot(event.DisputeGame.DisputeGameData.RootClaim, event.DisputeGame.DisputeGameData.L2blockNumber)
		if err != nil {
			return fmt.Errorf("failed to check validGameRootClaim on L2: %w", err)
		}
		event.validGameRootClaim = validGameRootClaim
	}

	//check if the withdrawal exists on L2 only if not confirmed already that is on L2
	if !event.withdrawalHashPresentOnL2 || !event.Enriched {
		withdrawalHashPresentOnL2, err := m.l2ToL1MessagePasserHelper.WithdrawalExistsOnL2(event.Event.WithdrawalHash)
		if err != nil {
			return fmt.Errorf("failed to check withdrawal existence on L2: %w", err)
		}
		event.withdrawalHashPresentOnL2 = withdrawalHashPresentOnL2
	}

	event.Enriched = true
	return nil
}

func (m *WithdrawalValidator) GetEnrichedWithdrawalEvent(withdrawalEvent *WithdrawalProvenExtension1Event) (*EnrichedWithdrawalEvent, error) {
	disputeGameProxy, err := m.getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalEvent.WithdrawalHash, withdrawalEvent.ProofSubmitter)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute games: %w", err)
	}

	enrichedWithdrawalEvent := EnrichedWithdrawalEvent{
		Event:              withdrawalEvent,
		DisputeGame:        disputeGameProxy,
		validGameRootClaim: false,
		Blacklisted:        false,
		Enriched:           false,
	}

	return &enrichedWithdrawalEvent, nil

}

func (m *WithdrawalValidator) ValidateWithdrawal(enrichedWithdrawalEvent *EnrichedWithdrawalEvent) ValidateProofWithdrawalState {

	if enrichedWithdrawalEvent.Blacklisted {
		return PROOF_ON_BLACKLISTED_GAME
	}

	if enrichedWithdrawalEvent.validGameRootClaim {
		// Assuming the l1 and l2 node are not malicious. We have a valid Game Root Claim.
		if enrichedWithdrawalEvent.withdrawalHashPresentOnL2 {
			// In this case all the information matches. The withdrawal is valid.
			return VALID_PROOF
		} else {
			return INVALID_PROOF_FORGERY_DETECTED
		}
	} else {
		if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == IN_PROGRESS {
			// The game is still in progress. We can't make a decision yet.
			return INVALID_PROPOSAL_INPROGRESS
		} else if enrichedWithdrawalEvent.DisputeGame.DisputeGameData.Status == DEFENDER_WINS {
			// The game is resolved and the defender won. This is a forgery.
			return INVALID_PROPOSAL_FORGERY_DETECTED
		} else {
			// The game is resolved and the challenger won. The withdrawal is not valid.
			return INVALID_PROPOSAL_CORRECTLY_RESOLVED
		}
	}
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
func (m *WithdrawalValidator) getDisputeGamesFromWithdrawalhashAndProofSubmitter(withdrawalHash [32]byte, proofSubmitter common.Address) (*FaultDisputeGameProxy, error) {

	submittedProofData, error := m.optimismPortal2Helper.GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitter)
	if error != nil {
		return nil, fmt.Errorf("failed to get games addresses: %w", error)
	}
	disputeGameProxyAddress := submittedProofData.disputeGameProxyAddress
	disputeGame, error := m.faultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if error != nil {
		return nil, fmt.Errorf("failed to get games: %w", error)
	}

	return disputeGame, nil
}
