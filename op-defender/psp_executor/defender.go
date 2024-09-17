package psp_executor

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

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
)

// **********************************************************************
// *                        Informations:                               *
// **********************************************************************
// * Sepolia:                                                           *
// * deputyGuardianSepolia: 0x4220C5deD9dC2C8a8366e684B098094790C72d3c *
// * SuperChainConfigSepolia: 0xC2Be75506d5724086DEB7245bd260Cc9753911Be *
// * FoSSepolia: 0x837DE453AD5F21E89771e3c06239d8236c0EFd5E        *
// **********************************************************************
// * Mainnet:                                                           *
// * deputyGuardianMainnet: 0x5dC91D01290af474CE21DE14c17335a6dEe4d2a8
// * SuperChainConfigMainnet: 0x95703e0982140D16f8ebA6d158FccEde42f04a4C
// * FoSMainnet: 0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
// **********************************************************************

const (
	MetricsNamespace   = "psp_executor"
	MaxRequestBodySize = 1 * 1024 * 1024 // 1MB in bytes
	DefaultGasLimit    = 21000
)

// DefenderExecutor is a struct that implements the Executor interface.
type DefenderExecutor struct{}

// Executor is an interface that defines the FetchAndExecute method.
type Executor interface {
	FetchAndExecute(d *Defender) (common.Hash, error) // For documentation, see directly the `FetchAndExecute()` function below.
	ReturnCorrectChainID(l1client *ethclient.Client, chainID uint64) (*big.Int, error)
}

