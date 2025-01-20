package validator

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l2"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type WithdrawalValidation struct {
	DisputeGameEvent                       DisputeGameEvent
	TrustedRoots                           [32]byte
	WithdrawalPresentOnL2ToL1MessagePasser bool
	BlockPresentOnL2                       bool
	IsWithdrawalValid                      bool
}

func (dgcd *WithdrawalValidation) String() string {
	return fmt.Sprintf("DisputeGameEvent: %v, TrustedRoots: 0x%v, WithdrawalPresentOnL2ToL1MessagePasser: %v, BlockPresentOnL2: %v, IsWithdrawalValid: %v", dgcd.DisputeGameEvent, hex.EncodeToString(dgcd.TrustedRoots[:]), dgcd.WithdrawalPresentOnL2ToL1MessagePasser, dgcd.BlockPresentOnL2, dgcd.IsWithdrawalValid)
}

type L2Proxy struct {
	ctx                 *context.Context
	l2NodeClient        *ethclient.Client // The op-node (consensus) client.
	l2GethClient        *ethclient.Client // The op-geth client.
	rpc_l2Client        *rpc.Client       // The RPC client for the L2 node.
	l2ToL1MessagePasser *l2.L2ToL1MessagePasser
	ConnectionState     *L2ConnectionState
}

type L2ConnectionState struct {
	ProxyConnection       uint64
	ProxyConnectionFailed uint64
}

// OutputResponse represents the response structure for output-related data.
type OutputResponse struct {
	Version    string `json:"version"`    // The version of the output.
	OutputRoot string `json:"outputRoot"` // The output root associated with the response.
}

func NewL2Proxy(ctx *context.Context, l2GethURL string, l2NodeURL string) (*L2Proxy, error) {
	connectionState := &L2ConnectionState{
		ProxyConnection:       0,
		ProxyConnectionFailed: 0,
	}

	l2GethClient, err := ethclient.Dial(l2GethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}
	l2NodeClient, err := ethclient.Dial(l2NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	connectionState.ProxyConnection++
	l2ToL1MessagePasser, err := l2.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2GethClient)
	if err != nil {
		connectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to bind to L2ToL1MessagePasser: %w", err)
	}

	rpc_l2Client := l2NodeClient.Client()

	return &L2Proxy{
		l2NodeClient:        l2NodeClient,
		l2GethClient:        l2GethClient,
		rpc_l2Client:        rpc_l2Client,
		l2ToL1MessagePasser: l2ToL1MessagePasser,
		ctx:                 ctx,
		ConnectionState:     connectionState,
	}, nil
}

func (l2Proxy *L2Proxy) GetWithdrawalValidation(disputeGameEvent DisputeGameEvent) (*WithdrawalValidation, error) {

	// fmt.Print("Game: ", disputeGame, "\n")
	withdrawalHash := disputeGameEvent.EventRef.WithdrawalHash

	l2Proxy.ConnectionState.ProxyConnection++
	withdrawalPresentOnL2ToL1MessagePasser, err := l2Proxy.l2ToL1MessagePasser.L2ToL1MessagePasserCaller.SentMessages(nil, withdrawalHash)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to check if withdrawal exists on L2: %w", err)
	}

	blockNumber := disputeGameEvent.DisputeGame.DisputeGameClaimData.L2blockNumber
	l2Proxy.ConnectionState.ProxyConnection++
	blockPresentOnL2, err := l2Proxy.BlockByNumber(blockNumber)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
	}

	l2Proxy.ConnectionState.ProxyConnection++
	trustedRootProof, err := l2Proxy.getOutputRootFromTrustedL2Node(blockNumber)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
	}

	return &WithdrawalValidation{
		DisputeGameEvent:                       disputeGameEvent,
		TrustedRoots:                           trustedRootProof,
		WithdrawalPresentOnL2ToL1MessagePasser: withdrawalPresentOnL2ToL1MessagePasser,
		BlockPresentOnL2:                       blockPresentOnL2 != nil,
		IsWithdrawalValid:                      withdrawalPresentOnL2ToL1MessagePasser && blockPresentOnL2 != nil && trustedRootProof == disputeGameEvent.DisputeGame.DisputeGameClaimData.RootClaim,
	}, nil
}

// GetOutputRootFromTrustedL2Node retrieves the output root for a given L2 block number from a trusted L2 node.
// It returns the output root as a Bytes32 array.
func (l2Proxy *L2Proxy) getOutputRootFromTrustedL2Node(l2blockNumber *big.Int) ([32]byte, error) {

	var result OutputResponse
	l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

	err := l2Proxy.rpc_l2Client.CallContext(*l2Proxy.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
	//check if error contains "failed to determine L2BlockRef of height"
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
	}
	trustedRootProof, err := StringToBytes32(result.OutputRoot)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to convert output root to Bytes32: %w", err)
	}
	return [32]byte(trustedRootProof), nil
}

// GetOutputRootFromCalculation retrieves the output root by calculating it from the given block number.
// It returns the calculated output root as a Bytes32 array.
func (l2Proxy *L2Proxy) getOutputRootFromCalculation(blockNumber *big.Int) ([32]byte, error) {
	block, err := l2Proxy.l2GethClient.BlockByNumber(*l2Proxy.ctx, blockNumber)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get block by number: %d %w", blockNumber, err)
	}

	proof := struct{ StorageHash common.Hash }{}
	err = l2Proxy.l2GethClient.Client().CallContext(*l2Proxy.ctx, &proof, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, nil, hexutil.EncodeBig(blockNumber))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get proof: %w", err)
	}

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: [32]byte(block.Root()), MessagePasserStorageRoot: [32]byte(proof.StorageHash), BlockHash: block.Hash()})
	return outputRoot, nil
}

func (l2Proxy *L2Proxy) LatestHeight() (uint64, error) {
	l2Proxy.ConnectionState.ProxyConnection++
	block, err := l2Proxy.l2GethClient.BlockByNumber(*l2Proxy.ctx, nil)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	return block.NumberU64(), nil
}

func (l2Proxy *L2Proxy) BlockByNumber(blockNumber *big.Int) (*types.Block, error) {
	l2Proxy.ConnectionState.ProxyConnection++
	block, err := l2Proxy.l2GethClient.BlockByNumber(*l2Proxy.ctx, blockNumber)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("L2Proxy failed to get block by number: :%d %w", blockNumber, err)
	}

	return block, nil
}

func (l2Proxy *L2Proxy) ChainID() (*big.Int, error) {
	l2Proxy.ConnectionState.ProxyConnection++
	chainID, err := l2Proxy.l2GethClient.ChainID(*l2Proxy.ctx)
	if err != nil {
		l2Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return chainID, nil
}

func (l2Proxy *L2Proxy) Close() {
	l2Proxy.l2GethClient.Close()
}
