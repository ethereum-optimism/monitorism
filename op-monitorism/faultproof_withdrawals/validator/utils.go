package validator

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Raw represents raw event data associated with a blockchain transaction.
type Raw struct {
	BlockNumber uint64      // The block number in which the transaction is included.
	TxHash      common.Hash // The hash of the transaction.
}

// String provides a string representation of Raw.
func (r Raw) String() string {
	return fmt.Sprintf("{BlockNumber: %d, TxHash: %s}", r.BlockNumber, r.TxHash.String())
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

// BlockFetcher defines the interface for fetching blocks
type BlockFetcher interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// RetryBlockNumber retries to get a block by number with a backoff.
func RetryBlockNumber(ctx context.Context, client BlockFetcher, log log.Logger, latestBlockNumber *big.Int, maxRetries int) (*types.Block, error) {
	var baseDelay = 1 * time.Second

	// Try forever if maxRetries is < 1
	for i := 0; maxRetries < 1 || i < maxRetries; i++ {
		block, err := client.BlockByNumber(ctx, latestBlockNumber)
		if err == nil {
			return block, nil
		}

		// Limit the delay to 20 seconds
		delay := baseDelay
		if i > 0 {
			delay = baseDelay * time.Duration(1<<uint(i))
		}
		if delay > 20*time.Second {
			delay = 20 * time.Second
		}
		log.Debug("Failed to get block, retrying with backoff",
			"attempt", i+1,
			"maxRetries", maxRetries,
			"delay", delay,
			"error", err)

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during block fetch retry")
		case <-time.After(delay):
			continue
		}
	}

	return nil, fmt.Errorf("failed to get block after all retries")
}

// RetryLatestBlock retries to get the latest block with a backoff.
func RetryLatestBlock(ctx context.Context, client BlockFetcher, log log.Logger, maxRetries int) (*types.Block, error) {
	return RetryBlockNumber(ctx, client, log, nil, maxRetries)
}
