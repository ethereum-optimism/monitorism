package psp_executor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"math/big"
	"net/http"
	"strings"
)

// **********************************************************************
// *                        Informations:                               *
// **********************************************************************
// * Sepolia:                                                           *
// * deputyGuardianSepolia: 0x4220C5deD9dC2C8a8366e684B098094790C72d3c *
// * SuperChainConfigSepolia: 0xC2Be75506d5724086DEB7245bd260Cc9753911Be *
// * FoS on Sepolia: 0x837DE453AD5F21E89771e3c06239d8236c0EFd5E        *
// **********************************************************************
// * Mainnet:                                                           *
// * deputyGuardianMainnet: 0x5dC91D01290af474CE21DE14c17335a6dEe4d2a8  *
// **********************************************************************

const (
	MetricsNamespace = "psp_executor"
	SepoliaRPC       = "https://proxyd-l1-consensus.primary.sepolia.prod.oplabs.cloud"
	MainnetRPC       = "https://proxyd-l1-consensus.primary.mainnet.prod.oplabs.cloud"
	LocalhostRPC     = "http://localhost:8545"
)

// DefenderExecutor is a struct that implements the Executor interface.
type DefenderExecutor struct{}

// Executor is an interface that defines the FetchAndExecute method.
type Executor interface {
	FetchAndExecute(d *Defender) // For doc see the `FetchAndExecute()` function.
}

// Defender is a struct that represents the Defender API server.
type Defender struct {
	log                     log.Logger
	port                    string
	SuperChainConfigAddress string
	l1Client                *ethclient.Client
	router                  *mux.Router
	executor                Executor
	privatekey              string
	// metrics
	latestPspNonce      *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// Define a struct that represents the data structure of your response through the HTTP API.
type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// Define a struct that represents the data structure of your request through the HTTP API.
type RequestData struct {
	Pause     bool   `json:"pause"`
	Timestamp int64  `json:"timestamp"`
	Operator  string `json:"operator"`
}

// handlePost handles POST requests and processes the JSON body
func (d *Defender) handlePost(w http.ResponseWriter, r *http.Request) {
	// Decode the JSON body into a map
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	var requestMap map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestMap); err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Check for exactly 3 fields
	if len(requestMap) != 3 {
		http.Error(w, "Request must contain exactly 3 fields: Pause, Timestamp, and Operator", http.StatusBadRequest)
		return
	}

	// Check for the presence of required fields and their types
	pause, ok := requestMap["Pause"].(bool)
	if !ok {
		http.Error(w, "Pause field must be a boolean", http.StatusBadRequest)
		return
	}

	timestamp, ok := requestMap["Timestamp"].(float64)
	if !ok {
		http.Error(w, "Timestamp field must be a number", http.StatusBadRequest)
		return
	}

	operator, ok := requestMap["Operator"].(string)
	if !ok {
		http.Error(w, "Operator field must be a string", http.StatusBadRequest)
		return
	}

	// Create the RequestData struct with the validated fields
	data := RequestData{
		Pause:     pause,
		Timestamp: int64(timestamp),
		Operator:  operator,
	}

	// Check that all the fields are set with valid values
	if !data.Pause || data.Timestamp == 0 || data.Operator == "" {
		http.Error(w, "Missing or invalid fields in the request", http.StatusBadRequest)
		return
	}

	// Execute the PSP on the chain by calling the FetchAndExecute method of the executor.
	d.executor.FetchAndExecute(d)
	return
}

// NewAPI creates a new HTTP API Server for the PSP Executor and starts listening on the specified port from the args passed.
func NewDefender(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig, executor Executor) (*Defender, error) {
	// Set the route and handler function for the `/api/psp_execution` endpoint.
	l1client, err := CheckAndReturnRPC(cfg.NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch l1 RPC: %w", err)
	}
	privatekey, err := CheckAndReturnPrivateKey(cfg.privatekeyflag)
	if err != nil {
		return nil, fmt.Errorf("failed to return the privatekey: %w", err)
	}

	if cfg.PortAPI == "" {
		return nil, fmt.Errorf("port.api is not set.")
	}

	defender := &Defender{
		log:        log,
		l1Client:   l1client,
		port:       cfg.PortAPI,
		executor:   executor,
		privatekey: privatekey,
	}

	defender.router = mux.NewRouter()
	defender.router.HandleFunc("/api/psp_execution", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1048576) // Limit payload to 1MB
		defender.handlePost(w, r)
	}).Methods("POST")
	defender.log.Info("Starting HTTP JSON API PSP Execution server...", "port", defender.port)
	return defender, nil
}

