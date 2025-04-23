package psp_executor

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"

	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
)

type SimpleExecutor struct{}

func (e *SimpleExecutor) FetchAndExecute(d *Defender) (common.Hash, error) {
	// Do nothing for now, for mocking purposes
	return common.Hash{}, nil
}

func (e *SimpleExecutor) ReturnCorrectChainID(l1client *ethclient.Client, chainID uint64) (*big.Int, error) { // Do nothing for now, for mocking purposes
	return big.NewInt(1), nil
}

func (e *SimpleExecutor) FetchAndSimulateAtBlock(ctx context.Context, d *Defender, blocknumber *uint64, nonce uint64) ([]byte, error) { // Do nothing for now, for mocking purposes
	return []byte{}, nil
}

// GeneratePrivatekey generates a private key of the given size useful for testing.
func GeneratePrivatekey(size int) string {
	// Generate a random byte slice of the specified size
	privateKeyBytes := make([]byte, size)
	_, err := rand.Read(privateKeyBytes)
	if err != nil {
		return ""
	}

	// Convert the byte slice to a hexadecimal string
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Add the "0x" prefix to the hexadecimal string
	return "0x" + privateKeyHex
}

// TestHTTPServerHasCorrectRoute tests if the HTTP server has the correct route with "/api/psp_execution" path and "POST" method and the "/api/healthcheck" path and "GET" method.
func TestHTTPServerHasCorrectRoute(t *testing.T) {
	// Mock dependencies or create real ones depending on your test needs
	logger := testlog.Logger(t, log.LvlInfo)
	executor := &SimpleExecutor{}
	metricsfactory := opmetrics.With(opmetrics.NewRegistry())
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Path:                    "/tmp",
		SafeAddress:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		ChainID:                 1,
		BlockDuration:           12,
	}
	// Initialize the Defender with necessary mock or real components
	defender, err := NewDefender(context.Background(), logger, metricsfactory, cfg, executor)

	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

	// We Check if the router has two routes
	routeCount := 0
	expectedRoutes := map[string]string{
		"/api/psp_execution": "POST",
		"/api/healthcheck":   "GET",
	}
	foundRoutes := make(map[string]string)

	err = defender.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		if len(methods) > 0 {
			foundRoutes[path] = methods[0]
			routeCount++
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to walk routes: %v", err)
	}

	if routeCount != 2 {
		t.Errorf("Expected 2 routes, but got %d", routeCount)
	}

	for path, method := range expectedRoutes {
		if foundMethod, ok := foundRoutes[path]; !ok {
			t.Errorf("Expected route %s not found", path)
		} else if foundMethod != method {
			t.Errorf("Expected method %s for path %s, but got %s", method, path, foundMethod)
		}
	}
}

// TestDefenderInitialization tests the initialization of the Defender struct with mock dependencies.
func TestDefenderInitialization(t *testing.T) {
	// Mock dependencies or create real ones depending on your test needs
	logger := testlog.Logger(t, log.LvlInfo)
	executor := &SimpleExecutor{}
	metricsfactory := opmetrics.With(opmetrics.NewRegistry())
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Path:                    "/tmp",
		SafeAddress:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockDuration:           12,
	}
	// Initialize the Defender with necessary mock or real components
	_, err := NewDefender(context.Background(), logger, metricsfactory, cfg, executor)

	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

}

