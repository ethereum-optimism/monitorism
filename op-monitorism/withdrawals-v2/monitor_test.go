package withdrawalsv2

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals-v2/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedactURL proves the review comment #6 fix: the startup log shows only the
// ends of the RPC URL, never the embedded credential/token in the middle.
func TestRedactURL(t *testing.T) {
	secret := "https://user:SUPERSECRETTOKEN@ci-mainnet-l1-archive.optimism.io/rpc?key=abcdef123456"
	got := redactURL(secret)
	assert.NotContains(t, got, "SUPERSECRETTOKEN")
	assert.NotContains(t, got, "abcdef123456")
	assert.True(t, strings.HasPrefix(got, "https://user"))
	assert.Contains(t, got, "...")

	// Short strings reveal nothing.
	assert.Equal(t, "***", redactURL("http://short"))
	assert.Equal(t, "***", redactURL(""))
}

// TestPendingStore proves the review comment #4 fix: unresolved events are parked
// and only released on a terminal verdict, and the tracking gauges reflect the
// backlog. enqueue is idempotent (one entry per event, counter-friendly).
func TestPendingStore(t *testing.T) {
	m := newTestMonitor(t)
	m.pending = make(map[string]*pendingEvent)
	m.metrics.pending = prometheus.NewGauge(prometheus.GaugeOpts{Name: "pending"})
	m.metrics.oldestPendingSeconds = prometheus.NewGauge(prometheus.GaugeOpts{Name: "oldest"})

	logA := types.Log{TxHash: common.HexToHash("0xaa"), Index: 0}
	logB := types.Log{TxHash: common.HexToHash("0xaa"), Index: 1} // same tx, different log

	assert.True(t, m.enqueuePending(logA, [32]byte{0x1}), "first enqueue is new")
	assert.False(t, m.enqueuePending(logA, [32]byte{0x1}), "re-enqueue of same event is not new")
	assert.True(t, m.enqueuePending(logB, [32]byte{0x2}), "distinct log position is a distinct event")
	assert.Equal(t, float64(2), testutil.ToFloat64(m.metrics.pending))

	m.updatePendingGauges()
	assert.GreaterOrEqual(t, testutil.ToFloat64(m.metrics.oldestPendingSeconds), float64(0))

	m.resolvePending(logA)
	assert.Equal(t, float64(1), testutil.ToFloat64(m.metrics.pending))
	m.resolvePending(logB)
	assert.Equal(t, float64(0), testutil.ToFloat64(m.metrics.pending))

	m.updatePendingGauges()
	assert.Equal(t, float64(0), testutil.ToFloat64(m.metrics.oldestPendingSeconds), "no pending -> oldest age resets to 0")
}

func TestComputeWithdrawalStorageKey(t *testing.T) {
	tests := []struct {
		name           string
		withdrawalHash [32]byte
		expectedKey    common.Hash
	}{
		{
			name:           "zero hash",
			withdrawalHash: [32]byte{},
			expectedKey:    crypto.Keccak256Hash(append(make([]byte, 32), make([]byte, 32)...)),
		},
		{
			name:           "non-zero hash",
			withdrawalHash: [32]byte{0x01, 0x02, 0x03},
			expectedKey:    crypto.Keccak256Hash(append([]byte{0x01, 0x02, 0x03}, append(make([]byte, 29), make([]byte, 32)...)...)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedKey, computeWithdrawalStorageKey(tt.withdrawalHash))
		})
	}
}

// testProof returns a decodedProof whose outputRootProof hashes to the returned
// root claim (so check (a) passes), letting tests exercise check (b) in isolation.
func testProof(withdrawalProof [][]byte) (*decodedProof, [32]byte) {
	orp := bindings.TypesOutputRootProof{
		Version:                  [32]byte{},
		StateRoot:                [32]byte{0x11},
		MessagePasserStorageRoot: [32]byte{0x22},
		LatestBlockhash:          [32]byte{0x33},
	}
	rootClaim := crypto.Keccak256Hash(
		orp.Version[:], orp.StateRoot[:], orp.MessagePasserStorageRoot[:], orp.LatestBlockhash[:],
	)
	return &decodedProof{outputRootProof: orp, withdrawalProof: withdrawalProof}, [32]byte(rootClaim)
}

