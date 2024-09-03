package psp_executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"testing"
)

type SimpleExecutor struct{}

func (e *SimpleExecutor) FetchAndExecute(d *Defender) {
	// Do nothing for now, for mocking purposes
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

// TestHTTPServerHasCorrectRoute() tests if the HTTP server has the correct route with "/api/psp_execution" path and "POST" method and the "/api/healthcheck" path and "GET" method.
func TestHTTPServerHasCorrectRoute(t *testing.T) {
	// Mock dependencies or create real ones depending on your test needs
	logger := log.New() //@TODO: replace with testlog  https://github.com/ethereum-optimism/optimism/blob/develop/op-service/testlog/testlog.go#L61
	executor := &SimpleExecutor{}
	metricsfactory := opmetrics.With(opmetrics.NewRegistry())
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: "0x1234567890abcdef1234567890abcdef12345678",
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

	defender.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		if len(methods) > 0 {
			foundRoutes[path] = methods[0]
			routeCount++
		}
		return nil
	})

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
	logger := log.New() //@TODO: replace with testlog  https://github.com/ethereum-optimism/optimism/blob/develop/op-service/testlog/testlog.go#L61
	executor := &SimpleExecutor{}
	metricsfactory := opmetrics.With(opmetrics.NewRegistry())
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: "0x1234567890abcdef1234567890abcdef12345678",
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
	logger := log.New() //@TODO: replace with testlog  https://github.com/ethereum-optimism/optimism/blob/develop/op-service/testlog/testlog.go#L61
	metricsRegistry := opmetrics.NewRegistry()
	metricsfactory := opmetrics.With(metricsRegistry)
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	executor := &SimpleExecutor{}
	cfg := CLIConfig{
		NodeURL:                 mockNodeUrl,
		PortAPI:                 "8080",
		privatekeyflag:          GeneratePrivatekey(32),
		SuperChainConfigAddress: "0x1234567890abcdef1234567890abcdef12345678",
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
			name:           "Invalid JSON", // Check if the JSON is invalid return the 400 status code.
			body:           `{"Pause":true, "Timestamp":"invalid","Operator":}`,
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
	tests := []struct {
		name      string
		rpcURL    string
		expectErr bool
	}{
		{"Empty URL", "", true},
		{"Production URL", "https://mainnet.infura.io", true},
		{"Valid Tenderly Fork URL", "https://rpc.tenderly.co/fork/some-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := CheckAndReturnRPC(tt.rpcURL)
			if (err != nil) != tt.expectErr {
				t.Errorf("Test: \"%s\" Expected error = %v, but got %v", tt.name, tt.expectErr, err)

				return
			}
			if !tt.expectErr && client == nil {
				t.Errorf("Test: \"%s\" Expected no error but got \"client=<nil>\"", tt.name)
			}
		})
	}
}
func TestCheckAndReturnPrivateKey(t *testing.T) {
	validPrivateKeyGenerated := GeneratePrivatekey(32)
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{"Valid private key", "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"Valid private key without 0x", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"Valid private key Generated", validPrivateKeyGenerated, validPrivateKeyGenerated[2:], false},
		{"Empty string", "", "", true},
		{"Invalid hex string", "0xInvalidHex", "", true},
		{"Incorrect length", "0x1234", "", true},
		{"Invalid private key", "0x0000000000000000000000000000000000000000000000000000000000000000", "", true},
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
				if result != tt.expected {
					t.Errorf("Test: \"%s\" Expected %s, but got %s", tt.name, tt.expected, result)
				}
			}
		})
	}
}
