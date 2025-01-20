package validator

import (
	"math/big"
)

// L1ProxyInterface defines the interface for L1 chain interactions
type L1ProxyInterface interface {
	// Gets dispute games for withdrawal events within a block range
	GetDisputeGamesEvents(start uint64, end uint64) ([]DisputeGameEvent, error)

	// Gets updates for a specific dispute game
	GetDisputeGameProxyUpdates(disputeGame *DisputeGame) (*DisputeGame, error)

	// Returns the current connection state
	GetConnectionState() *L1ConnectionState

	// Gets the latest block height and timestamp
	LatestHeight() (BlockInfo, error)

	// Gets a block by number
	BlockByNumber(blockNumber *big.Int) (BlockInfo, error)

	// Gets the chain ID
	ChainID() (*big.Int, error)

	// Close the L1 proxy
	Close()
}

// L2ProxyInterface defines the interface for L2 chain interactions
type L2ProxyInterface interface {
	// Validates a withdrawal based on a dispute game
	GetWithdrawalValidation(disputeGameEvent DisputeGameEvent) (*WithdrawalValidationRef, error)

	// Returns the current connection state
	GetConnectionState() *L2ConnectionState

	// Gets the latest block height and timestamp
	LatestHeight() (BlockInfo, error)

	// Gets a block by number
	BlockByNumber(blockNumber *big.Int) (BlockInfo, error)

	// Gets the chain ID
	ChainID() (*big.Int, error)

	// Close the L2 proxy
	Close()
}

// Ensure L1Proxy and L2Proxy implement their respective interfaces
var _ L1ProxyInterface = (*L1Proxy)(nil)
var _ L2ProxyInterface = (*L2Proxy)(nil)

// Add these methods to the existing L1Proxy and L2Proxy structs
func (l1Proxy *L1Proxy) GetConnectionState() *L1ConnectionState {
	return l1Proxy.ConnectionState
}

func (l2Proxy *L2Proxy) GetConnectionState() *L2ConnectionState {
	return l2Proxy.ConnectionState
}
