//go:build live
// +build live

package faultproof_withdrawals

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"testing"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// NewTestMonitorMainnet initializes and returns a new Monitor instance for testing.
// It sets up the necessary environment variables and configurations required for the monitor.
func NewTestMonitorMainnet() *Monitor {
	loadEnv(".env.op.mainnet")
	ctx := context.Background()
	L1GethURL := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL")
	L2OpNodeURL := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL")
	L2OpGethURL := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL")
	EventBlockRangeStr := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE")
	EventBlockRange, err := strconv.ParseUint(EventBlockRangeStr, 10, 64)
	require.NoError(err)

	StartingL1BlockHeightStr := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT")
	StartingL1BlockHeight, err := strconv.ParseInt(StartingL1BlockHeightStr, 10, 64)
	require.NoError(err)

	cfg := CLIConfig{
		L1GethURL:             L1GethURL,
		L2OpGethURL:           L2OpGethURL,
		L2OpNodeURL:           L2OpNodeURL,
		EventBlockRange:       EventBlockRange,
		StartingL1BlockHeight: StartingL1BlockHeight,
		OptimismPortalAddress: common.HexToAddress(os.Getenv("FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL")),
	}

	clicfg := oplog.DefaultCLIConfig()
	output_writer := io.Discard // discard log output during tests to avoid pollution of the standard output
	log := oplog.NewLogger(output_writer, clicfg)

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := NewMonitor(ctx, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		panic(err)
	}
	return monitor
}

// TestSingleRunMainnet tests a single execution of the monitor's Run method.
// It verifies that the state updates correctly after running.
func TestSingleRunMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	initialBlock := uint64(20872390) // this block is known to have events with errors
	blockIncrement := uint64(1000)
	// finalBlock := initialBlock + blockIncrement

	test_monitor.state.nextL1Height = initialBlock
	test_monitor.maxBlockRange = blockIncrement
	test_monitor.Run(test_monitor.ctx)
	fmt.Printf("State: %+v\n", test_monitor.state)

	require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestRun30Cycle1000BlocksMainnet tests multiple executions of the monitor's Run method over several cycles.
// It verifies that the state updates correctly after each cycle.
func TestRun30Cycle1000BlocksMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	maxCycle := 30
	initialBlock := uint64(20872390) // this block is known to have events with errors
	blockIncrement := uint64(1000)
	// finalBlock := initialBlock + blockIncrement

	test_monitor.state.nextL1Height = initialBlock
	test_monitor.maxBlockRange = blockIncrement

	for cycle := 1; cycle <= maxCycle; cycle++ {
		fmt.Println("-----------")
		fmt.Printf("Cycle: %d\n", cycle)

		test_monitor.Run(test_monitor.ctx)
		fmt.Println("************")
		fmt.Printf("State: %v\n", test_monitor.state)
		fmt.Printf("Metrics: %v\n", &test_monitor.metrics)
		fmt.Println("###########")

	}
}

func TestRunSingleBlocksMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	maxCycle := 1
	initialBlock := uint64(20873192) // this block is known to have events with errors
	blockIncrement := uint64(1)
	finalBlock := initialBlock + blockIncrement

	test_monitor.state.nextL1Height = initialBlock
	test_monitor.maxBlockRange = blockIncrement

	for cycle := 1; cycle <= maxCycle; cycle++ {
		fmt.Println("-----------")
		fmt.Printf("Cycle: %d\n", cycle)

		test_monitor.Run(test_monitor.ctx)
		fmt.Println("************")
		fmt.Printf("State: %v\n", test_monitor.state)
		fmt.Printf("Metrics: %v\n", &test_monitor.metrics)
		fmt.Println("###########")
	}

	require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	require.Equal(t, test_monitor.state.withdrawalsProcessed, uint64(1))
	require.Equal(t, test_monitor.state.eventsProcessed, uint64(1))
	require.Equal(t, test_monitor.state.numberOfPotentialAttackOnDefenderWinsGames, uint64(0))
	require.Equal(t, len(test_monitor.state.potentialAttackOnDefenderWinsGames), 0)
	require.Equal(t, len(test_monitor.state.potentialAttackOnInProgressGames), 0)
}

func TestInvalidWithdrawalsOnMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	// On mainnet for OP OptimismPortal, the block number 20873192 is known to have only 1 event
	start := uint64(20873192)
	stop := uint64(20873193)
	newEvents, err := test_monitor.withdrawalValidator.GetEnrichedWithdrawalsEvents(start, &stop)
	require.NoError(t, err)
	require.Equal(t, len(newEvents), 1)

	event := newEvents[0]
	require.NotNil(t, event)

	// Expected event:
	//{WithdrawalHash: 0x45fd4bbcf3386b1fdf75929345b9243c05cd7431a707e84c293b710d40220ebd, ProofSubmitter: 0x394400571C825Da37ca4D6780417DFB514141b1f}
	require.Equal(t, event.Event.WithdrawalHash, [32]byte(common.HexToHash("0x45fd4bbcf3386b1fdf75929345b9243c05cd7431a707e84c293b710d40220ebd")))
	require.Equal(t, event.Event.ProofSubmitter, common.HexToAddress("0x394400571C825Da37ca4D6780417DFB514141b1f"))

	//Expected DisputeGameData:
	// Game address: 0x52cE243d552369b11D6445Cd187F6393d3B42D4a
	require.Equal(t, event.DisputeGame.DisputeGameData.ProxyAddress, common.HexToAddress("0x52cE243d552369b11D6445Cd187F6393d3B42D4a"))

	// Expected Game root claim
	// 0xbc1c5ba13b936c6c23b7c51d425f25a8c9444771e851b6790f817a6002a14a33
	require.Equal(t, event.DisputeGame.DisputeGameData.RootClaim, [32]byte(common.HexToHash("0xbc1c5ba13b936c6c23b7c51d425f25a8c9444771e851b6790f817a6002a14a33")))

	// Expected L2 block number 1276288764
	require.Equal(t, event.DisputeGame.DisputeGameData.L2blockNumber, big.NewInt(1276288764))

	isValid, err := test_monitor.withdrawalValidator.IsWithdrawalEventValid(&event)
	require.EqualError(t, err, "trustedRootClaim is nil, game not enriched")
	fmt.Printf("isValid: %+v\n", isValid)
	fmt.Printf("event: %+v\n", event)
	err = test_monitor.withdrawalValidator.UpdateEnrichedWithdrawalEvent(&event)
	fmt.Printf("event: %+v\n", event)
	fmt.Printf("err: %+v\n", err)

	require.NoError(t, err)

}