// TestHandlePostMockFetch tests the handlePost function with HTTP status code to make sure HTTP code returned are expected in every possible cases.
func TestHandlePostMockFetch(t *testing.T) {
	// Initialize the Defender with necessary mock or real components
	logger := testlog.Logger(t, log.LvlInfo)
	metricsRegistry := opmetrics.NewRegistry()
	metricsfactory := opmetrics.With(metricsRegistry)
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	executor := &SimpleExecutor{}
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Path:                    "/tmp",
		SafeAddress:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockDuration:           12,
	}

	defender, err := NewDefender(context.Background(), logger, metricsfactory, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

	// Create a large request body (> 1MB)
	largeBody := make([]byte, 1048577) // 1MB + 1 byte
	for i := range largeBody {
		largeBody[i] = 'a'
	}
	// Define test cases
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		path           string
	}{
		{
			path:           "/api/psp_execution",
			name:           "Valid Request", // Check if the request is valid as expected return the 200 status code.
			body:           `{"Pause":true,"Timestamp":1596240000,"Operator":"0x123"}`,
			expectedStatus: http.StatusOK,
		},
		{
			path:           "/api/psp_execution",
			name:           "Invalid request with a non boolean for the pause field",
			body:           `{"Pause":"not a boolean","Timestamp":1596240000,"Operator":"0x123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Invalid timestamp",
			body:           `{"Pause":true, "Timestamp":"invalid","Operator":"0x123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Empty operator",
			body:           `{"Pause":true, "Timestamp":1596240000,"Operator":""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Empty operator",
			body:           `{"Pause":true, "Timestamp":1596240000,"Operator":[""]}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Invalid JSON", // Check if the JSON is invalid return the 400 status code.
			body:           `{"Pause":true, "Timestamp":"1596240000","Operator":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Missing Fields", // Check if the required fields are missing return the 400 status code.
			body:           `{"Timestamp":1596240000}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Too Many Fields", // Check if there are extra fields present and return the 400 status code.
			body:           `{"Pause":true,"Timestamp":1596240000,"Operator":"0x123", "extra":"unnecessary_value"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Payload Size Greater Than Limit", // Check if the path is incorrect return the 404 status code.
			body:           `{"Pause":true,"Timestamp":1596240000,"Operator":"` + string(largeBody) + `"}`,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			path:           "/api/",
			name:           "Incorrect Path Fields", // Check if the path is incorrect return the 404 status code.
			body:           `{"Pause":true,"Timestamp":1596240000,"Operator":"0x123"}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", tc.path, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("Could not create request: %v", err)
			}
			recorder := httptest.NewRecorder()

			// Get the servermux of the defender.router to check the routes
			muxrouter := defender.router
			// Use the mux to serve the request
			muxrouter.ServeHTTP(recorder, req)

			if status := recorder.Code; status != tc.expectedStatus {
				t.Errorf("handler \"%s\" returned wrong status code: Expected %v but got %v",
					tc.name, tc.expectedStatus, status)
			}
		})
	}
}

// TestCheckAndReturnRPC tests that the CheckAndReturnRPC function returns the correct client or error for an incorrect URL provided.
func TestCheckAndReturnRPC(t *testing.T) {
	nullByte, _ := hex.DecodeString("00")
	tests := []struct {
		name        string
		rpcURL      string
		expectError bool
	}{
		{"Empty URL", "", true},
		{"Production URL Mainnet", "https://mainnet.infura.io", true},
		{"Invalid URL", "https://sepolia.infura.io/" + string(nullByte), true},
		{"Production URL Sepolia", "https://sepolia.infura.io", false},
		{"Valid Tenderly Fork URL", "https://rpc.tenderly.co/fork/some-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := CheckAndReturnRPC(tt.rpcURL)
			if (err != nil) != tt.expectError {
				t.Errorf("Test: \"%s\" Expected error = %v, but got %v", tt.name, tt.expectError, err)

				return
			}
			if !tt.expectError && client == nil {
				t.Errorf("Test: \"%s\" Expected no error but got \"client=<nil>\"", tt.name)
			}
		})
	}
}

func TestCheckAndReturnPrivateKey(t *testing.T) {
	validPrivateKeyGeneratedStr := GeneratePrivatekey(32)
	validPrivateKeyGenerated, _ := crypto.HexToECDSA(validPrivateKeyGeneratedStr[2:])
	hardCodedTestPrivateKey, _ := crypto.HexToECDSA("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	tests := []struct {
		name        string
		input       string
		expected    *ecdsa.PrivateKey
		expectError bool
	}{
		{"Valid private key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", hardCodedTestPrivateKey, false},
		{"Valid private key without 0x", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", hardCodedTestPrivateKey, false},
		{"Valid private key Generated", validPrivateKeyGeneratedStr, validPrivateKeyGenerated, false},
		{"Empty string", "", nil, true},
		{"Invalid hex string", "0xInvalidHex", nil, true},
		{"Incorrect length (2 bytes)", "0x1234", nil, true},
		{"Incorrect length (38 bytes)", "0x1234123412341234123412341234123412341234123412341234123412341234123412341234", nil, true},
		{"Invalid private key", "0x0000000000000000000000000000000000000000000000000000000000000000", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CheckAndReturnPrivateKey(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !result.Equal(tt.expected) {
					t.Errorf("Test: \"%s\" Expected %s, but got %s", tt.name, tt.expected, result)
				}
			}
		})
	}
}

