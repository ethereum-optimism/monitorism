package global_events

import (
	//	"strings"
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

// func TestFunctionSelector(t *testing.T) {
// 	expectedSelector := "70a08231" // known selector for "balanceOf(address)"
// 	formattedSignature := "balanceOf(address)"
// 	hash := crypto.Keccak256([]byte(formattedSignature))
// 	selector := strings.ToLower(fmt.Sprintf("%x", hash[:4]))
//
// 	if selector != expectedSelector {
// 		t.Errorf("Selector calculation failed: expected %s but got %s", expectedSelector, selector)
// 	}
// }
