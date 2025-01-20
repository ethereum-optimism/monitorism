package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum/go-ethereum/log"
)

// getBlockAtApproximateTimeBinarySearch finds the block number corresponding to the timestamp from two weeks ago using a binary search approach.
func GetBlockAtApproximateTimeBinarySearch(ctx context.Context, l1Proxy validator.L1ProxyInterface, latestBlockNumber validator.BlockInfo, hoursInThePast *big.Int, logger log.Logger) (validator.BlockInfo, error) {

	latestL1HeightBigInt := new(big.Int).SetUint64(latestBlockNumber.BlockNumber)

	secondsInThePast := hoursInThePast.Mul(hoursInThePast, big.NewInt(60*60))
	logger.Info("Looking for a block at approximate time of hours back",
		"secondsInThePast", fmt.Sprintf("%v", secondsInThePast),
		"time", fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05 MST")),
		"latestBlockNumber", fmt.Sprintf("%v", latestL1HeightBigInt))
	// Calculate the total seconds in two weeks
	targetTime := big.NewInt(time.Now().Unix())
	targetTime.Sub(targetTime, secondsInThePast)

	// Initialize the search range
	left := big.NewInt(0)
	right := new(big.Int).Set(latestL1HeightBigInt)

	var mid *big.Int
	acceptablediff := big.NewInt(60 * 60) //60 minutes

	// Perform binary search
	for left.Cmp(right) <= 0 {
		//interrupt in case of context cancellation
		select {
		case <-ctx.Done():
			return validator.BlockInfo{}, fmt.Errorf("context cancelled")
		default:
		}

		// Calculate the midpoint
		mid = new(big.Int).Add(left, right)
		mid.Div(mid, big.NewInt(2))

		// Get the block at mid
		block, err := l1Proxy.BlockByNumber(mid)
		if err != nil {
			return validator.BlockInfo{}, err
		}

		// Check the block's timestamp
		blockTime := big.NewInt(int64(block.BlockTime))

		// If block time is less than or equal to target time, check if we need to search to the right
		if blockTime.Cmp(targetTime) <= 0 {
			left.Set(mid) // Move left boundary up to mid
		} else {
			right.Sub(mid, big.NewInt(1)) // Move right boundary down
		}

		//calculate the difference between the block time and the target time
		diff := new(big.Int).Sub(blockTime, targetTime)
		if new(big.Int).Abs(diff).Cmp(acceptablediff) <= 0 {
			// log the block number closest to the target time and the time
			logger.Info("block number closest to target time", "block", block)
			//if the difference is less than or equal to 1 hour, we can consider this block as the block closest to the target time
			return block, nil
		}

	}

	return validator.BlockInfo{}, fmt.Errorf("failed to find block at approximate time")
}

func GetStartingBlock(ctx context.Context, cfg CLIConfig, latestL1Height validator.BlockInfo, l1Proxy validator.L1ProxyInterface, logger log.Logger) (validator.BlockInfo, error) {
	hoursInThePastToStartFrom := cfg.HoursInThePastToStartFrom

	// In this case StartingL1BlockHeight is not set
	if cfg.StartingL1BlockHeight == -1 {
		// in this case is not set how many hours in the past to start from, we use default value that is 14 days.
		if hoursInThePastToStartFrom == 0 {
			hoursInThePastToStartFrom = DefaultHoursInThePastToStartFrom
		}

		// get the block number closest to the timestamp from two weeks ago
		startingL1BlockHeightSearch, err := GetBlockAtApproximateTimeBinarySearch(ctx, l1Proxy, latestL1Height, big.NewInt(int64(hoursInThePastToStartFrom)), logger)
		if err != nil {
			return validator.BlockInfo{}, fmt.Errorf("failed to get block at approximate time: %w", err)
		} else {
			return startingL1BlockHeightSearch, nil
		}

	}

	// In this case StartingL1BlockHeight is set
	block, error := l1Proxy.BlockByNumber(big.NewInt(cfg.StartingL1BlockHeight))
	if error != nil {
		return validator.BlockInfo{}, fmt.Errorf("failed to get block by number: %w", error)
	}

	return block, error
}