func TestIsValidHex(t *testing.T) {
	tests := []struct {
		name       string
		hexString  string
		returnBool bool
	}{
		{"Valid hex string", "414141", true},
		{"Invalid hex string with 0x", "0x414141", false},
		{"Invalid hex string", "zzzz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := isValidHexString(tt.hexString)
			if ok != tt.returnBool {
				t.Errorf("Test: \"%s\" Expected = %t, but got %t", tt.name, tt.returnBool, ok)
			}
		})
	}
}

func TestGetLatestPSP(t *testing.T) {
	var PSPTest1 = PSP{
		SafeNonce: 123456789,
	}
	var PSPTest2 = PSP{
		SafeNonce: 111111111,
	}
	tests := []struct {
		name        string
		PSPs        []PSP
		expected    PSP
		expectError bool
	}{
		{"PSP slice with expected safeNonce", []PSP{PSPTest1, PSPTest2}, PSPTest1, false},
		{"PSP slice without the expected safeNonce", []PSP{PSPTest2}, PSP{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PSPData, err := getLatestPSP(tt.PSPs, 123456789)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if PSPData.SafeNonce != tt.expected.SafeNonce {
					t.Errorf("Test: \"%s\" Expected %#v, but got %#v", tt.name, tt.expected, PSPData)
				}
			}

		})
	}
}

func TestGetPSPbyNonceFromFile(t *testing.T) {
	const filename = "testfile_TestGetPSPbyNonceFromFile.txt"
	const PSPsValid = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  },
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  }
]`

	const PSPIncorrectJSON = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
`
	const PSPNoData = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  }]`
	const PSPInvalidData = `[
    {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xzz",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  }]`
	const PSPNoCalldata = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0x"
  }]`
	const PSPInvalidCallData = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xzz"
  }]`
	const PSPInvalidSafeNonce = `[
  {
    "chain_id": "11155111",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "abc",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  }]`
	const PSPInvalidChainID = `[
  {
    "chain_id": "chainID",
    "rpc_url": "https://ethereum-sepolia.publicnode.com",
    "created_at": "2024-08-22T20:00:06+02:00",
    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
    "safe_nonce": "0",
    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
    "script_name": "PresignPauseFromJson.s.sol",
    "data": "0xe4b2f9f3",
    "signatures": [
      {
        "signer": "0x0000000000000000000000000000000000003333",
        "signature": "DEADBEEF"
      },
      {
        "signer": "0x0000000000000000000000000000000000004444",
        "signature": "DEADBEEF"
      }
    ],
    "calldata": "0xe4b2f9f3"
  }]`
	/*const PSPNoSignature = `[
	  {
	    "chain_id": "11155111",
	    "rpc_url": "https://ethereum-sepolia.publicnode.com",
	    "created_at": "2024-08-22T20:00:06+02:00",
	    "safe_addr": "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E",
	    "safe_nonce": "0",
	    "target_addr": "0xfd7E6Ef1f6c9e4cC34F54065Bf8496cE41A4e2e8",
	    "script_name": "PresignPauseFromJson.s.sol",
	    "data": "0xe4b2f9f3",
	    "signatures": [

	    ],
	    "calldata": "0xe4b2f9f3"
	  }]`*/
	tests := []struct {
		name                string
		PSPs                string
		nonce               uint64
		expectedSafeAddress common.Address
		expectedData        []byte
		expectError         bool
	}{
		{"Valid PSP", PSPsValid, 0, common.HexToAddress("0x837DE453AD5F21E89771e3c06239d8236c0EFd5E"), common.FromHex("e4b2f9f3"), false},
		{"Valid PSP with unknown nonce", PSPsValid, 1, common.Address{}, []byte{}, true},
		{"PSP with incorrect JSON", PSPIncorrectJSON, 0, common.Address{}, []byte{}, true},
		{"PSP with no data", PSPNoData, 0, common.Address{}, []byte{}, true},
		{"PSP with invalid data", PSPInvalidData, 0, common.Address{}, []byte{}, true},
		{"PSP with no calldata", PSPNoCalldata, 0, common.Address{}, []byte{}, true},
		{"PSP with an invalid calldata", PSPInvalidCallData, 0, common.Address{}, []byte{}, true},
		{"PSP with invalid safe nonce", PSPInvalidSafeNonce, 0, common.Address{}, []byte{}, true},
		{"PSP with invalid chain id", PSPInvalidChainID, 0, common.Address{}, []byte{}, true},
		// {"PSP with no signature", PSPNoSignature, 0, common.Address{}, []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.WriteFile(filename, []byte(tt.PSPs), 0644)
			if err != nil {
				t.Errorf("Fail to write %s file: %v", filename, err)
			}
			safeAddress, data, err := GetPSPbyNonceFromFile(tt.nonce, filename)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if safeAddress != tt.expectedSafeAddress {
					t.Errorf("Test: \"%s\" Expected %#v, but got %#v", tt.name, tt.expectedSafeAddress, safeAddress)
				}

				if !bytes.Equal(data, tt.expectedData) {
					t.Errorf("Test: \"%s\" Expected %#v, but got %#v", tt.name, tt.expectedData, data)
				}
			}
			os.Remove(filename)
		})
	}

	name := "File does not exist"
	t.Run(name, func(t *testing.T) {
		_, _, err := GetPSPbyNonceFromFile(0, "file notfound")
		if err == nil {
			t.Errorf("Test: \"%s\" Expected an error, but got no error", name)
		}
	})

}

