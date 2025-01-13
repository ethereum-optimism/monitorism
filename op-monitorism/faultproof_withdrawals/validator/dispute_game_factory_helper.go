package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DisputeGameFactoryCoordinates holds the details of a dispute game.
type DisputeGameFactoryCoordinates struct {
	GameType                  uint32         // The type of the dispute game.
	GameIndex                 uint64         // The index of the dispute game.
	disputeGameProxyAddress   common.Address // The address of the dispute game proxy.
	disputeGameProxyTimestamp uint64         // The timestamp of the dispute game proxy.
}

// DisputeFactoryGameHelper assists in interacting with the dispute game factory.
type DisputeFactoryGameHelper struct {
	// objects
	l1Client                 *ethclient.Client                // The L1 Ethereum client.
	DisputeGameFactoryCaller dispute.DisputeGameFactoryCaller // Caller for the dispute game factory contract.
}

// DisputeGameFactoryIterator iterates through dispute games.
type DisputeGameFactoryIterator struct {
	DisputeGameFactoryCaller      *dispute.DisputeGameFactoryCaller // Caller for the dispute game factory contract.
	currentIndex                  uint64                            // The current index in the iteration.
	gameCount                     uint64                            // Total number of games available.
	init                          bool                              // Indicates if the iterator has been initialized.
	DisputeGameFactoryCoordinates *DisputeGameFactoryCoordinates    // Coordinates for the current dispute game.
}

// NewDisputeGameFactoryHelper initializes a new DisputeFactoryGameHelper.
// It binds to the dispute game factory contract and returns a helper instance.
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

// GetDisputeGameCoordinatesFromGameIndex retrieves the coordinates of a dispute game by its index.
// It returns the coordinates including game type, index, proxy address, and timestamp.
func (df *DisputeFactoryGameHelper) GetDisputeGameCoordinatesFromGameIndex(gameIndex uint64) (*DisputeGameFactoryCoordinates, error) {
	gameDetails, err := df.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(gameIndex)))
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

// GetDisputeGameCount returns the total count of dispute games available in the factory.
func (df *DisputeFactoryGameHelper) GetDisputeGameCount() (uint64, error) {
	gameCountBigInt, err := df.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	return gameCountBigInt.Uint64(), nil
}

// GetDisputeGameIteratorFromDisputeGameFactory creates an iterator for the dispute games in the factory.
// It returns the iterator with the total number of games.
func (df *DisputeFactoryGameHelper) GetDisputeGameIteratorFromDisputeGameFactory() (*DisputeGameFactoryIterator, error) {
	gameCountBigInt, err := df.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	gameCount := gameCountBigInt.Uint64()

	return &DisputeGameFactoryIterator{
		DisputeGameFactoryCaller:      &df.DisputeGameFactoryCaller,
		currentIndex:                  0,
		gameCount:                     gameCount,
		DisputeGameFactoryCoordinates: nil,
	}, nil
}

// RefreshElements refreshes the game count for the iterator.
func (dgf *DisputeGameFactoryIterator) RefreshElements() error {
	gameCountBigInt, err := dgf.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return fmt.Errorf("failed to get num dispute games: %w", err)
	}
	dgf.gameCount = gameCountBigInt.Uint64()
	return nil
}

// Next moves the iterator to the next dispute game.
// It returns true if there is a next game; otherwise, false.
func (dgf *DisputeGameFactoryIterator) Next() bool {
	if dgf.currentIndex >= dgf.gameCount-1 {
		return false
	}

	var currentIndex uint64 = 0
	if dgf.init {
		currentIndex = dgf.currentIndex + 1
	}

	gameDetails, err := dgf.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(currentIndex)))
	if err != nil {
		return false
	}

	dgf.init = true
	dgf.currentIndex = currentIndex

	dgf.DisputeGameFactoryCoordinates = &DisputeGameFactoryCoordinates{
		GameType:                  gameDetails.GameType,
		GameIndex:                 currentIndex,
		disputeGameProxyAddress:   gameDetails.Proxy,
		disputeGameProxyTimestamp: gameDetails.Timestamp,
	}

	return true
}
