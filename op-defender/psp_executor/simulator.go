package psp_executor

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SimulateTransaction will simulate a transaction onchain if the blockNumber is `nil` it will simulate the transaction on the latest block.
func SimulateTransaction(ctx context.Context, client *ethclient.Client, blockNumber *uint64, fromAddress common.Address, contractAddress common.Address, data []byte) ([]byte, error) {
	// Create a call message
	var blockNumber_bigint *big.Int

	callMsg := ethereum.CallMsg{
		From:  fromAddress,
		To:    &contractAddress,
		Data:  data,
		Value: &big.Int{},
	}
	// If the blockNumber is not nil, set the blockNumber_bigint to the blockNumber provided.
	if blockNumber != nil {
		blockNumber_bigint = new(big.Int).SetUint64(*blockNumber)
	}
	// Simulate the transaction if the blockNumber_bigint is nil it will simulate the transaction on the latest block.
	simulation, err := client.CallContract(ctx, callMsg, blockNumber_bigint)
	if err != nil {

		return nil, err
	}
	return simulation, nil
}

// FetchAndSimulate will fetch the PSP from a file and simulate it this onchain.
func (e *DefenderExecutor) FetchAndSimulateAtBlock(ctx context.Context, d *Defender, blocknumber *uint64, nonce uint64) ([]byte, error) {
	operationSafe, data, err := GetPSPbyNonceFromFile(nonce, d.path) // return the PSP that has the correct nonce.
	if err != nil {
		return nil, err
	}
	// Check that operationSafe is the same as the config provided.
	if operationSafe != d.safeAddress {
		return nil, err
	}
	// Then simulate PSP with the correct nonce onchain with the PSP data through the `SimulateTransaction()` function.
	simulation, err := SimulateTransaction(ctx, d.l1Client, blocknumber, d.senderAddress, operationSafe, data)
	if err != nil {
		return nil, err
	}
	return simulation, nil
}
