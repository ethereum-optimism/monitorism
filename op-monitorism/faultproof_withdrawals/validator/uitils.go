package validator

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
)

type Raw struct {
	BlockNumber uint64
	TxHash      common.Hash
}

type Timestamp uint64

func (timestamp Timestamp) String() string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format("2006-01-02 15:04:05 MST")
}

func StringToBytes32(input string) ([32]uint8, error) {

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
