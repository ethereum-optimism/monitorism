package psp_executor

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func SimulateTransaction(ctx context.Context, client *ethclient.Client, blockNumber *uint64, fromAddress common.Address, contractAddress common.Address, data []byte) ([]byte, error) {
	// Create a call message

	callMsg := ethereum.CallMsg{
		From:  fromAddress,
		To:    &contractAddress,
		Data:  data,
		Value: &big.Int{},
	}

	// Context with a background
	// Simulate the transaction
	simulation, err := client.CallContract(ctx, callMsg, new(big.Int).SetUint64(*blockNumber))

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
	if operationSafe != d.safeAddress {
		return nil, err
	}
	// When the PSP is fetched correctly then simulate it onchain with the PSP data through the `SimulateTransaction()` function.
	simulation, err := SimulateTransaction(ctx, d.l1Client, blocknumber, d.senderAddress, operationSafe, data)
	if err != nil {
		return nil, err
	}
	return simulation, nil
}