func TestGetNonceSafe(t *testing.T) {
	const safeAddressMainnet = "0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A"
	const safeAddressSepolia = "0x837DE453AD5F21E89771e3c06239d8236c0EFd5E"
	const rpcURLMainnet = "https://ethereum-rpc.publicnode.com"
	const rpcURLSepolia = "https://ethereum-sepolia-rpc.publicnode.com"

	l1ClientMainnet, err := ethclient.Dial(rpcURLMainnet)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	l1ClientSepolia, err := ethclient.Dial(rpcURLSepolia)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	safeAddressMainnetBindings, _ := bindings.NewSafe(common.HexToAddress(safeAddressMainnet), l1ClientMainnet)
	safeAddressSepoliaBindings, _ := bindings.NewSafe(common.HexToAddress(safeAddressSepolia), l1ClientSepolia)

	// Initialize the Defender with necessary mock or real components
	logger := testlog.Logger(t, log.LvlInfo)
	metricsRegistry_mainnet := opmetrics.NewRegistry()
	metricsRegistry_sepolia := opmetrics.NewRegistry()
	metricsRegistry_sepolia_error := opmetrics.NewRegistry()
	metricsfactory_mainnet := opmetrics.With(metricsRegistry_mainnet)
	metricsfactory_sepolia := opmetrics.With(metricsRegistry_sepolia)
	metricsfactory_sepolia_error := opmetrics.With(metricsRegistry_sepolia_error)
	executor := &SimpleExecutor{}
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.

	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Path:                    "/tmp",
		SafeAddress:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockDuration:           12,
	}

	defenderMainnet, err := NewDefender(context.Background(), logger, metricsfactory_mainnet, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}
	defenderMainnet.l1Client = l1ClientMainnet
	defenderMainnet.operationSafe = safeAddressMainnetBindings

	defenderSepolia, err := NewDefender(context.Background(), logger, metricsfactory_sepolia, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}
	defenderSepolia.l1Client = l1ClientSepolia
	defenderSepolia.operationSafe = safeAddressSepoliaBindings

	defenderError, err := NewDefender(context.Background(), logger, metricsfactory_sepolia_error, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

	tests := []struct {
		name                  string
		defender              *Defender
		expectedSuperiorNonce uint64
		expectError           bool
	}{
		{"Nonce from Mainnet", defenderMainnet, 95, false},
		{"Nonce from Sepolia", defenderSepolia, 0, false},
		{"Nonce with an incorrect RPC", defenderError, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nonce, err := tt.defender.getNonceSafe(context.Background())
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if nonce < tt.expectedSuperiorNonce {
					t.Errorf("Test: \"%s\" Expected a nonce > %#v, but got %#v", tt.name, tt.expectedSuperiorNonce, nonce)
				}
			}

		})
	}

}