func TestVerifyProof_BadOutputRootBinding(t *testing.T) {
	// The output-root proof does NOT hash to the game's root claim: the portal
	// accepted a proof bound to a different output root -> P0.
	dp, _ := testProof(nil)
	mismatched := [32]byte{0xde, 0xad, 0xbe, 0xef}
	reason := verifyProof([32]byte{0x01}, mismatched, dp)
	assert.Equal(t, reasonBadOutputRootBinding, reason)
}

func TestVerifyProof_BadWithdrawalProof(t *testing.T) {
	// Check (a) passes (root claim matches) but the storage proof is garbage and
	// cannot prove inclusion of the withdrawal -> P0.
	dp, rootClaim := testProof([][]byte{{0xde, 0xad}, {0xbe, 0xef}})
	reason := verifyProof([32]byte{0x01}, rootClaim, dp)
	assert.Equal(t, reasonBadWithdrawalProof, reason)
}

func TestVerifyProof_EmptyProof(t *testing.T) {
	dp, rootClaim := testProof(nil)
	reason := verifyProof([32]byte{0x01}, rootClaim, dp)
	assert.Equal(t, reasonBadWithdrawalProof, reason)
}

func proveABI(t *testing.T) *abi.ABI {
	t.Helper()
	a, err := bindings.OptimismPortal2MetaData.GetAbi()
	require.NoError(t, err)
	return a
}

func newTestMonitor(t *testing.T) *Monitor {
	t.Helper()
	a := proveABI(t)
	var sel [4]byte
	copy(sel[:], a.Methods[proveMethodName].ID)
	wdTxArgs, err := newWithdrawalTxArgs()
	require.NoError(t, err)
	return &Monitor{
		portalAddress: common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
		portalABI:     a,
		proveSelector: sel,
		wdTxArgs:      wdTxArgs,
	}
}

// packProve builds calldata for a proveWithdrawalTransaction call.
func packProve(t *testing.T, orp bindings.TypesOutputRootProof, proof [][]byte) []byte {
	t.Helper()
	a := proveABI(t)
	tx := bindings.TypesWithdrawalTransaction{
		Nonce:    big.NewInt(1),
		Sender:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Target:   common.HexToAddress("0x2222222222222222222222222222222222222222"),
		Value:    big.NewInt(0),
		GasLimit: big.NewInt(100000),
		Data:     []byte{},
	}
	packed, err := a.Pack(proveMethodName, tx, big.NewInt(7), orp, proof)
	require.NoError(t, err)
	return packed
}

func TestDecodeProveInput_RoundTrip(t *testing.T) {
	m := newTestMonitor(t)
	orp := bindings.TypesOutputRootProof{
		Version:                  [32]byte{0x00},
		StateRoot:                [32]byte{0xaa},
		MessagePasserStorageRoot: [32]byte{0xbb},
		LatestBlockhash:          [32]byte{0xcc},
	}
	proof := [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}}
	input := packProve(t, orp, proof)

	dp, err := m.decodeProveInput(input)
	require.NoError(t, err)
	assert.Equal(t, orp, dp.outputRootProof)
	assert.Equal(t, proof, dp.withdrawalProof)
}

func TestCollectProveInputs_NestedWrapper(t *testing.T) {
	m := newTestMonitor(t)
	input := packProve(t, bindings.TypesOutputRootProof{}, [][]byte{{0x01}})

	// A wrapper contract calls the portal from an internal frame; the top-level
	// call is to some other contract.
	frame := &callFrame{
		Type:  "CALL",
		To:    "0x43edb88c4b80fdd2adff2412a7bebf9df42cb40e", // wrapper
		Input: "0xabcdef",
		Calls: []callFrame{
			{
				Type:  "CALL",
				To:    m.portalAddress.Hex(),
				Input: common.Bytes2Hex(input), // no 0x prefix, FromHex handles both
			},
		},
	}
	var got [][]byte
	m.collectProveInputs(frame, &got)
	require.Len(t, got, 1)
	assert.Equal(t, input, got[0])
}

