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

// newEnrichedEvent builds an already-enriched withdrawal event so that
// ConsumeEvent's categorization can be exercised deterministically, without
// depending on live enrichment against historical (possibly pruned or
// pre-Isthmus) L2 state.
func newEnrichedEvent(status validator.GameStatus, trusted, blacklisted, preIsthmus bool) *validator.EnrichedProvenWithdrawalEvent {
	return &validator.EnrichedProvenWithdrawalEvent{
		DisputeGame: &validator.FaultDisputeGameProxy{
			DisputeGameData: &validator.DisputeGameData{
				ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
				RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
				L2blockNumber: big.NewInt(12030787),
				L2ChainID:     big.NewInt(11155420),
				Status:        status,
				CreatedAt:     1730000000,
				ResolvedAt:    1730000000,
			},
			FaultDisputeGame: nil,
		},
		DisputeGameRootClaimIsTrusted: trusted,
		WithdrawalHashPresentOnL2:     trusted,
		PreIsthmusUnverifiable:        preIsthmus,
		Blacklisted:                   blacklisted,
		Enriched:                      true,
		Event: &validator.WithdrawalProvenExtension1Event{
			WithdrawalHash: common.HexToHash("0xedbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"),
			ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
			Raw: validator.Raw{
				BlockNumber: 5915676,
				TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
			},
		},
	}
}

// TestConsumeEventValid_DEFENDER_WINS_Sepolia: a canonical root claim on a
// resolved game is a valid withdrawal.
func TestConsumeEventValid_DEFENDER_WINS_Sepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.DEFENDER_WINS, true, false, false))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 0, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, m.state.suspiciousEventsOnChallengerWinsGames.Len())
	require.Equal(t, uint64(0), m.state.numberOfPreIsthmusUnverifiable)
}

// TestConsumeEventForgeryDefenderWinsSepolia: a non-canonical root claim on a
// resolved DEFENDER_WINS game is a forgery on a resolved game.
func TestConsumeEventForgeryDefenderWinsSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.DEFENDER_WINS, false, false, false))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 1, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventForgeryInProgressSepolia: a non-canonical root claim on a game
// still in progress is a potential attack pending fault-proof resolution.
func TestConsumeEventForgeryInProgressSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.IN_PROGRESS, false, false, false))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 0, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 1, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventChallengerWinsSepolia: a withdrawal proven against a game that
// resolved CHALLENGER_WINS is suspicious but not a resolved-game forgery.
func TestConsumeEventChallengerWinsSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.CHALLENGER_WINS, false, false, false))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.withdrawalsProcessed)
	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 0, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 1, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventBlacklistedSepolia: an invalid withdrawal on a blacklisted game
// is routed to the suspicious bucket, not the resolved-game forgery bucket.
func TestConsumeEventBlacklistedSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.DEFENDER_WINS, false, true, false))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 0, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 1, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventPreIsthmusSepolia: a withdrawal against a pre-Isthmus L2 block
// cannot be header-verified and is flagged for security triage — never counted
// as valid nor as a forgery.
func TestConsumeEventPreIsthmusSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	err := m.ConsumeEvent(newEnrichedEvent(validator.DEFENDER_WINS, false, false, true))
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, uint64(1), m.state.numberOfPreIsthmusUnverifiable)
	require.Equal(t, 0, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}

// TestConsumeEventTrustedButAbsentSepolia: a canonical root claim whose withdrawal
// is NOT present in the message passer at head (the Portal-inclusion-bug / fabricated
// withdrawal case). Presence is an independent condition, so this is a forgery on a
// resolved game even though the root claim is trusted.
func TestConsumeEventTrustedButAbsentSepolia(t *testing.T) {
	m := NewTestMonitorSepolia()

	event := newEnrichedEvent(validator.DEFENDER_WINS, true, false, false)
	event.WithdrawalHashPresentOnL2 = false // canonical root, but withdrawal absent at head

	err := m.ConsumeEvent(event)
	require.NoError(t, err)

	require.Equal(t, uint64(1), m.state.eventsProcessed)
	require.Equal(t, 1, len(m.state.potentialAttackOnDefenderWinsGames))
	require.Equal(t, 0, len(m.state.potentialAttackOnInProgressGames))
	require.Equal(t, 0, m.state.suspiciousEventsOnChallengerWinsGames.Len())
}