func TestReturnCorrectChainID(t *testing.T) {
	const rpcURLMainnet = "https://ethereum-rpc.publicnode.com"
	const rpcURLSepolia = "https://ethereum-sepolia-rpc.publicnode.com"
	const rpcURLInvalid = "http://www.google.com"
	l1ClientMainnet, err := ethclient.Dial(rpcURLMainnet)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	l1ClientSepolia, err := ethclient.Dial(rpcURLSepolia)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	l1InvalidClient, err := ethclient.Dial(rpcURLInvalid)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	executor := &DefenderExecutor{}
	tests := []struct {
		name            string
		l1Client        *ethclient.Client
		chainID         uint64
		expectedChainID *big.Int
		expectError     bool
	}{
		{"Check chain id on Mainnet", l1ClientMainnet, 1, big.NewInt(1), false},
		{"Check chain id on Sepolia", l1ClientSepolia, 11155111, big.NewInt(11155111), false},
		{"Invalid chain id on Sepolia", l1ClientSepolia, 100, big.NewInt(0), true},
		{"Not chain id configured on Sepolia", l1ClientSepolia, 0, big.NewInt(0), true},
		{"nil RPC", nil, 1, big.NewInt(0), true},
		{"Invalid RPC", l1InvalidClient, 1, big.NewInt(0), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainID, err := executor.ReturnCorrectChainID(tt.l1Client, tt.chainID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if chainID.Cmp(tt.expectedChainID) != 0 {
					t.Errorf("Test: \"%s\" Expected %#v, but got %#v", tt.name, tt.expectedChainID, chainID)
				}
			}

		})
	}

}

func TestAddressFromPrivateKey(t *testing.T) {
	validPrivateKeyGeneratedStr := GeneratePrivatekey(32)
	validPrivateKeyGenerated, _ := crypto.HexToECDSA(validPrivateKeyGeneratedStr[2:])
	validPrivateKeyGeneratedBytes := crypto.FromECDSAPub(validPrivateKeyGenerated.Public().(*ecdsa.PublicKey))
	hash := sha3.NewLegacyKeccak256()
	hash.Write(validPrivateKeyGeneratedBytes[1:])
	hardCodedTestPrivateKey, _ := crypto.HexToECDSA("1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")

	tests := []struct {
		name            string
		privateKey      *ecdsa.PrivateKey
		expectedAddress common.Address
		expectError     bool
	}{
		{"Valid private key", hardCodedTestPrivateKey, common.HexToAddress("0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb"), false},
		{"Valid private key without 0x", hardCodedTestPrivateKey, common.HexToAddress("0x1Be31A94361a391bBaFB2a4CCd704F57dc04d4bb"), false},
		{"Valid private key generated", validPrivateKeyGenerated, common.BytesToAddress(hash.Sum(nil)[12:]), false},
		{"Empty private key", nil, common.Address{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AddressFromPrivateKey(tt.privateKey)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.expectedAddress != result {
					t.Errorf("Test: \"%s\" Expected %s, but got %s", tt.name, tt.expectedAddress, result)
				}
			}
		})
	}
}

