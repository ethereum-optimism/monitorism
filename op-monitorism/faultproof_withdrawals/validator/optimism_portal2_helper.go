package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SubmittedProofData holds data about a submitted proof.
type SubmittedProofData struct {
	proofSubmitterAddress     common.Address // Address of the proof submitter.
	withdrawalHash            [32]byte       // Hash of the withdrawal.
	disputeGameProxyAddress   common.Address // Address of the dispute game proxy.
	disputeGameProxyTimestamp uint64         // Timestamp of the dispute game proxy.
}

// WithdrawalProvenExtension1Event represents an event for a proven withdrawal.
type WithdrawalProvenExtension1Event struct {
	WithdrawalHash                         [32]byte       // Hash of the withdrawal.
	ProofSubmitter                         common.Address // Address of the proof submitter.
	IsWithdrawalHashPresentOnL2TrustedNode bool           // Indicates if the withdrawal hash is present on L2 trusted node.
	Raw                                    Raw            // Raw event data.
}

// OptimismPortal2Helper assists in interacting with the Optimism Portal 2.
type OptimismPortal2Helper struct {
	// objects
	l1Client        *ethclient.Client   // The L1 Ethereum client.
	optimismPortal2 *l1.OptimismPortal2 // The Optimism Portal 2 contract instance.
	ctx             context.Context     // Context for managing cancellation and timeouts.
}

// String provides a string representation of WithdrawalProvenExtension1Event.
func (e WithdrawalProvenExtension1Event) String() string {
	return fmt.Sprintf("WithdrawalHash: %s, ProofSubmitter: %v, IsWithdrawalHashPresentOnL2TrustedNode:%v, Raw: %v", common.BytesToHash(e.WithdrawalHash[:]), e.ProofSubmitter, e.IsWithdrawalHashPresentOnL2TrustedNode, e.Raw)
}

// String provides a string representation of SubmittedProofData.
func (p *SubmittedProofData) String() string {
	return fmt.Sprintf("proofSubmitterAddress: %x, withdrawalHash: %x, disputeGameProxyAddress: %x, disputeGameProxyTimestamp: %d", p.proofSubmitterAddress, p.withdrawalHash, p.disputeGameProxyAddress, p.disputeGameProxyTimestamp)
}

// NewOptimismPortal2Helper initializes a new OptimismPortal2Helper.
// It binds to the Optimism Portal 2 contract and returns the helper instance.
func NewOptimismPortal2Helper(ctx context.Context, l1Client *ethclient.Client, optimismPortalAddress common.Address) (*OptimismPortal2Helper, error) {
	optimismPortal, err := l1.NewOptimismPortal2(optimismPortalAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	return &OptimismPortal2Helper{
		l1Client:        l1Client,
		optimismPortal2: optimismPortal,
		ctx:             ctx,
	}, nil
}

// IsGameBlacklisted checks if a dispute game is blacklisted.
// It returns true if the game is blacklisted, otherwise returns false along with any error encountered.
func (op *OptimismPortal2Helper) IsGameBlacklisted(disputeGameProxy *FaultDisputeGameProxy) (bool, error) {
	isBlacklisted, err := op.optimismPortal2.DisputeGameBlacklist(nil, disputeGameProxy.DisputeGameData.ProxyAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get dispute game blacklist status: %w", err)
	}

	return isBlacklisted, nil
}

// GetDisputeGameFactoryAddress retrieves the address of the dispute game factory.
// It returns the address along with any error encountered.
func (op *OptimismPortal2Helper) GetDisputeGameFactoryAddress() (common.Address, error) {
	disputeGameFactoryAddress, err := op.optimismPortal2.DisputeGameFactory(nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get dispute game factory address: %w", err)
	}
	return disputeGameFactoryAddress, nil
}

// GetProvenWithdrawalsEventsIterartor creates an iterator for proven withdrawal events within the specified block range.
// It returns the iterator along with any error encountered.
func (op *OptimismPortal2Helper) GetProvenWithdrawalsEventsIterartor(start uint64, end *uint64) (*l1.OptimismPortal2WithdrawalProvenIterator, error) {
	filterOpts := &bind.FilterOpts{Context: op.ctx, Start: start, End: end}
	iterator, err := op.optimismPortal2.FilterWithdrawalProven(filterOpts, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to filter withdrawal proven start_block:%d end_block:%d error:%w", start, *end, err)
	}

	return iterator, nil
}

// GetProvenWithdrawalsExtension1EventsIterator creates an iterator for proven withdrawal extension 1 events within the specified block range.
// It returns the iterator along with any error encountered.
func (op *OptimismPortal2Helper) GetProvenWithdrawalsExtension1EventsIterator(start uint64, end *uint64) (*l1.OptimismPortal2WithdrawalProvenExtension1Iterator, error) {
	filterOpts := &bind.FilterOpts{Context: op.ctx, Start: start, End: end}
	iterator, err := op.optimismPortal2.FilterWithdrawalProvenExtension1(filterOpts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to filter withdrawal proven start_block:%d end_block:%d error:%w", start, *end, err)
	}

	return iterator, nil
}

// GetProvenWithdrawalsExtension1Events retrieves proven withdrawal extension 1 events within the specified block range.
// It returns a slice of WithdrawalProvenExtension1Event along with any error encountered.
func (op *OptimismPortal2Helper) GetProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtension1Event, error) {
	iterator, err := op.GetProvenWithdrawalsExtension1EventsIterator(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 iterator error:%w", err)
	}

	events := make([]WithdrawalProvenExtension1Event, 0)
	for iterator.Next() {
		event := iterator.Event
		events = append(events, WithdrawalProvenExtension1Event{
			WithdrawalHash: event.WithdrawalHash,
			ProofSubmitter: event.ProofSubmitter,
			Raw: Raw{
				BlockNumber: event.Raw.BlockNumber,
				TxHash:      event.Raw.TxHash,
			},
		})
	}

	return events, nil
}

