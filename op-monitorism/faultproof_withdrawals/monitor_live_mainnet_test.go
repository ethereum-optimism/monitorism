//go:build live
// +build live

package faultproof_withdrawals

import (
	"context"
	"io"
	"math/big"
	"testing"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// NewTestMonitorMainnet initializes and returns a new Monitor instance for testing.
// It sets up the necessary environment variables and configurations required for the monitor.
func NewTestMonitorMainnet() *Monitor {
	envmap, err := godotenv.Read(".env.op.mainnet")
	if err != nil {
		panic("error")
	}

	ctx := context.Background()
	L1GethURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL"]
	L2OpNodeURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL"]
	L2OpGethURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL"]

	FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL := "0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"
	FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE := uint64(1000)
	FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT := int64(6789100)

	cfg := CLIConfig{
		L1GethURL:             L1GethURL,
		L2OpGethURL:           L2OpGethURL,
		L2OpNodeURL:           L2OpNodeURL,
		EventBlockRange:       FAULTPROOF_WITHDRAWAL_MON_EVENT_BLOCK_RANGE,
		StartingL1BlockHeight: FAULTPROOF_WITHDRAWAL_MON_START_BLOCK_HEIGHT,
		OptimismPortalAddress: common.HexToAddress(FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL),
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

	initialBlock := test_monitor.state.nextL1Height
	blockIncrement := test_monitor.maxBlockRange
	finalBlock := initialBlock + blockIncrement

	test_monitor.Run(test_monitor.ctx)

	require.Equal(t, finalBlock, test_monitor.state.nextL1Height)
	require.Equal(t, uint64(0), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(0), test_monitor.state.eventsProcessed)
	require.Equal(t, uint64(0), test_monitor.state.numberOfPotentialAttackOnInProgressGames)
	require.Equal(t, uint64(0), test_monitor.state.numberOfPotentialAttacksOnDefenderWinsGames)
	require.Equal(t, uint64(0), test_monitor.state.numberOfSuspiciousEventsOnChallengerWinsGames)

	require.Equal(t, test_monitor.state.numberOfPotentialAttackOnInProgressGames, uint64(len(test_monitor.state.potentialAttackOnInProgressGames)))
	require.Equal(t, test_monitor.state.numberOfPotentialAttacksOnDefenderWinsGames, uint64(len(test_monitor.state.potentialAttackOnDefenderWinsGames)))
	require.Equal(t, test_monitor.state.numberOfSuspiciousEventsOnChallengerWinsGames, uint64(test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len()))

}

// TestRun5Cycle1000BlocksMainnet tests multiple executions of the monitor's Run method over several cycles.
// It verifies that the state updates correctly after each cycle.
func TestRun5Cycle1000BlocksMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	maxCycle := uint64(5)
	initialBlock := test_monitor.state.nextL1Height
	blockIncrement := test_monitor.maxBlockRange

	for cycle := uint64(1); cycle <= maxCycle; cycle++ {
		test_monitor.Run(test_monitor.ctx)
	}

	initialL1HeightGaugeValue, _ := GetGaugeValue(test_monitor.state.metrics.InitialL1HeightGauge)
	nextL1HeightGaugeValue, _ := GetGaugeValue(test_monitor.state.metrics.NextL1HeightGauge)

	withdrawalsProcessedCounterValue, _ := GetCounterValue(test_monitor.state.metrics.WithdrawalsProcessedCounter)
	eventsProcessedCounterValue, _ := GetCounterValue(test_monitor.state.metrics.EventsProcessedCounter)

	nodeConnectionFailuresCounterValue, _ := GetCounterValue(test_monitor.state.metrics.NodeConnectionFailuresCounter)

	expected_end_block := blockIncrement*maxCycle + initialBlock
	require.Equal(t, uint64(initialBlock), uint64(initialL1HeightGaugeValue))
	require.Equal(t, uint64(expected_end_block), uint64(nextL1HeightGaugeValue))

	require.Equal(t, uint64(0), uint64(eventsProcessedCounterValue))
	require.Equal(t, uint64(0), uint64(withdrawalsProcessedCounterValue))
	require.Equal(t, uint64(0), uint64(nodeConnectionFailuresCounterValue))

	require.Equal(t, uint64(0), test_monitor.state.metrics.previousEventsProcessed)
	require.Equal(t, uint64(0), test_monitor.state.metrics.previousWithdrawalsProcessed)

}

func TestRunSingleBlocksMainnet(t *testing.T) {
	test_monitor := NewTestMonitorMainnet()

	maxCycle := 1
	initialBlock := test_monitor.state.nextL1Height
	blockIncrement := test_monitor.maxBlockRange
	finalBlock := initialBlock + blockIncrement

	for cycle := 1; cycle <= maxCycle; cycle++ {
		test_monitor.Run(test_monitor.ctx)
	}

	require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	require.Equal(t, uint64(0), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(0), test_monitor.state.eventsProcessed)
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())
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

	err = test_monitor.withdrawalValidator.UpdateEnrichedWithdrawalEvent(&event)
	require.NoError(t, err)
	isValid := event.DisputeGame.DisputeGameData.RootClaim == event.ExpectedRootClaim && event.WithdrawalHashPresentOnL2

	require.False(t, isValid)

}
