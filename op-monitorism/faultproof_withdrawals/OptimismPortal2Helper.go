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
	proofSubmitterIndex       uint64
	disputeGameProxyAddress   common.Address
	disputeGameProxyTimestamp uint64
}

type WithdrawalEvent struct {
	WithdrawalHash [32]byte
	BlockNumber    uint64
	TxHash         common.Hash
}

type OptimismPortal2Helper struct {
	//strings
	optimismPortalAddress common.Address

	//objects
	l1Client        *ethclient.Client
	optimismPortal2 *l1.OptimismPortal2

	ctx context.Context
}

func NewOptimismPortal2Helper(ctx context.Context, l1Client *ethclient.Client, optimismPortalAddress common.Address) (*OptimismPortal2Helper, error) {

	optimismPortal, err := l1.NewOptimismPortal2(optimismPortalAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	return &OptimismPortal2Helper{
		optimismPortalAddress: optimismPortalAddress,

		l1Client:        l1Client,
		optimismPortal2: optimismPortal,
		ctx:             ctx,
	}, nil
}

func (op *OptimismPortal2Helper) IsGameBlacklisted(disputeGame *DisputeGame) (bool, error) {

	isBlacklisted, err := op.optimismPortal2.DisputeGameBlacklist(nil, disputeGame.disputeGameProxyAddress)
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

func (op *OptimismPortal2Helper) GetSumittedProofsDataFromWithdrawalhash(withdrawalHash [32]byte) ([]SubmittedProofData, error) {

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
			proofSubmitterIndex:       uint64(i),
			disputeGameProxyAddress:   gameProxyStruct.DisputeGameProxy,
			disputeGameProxyTimestamp: gameProxyStruct.Timestamp,
		}

	}

	return withdrawals, nil
}
