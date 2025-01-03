package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// getBlockAtApproximateTimeBinarySearch finds the block number corresponding to the timestamp from two weeks ago using a binary search approach.
func GetBlockAtApproximateTimeBinarySearch(ctx context.Context, client *ethclient.Client, latestBlockNumber *big.Int, hoursInThePast *big.Int, logger log.Logger) (*big.Int, error) {

	secondsInThePast := hoursInThePast.Mul(hoursInThePast, big.NewInt(60*60))
	logger.Info("Looking for a block at approximate time of hours back",
		"secondsInThePast", fmt.Sprintf("%v", secondsInThePast),
		"time", fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05 MST")),
		"latestBlockNumber", fmt.Sprintf("%v", latestBlockNumber))
	// Calculate the total seconds in two weeks
	targetTime := big.NewInt(time.Now().Unix())
	targetTime.Sub(targetTime, secondsInThePast)

	// Initialize the search range
	left := big.NewInt(0)
	right := new(big.Int).Set(latestBlockNumber)

	var mid *big.Int
	acceptablediff := big.NewInt(60 * 60) //60 minutes

	// Perform binary search
	for left.Cmp(right) <= 0 {
		//interrupt in case of context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled")
		default:
		}

		// Calculate the midpoint
		mid = new(big.Int).Add(left, right)
		mid.Div(mid, big.NewInt(2))

		// Get the block at mid
		block, err := client.BlockByNumber(context.Background(), mid)
		if err != nil {
			return nil, err
		}

		// Check the block's timestamp
		blockTime := big.NewInt(int64(block.Time()))

		//calculate the difference between the block time and the target time
		diff := new(big.Int).Sub(blockTime, targetTime)

		// If block time is less than or equal to target time, check if we need to search to the right
		if blockTime.Cmp(targetTime) <= 0 {
			left.Set(mid) // Move left boundary up to mid
		} else {
			right.Sub(mid, big.NewInt(1)) // Move right boundary down
		}
		if new(big.Int).Abs(diff).Cmp(acceptablediff) <= 0 {
			//if the difference is less than or equal to 1 hour, we can consider this block as the block closest to the target time
			break
		}

	}

	// log the block number closest to the target time and the time
	logger.Info("block number closest to target time", "block", fmt.Sprintf("%v", left), "time", time.Unix(targetTime.Int64(), 0))
	// After exiting the loop, left should be the block number closest to the target time
	return left, nil
}
