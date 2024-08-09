package psp_executor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
)

const (
	MetricsNamespace = "psp_executor"
)

type Account struct {
	Address  common.Address
	Nickname string
}

type Monitor struct {
	log log.Logger

	rpc      client.RPC
	accounts []Account

	// metrics
	balances            *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// For now new API will serve the purpose of sending a transaction
func NewAPI(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("Creating the API psp_executor.")
	client, err := ethclient.Dial(cfg.NodeUrl)
	if cfg.NodeUrl != "http://localhost:8545" {
		log.Warn("This is not the RPC localhost are you sure you want to continue (yes/no)")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			log.Crit("Not yes, We Exiting the program.")
		}
	}
	if err != nil {
		log.Crit("Failed to connect to the Ethereum client: %v", err.Error())
	}
	data, err := hex.DecodeString(cfg.hexString)
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return &Monitor{}, err
	}

	txHash, err := sendTransaction(client, cfg.privatekeyflag, cfg.ReceiverAddressFlagName, big.NewInt(1), data) // 1 wei
	if err != nil {
		log.Crit("Failed to send transaction: %v", err)
	}

	fmt.Printf("Transaction sent! Tx Hash: %s\n", txHash)

	return &Monitor{}, errors.New("Not implemented")
}

func PSPExecutionPause(request_operator string, request_timestamp string) {
	fmt.Println("PSP Execution Pause")

	// txHash, err := sendTransaction(client, "YOUR_PRIVATE_KEY", "RECIPIENT_ADDRESS", big.NewInt(1000000000000000000)) // 1 ETH
	// if err != nil {
	// 	log.Crit("Failed to send transaction: %v", err)
	// }
	// fmt.Printf("Transaction sent! Tx Hash: %s\n", txHash)

}

func (m *Monitor) Run(ctx context.Context) {
}

func (m *Monitor) Close(_ context.Context) error {
	m.rpc.Close()
	return nil
}

func sendTransaction(client *ethclient.Client, privateKeyStr string, toAddressStr string, amount *big.Int, data []byte) (string, error) {
	// Convert the private key string to a private key type
	// TODO: Need to check if there is the `0x` if yes remove it from the string.
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	// Derive the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("error casting public key to ECDSA")
	}

	// Derive the sender address from the public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Ensure the toAddress is valid
	toAddress := common.HexToAddress(toAddressStr)
	if !common.IsHexAddress(toAddressStr) {
		return "", fmt.Errorf("invalid to address: %s", toAddressStr)
	}

	// Get the nonce for the next transaction
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Set up the transaction parameters
	value := amount                // Amount of Ether to send
	gasLimit := uint64(10 * 21000) // In units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to suggest gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	// Sign the transaction with the private key
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get network ID: %v", err)
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
