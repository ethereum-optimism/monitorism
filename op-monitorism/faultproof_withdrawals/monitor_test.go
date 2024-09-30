//go:build live
// +build live

package faultproof_withdrawals

import (
	"context"
	"io"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

// TestMain runs the tests in the package and exits with the appropriate exit code.
func TestMain(m *testing.M) {
	exitVal := m.Run()
	os.Exit(exitVal)
}

// loadEnv loads environment variables from the specified .env file.
func loadEnv(env string) error {
	return godotenv.Load(env)
}

// NewTestMonitor initializes and returns a new Monitor instance for testing.
// It sets up the necessary environment variables and configurations required for the monitor.
func NewTestMonitor() *Monitor {
	loadEnv(".env.op.sepolia")
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
	StartingL1BlockHeight, err := strconv.ParseUint(StartingL1BlockHeightStr, 10, 64)
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
	output_writer := io.Discard // discard log output during tests to avoid pollution of the standard output
	log := oplog.NewLogger(output_writer, clicfg)

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := NewMonitor(ctx, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		panic(err)
	}
	return monitor
}

// TestSingleRun tests a single execution of the monitor's Run method.
// It verifies that the state updates correctly after running.
func TestSingleRun(t *testing.T) {
	test_monitor := NewTestMonitor()

	initialBlock := uint64(5914813)
	blockIncrement := uint64(1000)
	finalBlock := initialBlock + blockIncrement

	test_monitor.state.nextL1Height = initialBlock
	test_monitor.maxBlockRange = blockIncrement
	test_monitor.Run(test_monitor.ctx)

	require.Equal(t, test_monitor.state.nextL1Height, finalBlock)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEvents tests the consumption of enriched withdrawal events.
// It verifies that new events can be processed correctly.
func TestConsumeEvents(t *testing.T) {
	test_monitor := NewTestMonitor()

	initialBlock := uint64(5914813)
	blockIncrement := uint64(1000)
	finalBlock := initialBlock + blockIncrement

	newEvents, err := test_monitor.withdrawalValidator.GetEnrichedWithdrawalsEvents(initialBlock, &finalBlock)
	require.NoError(t, err)
	require.NotEqual(t, len(newEvents), 0)

	newInvalidProposalWithdrawalsEvents, err := test_monitor.ConsumeEvents(newEvents)
	require.NoError(t, err)
	require.Equal(t, len(*newInvalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEventValid_DEFENDER_WINS tests the consumption of a valid event where the defender wins.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_DEFENDER_WINS(t *testing.T) {
	test_monitor := NewTestMonitor()

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

	consumedEvent, err := test_monitor.ConsumeEvent(validEvent)
	require.NoError(t, err)
	require.True(t, consumedEvent)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEventValid_CHALLENGER_WINS tests the consumption of a valid event where the challenger wins.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_CHALLENGER_WINS(t *testing.T) {
	test_monitor := NewTestMonitor()

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

	consumedEvent, err := test_monitor.ConsumeEvent(event)
	require.NoError(t, err)
	require.True(t, consumedEvent)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEventValid_Blacklisted tests the consumption of a valid event that is blacklisted.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventValid_Blacklisted(t *testing.T) {
	test_monitor := NewTestMonitor()

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

	consumedEvent, err := test_monitor.ConsumeEvent(event)
	require.NoError(t, err)
	require.True(t, consumedEvent)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(1))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(0))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 0)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEventForgery1 tests the consumption of an event that indicates a forgery.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventForgery1(t *testing.T) {
	test_monitor := NewTestMonitor()

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

	consumedEvent, err := test_monitor.ConsumeEvent(validEvent)
	require.NoError(t, err)
	require.True(t, consumedEvent)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(0))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(1))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 1)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}

// TestConsumeEventForgery2 tests the consumption of another event that indicates a forgery.
// It checks that the state updates correctly after processing the event.
func TestConsumeEventForgery2(t *testing.T) {
	test_monitor := NewTestMonitor()

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

	consumedEvent, err := test_monitor.ConsumeEvent(event)
	require.NoError(t, err)
	require.True(t, consumedEvent)
	require.Equal(t, test_monitor.state.withdrawalsValidated, uint64(0))
	require.Equal(t, test_monitor.state.processedProvenWithdrawalsExtension1Events, uint64(1))
	require.Equal(t, test_monitor.state.numberOfDetectedForgery, uint64(1))
	require.Equal(t, len(test_monitor.state.forgeriesWithdrawalsEvents), 1)
	require.Equal(t, len(test_monitor.state.invalidProposalWithdrawalsEvents), 0)
}
