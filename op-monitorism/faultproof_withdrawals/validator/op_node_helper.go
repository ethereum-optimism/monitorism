package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
)

// OpNodeHelper assists in interacting with the op-node
type OpNodeHelper struct {
	// objects
	l2OpGethClient    *ethclient.Client // The op-geth client.
	ctx               context.Context   // Context for managing cancellation and timeouts.
	l2OutputRootCache *lru.Cache        // Cache for storing L2 output roots.
}

const outputRootCacheSize = 1000 // Size of the output root cache.

// NewOpNodeHelper initializes a new OpNodeHelper.
// It creates a cache for storing output roots and binds to the L2 node client.
func NewOpNodeHelper(ctx context.Context, l2OpGethClient *ethclient.Client) (*OpNodeHelper, error) {
	l2OutputRootCache, err := lru.New(outputRootCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	ret := OpNodeHelper{
		l2OpGethClient:    l2OpGethClient,
		ctx:               ctx,
		l2OutputRootCache: l2OutputRootCache,
	}

	return &ret, nil

}

// get latest known L2 block number
func (on *OpNodeHelper) BlockNumber() (uint64, error) {
	return on.l2OpGethClient.BlockNumber(on.ctx)
}

// GetOutputRootFromCalculation retrieves the output root by calculating it from the given block number.
// It returns the calculated output root as a Bytes32 array.
func (on *OpNodeHelper) GetOutputRootFromCalculation(blockNumber *big.Int) ([32]byte, error) {
	// We get the block from our trusted op-geth node
	block, err := on.l2OpGethClient.BlockByNumber(on.ctx, blockNumber)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get output at block for game blockInt:%v error:%w", blockNumber, err)
	}

	// We get proof from our trusted op-geth node if present
	accountResult, err := on.RetrieveEthProof(blockNumber)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get proof: %w", err)
	}
	// verify the proof when this comes from untrusted node (merkle trie)
	err = accountResult.Verify(block.Root())
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to verify proof: %w", err)
	}
	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: [32]byte(block.Root()), MessagePasserStorageRoot: [32]byte(accountResult.StorageHash), BlockHash: block.Hash()})
	return outputRoot, nil
}

// we retrieve the proof from the truested op-geth node and eventually from backup nodes if present
func (on *OpNodeHelper) RetrieveEthProof(blockNumber *big.Int) (AccountResult, error) {
	accountResult := AccountResult{}
	err := on.l2OpGethClient.Client().CallContext(on.ctx, &accountResult, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(blockNumber))
	if err != nil {
		return AccountResult{}, fmt.Errorf("failed to get proof: %w", err)
	}
	return accountResult, nil
}