func TestCollectProveInputs_WrongSelectorIgnored(t *testing.T) {
	m := newTestMonitor(t)
	// A call to the portal but with a different selector (e.g. finalize) must not match.
	frame := &callFrame{
		Type:  "CALL",
		To:    m.portalAddress.Hex(),
		Input: "0xdeadbeef" + "00",
	}
	var got [][]byte
	m.collectProveInputs(frame, &got)
	assert.Empty(t, got)
}

func TestCollectProveInputs_NonPortalIgnored(t *testing.T) {
	m := newTestMonitor(t)
	input := packProve(t, bindings.TypesOutputRootProof{}, [][]byte{{0x01}})
	// Correct selector but to a non-portal address -> ignored.
	frame := &callFrame{
		Type:  "CALL",
		To:    "0x9999999999999999999999999999999999999999",
		Input: "0x" + common.Bytes2Hex(input),
	}
	var got [][]byte
	m.collectProveInputs(frame, &got)
	assert.Empty(t, got)
}

// TestIsSuperGame proves the review comment #3 fix: super game types are routed
// to rootClaimByChainId and legacy types to rootClaim, exactly as the portal's
// GameTypes.isSuperGame decides. Kept in sync with src/dispute/lib/Types.sol.
func TestIsSuperGame(t *testing.T) {
	super := []uint32{4, 5, 7, 9} // SUPER_CANNON, SUPER_PERMISSIONED_CANNON, SUPER_ASTERISC_KONA, SUPER_CANNON_KONA
	legacy := []uint32{0, 1, 2, 3, 6, 8, 10, 254, 255}
	for _, gt := range super {
		assert.True(t, isSuperGame(gt), "game type %d is a super game", gt)
	}
	for _, gt := range legacy {
		assert.False(t, isSuperGame(gt), "game type %d is a legacy game", gt)
	}
}

// TestCollectProveInputs_SkipsRevertedDecoy proves the review comment #1 fix: a
// relayer prepends a reverting portal call carrying the prove selector (a decoy),
// then makes the real successful prove call. The reverted frame must be skipped
// so only the real prove input is collected.
func TestCollectProveInputs_SkipsRevertedDecoy(t *testing.T) {
	m := newTestMonitor(t)
	real := packProve(t, bindings.TypesOutputRootProof{}, [][]byte{{0x01}})
	decoy := "0x" + common.Bytes2Hex(m.proveSelector[:]) + "deadbeef" // selector + garbage args

	frame := &callFrame{
		Type: "CALL", To: "0x43edb88c4b80fdd2adff2412a7bebf9df42cb40e", Input: "0xabcdef",
		Calls: []callFrame{
			// Reverted decoy to the portal, with a subtree that must also be skipped.
			{Type: "CALL", To: m.portalAddress.Hex(), Input: decoy, Error: "execution reverted",
				Calls: []callFrame{{Type: "CALL", To: m.portalAddress.Hex(), Input: decoy}}},
			// The real, successful prove call.
			{Type: "CALL", To: m.portalAddress.Hex(), Input: "0x" + common.Bytes2Hex(real)},
		},
	}

	var got [][]byte
	m.collectProveInputs(frame, &got)
	require.Len(t, got, 1, "only the successful prove call is collected")
	assert.Equal(t, real, got[0])
}

// packProveTx builds calldata for a proveWithdrawalTransaction call with an
// explicit withdrawal tx and dispute-game index, so tests can construct several
// prove calls for the SAME withdrawal against DIFFERENT games.
func packProveTx(t *testing.T, tx bindings.TypesWithdrawalTransaction, index *big.Int, orp bindings.TypesOutputRootProof, proof [][]byte) []byte {
	t.Helper()
	a := proveABI(t)
	packed, err := a.Pack(proveMethodName, tx, index, orp, proof)
	require.NoError(t, err)
	return packed
}

