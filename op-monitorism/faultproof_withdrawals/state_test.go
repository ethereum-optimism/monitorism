package faultproof_withdrawals

import (
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
	"github.com/ethereum/go-ethereum/common"
	oplog "github.com/ethereum/go-ethereum/log"
	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/require"
)

// newTestState creates a minimal State instance with the internal maps / cache
// initialised so that the increment-helpers can run without panicking.
func newTestState() *State {
	cache, _ := lru.New(10)

	return &State{
		logger:                                oplog.New(),
		potentialAttackOnDefenderWinsGames:    make(map[common.Hash]*validator.EnrichedProvenWithdrawalEvent),
		potentialAttackOnInProgressGames:      make(map[common.Hash]*validator.EnrichedProvenWithdrawalEvent),
		suspiciousEventsOnChallengerWinsGames: cache,
	}
}

// newTestEvent builds a synthetic EnrichedProvenWithdrawalEvent whose
// TxHash is derived from the supplied hex string.
func newTestEvent(txHashHex string) *validator.EnrichedProvenWithdrawalEvent {
	txHash := common.HexToHash(txHashHex)
	withdrawalEvent := &validator.WithdrawalProvenExtension1Event{
		Raw: validator.Raw{
			BlockNumber: 1,
			TxHash:      txHash,
		},
	}

	return &validator.EnrichedProvenWithdrawalEvent{
		Event: withdrawalEvent,
	}
}

func TestIncrementWithdrawalsValidated(t *testing.T) {
	st := newTestState()
	evt := newTestEvent("0x1")

	st.IncrementWithdrawalsValidated(evt)

	require.Equal(t, uint64(1), st.withdrawalsProcessed)
	require.NotZero(t, evt.ProcessedTimeStamp)
}

func TestIncrementPotentialAttackOnDefenderWinsGames(t *testing.T) {
	st := newTestState()
	evt := newTestEvent("0x2")

	st.IncrementPotentialAttackOnDefenderWinsGames(evt)

	require.Equal(t, uint64(1), st.numberOfPotentialAttacksOnDefenderWinsGames)
	require.Equal(t, 1, len(st.potentialAttackOnDefenderWinsGames))
	require.Equal(t, uint64(1), st.withdrawalsProcessed)
	require.NotZero(t, evt.ProcessedTimeStamp)
}

func TestIncrementPotentialAttackOnInProgressGames(t *testing.T) {
	st := newTestState()
	evt := newTestEvent("0x3")

	st.IncrementPotentialAttackOnInProgressGames(evt)

	require.Equal(t, uint64(1), st.numberOfPotentialAttackOnInProgressGames)
	require.Equal(t, 1, len(st.potentialAttackOnInProgressGames))
	require.Equal(t, uint64(0), st.withdrawalsProcessed)
	require.NotZero(t, evt.ProcessedTimeStamp)
}

func TestIncrementSuspiciousEventsOnChallengerWinsGames(t *testing.T) {
	st := newTestState()
	evt := newTestEvent("0x4")

	// First mark the event as in-progress to exercise the removal path.
	st.IncrementPotentialAttackOnInProgressGames(evt)

	require.Equal(t, uint64(1), st.numberOfPotentialAttackOnInProgressGames, "setup failed: expected 1 in-progress event")

	st.IncrementSuspiciousEventsOnChallengerWinsGames(evt)

	require.Equal(t, uint64(1), st.numberOfSuspiciousEventsOnChallengerWinsGames)
	require.Equal(t, 1, st.suspiciousEventsOnChallengerWinsGames.Len())
	require.Equal(t, uint64(0), st.numberOfPotentialAttackOnInProgressGames, "expected 0 after removal")
	require.Equal(t, uint64(1), st.withdrawalsProcessed)
	require.NotZero(t, evt.ProcessedTimeStamp)
}
