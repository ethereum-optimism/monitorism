package faultproof_withdrawals

import (
	"encoding/hex"
	"strings"

	"github.com/joho/godotenv"
)

func stringToBytes32(input string) ([32]uint8, error) {

	if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
		input = input[2:]
	}

	bytes, err := hex.DecodeString(input)
	if err != nil {
		return [32]uint8{}, err
	}

	// Convert bytes to [32]uint8
	var array [32]uint8
	copy(array[:], bytes)
	return array, nil
}

func loadEnv(env string) error {
	return godotenv.Load(env)
}
