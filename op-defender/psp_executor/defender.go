package psp_executor

import (
	"context"
	"crypto/ecdsa"
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

type Account struct {
	Address  common.Address
	Nickname string
}

type Defender struct {
	log                     log.Logger
	port                    string
	SuperChainConfigAddress string
	l1Client                *ethclient.Client
	router                  *mux.Router
	// metrics
	latestPspNonce      *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// Define a struct that represents the data structure of your response
type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type RequestData struct {
	Pause     bool   `json:"pause"`
	Timestamp int64  `json:"timestamp"`
	Operator  string `json:"operator"`
	Calldata  string `json:"calldata"` //temporary field as the calldata will be fetched from the GCP in the future (so will be removed in the future PR).
}

// handlePost handles POST requests and processes the JSON body
func handlePost(w http.ResponseWriter, r *http.Request) {
	var data RequestData
	// Decode the JSON body into the struct
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// return HTTP code 511 for the first PR.
	http.Error(w, "Network Authentication Required", 511)
	return

	// The next code after the `return` is commented for next PRs and don't require review for now.

	// if data.Pause == false || data.Timestamp == 0 || data.Operator == "" || data.Calldata == "" {
	// 	http.Error(w, "All fields are required and must be non-zero", http.StatusBadRequest)
	// 	log.Warn("A field is set to empty or 0", "data", data)
	// 	return
	// }
	// // Log the received data
	// log.Info("HandlePost Received data", "data", data)
	//
	// // Call the Fetch and Execute -> This will be commented for the first PR.
	// // FetchAndExecute()
	//
	// // Respond back with the received data or a success message
	// response := map[string]interface{}{
	// 	"status": "success",
	// 	"data":   data,
	// }
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(response)
}

// NewAPI creates a new HTTP API Server for the PSP Executor and starts listening on the specified port from the args passed.
func NewDefender(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Defender, error) {
	// Set the route and handler function for the `/api/psp_execution` endpoint.
	router := mux.NewRouter()
	router.HandleFunc("/api/psp_execution", handlePost).Methods("POST")
	l1client, err := GetTheL1Client()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch l1 RPC: %w", err)
	}

	if cfg.portapi == "" {
		return nil, fmt.Errorf("port.api is not set.")
	}

	defender := &Defender{
		log:      log,
		l1Client: l1client,
		port:     cfg.portapi,
		router:   router,
	}

	defender.log.Info("Starting HTTP JSON API PSP Execution server...", "port", defender.port)
	return defender, nil
}

// FetchAndExecute() will fetch the privatekey, and correct PSP from GCP and execute it on the correct chain.
func FetchAndExecute() {
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

// GetTheL1Client() will return the L1 client based on the RPC provided in the config and ensure that the RPC is not production one.
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
	return "2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6", nil // Mock with a well-known private key from test test ... test junk derivation (9).
}

// FetchPSPInGCP() will fetch the correct PSPs into GCP and return the Data.
func FetchPSPInGCP() (string, string, []byte, error) { //superchainconfig_address, safe_address, data, error
	// need to fetch check first the nonce with the same method with `checkPauseStatus` and then return the data for this PSP.

	return "0xC2Be75506d5724086DEB7245bd260Cc9753911Be", "0x4141414142424242414141414242424241414141", []byte{0x41, 0x42, 0x43}, nil //errors.New("Not implemented") mock with simple value to make a call on L1.
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

func (d *Defender) Run(ctx context.Context) {
	err := http.ListenAndServe(":"+d.port, d.router) // Start the HTTP server blocking thread for now.
	if err != nil {
		log.Crit("Failed to start the API server", "error", err)
	}
}

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
