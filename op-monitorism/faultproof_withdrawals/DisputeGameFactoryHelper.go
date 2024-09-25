package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type DisputeGameFactoryCoordinates struct {
	GameType                  uint32
	GameIndex                 uint64
	disputeGameProxyAddress   common.Address
	disputeGameProxyTimestamp uint64
}

type DisputeFactoryGameHelper struct {
	//objects
	l1Client                 *ethclient.Client
	DisputeGameFactoryCaller dispute.DisputeGameFactoryCaller
}

type DisputeGameFactoryIterator struct {
	DisputeGameFactoryCaller      *dispute.DisputeGameFactoryCaller
	currentIndex                  uint64
	gameCount                     uint64
	init                          bool
	DisputeGameFactoryCoordinates *DisputeGameFactoryCoordinates
}

func NewDisputeGameFactoryHelper(ctx context.Context, l1Client *ethclient.Client, disputeGameFactoryAddress common.Address) (*DisputeFactoryGameHelper, error) {

	disputeGameFactory, err := dispute.NewDisputeGameFactory(disputeGameFactoryAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to dispute game factory: %w", err)
	}
	disputeGameFactoryCaller := disputeGameFactory.DisputeGameFactoryCaller

	return &DisputeFactoryGameHelper{
		l1Client:                 l1Client,
		DisputeGameFactoryCaller: disputeGameFactoryCaller,
	}, nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameCoordinatesFromGameIndex(gameIndex uint64) (*DisputeGameFactoryCoordinates, error) {
	gameDetails, err := op.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(gameIndex)))
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute game details: %w", err)
	}

	return &DisputeGameFactoryCoordinates{
		GameType:                  gameDetails.GameType,
		GameIndex:                 gameIndex,
		disputeGameProxyAddress:   gameDetails.Proxy,
		disputeGameProxyTimestamp: gameDetails.Timestamp,
	}, nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameCount() (uint64, error) {
	gameCountBigInt, err := op.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	return gameCountBigInt.Uint64(), nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameIteratorFromDisputeGameFactory() (*DisputeGameFactoryIterator, error) {

	gameCountBigInt, err := op.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	gameCount := gameCountBigInt.Uint64()

	return &DisputeGameFactoryIterator{
		DisputeGameFactoryCaller:      &op.DisputeGameFactoryCaller,
		currentIndex:                  0,
		gameCount:                     gameCount,
		DisputeGameFactoryCoordinates: nil,
	}, nil
}

func (it *DisputeGameFactoryIterator) RefreshElements() error {
	gameCountBigInt, err := it.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return fmt.Errorf("failed to get num dispute games: %w", err)
	}
	it.gameCount = gameCountBigInt.Uint64()
	return nil
}

func (it *DisputeGameFactoryIterator) Next() bool {
	if it.currentIndex >= it.gameCount-1 {
		return false
	}

	var currentIndex uint64 = 0
	if it.init {
		currentIndex = it.currentIndex + 1
	}

	gameDetails, err := it.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(currentIndex)))
	if err != nil {
		return false
	}

	it.init = true
	it.currentIndex = currentIndex

	it.DisputeGameFactoryCoordinates = &DisputeGameFactoryCoordinates{
		GameType:                  gameDetails.GameType,
		GameIndex:                 currentIndex,
		disputeGameProxyAddress:   gameDetails.Proxy,
		disputeGameProxyTimestamp: gameDetails.Timestamp,
	}

	return true
}
