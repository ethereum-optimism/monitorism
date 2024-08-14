package psp_executor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandlePost tests the handlePost function for various scenarios.
func TestHandlePost(t *testing.T) {
	tests := []struct {
		name           string
		body           RequestData
		expectedStatus int
		expectedBody   string
	}{
		// The next tests case are commented for next PRs and don't require review for now.

		// {
		// 	name: "Valid request",
		// 	body: RequestData{
		// 		Pause:     true,
		// 		Timestamp: 123456789,
		// 		Operator:  "0x123",
		// 		Calldata:  "0x456",
		// 	},
		// 	expectedStatus: http.StatusOK,
		// 	expectedBody:   `{"status":"success","data":{"pause":true,"timestamp":123456789,"operator":"0x123","calldata":"0x456"}}`,
		// },
		// {
		// 	name: "Invalid request with empty fields",
		// 	body: RequestData{
		// 		Pause:     false,
		// 		Timestamp: 0,
		// 		Operator:  "",
		// 		Calldata:  "",
		// 	},
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedBody:   "All fields are required and must be non-zero",
		// },
		{
			name: "Network Authentication Required",
			body: RequestData{
				Pause:     false,
				Timestamp: 0,
				Operator:  "",
				Calldata:  "",
			},
			expectedStatus: http.StatusNetworkAuthenticationRequired,
			expectedBody:   "Network Authentication Required\n", //do not forget the newline character.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert the body to JSON
			jsonBody, _ := json.Marshal(tt.body)
			req, err := http.NewRequest("POST", "/api/psp_execution", bytes.NewBuffer(jsonBody))
			if err != nil {
				t.Fatal(err)
			}

			// Create a ResponseRecorder to record the response.
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlePost) // Call the `handlePost` that is the entrypoint of the API.

			// Call the handler function
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got `%v` want `%v`", status, tt.expectedStatus)
			}

			// Check the response body
			if rr.Body.String() != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got `%v` want `%v`", rr.Body.String(), tt.expectedBody)
			}
		})
	}
}
