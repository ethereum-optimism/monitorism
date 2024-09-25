package psp_executor

import (
	"math/big"
)

// weiToEther converts a wei value to ether return a float64.
func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(1e18, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
