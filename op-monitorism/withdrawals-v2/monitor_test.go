package withdrawalsv2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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
		{
			name:           "specific test vector",
			withdrawalHash: [32]byte{0xaa, 0xbb, 0xcc, 0xdd},
			expectedKey:    crypto.Keccak256Hash(append([]byte{0xaa, 0xbb, 0xcc, 0xdd}, append(make([]byte, 28), make([]byte, 32)...)...)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeWithdrawalStorageKey(tt.withdrawalHash)
			assert.Equal(t, tt.expectedKey, result)
		})
	}
}

func TestDetermineFailureReason(t *testing.T) {
	tests := []struct {
		name                 string
		computedOutputRoot   eth.Bytes32
		disputeGameRootClaim [32]byte
		expectedReason       string
		description          string
	}{
		{
			name:                 "output roots match - bad withdrawal proof",
			computedOutputRoot:   eth.Bytes32{0xaa, 0xbb, 0xcc},
			disputeGameRootClaim: [32]byte{0xaa, 0xbb, 0xcc},
			expectedReason:       "bad_withdrawal_proof",
			description:          "When output roots match, the dispute game is correct but withdrawal proof is invalid (P0 - serious issue)",
		},
		{
			name:                 "output roots don't match - bad output root",
			computedOutputRoot:   eth.Bytes32{0xaa, 0xbb, 0xcc},
			disputeGameRootClaim: [32]byte{0xdd, 0xee, 0xff},
			expectedReason:       "bad_output_root",
			description:          "When output roots don't match, dispute game has wrong output root (P3 - acceptable)",
		},
		{
			name:                 "zero values don't match",
			computedOutputRoot:   eth.Bytes32{},
			disputeGameRootClaim: [32]byte{0x01},
			expectedReason:       "bad_output_root",
			description:          "Zero vs non-zero should be bad output root",
		},
		{
			name:                 "both zero values",
			computedOutputRoot:   eth.Bytes32{},
			disputeGameRootClaim: [32]byte{},
			expectedReason:       "bad_withdrawal_proof",
			description:          "Both zero means they match - bad withdrawal proof",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineFailureReason(tt.computedOutputRoot, tt.disputeGameRootClaim)
			assert.Equal(t, tt.expectedReason, result, tt.description)
		})
	}
}

func TestFailureReasonPriority(t *testing.T) {
	matchingRoots := eth.Bytes32{0x12, 0x34, 0x56}
	sameRootClaim := [32]byte{0x12, 0x34, 0x56}

	reason := determineFailureReason(matchingRoots, sameRootClaim)
	assert.Equal(t, "bad_withdrawal_proof", reason,
		"When output roots match, it should indicate bad_withdrawal_proof (P0 - serious issue)")

	differentRootClaim := [32]byte{0x78, 0x9a, 0xbc}
	reason = determineFailureReason(matchingRoots, differentRootClaim)
	assert.Equal(t, "bad_output_root", reason,
		"When output roots don't match, it should indicate bad_output_root (P3 - acceptable)")
}
