package psp_executor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
)

// Informations:
// deputyGuardianSepolia: 0x4220C5deD9dC2C8a8366e684B098094790C72d3c
// SuperChainConfigSepolia: 0xC2Be75506d5724086DEB7245bd260Cc9753911Be
// deputyGuardianMainnet: 0x5dC91D01290af474CE21DE14c17335a6dEe4d2a8

const (
	MetricsNamespace = "psp_executor"
	SepoliaRPC       = "https://proxyd-l1-consensus.primary.sepolia.prod.oplabs.cloud"
	MainnetRPC       = "https://proxyd-l1-consensus.primary.mainnet.prod.oplabs.cloud"
	LocalhostRPC     = "http://localhost:8545"
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
	println("hexString", cfg.hexString)
	data, err := hex.DecodeString(cfg.hexString)
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return &Monitor{}, err
	}
	println("Data: ", string(data))
	superchainconfig_address := "0xC2Be75506d5724086DEB7245bd260Cc9753911Be" //for now hardcoded for sepolia but will dynamically get from the config in the future
	// Here need to extract all the information from the calldata to retrieve the address of the superchainConfig.

	PspExecutionOnChain(ctx, client, superchainconfig_address, cfg.privatekeyflag, cfg.receiverAddress, data)

	return &Monitor{}, errors.New("")
}
func main() {
	//1. Fetch the privatekey of the account in GCP secret Manager.
	privatekey, err := FetchPrivateKeyInGcp()
	if err != nil {
		log.Crit("Failed to fetch the privatekey from GCP secret manager: %v", err.Error())
	}
	// 2. Fetch the CORRECT Nounce PSP in the GCP secret Manager and return the data to execute.
	superchainconfig_address, safe_address, data, err := FetchPSPInGCP()
	if err != nil {
		log.Crit("Failed to fetch the PSP from GCP secret manager: %v", err.Error())
	}

	// 3. Get the L1 client and ensure RPC is correct.
	l1client, err := GetTheL1Client()
	if err != nil {
		log.Crit("Failed to get the L1client: %v", err.Error())
	}

	// 4. Execute the PSP on the chain.
	ctx := context.Background() //TODO: Check if we really do need to context if yes we will keep it otherwise we will remove this.
	PspExecutionOnChain(ctx, l1client, superchainconfig_address, privatekey, safe_address, data)
}

func GetTheL1Client() (*ethclient.Client, error) {
	client, err := ethclient.Dial(LocalhostRPC) //Need to change this to the correct RPC (mainnet or sepolia) but for now hardcoded to localhost.
	if LocalhostRPC != "http://localhost:8545" {
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
	return client, nil
}

// FetchPrivateKey() will fetch the privatekey of the account that will execute the pause (from the GCP secret manager).
func FetchPrivateKeyInGcp() (string, error) {
	return "", errors.New("Not implemented")
}

// FetchPSPInGCP() will fetch the correct PSPs into GCP and return the Data.
func FetchPSPInGCP() (string, string, []byte, error) {
	// need to fetch check first the nonce with the same method with `checkPauseStatus` and then return the data for this PSP.

	return "", "", []byte{}, errors.New("Not implemented")
}

// PSPexecution(): PSPExecutionOnChain is a core function that will check that status of the superchain is not paused and then send onchain transaction to pause the superchain.
func PspExecutionOnChain(ctx context.Context, l1client *ethclient.Client, superchainconfig_address string, privatekey string, safe_address string, data []byte) {
	fmt.Println("PSP Execution Pause")
	pause_before_transaction := checkPauseStatus(ctx, l1client, superchainconfig_address)
	if pause_before_transaction {
		log.Crit("The SuperChainConfig is already paused! Exiting the program.")
	}
	println("Before the transaction the status of the `pause` is set to: ", pause_before_transaction)
	txHash, err := sendTransaction(l1client, privatekey, safe_address, big.NewInt(1), data) // 1 wei
	if err != nil {
		log.Crit("Failed to send transaction:", "error", err)
	}

	fmt.Printf("Transaction sent! Tx Hash: %s\n", txHash)

	pause_after_transaction := checkPauseStatus(ctx, l1client, superchainconfig_address)
	println("After the transaction the status of the `pause` is set to: ", pause_after_transaction)

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
		return "", fmt.Errorf("Invalid private key: %v", err)
	}

	// Derive the public key from the private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("Error casting public key to ECDSA")
	}

	// Derive the sender address from the public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Ensure the toAddress is valid
	toAddress := common.HexToAddress(toAddressStr)
	if !common.IsHexAddress(toAddressStr) {
		return "", fmt.Errorf("Invalid to address: %s", toAddressStr)
	}

	// Get the nonce for the next transaction
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Set up the transaction parameters
	value := amount                  // Amount of Ether to send
	gasLimit := uint64(1000 * 21008) // In units TODO: Need to use `estimateGas()` to get the correct value.
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("Failed to suggest gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	// Sign the transaction with the private key
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("Failed to get network ID: %v", err)
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("Failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("Failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

// checkPauseStatus(): Is a function made for checking the pause status of the SuperChainConfigAddress
func checkPauseStatus(ctx context.Context, l1client *ethclient.Client, SuperChainConfigAddress string) bool {
	// Get the contract instance
	contractAddress := common.HexToAddress(SuperChainConfigAddress)
	optimismPortal, err := bindings.NewSuperchainConfig(contractAddress, l1client)
	if err != nil {
		log.Crit("Failed to create SuperChainConfig instance: %v", err)
	}

	paused, err := optimismPortal.Paused(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Error("failed to query OptimismPortal paused status", "err", err)
		return false
	}

	return paused
}
func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
