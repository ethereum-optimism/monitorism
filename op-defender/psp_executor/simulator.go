package psp_executor

import (
	"context"
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
	_, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {

		return nil, err
	}
	return &callMsg, nil
}
