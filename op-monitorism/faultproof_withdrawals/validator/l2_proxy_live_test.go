//go:build live
// +build live

package validator

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestVerifyRootClaimFromHeaderMainnet exercises header-based verification against
// real OP Mainnet data. Set FPW_L2_RPC to an OP Mainnet execution RPC.
//
//	FPW_L2_RPC=<rpc> go test -tags live -run TestVerifyRootClaimFromHeaderMainnet ./validator/
func TestVerifyRootClaimFromHeaderMainnet(t *testing.T) {
	l2URL := os.Getenv("FPW_L2_RPC")
	require.NotEmpty(t, l2URL, "FPW_L2_RPC must be set")

	l2, err := NewL2Proxy(context.Background(), l2URL, nil)
	require.NoError(t, err)

	// Real proven withdrawal, dispute game index 18180, from L1 tx
	// 0x4b4cf0681ae913d4124e10347464987535756ebb0be16da5a420b8f7242f0205.
	postIsthmusBlock := big.NewInt(153912395)
	rootClaim := [32]byte(common.HexToHash("0x19c57f7983d2c80a343784284b642da8a4ac4594982dcec29c3b65708dbfd57a"))

	t.Run("post-isthmus honest game verifies from header", func(t *testing.T) {
		trusted, preIsthmus, err := l2.VerifyRootClaimFromHeader(postIsthmusBlock, rootClaim)
		require.NoError(t, err)
		require.False(t, preIsthmus)
		require.True(t, trusted, "header-derived output root should match the game root claim")
	})

	t.Run("wrong root claim does not verify (would be a forgery)", func(t *testing.T) {
		wrong := [32]byte(common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"))
		trusted, preIsthmus, err := l2.VerifyRootClaimFromHeader(postIsthmusBlock, wrong)
		require.NoError(t, err)
		require.False(t, preIsthmus)
		require.False(t, trusted)
	})

	t.Run("pre-isthmus block is flagged, not alarmed", func(t *testing.T) {
		// Block 125M carries the pre-Isthmus sentinel withdrawals root.
		trusted, preIsthmus, err := l2.VerifyRootClaimFromHeader(big.NewInt(125000000), rootClaim)
		require.NoError(t, err)
		require.True(t, preIsthmus, "pre-Isthmus block must be flagged for triage")
		require.False(t, trusted)
	})
}
