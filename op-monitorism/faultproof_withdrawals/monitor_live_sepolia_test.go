//go:build live
// +build live

package faultproof_withdrawals

import (
	"context"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// NewTestMonitorSepolia initializes and returns a new Monitor instance for testing.
// It sets up the necessary environment variables and configurations required for the monitor.
func NewTestMonitorSepolia() *Monitor {
	envmap, err := godotenv.Read(".env.op.sepolia")
	if err != nil {
		panic("error")
	}

	ctx := context.Background()
	L1GethURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL"]
	L2OpNodeURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL"]
	L2OpGethURL := envmap["FAULTPROOF_WITHDRAWAL_MON_L2_OP_GETH_URL"]

	FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL := "0x16Fc5058F25648194471939df75CF27A2fdC48BC"
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

// TestSingleRunSepolia tests a single execution of the monitor's Run method.
// It verifies that the state updates correctly after running.
func TestSingleRunSepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	initialBlock := test_monitor.state.nextL1Height
	blockIncrement := test_monitor.maxBlockRange
	finalBlock := initialBlock + blockIncrement

	test_monitor.Run(context.Background())

	require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventsSepolia tests the consumption of enriched withdrawal events.
// It verifies that new events can be processed correctly.
func TestConsumeEventsSepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	initialBlock := test_monitor.state.nextL1Height
	blockIncrement := test_monitor.maxBlockRange
	finalBlock := initialBlock + blockIncrement

	newEvents, err := test_monitor.withdrawalValidator.GetEnrichedWithdrawalsEventsMap(initialBlock, &finalBlock)
	require.NoError(t, err)
	require.NotEqual(t, 0, len(newEvents))

	err = test_monitor.ConsumeEvents(newEvents)
	require.NoError(t, err)
}

// TestConsumeEventValid_DEFENDER_WINS_Sepolia tests the consumption of a valid event where the defender wins.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_DEFENDER_WINS_Sepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	expectedRootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")

	validEvent := validator.EnrichedProvenWithdrawalEvent{
		ExpectedRootClaim: expectedRootClaim,
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     expectedRootClaim,
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        validator.DEFENDER_WINS,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		WithdrawalHashPresentOnL2: true,
		Blacklisted:               false,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: func() [32]byte {
				var arr [32]byte
				copy(arr[:], common.Hex2Bytes("edbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"))
				return arr
			}(),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}

	eventsMap := map[common.Hash]validator.EnrichedProvenWithdrawalEvent{
		validEvent.Event.WithdrawalHash: validEvent,
	}
	err := test_monitor.ConsumeEvents(eventsMap)
	require.NoError(t, err)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())

}

// TestConsumeEventValid_CHALLENGER_WINS_Sepolia tests the consumption of a valid event where the challenger wins.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_CHALLENGER_WINS_Sepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	expectedRootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")
	rootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7596") // different root claim, last number is 6 instead of 7

	event := validator.EnrichedProvenWithdrawalEvent{
		ExpectedRootClaim: expectedRootClaim,
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     rootClaim,
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        validator.CHALLENGER_WINS,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		WithdrawalHashPresentOnL2: true,
		Blacklisted:               false,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: func() [32]byte {
				var arr [32]byte
				copy(arr[:], common.Hex2Bytes("edbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"))
				return arr
			}(),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}

	eventsMap := map[common.Hash]validator.EnrichedProvenWithdrawalEvent{
		event.Event.WithdrawalHash: event,
	}
	err := test_monitor.ConsumeEvents(eventsMap)
	require.NoError(t, err)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 1, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())

}

// TestConsumeEventValid_BlacklistedSepolia tests the consumption of a valid event that is blacklisted.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_BlacklistedSepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	expectedRootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")
	rootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7596") // different root claim, last number is 6 instead of 7

	event := validator.EnrichedProvenWithdrawalEvent{
		ExpectedRootClaim: expectedRootClaim,
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     rootClaim,
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        validator.DEFENDER_WINS,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		WithdrawalHashPresentOnL2: true,
		Blacklisted:               true,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: func() [32]byte {
				var arr [32]byte
				copy(arr[:], common.Hex2Bytes("edbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"))
				return arr
			}(),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}

	eventsMap := map[common.Hash]validator.EnrichedProvenWithdrawalEvent{
		event.Event.WithdrawalHash: event,
	}
	err := test_monitor.ConsumeEvents(eventsMap)
	require.NoError(t, err)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 1, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())

}

// TestConsumeEventForgery1Sepolia tests the consumption of an event that indicates a forgery.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventForgery1Sepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	expectedRootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")

	validEvent := validator.EnrichedProvenWithdrawalEvent{
		ExpectedRootClaim: expectedRootClaim,
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     expectedRootClaim,
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        validator.DEFENDER_WINS,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		WithdrawalHashPresentOnL2: false, // this is the forgery
		Blacklisted:               false,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: func() [32]byte {
				var arr [32]byte
				copy(arr[:], common.Hex2Bytes("edbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"))
				return arr
			}(),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}

	eventsMap := map[common.Hash]validator.EnrichedProvenWithdrawalEvent{
		validEvent.Event.WithdrawalHash: validEvent,
	}
	err := test_monitor.ConsumeEvents(eventsMap)
	require.NoError(t, err)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventForgery2Sepolia tests the consumption of another event that indicates a forgery.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventForgery2Sepolia(t *testing.T) {
	test_monitor := NewTestMonitorSepolia()

	expectedRootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")
	rootClaim := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7596") // different root claim, last number is 6 instead of 7

	event := validator.EnrichedProvenWithdrawalEvent{
		ExpectedRootClaim: expectedRootClaim,
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     rootClaim,
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        validator.DEFENDER_WINS,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		WithdrawalHashPresentOnL2: true,
		Blacklisted:               false,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: func() [32]byte {
				var arr [32]byte
				copy(arr[:], common.Hex2Bytes("edbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"))
				return arr
			}(),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}

	eventsMap := map[common.Hash]validator.EnrichedProvenWithdrawalEvent{
		event.Event.WithdrawalHash: event,
	}
	err := test_monitor.ConsumeEvents(eventsMap)
	require.NoError(t, err)
	require.Equal(t, uint64(1), test_monitor.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), test_monitor.state.eventsProcessed)
	require.Equal(t, 1, len(test_monitor.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(test_monitor.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, test_monitor.state.suspiciousEventsOnChallengerWinsGames.Len())

}