// TestDecodeProveInput_DisputeGameIndex proves the game index is recovered from
// the prove call itself. This is the core of PR #176 review comment #2: the game
// is resolved by the call's own _disputeGameIndex (immutable via gameAtIndex),
// not by reading the mutable provenWithdrawals mapping which a re-prove overwrites.
func TestDecodeProveInput_DisputeGameIndex(t *testing.T) {
	m := newTestMonitor(t)
	tx := bindings.TypesWithdrawalTransaction{
		Nonce: big.NewInt(9), Sender: common.HexToAddress("0xaa"), Target: common.HexToAddress("0xbb"),
		Value: big.NewInt(0), GasLimit: big.NewInt(21000), Data: []byte{},
	}
	for _, idx := range []int64{7, 42, 100} {
		input := packProveTx(t, tx, big.NewInt(idx), bindings.TypesOutputRootProof{}, [][]byte{{0x01}})
		dp, err := m.decodeProveInput(input)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(idx), dp.disputeGameIndex)
	}
}

// TestPositionalPairing_SameHashTwoGames proves the same-tx duplicate case from
// review comment #2: a single tx proves ONE withdrawal hash twice, against two
// different games. collectProveInputs must preserve call order, and the two
// decoded calls must share the withdrawal hash but carry distinct game indices —
// so pairing the k-th event to the k-th call selects the correct game.
func TestPositionalPairing_SameHashTwoGames(t *testing.T) {
	m := newTestMonitor(t)
	tx := bindings.TypesWithdrawalTransaction{
		Nonce: big.NewInt(1), Sender: common.HexToAddress("0xaa"), Target: common.HexToAddress("0xbb"),
		Value: big.NewInt(0), GasLimit: big.NewInt(21000), Data: []byte{},
	}
	wdHash, err := m.withdrawalHash(tx)
	require.NoError(t, err)

	first := packProveTx(t, tx, big.NewInt(7), bindings.TypesOutputRootProof{}, [][]byte{{0x01}})
	second := packProveTx(t, tx, big.NewInt(42), bindings.TypesOutputRootProof{}, [][]byte{{0x02}})

	// Two sibling internal calls to the portal, in order, inside a wrapper tx.
	frame := &callFrame{
		Type: "CALL", To: "0x43edb88c4b80fdd2adff2412a7bebf9df42cb40e", Input: "0xabcdef",
		Calls: []callFrame{
			{Type: "CALL", To: m.portalAddress.Hex(), Input: "0x" + common.Bytes2Hex(first)},
			{Type: "CALL", To: m.portalAddress.Hex(), Input: "0x" + common.Bytes2Hex(second)},
		},
	}

	var inputs [][]byte
	m.collectProveInputs(frame, &inputs)
	require.Len(t, inputs, 2)

	// Collect candidates for this hash in call order, mirroring recoverProof.
	var candidates []*decodedProof
	for _, in := range inputs {
		dp, derr := m.decodeProveInput(in)
		require.NoError(t, derr)
		h, herr := m.withdrawalHash(dp.withdrawalTx)
		require.NoError(t, herr)
		if h == wdHash {
			candidates = append(candidates, dp)
		}
	}

	require.Len(t, candidates, 2, "both proves target the same withdrawal hash")
	// Positional pairing: ordinal 0 -> game 7, ordinal 1 -> game 42.
	assert.Equal(t, big.NewInt(7), candidates[0].disputeGameIndex)
	assert.Equal(t, big.NewInt(42), candidates[1].disputeGameIndex)
}

// TestWithdrawalHashMatching proves the batcher disambiguation: a tx with two
// prove calls yields two inputs, and each decodes to a distinct withdrawal hash so
// the right proof is selected per event.
func TestWithdrawalHashMatching(t *testing.T) {
	m := newTestMonitor(t)
	txA := bindings.TypesWithdrawalTransaction{
		Nonce: big.NewInt(1), Sender: common.HexToAddress("0xaa"), Target: common.HexToAddress("0xbb"),
		Value: big.NewInt(0), GasLimit: big.NewInt(21000), Data: []byte{},
	}
	txB := bindings.TypesWithdrawalTransaction{
		Nonce: big.NewInt(2), Sender: common.HexToAddress("0xcc"), Target: common.HexToAddress("0xdd"),
		Value: big.NewInt(5), GasLimit: big.NewInt(21000), Data: []byte{0x01},
	}
	hA, err := m.withdrawalHash(txA)
	require.NoError(t, err)
	hB, err := m.withdrawalHash(txB)
	require.NoError(t, err)
	assert.NotEqual(t, hA, hB, "distinct withdrawal txs must hash differently")
}
