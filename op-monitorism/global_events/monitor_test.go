package global_events

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
			name:           "Uniswap swap",
			input:          "Swap (address sender,address recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)",
			expectedOutput: "Swap(address,address,int256,int256,uint160,uint128,int24)",
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

func TestFormatAndHash(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput common.Hash
	}{
		{
			name:           "Uniswap swap",
			input:          "Swap (address indexed sender,address recipient, int256 amount0, int256 amount1, uint160 sqrtPriceX96, uint128 liquidity, int24 tick)",
			expectedOutput: common.HexToHash("0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := FormatAndHash(test.input)
			if output != test.expectedOutput {
				t.Errorf("Failed %s: expected %q but got %q", test.name, test.expectedOutput, output)
			}
		})
	}
}
