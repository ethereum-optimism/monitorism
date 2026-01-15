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
	"github.com/ethereum/go-ethereum/ethclient"
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

// Needed so that we can mock the interface for tests
type BlockFetcher interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type BlockFetcherWrapper struct {
	client *ethclient.Client
}

func NewBlockFetcher(client *ethclient.Client) BlockFetcher {
	return &BlockFetcherWrapper{client: client}
}

func (w *BlockFetcherWrapper) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return w.client.BlockByNumber(ctx, number)
}

func (w *BlockFetcherWrapper) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return w.client.HeaderByNumber(ctx, number)
}