// GetSubmittedProofsDataFromWithdrawalhash retrieves submitted proof data associated with the given withdrawal hash.
// It returns a slice of SubmittedProofData along with any error encountered.
func (op *OptimismPortal2Helper) GetSubmittedProofsDataFromWithdrawalhash(withdrawalHash [32]byte) ([]SubmittedProofData, error) {
	numProofSubmitters, err := op.optimismPortal2.NumProofSubmitters(nil, withdrawalHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get num proof submitters for withdrawal hash:%x error:%w", withdrawalHash, err)
	}

	withdrawals := make([]SubmittedProofData, numProofSubmitters.Int64())

	for i := 0; i < int(numProofSubmitters.Int64()); i++ {
		proofSubmitterAddress, err := op.optimismPortal2.ProofSubmitters(nil, withdrawalHash, big.NewInt(int64(i)))
		if err != nil {
			return nil, fmt.Errorf("failed to get proof submitter for withdrawal hash:%x index:%d error:%w", withdrawalHash, i, err)
		}
		gameProxyStruct, err := op.optimismPortal2.ProvenWithdrawals(nil, withdrawalHash, proofSubmitterAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to get proven withdrawal for withdrawal hash:%x proof submitter:%x error:%w", withdrawalHash, proofSubmitterAddress, err)
		}

		withdrawals[i] = SubmittedProofData{
			proofSubmitterAddress:     proofSubmitterAddress,
			withdrawalHash:            withdrawalHash,
			disputeGameProxyAddress:   gameProxyStruct.DisputeGameProxy,
			disputeGameProxyTimestamp: gameProxyStruct.Timestamp,
		}
	}

	return withdrawals, nil
}

// GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress retrieves submitted proof data
// for the specified withdrawal hash and proof submitter address.
// It returns a pointer to SubmittedProofData along with any error encountered.
func (op *OptimismPortal2Helper) GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash [32]byte, proofSubmitterAddress common.Address) (*SubmittedProofData, error) {
	gameProxyStruct, err := op.optimismPortal2.ProvenWithdrawals(nil, withdrawalHash, proofSubmitterAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawal for withdrawal hash:%x proof submitter:%x error:%w", withdrawalHash, proofSubmitterAddress, err)
	}

	return &SubmittedProofData{
		proofSubmitterAddress:     proofSubmitterAddress,
		withdrawalHash:            withdrawalHash,
		disputeGameProxyAddress:   gameProxyStruct.DisputeGameProxy,
		disputeGameProxyTimestamp: gameProxyStruct.Timestamp,
	}, nil
}
