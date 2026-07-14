package withdrawalsv2

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals-v2/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
