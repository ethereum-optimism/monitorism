package psp_executor

import (
	"bytes"
	"context"
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

// TestHTTPServerHasOnlyPSPExecutionRoute tests if the HTTP server has only one route with "/api/psp_execution" path and "POST" method.
func TestHTTPServerHasOnlyPSPExecutionRoute(t *testing.T) {
	// Mock dependencies or create real ones depending on your test needs
	logger := log.New() //@TODO: replace with testlog  https://github.com/ethereum-optimism/optimism/blob/develop/op-service/testlog/testlog.go#L61
	executor := &SimpleExecutor{}
	metricsfactory := opmetrics.With(opmetrics.NewRegistry())
	mockNodeUrl := "http://rpc.tenderly.co/fork/" // Need to have the "fork" in the URL to avoid mistake for now.
	cfg := CLIConfig{
		NodeURL: mockNodeUrl,
		PortAPI: "8080",
	}
	// Initialize the Defender with necessary mock or real components
	defender, err := NewDefender(context.Background(), logger, metricsfactory, cfg, executor)

	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
	}

	// We Check if the router has only one route
	routeCount := 0
	expectedPath := "/api/psp_execution"
	expectedMethod := "POST"
	var foundRoute *mux.Route

	defender.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		routeCount++
		foundRoute = route
		return nil
	})

	if routeCount != 1 {
		t.Errorf("Expected 1 route, but found %d", routeCount)
	}

	if foundRoute != nil {
		path, _ := foundRoute.GetPathTemplate()
		methods, _ := foundRoute.GetMethods()

		if path != expectedPath {
			t.Errorf("Expected path %s, but found %s", expectedPath, path)
		}

		if len(methods) != 1 || methods[0] != expectedMethod {
			t.Errorf("Expected method %s, but found %v", expectedMethod, methods)
		}
	} else {
		t.Error("No route found")
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
		NodeURL: mockNodeUrl,
		PortAPI: "8080",
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
		NodeURL: mockNodeUrl,
		PortAPI: "8080",
	}

	defender, err := NewDefender(context.Background(), logger, metricsfactory, cfg, executor)
	if err != nil {
		t.Fatalf("Failed to create Defender: %v", err)
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
			body:           `{"pause":true,"timestamp":1596240000,"operator":"0x123","calldata":"0xabc"}`,
			expectedStatus: http.StatusOK,
		},
		{
			path:           "/api/psp_execution",
			name:           "Invalid JSON", // Check if the JSON is invalid return the 400 status code.
			body:           `{"pause":true, "timestamp":"invalid","operator":"0x123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/psp_execution",
			name:           "Missing Fields", // Check if the required fields are missing return the 400 status code.
			body:           `{"timestamp":1596240000}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			path:           "/api/",
			name:           "Incorrect Path Fields", // Check if the path is incorrect return the 404 status code.
			body:           `{"pause":true,"timestamp":1596240000,"operator":"0x123","calldata":"0xabc"}`,
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
				t.Errorf("handler \"%s\" returned wrong status code: got %v want %v",
					tc.name, status, tc.expectedStatus)
			}
		})
	}
}

// TestCheckAndReturnRPC tests that the CheckAndReturnRPC function returns the correct client or error for an incorrect URL provided.
func TestCheckAndReturnRPC(t *testing.T) {
	tests := []struct {
		name        string
		rpcURL      string
		expectedErr bool
	}{
		{"Empty URL", "", true},
		{"Production URL", "https://mainnet.infura.io", true},
		{"Valid Tenderly Fork URL", "https://rpc.tenderly.co/fork/some-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := CheckAndReturnRPC(tt.rpcURL)
			if (err != nil) != tt.expectedErr {
				t.Errorf("CheckAndReturnRPC() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			if !tt.expectedErr && client == nil {
				t.Errorf("CheckAndReturnRPC() returned nil client for valid URL")
			}
		})
	}
}