func TestCheckPauseStatus(t *testing.T) {
	const superChainAddressMainnet = "0x95703e0982140D16f8ebA6d158FccEde42f04a4C"
	const superChainAddressSepolia = "0xC2Be75506d5724086DEB7245bd260Cc9753911Be"
	const rpcURLMainnet = "https://ethereum-rpc.publicnode.com"
	const rpcURLSepolia = "https://ethereum-sepolia-rpc.publicnode.com"
	l1ClientMainnet, err := ethclient.Dial(rpcURLMainnet)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	l1ClientSepolia, err := ethclient.Dial(rpcURLSepolia)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	superChainAddressMainnetBindings, _ := bindings.NewSuperchainConfig(common.HexToAddress(superChainAddressMainnet), l1ClientMainnet)
	superChainAddressSepoliaBindings, _ := bindings.NewSuperchainConfig(common.HexToAddress(superChainAddressSepolia), l1ClientSepolia)

	// Initialize the Defender with necessary mock or real components
	logger := testlog.Logger(t, log.LvlInfo)
	metricsRegistry_mainnet := opmetrics.NewRegistry()
	metricsRegistry_sepolia := opmetrics.NewRegistry()
	metricsRegistry_sepolia_error := opmetrics.NewRegistry()
	metricsfactory_mainnet := opmetrics.With(metricsRegistry_mainnet)
	metricsfactory_sepolia := opmetrics.With(metricsRegistry_sepolia)
	metricsfactory_sepolia_error := opmetrics.With(metricsRegistry_sepolia_error)
	executor := &SimpleExecutor{}
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.

	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		Path:                    "/tmp",
		SafeAddress:             common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
		BlockDuration:           12,
	}

	defenderMainnet, err := NewDefender(context.Background(), logger, metricsfactory_mainnet, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}
	defenderMainnet.l1Client = l1ClientMainnet
	defenderMainnet.superChainConfig = superChainAddressMainnetBindings

	defenderSepolia, err := NewDefender(context.Background(), logger, metricsfactory_sepolia, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}
	defenderSepolia.l1Client = l1ClientSepolia
	defenderSepolia.superChainConfig = superChainAddressSepoliaBindings

	defenderError, err := NewDefender(context.Background(), logger, metricsfactory_sepolia_error, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

	tests := []struct {
		name           string
		defender       *Defender
		expectedResult bool
		expectError    bool
	}{
		{"Nonce from Mainnet", defenderMainnet, false, false},
		{"Nonce from Sepolia", defenderSepolia, false, false},
		{"Nonce with an incorrect RPC", defenderError, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.defender.checkPauseStatus(context.Background())
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expectedResult {
					t.Errorf("Test: \"%s\" Expected a nonce > %#v, but got %#v", tt.name, tt.expectedResult, result)
				}
			}

		})
	}
}

func TestSendTransaction(t *testing.T) {

	validPrivateKeyGeneratedStr := GeneratePrivatekey(32)
	validPrivateKeyGenerated, _ := crypto.HexToECDSA(validPrivateKeyGeneratedStr[2:])

	const rpcURLMainnet = "https://ethereum-rpc.publicnode.com"
	const rpcURLSepolia = "https://ethereum-sepolia-rpc.publicnode.com"
	const rpcURLInvalid = "http://www.google.com"

	l1ClientSepolia, err := ethclient.Dial(rpcURLSepolia)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	l1InvalidClient, err := ethclient.Dial(rpcURLInvalid)
	if err != nil {
		t.Errorf("Fail to connet to RPC %s: %v", rpcURLMainnet, err)
	}

	tests := []struct {
		name         string
		l1Client     *ethclient.Client
		chainID      *big.Int
		privateKey   *ecdsa.PrivateKey
		toAddress    common.Address
		amount       *big.Int
		data         []byte
		expectedHash common.Hash
		expectError  bool
	}{
		{"Send transaction on Sepolia", l1ClientSepolia, big.NewInt(11155111), validPrivateKeyGenerated, common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be"), big.NewInt(1), []byte("test"), common.Hash{}, true},
		{"Send transaction on Sepolia with an invalid chain id", l1ClientSepolia, big.NewInt(1), validPrivateKeyGenerated, common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be"), big.NewInt(1), []byte("test"), common.Hash{}, true},
		{"Send transaction on Sepolia with nil private key", l1ClientSepolia, big.NewInt(1), nil, common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be"), big.NewInt(1), []byte("test"), common.Hash{}, true},
		{"Send transaction on Sepolia with empty private key", l1ClientSepolia, big.NewInt(1), &ecdsa.PrivateKey{}, common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be"), big.NewInt(1), []byte("test"), common.Hash{}, true},
		{"Send transaction on Sepolia with invalid toAddress", l1ClientSepolia, big.NewInt(1), validPrivateKeyGenerated, common.Address{}, big.NewInt(1), []byte("test"), common.Hash{}, true},
		{"Send transaction on Sepolia with invalid RPC", l1InvalidClient, big.NewInt(1), validPrivateKeyGenerated, common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be"), big.NewInt(1), []byte("test"), common.Hash{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := sendTransaction(tt.l1Client, tt.chainID, tt.privateKey, tt.toAddress, tt.amount, tt.data)
			if tt.expectError {
				if err == nil {
					t.Errorf("Test: \"%s\" Expected an error, but got no error", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hash != tt.expectedHash {
					t.Errorf("Test: \"%s\" Expected %#v, but got %#v", tt.name, tt.expectedHash, hash)
				}
			}

		})
	}

}
