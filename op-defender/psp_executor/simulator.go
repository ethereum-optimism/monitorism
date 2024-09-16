package psp_executor

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func SimulateTransaction(client *ethclient.Client, fromAddress common.Address, contractAddress common.Address, data []byte) (*ethereum.CallMsg, error) {
	// Create a call message
	callMsg := ethereum.CallMsg{
		From:  fromAddress,
		To:    &contractAddress,
		Data:  data,
		Value: &big.Int{},
	}

	// Context with a background
	ctx := context.Background()

	// Simulate the transaction
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Simulation result: %x\n", result)
	return &callMsg, nil
}
