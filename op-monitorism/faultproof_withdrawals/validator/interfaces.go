package validator

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// L1ProxyInterface defines the interface for L1 chain interactions
type L1ProxyInterface interface {
	// Gets dispute games for withdrawal events within a block range
	GetDisputeGamesForWithdrawalsEvents(start uint64, end *uint64) ([]DisputeGame, error)

	// Gets updates for a specific dispute game
	GetDisputeGameProxyUpdates(disputeGame *DisputeGame) (*DisputeGame, error)

	// Returns the current connection state
	GetConnectionState() *L1ConnectionState

	// Gets the latest block height
	LatestHeight() (uint64, error)

	// Gets a block by number
	BlockByNumber(blockNumber *big.Int) (*types.Block, error)

	// Gets the chain ID
	ChainID() (*big.Int, error)

	// Close the L1 proxy
	Close()
}

// L2ProxyInterface defines the interface for L2 chain interactions
type L2ProxyInterface interface {
	// Validates a withdrawal based on a dispute game
	GetWithdrawalValidation(disputeGame *DisputeGame) (*WithdrawalValidation, error)

	// Returns the current connection state
	GetConnectionState() *L2ConnectionState

	// Gets the latest block height
	LatestHeight() (uint64, error)

	// Gets a block by number
	BlockByNumber(blockNumber *big.Int) (*types.Block, error)

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
