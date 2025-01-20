package validator

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Raw represents raw event data associated with a blockchain transaction.
// type Raw struct {
// 	BlockNumber uint64      // The block number in which the transaction is included.
// 	TxHash      common.Hash // The hash of the transaction.
// }

// // String provides a string representation of Raw.
// func (r Raw) String() string {
// 	return fmt.Sprintf("{BlockNumber: %d, TxHash: %s}", r.BlockNumber, r.TxHash.String())
// }

// Raw represents raw event data associated with a blockchain transaction.
type BlockInfo struct {
	BlockNumber uint64    // The block number in which the transaction is included.
	BlockTime   Timestamp // The timestamp of the block.
}

// String provides a string representation of BlockInfo.
func (b BlockInfo) String() string {
	return fmt.Sprintf("{BlockNumber: %d, BlockTime: %s}", b.BlockNumber, b.BlockTime.String())
}

// Timestamp represents a Unix timestamp.
type Timestamp uint64

// String converts a Timestamp to a formatted string representation.
// It returns the timestamp as a string in the format "2006-01-02 15:04:05 MST".
func (timestamp Timestamp) String() string {
	t := time.Unix(int64(timestamp), 0)
	return t.Format("2006-01-02 15:04:05 MST")
}

// StringToBytes32 converts a hexadecimal string to a [32]uint8 array.
// It returns the converted array and any error encountered during the conversion.
func StringToBytes32(input string) ([32]uint8, error) {
	// Remove the "0x" prefix if present
	if strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X") {
		input = input[2:]
	}

	// Decode the hexadecimal string
	bytes, err := hex.DecodeString(input)
	if err != nil {
		return [32]uint8{}, err
	}

	// Convert bytes to [32]uint8
	var array [32]uint8
	copy(array[:], bytes)
	return array, nil
}
