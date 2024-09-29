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

type OpNodeHelper struct {
	//objects
	l2OpNodeClient    *ethclient.Client
	rpc_l2Client      *rpc.Client
	ctx               context.Context
	l2OutputRootCache *lru.Cache
}

const outputRootCacheSize = 1000

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

func (on *OpNodeHelper) GetOutputRootFromTrustedL2Node(l2blockNumber *big.Int) ([32]byte, error) {

	ret, found := on.l2OutputRootCache.Get(l2blockNumber)
	if !found {

		var result OutputResponse
		l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

		err := on.rpc_l2Client.CallContext(on.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
		}
		trustedRootProof, err := StringToBytes32(result.OutputRoot)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to convert output root to bytes32: %w", err)
		}
		ret = trustedRootProof
	}

	return ret.([32]byte), nil
}

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

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: eth.Bytes32(block.Root()), MessagePasserStorageRoot: eth.Bytes32(proof.StorageHash), BlockHash: block.Hash()})
	return outputRoot, nil
}