// Defender is a struct that represents the Defender API server.
type Defender struct {
	// Infra data
	log      log.Logger
	port     string
	router   *mux.Router
	executor Executor
	// Onchain data
	l1Client   *ethclient.Client
	privatekey *ecdsa.PrivateKey
	path       string
	nonce      uint64
	chainID    *big.Int
	// superChainConfig
	superChainConfigAddress common.Address
	superChainConfig        *bindings.SuperchainConfig
	// Foundation Operation Safe
	safeAddress   common.Address
	operationSafe *bindings.Safe
	// Metrics
	latestPspNonce      *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// Define a struct that represents the data structure of your PSP.
type PSP struct {
	ChainID      uint64
	ChainIdStr   string         `json:"chain_id"`
	RPCURL       string         `json:"rpc_url"`
	CreatedAt    string         `json:"created_at"`
	SafeAddr     common.Address `json:"safe_addr"`
	SafeNonce    uint64
	SafeNonceStr string `json:"safe_nonce"`
	TargetAddr   string `json:"target_addr"`
	ScriptName   string `json:"script_name"`
	Data         []byte
	DataStr      string `json:"data"`
	Signatures   []struct {
		Signer    common.Address `json:"signer"`
		Signature string         `json:"signature"`
	} `json:"signatures"`
	Calldata    []byte
	CalldataStr string `json:"calldata"`
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

// handleHealthCheck is a GET method that returns the health status of the Defender
func (d *Defender) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

// handlePost handles POST requests and processes the JSON body
func (d *Defender) handlePost(w http.ResponseWriter, r *http.Request) {
	// Decode the JSON body into a map
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	var requestMap map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestMap); err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
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
	txHash, err := d.executor.FetchAndExecute(d)
	if (txHash == common.Hash{}) && (err != nil) { // If TxHash and Error then we return an error as the execution had an issue.
		response := Response{
			Message: "🛑" + err.Error() + "🛑",
			Status:  http.StatusInternalServerError,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response (Error)", http.StatusInternalServerError)
			return
		}
		return
	}
	if (txHash == common.Hash{}) && (err == nil) {
		response := Response{
			Message: "🛑 Unknown error, `TxHash` is set to `nil` 🛑",
			Status:  http.StatusInternalServerError,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response (`TxHash` is set to `nil`)", http.StatusInternalServerError)
			return
		}
		return
	}
	if (txHash != common.Hash{}) && (err != nil) { // If the transaction hash is not empty and the error is not nil we return the transaction hash.
		response := Response{
			Message: "🚧 Transaction Executed 🚧, but the SuperchainConfig is not *pause*. An error occured: " + err.Error() + ". The TxHash: " + txHash.Hex(),
			Status:  http.StatusInternalServerError,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response (SuperChain not Paused)", http.StatusInternalServerError)
			return
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := Response{
		Message: "PSP executed successfully ✅ Transaction hash: " + txHash.Hex() + "🎉",
		Status:  http.StatusOK,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response (PSP executed successfully)", http.StatusInternalServerError)
		return
	}
	return
}

// ReturnCorrectChainID is a function that will return the correct chainID based on the chainID provided in the config against the RPC url.
func (e *DefenderExecutor) ReturnCorrectChainID(l1client *ethclient.Client, chainID uint64) (*big.Int, error) {
	if l1client == nil {
		return &big.Int{}, fmt.Errorf("l1client is not set.")
	}
	if chainID == 0 {
		return &big.Int{}, fmt.Errorf("chainID is not set.")
	}
	chainID_RPC, err := l1client.ChainID(context.Background())
	if err != nil {
		return &big.Int{}, fmt.Errorf("failed to get network ID: %v", err)
	}
	if chainID_RPC.Uint64() != chainID {
		return &big.Int{}, fmt.Errorf("chainID mismatch: got %d, expected %d", chainID_RPC.Uint64(), chainID)
	}
	return chainID_RPC, nil
}

// AddressFromPrivateKey is a function that will return the address of the privatekey.
func AddressFromPrivateKey(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	if privateKey == nil {
		return common.Address{}, fmt.Errorf("private key is not set")
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}
	return crypto.PubkeyToAddress(*publicKeyECDSA), nil

}

// NewDefender creates a new HTTP API Server for the PSP Executor and starts listening on the specified port from the args passed.
func NewDefender(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig, executor Executor) (*Defender, error) {
	// Set the route and handler function for the `/api/psp_execution` endpoint.
	privatekey, err := CheckAndReturnPrivateKey(cfg.privatekeyflag)
	if err != nil {
		return nil, fmt.Errorf("failed to return the privatekey: %w", err)
	}
	address, err := AddressFromPrivateKey(privatekey)
	if err != nil {
		return nil, fmt.Errorf("failed to return the address associated to the private key: %w", err)
	}

	log.Info("============================ Configuration Info ================================")
	log.Info("cfg.nodeurl", "cfg.nodeurl", cfg.NodeURL)
	log.Info("cfg.portapi", "cfg.portapi", cfg.PortAPI)
	log.Info("cfg.path", "cfg.path", cfg.Path)
	log.Info("cfg.SuperChainConfigAddress", "cfg.SuperChainConfigAddress", cfg.SuperChainConfigAddress)
	log.Info("cfg.operationSafe", "cfg.operationSafe", cfg.SafeAddress)
	log.Info("cfg.chainID", "cfg.chainID", cfg.chainID)
	log.Info("defender address (from privatekey)", "address", address)

	log.Info("===============================================================================")

	l1client, err := CheckAndReturnRPC(cfg.NodeURL) //@TODO: Need to check if the latest blocknumber returned is 0.
	if err != nil {
		return nil, fmt.Errorf("failed to fetch l1 RPC: %w", err)
	}
	if cfg.PortAPI == "" {
		return nil, fmt.Errorf("port.api is not set.")
	}

	if cfg.Path == "" {
		return nil, fmt.Errorf("path is not set.")
	}

	if cfg.SuperChainConfigAddress == (common.Address{}) {
		return nil, fmt.Errorf("superchainconfig.address is not set.")
	}
	superchainconfig, err := bindings.NewSuperchainConfig(cfg.SuperChainConfigAddress, l1client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the SuperChainConfig: %w", err)
	}
	if cfg.SafeAddress == (common.Address{}) {
		return nil, fmt.Errorf("safe.address is not set.")
	}
	safe, err := bindings.NewSafe(cfg.SafeAddress, l1client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the GnosisSafe: %w", err)
	}

	defender := &Defender{
		log:                     log,
		l1Client:                l1client,
		port:                    cfg.PortAPI,
		executor:                executor,
		privatekey:              privatekey,
		superChainConfigAddress: cfg.SuperChainConfigAddress,
		superChainConfig:        superchainconfig,
		safeAddress:             cfg.SafeAddress,
		operationSafe:           safe,
		path:                    cfg.Path,
	}
	chainID, err := defender.executor.ReturnCorrectChainID(l1client, cfg.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to return the correct chainID: %w", err)
	}
	defender.chainID = chainID
	defender.router = mux.NewRouter()
	defender.router.HandleFunc("/api/psp_execution", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize) // Limit payload to 1MB
		defender.handlePost(w, r)
	}).Methods("POST")
	defender.router.HandleFunc("/api/healthcheck", defender.handleHealthCheck).Methods("GET")
	return defender, nil
}

// getNonceSafe is a function that will return the nonce of the operationSafe.
func (d *Defender) getNonceSafe(ctx context.Context) (uint64, error) {
	nonce, err := d.operationSafe.Nonce(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, err
	}
	return nonce.Uint64(), nil
}

// FetchAndExecute will fetch the PSP from a file and execute it this onchain.
func (e *DefenderExecutor) FetchAndExecute(d *Defender) (common.Hash, error) {
	ctx := context.Background()
	nonce, err := d.getNonceSafe(ctx) // Get the the current nonce of the operationSafe.
	if err != nil {
		d.log.Error("failed to get nonce", "error", err)
		return common.Hash{}, err
	}
	operationSafe, data, err := GetPSPbyNonceFromFile(nonce, d.path) // return the PSP that has the correct nonce.
	if err != nil {
		d.log.Error("failed to get the PSPs from a file", "error", err)
		return common.Hash{}, err
	}
	if operationSafe != d.safeAddress {
		d.log.Error("the safe address in the file is not the same as the one in the configuration!")
		return common.Hash{}, err
	}
	// When all the data is fetched correctly then execute the PSP onchain with the PSP data through the `ExecutePSPOnchain()` function.
	txHash, err := d.ExecutePSPOnchain(ctx, operationSafe, data)
	if err != nil {
		d.log.Error("failed to execute the PSP onchain", "error", err)
		return txHash, err
	}
	return txHash, nil
}

// CheckAndReturnRPC will return the L1 client based on the RPC provided in the config and ensure that the RPC is not production one.
func CheckAndReturnRPC(rpc_url string) (*ethclient.Client, error) {

	if rpc_url == "" {
		return nil, fmt.Errorf("rpc.url is not set.")
	}
	if !strings.Contains(rpc_url, "rpc.tenderly.co/fork") && !strings.Contains(rpc_url, "sepolia") && !strings.Contains(rpc_url, "localhost") { // Check if the RPC is a mainnet production. if yes return an error, as we should not execute the pause on the fork or sepolia or localhost chain in the first version
		return nil, fmt.Errorf("rpc.url doesn't contains \"fork\" or \"sepolia\" so this is a production RPC on mainnet")
	}

	client, err := ethclient.Dial(rpc_url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %v", err)
	}
	return client, nil
}

// CheckAndReturnPrivateKey will return the privatekey only if the privatekey is a valid one otherwise return an error.
func CheckAndReturnPrivateKey(privateKeyStr string) (*ecdsa.PrivateKey, error) {
	// Remove "0x" prefix if present
	privateKeyStr = strings.TrimPrefix(privateKeyStr, "0x")

	// Check if the private key is a valid hex string
	if !isValidHexString(privateKeyStr) {
		return nil, fmt.Errorf("invalid private key: not a valid hex string")
	}

	// Check if the private key has the correct length (32 bytes = 64 hex characters)
	if len(privateKeyStr) != 64 {
		return nil, fmt.Errorf("invalid private key: incorrect length")
	}

	// Attempt to parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	return privateKey, nil
}

// isValidHexString checks if a string is a valid hexadecimal string
func isValidHexString(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// GetPSPbyNonceFromFile will fetch the latest PSPs from a secret file and return the PSP that has the correct nonce.
func GetPSPbyNonceFromFile(nonce uint64, path string) (common.Address, []byte, error) {
	// Read the content of the file
	var pspData []PSP
	content, err := os.ReadFile(path)
	if err != nil {
		return common.Address{}, []byte{}, fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(content, &pspData); err != nil {
		return common.Address{}, []byte{}, fmt.Errorf("failed to parse JSON: %w", err)
	}
	// Iterate over the PSPs and populate the field with the correct field accordingly.
	for i, psp := range pspData {
		chainID, err := strconv.ParseUint(psp.ChainIdStr, 10, 64)
		if err != nil {
			return common.Address{}, []byte{}, fmt.Errorf("failed to parse chainID: %w", err)
		}
		pspData[i].ChainID = chainID

		safeNonce, err := strconv.ParseUint(psp.SafeNonceStr, 10, 64)
		if err != nil {
			return common.Address{}, []byte{}, fmt.Errorf("failed to parse safeNonce: %w", err)
		}
		pspData[i].SafeNonce = safeNonce

		if len(psp.CalldataStr) <= 2 {
			return common.Address{}, []byte{}, fmt.Errorf("calldata is empty")
		}
		callData, err := hex.DecodeString(psp.CalldataStr[2:])
		if err != nil {
			return common.Address{}, []byte{}, fmt.Errorf("failed to parse calldata %w", err)
		}
		pspData[i].Calldata = callData

		if len(psp.DataStr) <= 2 {
			return common.Address{}, []byte{}, fmt.Errorf("Data is empty")
		}
		Data, err := hex.DecodeString(psp.DataStr[2:])
		if err != nil {
			return common.Address{}, []byte{}, fmt.Errorf("failed to parse data %w", err)
		}
		pspData[i].Data = Data
	}

	current_psp, err := getLatestPSP(pspData, nonce)
	if err != nil {
		return common.Address{}, []byte{}, fmt.Errorf("failed to get the latest PSP: %w", err)
	}
	return current_psp.SafeAddr, current_psp.Calldata, nil
}

// getLatestPSP will return the PSP that has the correct nonce.
func getLatestPSP(pspData []PSP, nonce uint64) (PSP, error) {
	for _, psp := range pspData {
		if psp.SafeNonce == nonce {

			return psp, nil
		}
	}
	return PSP{}, fmt.Errorf("no PSP found with nonce %d", nonce)
}

// ExecutePSPOnchain is a core function that will check that status of the superchain is not paused and then send onchain transaction to pause the superchain.
// This function take the PSP data in parameter, we make sure before that the nonce is correct to execute the PSP.
func (d *Defender) ExecutePSPOnchain(ctx context.Context, safe_address common.Address, calldata []byte) (common.Hash, error) {
	pause_before_transaction, err := d.checkPauseStatus(ctx)
	if err != nil {
		log.Error("failed to check the pause status of the SuperChainConfig", "error", err, "superchainconfig_address", d.superChainConfigAddress)
		return common.Hash{}, err
	}
	if pause_before_transaction {

		return common.Hash{}, errors.New("the SuperChainConfig is already paused")

	}
	log.Info("[Before Transaction] status of the pause()", "pause", pause_before_transaction)
	log.Info("Current parameters", "SuperchainConfigAddress", d.superChainConfigAddress, "safe_address", d.safeAddress, "chainID", d.chainID)

	txHash, err := sendTransaction(d.l1Client, d.chainID, d.privatekey, safe_address, big.NewInt(0), calldata) // Send the transaction to the chain with 0 wei.
	if err != nil {
		return common.Hash{}, err
	}
	log.Info("Transaction sent!", "TxHash", txHash)

	pause_after_transaction, err := d.checkPauseStatus(ctx)
	if !pause_after_transaction {
		return txHash, fmt.Errorf("failed to pause the SuperChainConfig")
	}
	if err != nil {
		log.Error("failed to check the pause status of the SuperChainConfig", "error", err, "superchainconfig_address", d.superChainConfigAddress)
		return common.Hash{}, err
	}
	log.Info("[After Transaction] status of the pause()", "pause", pause_after_transaction)

	return txHash, nil

}

// Run() will start the Defender API server and block the thread.
func (d *Defender) Run(ctx context.Context) {
	err := http.ListenAndServe(":"+d.port, d.router) // Start the HTTP server blocking thread for now.
	if err != nil {
		log.Crit("failed to start the API server", "error", err)
	}
}

// Close will close the Defender API server and the L1 client.
func (d *Defender) Close(_ context.Context) error {
	d.l1Client.Close() //close the L1 client.
	return nil
}

// sendTransaction: Is a function made for sending a transaction on chain with the parameters : eth client, privatekey, toAddress, amount of eth in wei, data.
func sendTransaction(client *ethclient.Client, chainID *big.Int, privateKey *ecdsa.PrivateKey, toAddress common.Address, amount *big.Int, data []byte) (common.Hash, error) {

	if privateKey == nil || *privateKey == (ecdsa.PrivateKey{}) {
		return common.Hash{}, fmt.Errorf("private key is nil")
	}
	// Derive the public key from the private key.
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Hash{}, fmt.Errorf("error casting public key to ECDSA")
	}

	// Derive the sender address from the public key
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Ensure the recipient address is valid.
	if (toAddress == common.Address{}) {
		return common.Hash{}, fmt.Errorf("invalid recipient address (toAddress)")
	}
	// Get the nonce for the current transaction.
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %v", err)
	}

	// Set up the transaction parameters
	value := amount                            // Amount of ether to send in wei
	gasLimit := uint64(1000 * DefaultGasLimit) // In units TODO: Need to use `estimateGas()` to get the correct value.
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to suggest gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	// Sign the transaction with the private key
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash(), nil
}

// checkPauseStatus: Is a function made for checking the pause status of the SuperChainConfigAddress
func (d *Defender) checkPauseStatus(ctx context.Context) (bool, error) {
	// Get the contract instance
	paused, err := d.superChainConfig.Paused(&bind.CallOpts{Context: ctx})
	if err != nil {
		return false, err
	}

	return paused, nil
}
