package global_events

import (
	"testing"
)

func TestFormatSignature(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Basic Function",
			input:          "balanceOf(address owner)",
			expectedOutput: "balanceOf(address)",
		},
		{
			name:           "Function With Multiple Params",
			input:          "transfer(address to, uint256 amount)",
			expectedOutput: "transfer(address,uint256)",
		},
		{
			name:           "Function With No Params",
			input:          "pause()",
			expectedOutput: "pause()",
		},
		{
			name:           "Function With Extra Spaces",
			input:          " approve ( address spender , uint256 value ) ",
			expectedOutput: "approve(address,uint256)",
		},
		{
			name:           "Invalid Input",
			input:          "invalidInput",
			expectedOutput: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := formatSignature(test.input)
			if output != test.expectedOutput {
				t.Errorf("Failed %s: expected %q but got %q", test.name, test.expectedOutput, output)
			}
		})
	}
}
