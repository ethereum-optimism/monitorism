package faultproof_withdrawals

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"github.com/joho/godotenv"
)

type Raw struct {
	BlockNumber uint64
	TxHash      common.Hash
}

type Timestamp uint64

func (timestamp Timestamp) String() string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format("2006-01-02 15:04:05 MST")
}

type L2NodeHelper struct {
	//objects
	l2OpNodeClient    *ethclient.Client
	rpc_l2Client      *rpc.Client
	ctx               context.Context
	l2OutputRootCache *lru.Cache
}

func NewL2NodeHelper(ctx context.Context, l2OpNodeClient *ethclient.Client) (*L2NodeHelper, error) {
	l2OutputRootCache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	rpc_l2Client := l2OpNodeClient.Client()

	return &L2NodeHelper{
		l2OpNodeClient:    l2OpNodeClient,
		rpc_l2Client:      rpc_l2Client,
		ctx:               ctx,
		l2OutputRootCache: l2OutputRootCache,
	}, nil
}

func (op *L2NodeHelper) GetOutputRootFromTrustedL2Node(l2blockNumber *big.Int) ([32]byte, error) {

	ret, found := op.l2OutputRootCache.Get(l2blockNumber)
	if !found {
		// log.Info("Cache Miss", "l2blockNumber", l2blockNumber)

		var result OutputResponse
		l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

		err := op.rpc_l2Client.CallContext(op.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
		}
		trustedRootProof, err := stringToBytes32(result.OutputRoot)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to convert output root to bytes32: %w", err)
		}
		ret = trustedRootProof
	}

	return ret.([32]byte), nil
}

func (op *L2NodeHelper) IsValidOutputRoot(gameClaim [32]byte, l2blockNumber *big.Int) (bool, error) {
	trustedL2OutputRoot, err := op.GetOutputRootFromTrustedL2Node(l2blockNumber)
	if err != nil {
		return false, fmt.Errorf("failed to get root proof from trusted l2 node: %w", err)
	}
	return gameClaim == trustedL2OutputRoot, nil
}

func stringToBytes32(input string) ([32]uint8, error) {

	if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
		input = input[2:]
	}

	bytes, err := hex.DecodeString(input)
	if err != nil {
		return [32]uint8{}, err
	}

	// Convert bytes to [32]uint8
	var array [32]uint8
	copy(array[:], bytes)
	return array, nil
}

func loadEnv(env string) error {
	return godotenv.Load(env)
}
