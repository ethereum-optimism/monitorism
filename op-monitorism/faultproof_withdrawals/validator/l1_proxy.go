package validator

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type L1Proxy struct {
	l1GethClient           *ethclient.Client       // The L1 Ethereum client.
	optimismPortal2Helper  *OptimismPortal2Helper  // Helper for interacting with Optimism Portal 2.
	faultDisputeGameHelper *FaultDisputeGameHelper // Helper for dispute game interactions.
	ctx                    context.Context         // Context for managing cancellation and timeouts.
	Connections            uint64
	ConnectionErrors       uint64
}

func NewL1Proxy(ctx context.Context, l1GethClientURL string, OptimismPortalAddress common.Address) (*L1Proxy, error) {
	l1GethClient, err := ethclient.Dial(l1GethClientURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}

	optimismPortal2Helper, err := NewOptimismPortal2Helper(ctx, l1GethClient, OptimismPortalAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	faultDisputeGameHelper, err := NewFaultDisputeGameHelper(ctx, l1GethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create dispute game helper: %w", err)
	}

	return &L1Proxy{
		l1GethClient:           l1GethClient,
		optimismPortal2Helper:  optimismPortal2Helper,
		faultDisputeGameHelper: faultDisputeGameHelper,
		ctx:                    ctx,
		Connections:            0,
		ConnectionErrors:       0,
	}, nil
}

func (l1Proxy *L1Proxy) IsGameBlacklisted(disputeGame *FaultDisputeGameProxy) (bool, error) {
	l1Proxy.Connections++
	blacklisted, err := l1Proxy.optimismPortal2Helper.IsGameBlacklisted(disputeGame)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return false, fmt.Errorf("failed to check if game is blacklisted: %w", err)
	}
	return blacklisted, nil
}

func (l1Proxy *L1Proxy) GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash [32]byte, proofSubmitterAddress common.Address) (*SubmittedProofData, error) {
	l1Proxy.Connections++
	submittedProofData, err := l1Proxy.optimismPortal2Helper.GetSubmittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalHash, proofSubmitterAddress)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return nil, fmt.Errorf("failed to get submitted proofs data: %w", err)
	}
	return submittedProofData, nil
}

func (l1Proxy *L1Proxy) GetDisputeGameProxyFromAddress(disputeGameProxyAddress common.Address) (FaultDisputeGameProxy, error) {
	l1Proxy.Connections++
	disputeGameProxy, err := l1Proxy.faultDisputeGameHelper.GetDisputeGameProxyFromAddress(disputeGameProxyAddress)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return FaultDisputeGameProxy{}, fmt.Errorf("failed to get dispute game proxy: %w", err)
	}
	return disputeGameProxy, nil
}

func (l1Proxy *L1Proxy) GetProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtension1Event, error) {
	l1Proxy.Connections++
	events, err := l1Proxy.optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, end)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events: %w", err)
	}
	return events, nil
}

func (l1Proxy *L1Proxy) GetProvenWithdrawalsExtension1EventsIterator(start uint64, end *uint64) (*l1.OptimismPortal2WithdrawalProvenExtension1Iterator, error) {
	l1Proxy.Connections++
	eventsIterator, err := l1Proxy.optimismPortal2Helper.GetProvenWithdrawalsExtension1EventsIterator(start, end)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events iterator: %w", err)
	}
	return eventsIterator, nil
}

func (l1Proxy *L1Proxy) BlockNumber() (uint64, error) {
	l1Proxy.Connections++
	blockNumber, err := l1Proxy.l1GethClient.BlockNumber(l1Proxy.ctx)
	if err != nil {
		l1Proxy.ConnectionErrors++
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}
	return blockNumber, nil
}

func (l1Proxy *L1Proxy) GetTotalConnections() uint64 {
	return l1Proxy.Connections
}

func (l1Proxy *L1Proxy) GetTotalConnectionErrors() uint64 {
	return l1Proxy.ConnectionErrors
}