// FetchAndExecute() will fetch the PSP and execute it this onchain.
// For now, the function is not fully implemented and will make a dummy transaction on chain (see `pspExecutionOnChain()`).
// In the future, the function will fetch the PSPs from a secret file and execute it onchain through a EVM transaction.
func (e *DefenderExecutor) FetchAndExecute(d *Defender) {
	ctx := context.Background()
	configAddress, safeAddress, data, err := FetchPSPInGCP()
	if err != nil {
		d.log.Crit("Failed to fetch PSP data from GCP", "error", err)
		return
	}
	PspExecutionOnChain(ctx, d.l1Client, configAddress, d.privatekey, safeAddress, data)
}

// CheckAndReturnRPC() will return the L1 client based on the RPC provided in the config and ensure that the RPC is not production one.
func CheckAndReturnRPC(rpc_url string) (*ethclient.Client, error) {
	if rpc_url == "" {
		return nil, fmt.Errorf("rpc.url is not set.")
	}
	if !strings.Contains(rpc_url, "rpc.tenderly.co/fork") { // Check if the RPC is a production one. if yes return an error, as we should not execute the pause on the production chain in the first version.
		return nil, fmt.Errorf("rpc.url doesn't contains \"fork\" is a production RPC.")
	}

	client, err := ethclient.Dial(rpc_url)
	if err != nil {

		log.Crit("Failed to connect to the Ethereum client", "error", err)
	}
	return client, nil
}

// CheckAndReturnPrivateKey() will return the privatekey only if the privatekey is a valid one otherwise return an error.
func CheckAndReturnPrivateKey(privateKeyStr string) (string, error) {
	// Remove "0x" prefix if present
	privateKeyStr = strings.TrimPrefix(privateKeyStr, "0x")

	// Check if the private key is a valid hex string
	if !isValidHexString(privateKeyStr) {
		return "", fmt.Errorf("invalid private key: not a valid hex string")
	}

	// Check if the private key has the correct length (32 bytes = 64 hex characters)
	if len(privateKeyStr) != 64 {
		return "", fmt.Errorf("invalid private key: incorrect length")
	}

	// Attempt to parse the private key
	_, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %v", err)
	}

	return privateKeyStr, nil
}

// isValidHexString checks if a string is a valid hexadecimal string
func isValidHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// FetchPSPInGCP() will fetch the correct PSPs into GCP and return the Data.
func FetchPSPInGCP() (string, string, []byte, error) {
	//In the future, we need to check first the nonce and then `checkPauseStatus` and then return the data for the latest PSPs.
	return "0xC2Be75506d5724086DEB7245bd260Cc9753911Be", "0x4141414142424242414141414242424241414141", []byte{0x41, 0x42, 0x43}, nil
}

// PSPexecution(): PSPExecutionOnChain is a core function that will check that status of the superchain is not paused and then send onchain transaction to pause the superchain.
func PspExecutionOnChain(ctx context.Context, l1client *ethclient.Client, superchainconfig_address string, privatekey string, safe_address string, data []byte) {
	blocknumber, err := l1client.BlockNumber(ctx)
	log.Info("", "block number", blocknumber)
	pause_before_transaction := checkPauseStatus(ctx, l1client, superchainconfig_address)
	if pause_before_transaction {
		log.Crit("The SuperChainConfig is already paused! Exiting the program.")
	}
	log.Info("[Before Transaction] status of the pause()", "pause", pause_before_transaction)
	txHash, err := sendTransaction(l1client, privatekey, safe_address, big.NewInt(1), data) // 1 wei
	if err != nil {
		log.Crit("Failed to send transaction:", "error", err)
	}

	log.Info("Transaction sent!", "TxHash", txHash)

	pause_after_transaction := checkPauseStatus(ctx, l1client, superchainconfig_address)
	log.Info("[After Transaction] status of the pause()", "pause", pause_after_transaction)

}

// Run() will start the Defender API server and block the thread.
func (d *Defender) Run(ctx context.Context) {
	err := http.ListenAndServe(":"+d.port, d.router) // Start the HTTP server blocking thread for now.
	if err != nil {
		log.Crit("Failed to start the API server", "error", err)
	}
}

// Close() will close the Defender API server and the L1 client.
func (d *Defender) Close(_ context.Context) error {
	d.l1Client.Close() //close the L1 client.
	return nil
}

// sendTransaction(): Is a function made for sending a transaction on chain with the parameters : client, privatekey, toAddress, amount, data.
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
	log.Info("SuperChainConfigAddress", "SuperChainConfigAddress", SuperChainConfigAddress)
	log.Info("l1client", "l1client", l1client)
	superchainconfig, err := bindings.NewSuperchainConfig(common.HexToAddress(SuperChainConfigAddress), l1client)

	log.Info("superchainconfig", "superchainconfig address", superchainconfig)
	if err != nil {
		log.Crit("failed to create superchainconfig instance", "error", err)
	}

	paused, err := superchainconfig.Paused(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Error("failed to query superchainconfig paused status", "error", err)
		return false
	}

	return paused
}
