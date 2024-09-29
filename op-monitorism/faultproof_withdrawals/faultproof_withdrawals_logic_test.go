package faultproof_withdrawals

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func faultproof_withdrawals_logic_test_init() *struct{ empty bool } {
	return &struct{ empty bool }{}
}

func TestValidProofEvent(t *testing.T) {
	init_values := faultproof_withdrawals_logic_test_init()
	_ = init_values

	// https://sepolia.etherscan.io/tx/0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132
	withdrawalEvent := &WithdrawalProvenExtension1Event{
		WithdrawalHash: common.HexToHash("0xedbe26c8f9b11835295aee42123335f920599f01448e0ec697e9a47e69ed673e"),
		ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
		Raw: Raw{
			BlockNumber: 5915676,
			TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
		},
	}

	// https: //sepolia.etherscan.io/address/0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7#readContract
	expectedDisputeGameData := &DisputeGameData{
		ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
		L2blockNumber: big.NewInt(12030787),
		L2ChainID:     big.NewInt(11155420),
		Status:        DEFENDER_WINS,
		CreatedAt:     1715864520,
		ResolvedAt:    1716166980,
	}

	expectedFaultDisputeGameProxy := &FaultDisputeGameProxy{
		FaultDisputeGame: nil, //we want to ignore the pointer to the object in the check
		DisputeGameData:  expectedDisputeGameData,
	}

	expectedRootClaim := &expectedFaultDisputeGameProxy.DisputeGameData.RootClaim
	expectedEnrichedWithdrawalEvent := &EnrichedWithdrawalEvent{
		Event:                     withdrawalEvent,
		DisputeGame:               expectedFaultDisputeGameProxy,
		expectedRootClaim:         expectedRootClaim,
		Blacklisted:               false,
		withdrawalHashPresentOnL2: true,
		Enriched:                  false,
	}

	withdrawalValidator := &WithdrawalValidator{
		optimismPortal2Helper:     nil,
		l2NodeHelper:              nil,
		l2ToL1MessagePasserHelper: nil,
		faultDisputeGameHelper:    nil,
		ctx:                       nil,
	}
	expected_validateProofWithdrawalState, err := withdrawalValidator.ValidateWithdrawal(expectedEnrichedWithdrawalEvent)
	require.NoError(t, err)
	require.Equal(t, expected_validateProofWithdrawalState, VALID_PROOF)
}

func TestInValidProofEvent1(t *testing.T) {
	init_values := faultproof_withdrawals_logic_test_init()
	_ = init_values

	thisHashDoesNotExists := common.HexToHash("0xedbe26c8f9b11835295aee42123325f920599f01448e0ec697e9a47e69ed673e")
	// https://sepolia.etherscan.io/tx/0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132
	withdrawalEvent := &WithdrawalProvenExtension1Event{
		WithdrawalHash: thisHashDoesNotExists,
		ProofSubmitter: common.HexToAddress("0x4444d38c385d0969C64c4C8f996D7536d16c28B9"),
		Raw: Raw{
			BlockNumber: 5915676,
			TxHash:      common.HexToHash("0x38227b45af7eb20bfa341df89955f142a4de85add67e05cbac5d80c0d9cc6132"),
		},
	}

	// https: //sepolia.etherscan.io/address/0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7#readContract
	expectedDisputeGameData := &DisputeGameData{
		ProxyAddress:  common.HexToAddress("0xFA6b748abc490d3356585A1228c73BEd8DA2A3a7"),
		RootClaim:     common.HexToHash("0x763d50048ccdb85fded935ff88c9e6b2284fd981da8ed7ae892f36b8761f7597"),
		L2blockNumber: big.NewInt(12030787),
		L2ChainID:     big.NewInt(11155420),
		Status:        DEFENDER_WINS,
		CreatedAt:     1715864520,
		ResolvedAt:    1716166980,
	}

	expectedFaultDisputeGameProxy := &FaultDisputeGameProxy{
		FaultDisputeGame: nil, //we want to ignore the pointer to the object in the check
		DisputeGameData:  expectedDisputeGameData,
	}

	expectedRootClaim := &expectedFaultDisputeGameProxy.DisputeGameData.RootClaim
	expectedEnrichedWithdrawalEvent := &EnrichedWithdrawalEvent{
		Event:                     withdrawalEvent,
		DisputeGame:               expectedFaultDisputeGameProxy,
		expectedRootClaim:         expectedRootClaim,
		Blacklisted:               false,
		withdrawalHashPresentOnL2: true,
		Enriched:                  false,
	}

	withdrawalValidator := &WithdrawalValidator{
		optimismPortal2Helper:     nil,
		l2NodeHelper:              nil,
		l2ToL1MessagePasserHelper: nil,
		faultDisputeGameHelper:    nil,
		ctx:                       nil,
	}
	expected_validateProofWithdrawalState, err := withdrawalValidator.ValidateWithdrawal(expectedEnrichedWithdrawalEvent)
	require.Error(t, err)
	require.Equal(t, expected_validateProofWithdrawalState, VALID_PROOF)
}
