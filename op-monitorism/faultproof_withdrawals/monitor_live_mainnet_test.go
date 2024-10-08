//go:build live
// +build live

package faultproof_withdrawals

import (
	"context"
	"fmt"
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
	if err != nil {
		panic(err)
	}
	StartingL1BlockHeightStr := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT")
	StartingL1BlockHeight, err := strconv.ParseInt(StartingL1BlockHeightStr, 10, 64)
	if err != nil {
		panic(err)
	}

	cfg := CLIConfig{
		L1GethURL:             L1GethURL,
		L2OpGethURL:           L2OpGethURL,
		L2OpNodeURL:           L2OpNodeURL,
		EventBlockRange:       EventBlockRange,
		StartingL1BlockHeight: StartingL1BlockHeight,
		OptimismPortalAddress: common.HexToAddress(os.Getenv("FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL")),
	}

	clicfg := oplog.DefaultCLIConfig()
	// output_writer := io.Discard // discard log output during tests to avoid pollution of the standard output
	output_writer := os.Stdout
	log := oplog.NewLogger(output_writer, clicfg)

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := NewMonitor(ctx, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		panic(err)
	}
	return monitor
}

// TestSingleRuMainnet tests a single execution of the monitor's Run method.
// It verifies that the state updates correctly after running.
func TestSingleRuMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	initialBlock := uint64(20872390) // this block is known to have events with errors
	blockIncrement := uint64(1000)
	// finalBlock := initialBlock + blockIncrement

	test_monitor.state.nextL1Height = initialBlock
	test_monitor.maxBlockRange = blockIncrement
	test_monitor.Run(test_monitor.ctx)
	fmt.Printf("State: %+v\n", test_monitor.state)

	// require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	// require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	// require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	// require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	// require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	// require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestSingleRuMainnet tests a single execution of the monitor's Run method.
// It verifies that the state updates correctly after running.
func TestRun30Cycle1000BlocksMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	maxCycle := 1
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
	// require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	// require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	// require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	// require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	// require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	// require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// Cycle: 1
// t=2024-10-03T12:52:11+0200 lvl=info msg="processing withdrawal event" event="WithdrawalHash: 0x307834356664346262636633333836623166646637353932393334356239323433633035636437343331613730376538346332393362373130643430323230656264, ProofSubmitter: 0x394400571C825Da37ca4D6780417DFB514141b1f, Raw: {BlockNumber: 20873192, TxHash: 0x10b4e130eca6ad9466e35ce04a76e7e273456109820c6581993596882c4cdaee}"
// t=2024-10-03T12:52:11+0200 lvl=warn msg="WITHDRAWAL: is NOT valid, game is still in progress." enrichedWithdrawalEvent="{Event:WithdrawalHash: 0x307834356664346262636633333836623166646637353932393334356239323433633035636437343331613730376538346332393362373130643430323230656264, ProofSubmitter: 0x394400571C825Da37ca4D6780417DFB514141b1f, Raw: {BlockNumber: 20873192, TxHash: 0x10b4e130eca6ad9466e35ce04a76e7e273456109820c6581993596882c4cdaee} DisputeGame:FaultDisputeGameProxy[ DisputeGameData=DisputeGame[ disputeGameProxyAddress: 0x680ACF0B7B97F10A6AF0e291c03717e4e532E511 rootClaim: 0xe1e1cba54480c557f68e3781f90ac52fa023ffcbd56c69665e861567367c08a9 l2blockNumber: 126104291 l2ChainID: 10 status: IN_PROGRESS createdAt: 2024-10-01 20:49:47 CEST resolvedAt: 1970-01-01 01:00:00 CET ] ] ExpectedRootClaim:0xe1e1cba54480c557f68e3781f90ac52fa023ffcbd56c69665e861567367c08a9 Blacklisted:false WithdrawalHashPresentOnL2:false Enriched:true}"
// t=2024-10-03T12:52:11+0200 lvl=error msg="failed to update enriched withdrawal event" error="failed to get trustedRootClaim from Op-node: failed to get output at block for game block:0x4c129efc : failed to get L2 block ref with sync status: failed to determine L2BlockRef of height 1276288764, could not get payload: not found"
// t=2024-10-03T12:52:11+0200 lvl=error msg="failed to consume events" error="failed to get trustedRootClaim from Op-node: failed to get output at block for game block:0x4c129efc : failed to get L2 block ref with sync status: failed to determine L2BlockRef of height 1276288764, could not get payload: not found"
// ************

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
	require.Error(t, err, "trustedRootClaim is nil, game not enriched")
	fmt.Printf("isValid: %+v\n", isValid)
	fmt.Printf("event: %+v\n", event)
	err = test_monitor.withdrawalValidator.UpdateEnrichedWithdrawalEvent(&event)
	fmt.Printf("event: %+v\n", event)
	fmt.Printf("err: %+v\n", err)

	require.NoError(t, err)

}
