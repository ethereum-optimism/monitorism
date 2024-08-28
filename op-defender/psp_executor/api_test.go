package psp_executor

import (
	"bytes"
	"context"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"net/http/httptest"
	"testing"
)

type SimpleExecutor struct{}

func (e *SimpleExecutor) FetchAndExecute(d *Defender) {
	// Implement logic or return mock response
}

// TestDefenderInitialization tests the initialization of the Defender struct
func TestDefenderInitialization(t *testing.T) {
	// Mock dependencies or create real ones depending on your test needs
	var logger log.Logger
	var l1Client *ethclient.Client
	var router *mux.Router

	// You can use real values or mock values for testing
	port := "8080"
	superChainConfigAddress := "0x123"

	// Initialize Prometheus metrics for testing
	latestPspNonce := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "latest_psp_nonce",
		Help: "Latest PSP nonce",
	}, []string{"tag"})
	unexpectedRpcErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "unexpected_rpc_errors",
		Help: "Unexpected RPC errors count",
	}, []string{"errorType"})

	// Create an instance of Defender
	defender := Defender{
		log:                     logger,
		port:                    port,
		SuperChainConfigAddress: superChainConfigAddress,
		l1Client:                l1Client,
		router:                  router,
		latestPspNonce:          latestPspNonce,
		unexpectedRpcErrors:     unexpectedRpcErrors,
	}

	// Check if the Defender instance is initialized correctly
	if defender.port != port {
		t.Errorf("expected port %s, got %s", port, defender.port)
	}
	if defender.SuperChainConfigAddress != superChainConfigAddress {
		t.Errorf("expected SuperChainConfigAddress %s, got %s", superChainConfigAddress, defender.SuperChainConfigAddress)
	}
	// Add more checks as necessary for your application
}

// TestHandlePost tests the handlePost function for various scenarios
func TestHandlePostE2E(t *testing.T) {
	// Initialize the Defender with necessary mock or real components
	logger := log.New()
	// metricsfactory := prometheus.NewRegistry() // Use Prometheus for metrics
	metricsRegistry := opmetrics.NewRegistry()
	metricsfactory := opmetrics.With(metricsRegistry)
	executor := &SimpleExecutor{}
	cfg := CLIConfig{
		NodeUrl: "https://rpc.tenderly.co/fork/not_a_valid", // Example URL
		portapi: "8080",
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
	}{
		{
			name:           "Valid Request",
			body:           `{"pause":true,"timestamp":1596240000,"operator":"0x123","calldata":"0xabc"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON",
			body:           `{"pause":true, "timestamp":"invalid","operator":"0x123"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing Fields",
			body:           `{"timestamp":1596240000}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/psp_execution", bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("Could not create request: %v", err)
			}

			recorder := httptest.NewRecorder()
			handler := http.HandlerFunc(defender.handlePost)

			handler.ServeHTTP(recorder, req)

			if status := recorder.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}
		})
	}
}
