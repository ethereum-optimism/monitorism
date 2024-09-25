//go:build live
// +build live

package faultproof_withdrawals

import (
	"math/big"
	"os"
	"testing"

	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

/*
Package faultproof_withdrawals contains tests that are meant to test the functionalities against live nodes.
These tests specifically refer to the Optimism chain on the Sepolia network.
The tests have a custom setup tailored for this chain, including specific block ranges and expected events.
*/

var (
	l1GethClient           *ethclient.Client
	optimismPortal2Helper  *OptimismPortal2Helper
	faultDisputeGameHelper *FaultDisputeGameHelper
	l2NodeHelper           *L2NodeHelper
)

// TestMain sets up the environment and necessary connections before running the tests
func TestMain(m *testing.M) {
	err := loadEnv(".env.op.sepolia")
	if err != nil {
		panic("Failed to load environment variables: " + err.Error())
	}

	L1GethURL := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_L1_GETH_URL")
	L2OpNodeURL := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_L2_OP_NODE_URL")
	portalAddr := os.Getenv("FAULTPROOF_WITHDRAWAL_MON_OPTIMISM_PORTAL")
	OptimismPortalAddress := common.HexToAddress(portalAddr)

	ctx := context.Background()
	l1GethClient, err = ethclient.Dial(L1GethURL)
	if err != nil {
		panic("Failed to connect to L1 Geth client: " + err.Error())
	}
	l2OpNodeClient, err := ethclient.Dial(L2OpNodeURL)
	if err != nil {
		panic("Failed to connect to L2 Optimism Node client: " + err.Error())
	}

	optimismPortal2Helper, err = NewOptimismPortal2Helper(ctx, l1GethClient, OptimismPortalAddress)
	if err != nil {
		panic("Failed to initialize OptimismPortal2Helper: " + err.Error())
	}
	faultDisputeGameHelper, err = NewFaultDisputeGameHelper(ctx, l1GethClient)
	if err != nil {
		panic("Failed to initialize FaultDisputeGameHelper: " + err.Error())
	}
	l2NodeHelper, err = NewL2NodeHelper(ctx, l2OpNodeClient)
	if err != nil {
		panic("Failed to initialize L2NodeHelper: " + err.Error())
	}

	// Run the tests
	code := m.Run()

	// Perform any cleanup (if needed)
	os.Exit(code)
}

func TestGetProvenWithdrawalsExtension1Events(t *testing.T) {

	start := uint64(5914813) // Adjust according to your test case
	stop := start + 1000     // Adjust according to your test case

	// Event Topic: 0x798f9f13695f8f045aa5f80ed8efebb695f3c7fe65da381969f2f28bf3c60b97
	// Transaction MethodID: 0x4870496f (proveWithdrawalTransaction)
	expectedEvent := WithdrawalProvenExtension1Event{
		WithdrawalHash: common.HexToHash("0xedbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"),
		ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
		Raw: Raw{
			BlockNumber: 5915676,
			TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
		},
	}

	events, err := optimismPortal2Helper.GetProvenWithdrawalsExtension1Events(start, &stop)
	require.NoError(t, err)
	require.Equal(t, len(events), 1, "Expected 1 event")
	require.Equal(t, expectedEvent, events[0], "Expected event not found")
}

func TestGetProvenWithdrawalsEvents(t *testing.T) {
	start := uint64(5914813) // Adjust according to your test case
	stop := start + 1000     // Adjust according to your test case

	// Event Topic: 0x798f9f13695f8f045aa5f80ed8efebb695f3c7fe65da381969f2f28bf3c60b97
	// Transaction MethodID: 0x4870496f (proveWithdrawalTransaction)
	expectedEvent := WithdrawalProvenEvent{
		WithdrawalHash: common.HexToHash("0xedbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"),
		Raw: Raw{
			BlockNumber: 5915676,
			TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
		},
	}

	events, err := optimismPortal2Helper.GetProvenWithdrawalsEvents(start, &stop)
	require.NoError(t, err)
	require.Equal(t, len(events), 1, "Expected 1 event")
	require.Equal(t, expectedEvent, events[0], "Expected event not found")
}

func TestGetSumittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(t *testing.T) {

	// https://sepolia.etherscan.io/tx/0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132
	withdrawalEvent := WithdrawalProvenExtension1Event{
		WithdrawalHash: common.HexToHash("0xedbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"),
		ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
		Raw: Raw{
			BlockNumber: 5915676,
			TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
		},
	}

	expectedSumittedProofsData := &SubmittedProofData{
		proofSubmitterAddress:     withdrawalEvent.ProofSubmitter,
		withdrawalHash:            withdrawalEvent.WithdrawalHash,
		disputeGameProxyAddress:   common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		disputeGameProxyTimestamp: 1716028908,
	}
	sumittedProofsData, err := optimismPortal2Helper.GetSumittedProofsDataFromWithdrawalhashAndProofSubmitterAddress(withdrawalEvent.WithdrawalHash, withdrawalEvent.ProofSubmitter)
	require.NoError(t, err)
	require.Equal(t, expectedSumittedProofsData, sumittedProofsData, "Expected game not found")

	//https://sepolia.etherscan.io/address/0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7#readContract
	expectedDisputeGameData := &DisputeGameData{
		ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
		L2blockNumber: big.NewInt(12030787),
		L2ChainID:     big.NewInt(11155420),
		Status:        DEFENDER_WINS,
		CreatedAt:     1715864520,
		ResolvedAt:    1716166980,
	}

	expectedTrustedL2OutputRoot := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")
	trustedL2OutputRoot, err := l2NodeHelper.GetOutputRootFromTrustedL2Node(expectedDisputeGameData.L2blockNumber)
	require.NoError(t, err)
	require.Equal(t, true, trustedL2OutputRoot == expectedTrustedL2OutputRoot, "Expected root claim not found")

	disputeGameProxy, error := faultDisputeGameHelper.GetDisputeGameProxyFromAddress(sumittedProofsData.disputeGameProxyAddress)
	require.NoError(t, error)
	disputeGameData := disputeGameProxy.DisputeGameData
	require.Equal(t, expectedDisputeGameData, disputeGameData, "Expected Dispute Game not found")

	require.Equal(t, true, disputeGameData.RootClaim == trustedL2OutputRoot, "Expected root claim not found")

}

func TestGetOutputRootFromTrustedL2Node(t *testing.T) {

	//https://sepolia.etherscan.io/address/0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7#readContract
	expectedDisputeGameData := &DisputeGameData{
		ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
		L2blockNumber: big.NewInt(12030787),
		L2ChainID:     big.NewInt(11155420),
		Status:        DEFENDER_WINS,
		CreatedAt:     1715864520,
		ResolvedAt:    1716166980,
	}

	expectedTrustedL2OutputRoot := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")
	trustedL2OutputRoot, err := l2NodeHelper.GetOutputRootFromTrustedL2Node(expectedDisputeGameData.L2blockNumber)
	require.NoError(t, err)
	require.Equal(t, true, trustedL2OutputRoot == expectedTrustedL2OutputRoot, "Expected root claim not found")

	disputeGameProxy, error := faultDisputeGameHelper.GetDisputeGameProxyFromAddress(expectedDisputeGameData.ProxyAddress)
	require.NoError(t, error)
	disputeGameData := disputeGameProxy.DisputeGameData
	require.Equal(t, expectedDisputeGameData, disputeGameData, "Expected Dispute Game not found")

	require.Equal(t, true, disputeGameData.RootClaim == trustedL2OutputRoot, "Expected root claim not found")

}

func TestGetDisputeGameProxyFromAddress(t *testing.T) {

	//https://sepolia.etherscan.io/address/0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7#readContract
	expectedDisputeGameData := &DisputeGameData{
		ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
		L2blockNumber: big.NewInt(12030787),
		L2ChainID:     big.NewInt(11155420),
		Status:        DEFENDER_WINS,
		CreatedAt:     1715864520,
		ResolvedAt:    1716166980,
	}

	//block https://sepolia-optimism.etherscan.io/block/12030787
	expectedTrustedL2OutputRoot := common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597")

	disputeGameProxy, error := faultDisputeGameHelper.GetDisputeGameProxyFromAddress(expectedDisputeGameData.ProxyAddress)
	require.NoError(t, error)
	disputeGameData := disputeGameProxy.DisputeGameData
	require.Equal(t, expectedDisputeGameData, disputeGameData, "Expected Dispute Game not found")

	require.Equal(t, true, disputeGameData.RootClaim == expectedTrustedL2OutputRoot, "Expected root claim not found")

}
