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
	l2OpNodeClient    *ethclient.Client // The op-node (consensus) client.
	rpc_l2Client      *rpc.Client       // The RPC client for the L2 node.
	ctx               context.Context   // Context for managing cancellation and timeouts.
	l2OutputRootCache *lru.Cache        // Cache for storing L2 output roots.
}

const outputRootCacheSize = 1000 // Size of the output root cache.

// NewOpNodeHelper initializes a new OpNodeHelper.
// It creates a cache for storing output roots and binds to the L2 node client.
func NewOpNodeHelper(ctx context.Context, l2OpNodeClient *ethclient.Client) (*OpNodeHelper, error) {
	l2OutputRootCache, err := lru.New(outputRootCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	rpc_l2Client := l2OpNodeClient.Client()

	return &OpNodeHelper{
		l2OpNodeClient:    l2OpNodeClient,
		rpc_l2Client:      rpc_l2Client,
		ctx:               ctx,
		l2OutputRootCache: l2OutputRootCache,
	}, nil
}

// GetOutputRootFromTrustedL2Node retrieves the output root for a given L2 block number from a trusted L2 node.
// It returns the output root as a eth.Bytes32 array.
func (on *OpNodeHelper) GetOutputRootFromTrustedL2Node(l2blockNumber *big.Int) (eth.Bytes32, error) {
	ret, found := on.l2OutputRootCache.Get(l2blockNumber)

	if !found {
		var result OutputResponse
		l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

		err := on.rpc_l2Client.CallContext(on.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
		if err != nil {
			return eth.Bytes32{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
		}
		trustedRootProof, err := StringToBytes32(result.OutputRoot)
		if err != nil {
			return eth.Bytes32{}, fmt.Errorf("failed to convert output root to eth.Bytes32: %w", err)
		}
		ret = eth.Bytes32(trustedRootProof)
	}

	return ret.(eth.Bytes32), nil
}

// GetOutputRootFromCalculation retrieves the output root by calculating it from the given block number.
// It returns the calculated output root as a eth.Bytes32 array.
func (on *OpNodeHelper) GetOutputRootFromCalculation(blockNumber *big.Int) (eth.Bytes32, error) {
	block, err := on.l2OpNodeClient.BlockByNumber(on.ctx, blockNumber)
	if err != nil {
		return eth.Bytes32{}, fmt.Errorf("failed to get block by number: %w", err)
	}

	proof := struct{ StorageHash common.Hash }{}
	err = on.l2OpNodeClient.Client().CallContext(on.ctx, &proof, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(blockNumber))
	if err != nil {
		return eth.Bytes32{}, fmt.Errorf("failed to get proof: %w", err)
	}

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: eth.Bytes32(block.Root()), MessagePasserStorageRoot: eth.Bytes32(proof.StorageHash), BlockHash: block.Hash()})
	return outputRoot, nil
}
