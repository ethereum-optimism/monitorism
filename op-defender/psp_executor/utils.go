package psp_executor

import (
	"errors"
	"math/big"
)

const ether = 1e18

// weiToEther converts a wei value to ether return a float64 return an error if the float is toolarge
func weiToEther(wei *big.Int) (float64, error) {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(ether, 1)
	num = num.Quo(num, denom)
	f, isTooLarge := num.Float64()
	if isTooLarge {
		return 0, errors.New("number is too large to convert to float")
	}
	return f, nil
}
