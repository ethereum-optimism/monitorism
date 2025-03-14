package validator

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
)

type L2Proxy struct {
	l2GethClient          *ethclient.Client
	chainID               *big.Int
	ctx                   context.Context
	l2OpGethBackupClients map[string]*ethclient.Client
	ConnectionError       map[string]uint64
	Connections           map[string]uint64
}

func NewL2Proxy(ctx context.Context, l2GethClientURL string, l2GethBackupClientsURLs map[string]string) (*L2Proxy, error) {
	l2GethClient, err := ethclient.Dial(l2GethClientURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	chainID, err := l2GethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	// if backup urls are provided, create a backup client for each
	var l2OpGethBackupClients map[string]*ethclient.Client
	if len(l2GethBackupClientsURLs) > 0 {
		l2OpGethBackupClients, err = GethBackupClientsDictionary(ctx, l2GethBackupClientsURLs, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup clients: %w", err)
		}

	}

	return &L2Proxy{l2GethClient: l2GethClient, chainID: chainID, ctx: ctx, l2OpGethBackupClients: l2OpGethBackupClients, ConnectionError: make(map[string]uint64), Connections: make(map[string]uint64)}, nil
}

// get latest known L2 block number
func (l2Proxy *L2Proxy) BlockNumber() (uint64, error) {
	blockNumber, err := l2Proxy.l2GethClient.BlockNumber(l2Proxy.ctx)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}
	return blockNumber, nil
}

// GetOutputRootFromCalculation retrieves the output root by calculating it from the given block number.
// It returns the calculated output root as a Bytes32 array.
func (l2Proxy *L2Proxy) VerifyWithdrawalHashAndClaim(blockNumber *big.Int, withdrawalHash [32]byte, claim [32]byte) (bool, bool, string, error) {
	// We get the block from our trusted op-geth node
	block, err := l2Proxy.l2GethClient.BlockByNumber(l2Proxy.ctx, blockNumber)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, false, "", fmt.Errorf("failed to get output at block for game blockInt:%v error:%w", blockNumber, err)
	}

	// We get proof from our trusted op-geth node if present
	accountResult, clientUsed, err := l2Proxy.RetrieveEthProof(blockNumber, withdrawalHash)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, false, "", fmt.Errorf("failed to get proof: %w", err)
	}
	// verify the proof when this comes from untrusted node (merkle trie)
	err = accountResult.Verify(block.Root())
	if err != nil {
		return false, false, "", fmt.Errorf("failed to verify proof: %w", err)
	}
	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: [32]byte(block.Root()), MessagePasserStorageRoot: [32]byte(accountResult.StorageHash), BlockHash: block.Hash()})

	return bytes.Equal(outputRoot[:], claim[:]), false, clientUsed, nil
}

// VerifyRootClaimAndWithdrawalHash verifies that a given root claim matches the computed output root
// for a block and that the withdrawal hash proof is valid.
//
// Parameters:
//   - blockNumber: The L2 block number to verify
//   - rootClaim: The claimed output root to verify against
//   - withdrawalHash: The withdrawal hash to verify
//
// Returns:
//   - bool: True if the computed output root matches the claim
//   - bool: True if the withdrawal proof is valid
//   - string: The name of the client used to retrieve the proof
//   - error: Any error that occurred during verification
func (l2Proxy *L2Proxy) VerifyRootClaimAndWithdrawalHash(blockNumber *big.Int, rootClaim [32]byte, withdrawalHash [32]byte) (bool, bool, string, error) {
	// We get the block from our trusted op-geth node
	block, err := l2Proxy.l2GethClient.BlockByNumber(l2Proxy.ctx, blockNumber)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, false, "", fmt.Errorf("failed to get output at block for game blockInt:%v error:%w", blockNumber, err)
	}

	// We get proof from our trusted op-geth node if present
	accountResult, clientUsed, err := l2Proxy.RetrieveEthProof(blockNumber, withdrawalHash)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, false, "", fmt.Errorf("failed to get proof: %w", err)
	}

	// verify the proof when this comes from untrusted node (merkle trie)
	err = accountResult.Verify(block.Root())

	outputRoot := eth.OutputRoot(&eth.OutputV0{StateRoot: [32]byte(block.Root()), MessagePasserStorageRoot: [32]byte(accountResult.StorageHash), BlockHash: block.Hash()})
	return bytes.Equal(outputRoot[:], rootClaim[:]), err == nil, clientUsed, nil
}

// we retrieve the proof from the truested op-geth node and eventually from backup nodes if present
func (l2Proxy *L2Proxy) RetrieveEthProof(blockNumber *big.Int, withdrawalHash [32]byte) (AccountResult, string, error) {
	accountResult := AccountResult{}
	encodedBlock := hexutil.EncodeBig(blockNumber)

	// Create the ABI definition for "dummy(bytes32,uint256)"
	typeBytes32, _ := abi.NewType("bytes32", "", nil)
	typeUint256, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{
		{Type: typeBytes32},
		{Type: typeUint256},
	}

	// Pack the arguments (equivalent to abi-encode)
	zero := big.NewInt(0)
	packed, _ := arguments.Pack(withdrawalHash, zero)

	// Calculate keccak hash (equivalent to cast keccak)
	hash := crypto.Keccak256Hash(packed)

	storageKeys := []common.Hash{hash}
	fmt.Printf("Storage Keys: %v\n", storageKeys)

	l2Proxy.Connections["default"]++
	err := l2Proxy.l2GethClient.Client().CallContext(l2Proxy.ctx, &accountResult, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, storageKeys, encodedBlock)
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		for clientName, client := range l2Proxy.l2OpGethBackupClients {
			l2Proxy.Connections[clientName]++
			err = client.Client().CallContext(l2Proxy.ctx, &accountResult, "eth_getProof", predeploys.L2ToL1MessagePasserAddr, storageKeys, encodedBlock)
			// if we get a proof, we return it
			if err == nil {
				return accountResult, clientName, nil
			}
			l2Proxy.ConnectionError[clientName]++
		}

		return AccountResult{}, "", fmt.Errorf("failed to get proof from any node: %w", err)
	}
	return accountResult, "default", nil
}

func (l2Proxy *L2Proxy) GetTotalConnections() uint64 {
	totalConnections := uint64(0)
	for _, connection := range l2Proxy.Connections {
		totalConnections += connection
	}
	return totalConnections
}

func (l2Proxy *L2Proxy) GetTotalConnectionErrors() uint64 {
	totalConnectionErrors := uint64(0)
	for _, connectionError := range l2Proxy.ConnectionError {
		totalConnectionErrors += connectionError
	}
	return totalConnectionErrors
}
