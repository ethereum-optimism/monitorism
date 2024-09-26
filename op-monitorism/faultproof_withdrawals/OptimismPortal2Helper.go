package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type SubmittedProofData struct {
	proofSubmitterAddress     common.Address
	withdrawalHash            [32]byte
	disputeGameProxyAddress   common.Address
	disputeGameProxyTimestamp uint64
}

type WithdrawalProvenExtension1Event struct {
	WithdrawalHash [32]byte
	ProofSubmitter common.Address
	Raw            Raw
}

type WithdrawalProvenEvent struct {
	WithdrawalHash [32]byte
	Raw            Raw
}

type OptimismPortal2Helper struct {
	//objects
	l1Client        *ethclient.Client
	optimismPortal2 *l1.OptimismPortal2

	ctx context.Context
}

func (e *WithdrawalProvenExtension1Event) String() string {
	return fmt.Sprintf("WithdrawalHash: %x, ProofSubmitter: %v, Raw: %v", e.WithdrawalHash, e.ProofSubmitter, e.Raw)
}

func (e *WithdrawalProvenEvent) String() string {
	return fmt.Sprintf("WithdrawalHash: %x, Raw: %v", e.WithdrawalHash, e.Raw)
}

func (p *SubmittedProofData) String() string {
	return fmt.Sprintf("proofSubmitterAddress: %x, withdrawalHash: %x, disputeGameProxyAddress: %x, disputeGameProxyTimestamp: %d", p.proofSubmitterAddress, p.withdrawalHash, p.disputeGameProxyAddress, p.disputeGameProxyTimestamp)
}

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

func (op *OptimismPortal2Helper) IsGameBlacklisted(disputeGameProxy *FaultDisputeGameProxy) (bool, error) {

	isBlacklisted, err := op.optimismPortal2.DisputeGameBlacklist(nil, disputeGameProxy.DisputeGameData.ProxyAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get dispute game blacklist status: %w", err)
	}

	return isBlacklisted, err
}

func (op *OptimismPortal2Helper) GetDisputeGameFactoryAddress() (common.Address, error) {

	disputeGameFactoryAddress, err := op.optimismPortal2.DisputeGameFactory(nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get dispute game factory address: %w", err)
	}
	return disputeGameFactoryAddress, nil
}

func (op *OptimismPortal2Helper) GetProvenWithdrawalsEventsIterartor(start uint64, end *uint64) (*l1.OptimismPortal2WithdrawalProvenIterator, error) {

	filterOpts := &bind.FilterOpts{Context: op.ctx, Start: start, End: end}
	iterator, err := op.optimismPortal2.FilterWithdrawalProven(filterOpts, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to filter withdrawal proven start_block:%d end_block:%d error:%w", start, *end, err)
	}

	return iterator, nil
}

func (op *OptimismPortal2Helper) GetProvenWithdrawalsEvents(start uint64, end *uint64) ([]WithdrawalProvenEvent, error) {

	iterator, err := op.GetProvenWithdrawalsEventsIterartor(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 iterator error:%w", err)
	}

	events := make([]WithdrawalProvenEvent, 0)
	for iterator.Next() {
		event := iterator.Event
		events = append(events, WithdrawalProvenEvent{
			WithdrawalHash: event.WithdrawalHash,
			Raw: Raw{
				BlockNumber: event.Raw.BlockNumber,
				TxHash:      event.Raw.TxHash,
			},
		})
	}

	return events, nil

}

func (op *OptimismPortal2Helper) GetProvenWithdrawalsExtension1EventsIterartor(start uint64, end *uint64) (*l1.OptimismPortal2WithdrawalProvenExtension1Iterator, error) {

	filterOpts := &bind.FilterOpts{Context: op.ctx, Start: start, End: end}
	iterator, err := op.optimismPortal2.FilterWithdrawalProvenExtension1(filterOpts, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to filter withdrawal proven start_block:%d end_block:%d error:%w", start, *end, err)
	}

	return iterator, nil
}

func (op *OptimismPortal2Helper) GetProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtension1Event, error) {

	iterator, err := op.GetProvenWithdrawalsExtension1EventsIterartor(start, end)
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
