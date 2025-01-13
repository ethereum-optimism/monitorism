package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
)

// OpNodeHelper assists in interacting with the op-node
type OpNodeHelper struct {
	// objects
	l2OpNodeClient           *ethclient.Client // The op-node (consensus) client.
	l2OpGethClient           *ethclient.Client // The op-geth client.
	rpc_l2Client             *rpc.Client       // The RPC client for the L2 node.
	ctx                      context.Context   // Context for managing cancellation and timeouts.
	l2OutputRootCache        *lru.Cache        // Cache for storing L2 output roots.
	LatestKnownL2BlockNumber uint64            // The latest known L2 block number.
}

const outputRootCacheSize = 1000 // Size of the output root cache.

// NewOpNodeHelper initializes a new OpNodeHelper.
// It creates a cache for storing output roots and binds to the L2 node client.
func NewOpNodeHelper(ctx context.Context, l2OpNodeClient *ethclient.Client, l2OpGethClient *ethclient.Client) (*OpNodeHelper, error) {
	l2OutputRootCache, err := lru.New(outputRootCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	rpc_l2Client := l2OpNodeClient.Client()

	ret := OpNodeHelper{
		l2OpNodeClient:           l2OpNodeClient,
		l2OpGethClient:           l2OpGethClient,
		rpc_l2Client:             rpc_l2Client,
		ctx:                      ctx,
		l2OutputRootCache:        l2OutputRootCache,
		LatestKnownL2BlockNumber: 0,
	}

	//ignoring the return value as it is already stored in the struct by the method
	latestBlockNumber, err := ret.GetLatestKnownL2BlockNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest known L2 block number: %w", err)
	}

	ret.LatestKnownL2BlockNumber = latestBlockNumber
	return &ret, nil

}

// get latest known L2 block number
func (on *OpNodeHelper) GetLatestKnownL2BlockNumber() (uint64, error) {
	LatestKnownL2BlockNumber, err := on.l2OpGethClient.BlockNumber(on.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest known L2 block number: %w", err)
	}
	on.LatestKnownL2BlockNumber = LatestKnownL2BlockNumber
	return LatestKnownL2BlockNumber, nil
}

// GetOutputRootFromTrustedL2Node retrieves the output root for a given L2 block number from a trusted L2 node.
// It returns the output root as a Bytes32 array.
func (on *OpNodeHelper) GetOutputRootFromTrustedL2Node(l2blockNumber *big.Int) ([32]byte, error) {
	ret, found := on.l2OutputRootCache.Get(l2blockNumber)

	if !found {
		var result OutputResponse
		l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

		err := on.rpc_l2Client.CallContext(on.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
		//check if error contains "failed to determine L2BlockRef of height"
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
		}
		trustedRootProof, err := StringToBytes32(result.OutputRoot)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to convert output root to Bytes32: %w", err)
		}
		ret = [32]byte(trustedRootProof)
		on.l2OutputRootCache.Add(l2blockNumber, ret)
	}

	return ret.([32]byte), nil
}

// GetOutputRootFromCalculation retrieves the output root by calculating it from the given block number.
// It returns the calculated output root as a Bytes32 array.
func (on *OpNodeHelper) GetOutputRootFromCalculation(blockNumber *big.Int) ([32]byte, error) {
	block, err := on.l2OpNodeClient.BlockByNumber(on.ctx, blockNumber)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get block by number: %w", err)
	}

	proof := struct{ StorageHash common.Hash }{}
	err = on.l2OpNodeClient.Client().CallContext(on.ctx, &proof, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(blockNumber))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get proof: %w", err)
	}

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: [32]byte(block.Root()), MessagePasserStorageRoot: [32]byte(proof.StorageHash), BlockHash: block.Hash()})
	return outputRoot, nil
}
