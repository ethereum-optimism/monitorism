package transaction_monitor

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/op-monitorism/transaction_monitor/bindings/dispute"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CheckFunc is a type for address verification functions
type CheckFunc func(ctx context.Context, client *ethclient.Client, addr common.Address, params map[string]interface{}) (bool, error)

// AddressChecks maps check types to their implementation functions
var AddressChecks = map[CheckType]CheckFunc{
	ExactMatchCheck:  CheckExactMatch,
	DisputeGameCheck: CheckDisputeGame,
}

// CheckExactMatch verifies if the address matches exactly with the provided match parameter
func CheckExactMatch(ctx context.Context, client *ethclient.Client, addr common.Address, params map[string]interface{}) (bool, error) {
	match, ok := params["match"].(string)
	if !ok {
		return false, fmt.Errorf("match parameter not found or invalid")
	}
	return addr == common.HexToAddress(match), nil
}

// CheckDisputeGame verifies if the address is a valid dispute game created by the factory
func CheckDisputeGame(ctx context.Context, client *ethclient.Client, addr common.Address, params map[string]interface{}) (bool, error) {
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get code at address: %w", err)
	}
	if len(code) == 0 {
		return false, nil
	}

	factoryAddr, ok := params["disputeGameFactory"].(string)
	if !ok {
		return false, fmt.Errorf("disputeGameFactory parameter not found or invalid")
	}

	game, err := dispute.NewDisputeGame(addr, client)
	if err != nil {
		return false, fmt.Errorf("failed to create dispute game: %w", err)
	}

	factory, err := dispute.NewDisputeGameFactory(common.HexToAddress(factoryAddr), client)
	if err != nil {
		return false, fmt.Errorf("failed to create dispute game factory: %w", err)
	}

	gameType, err := game.GameType(&bind.CallOpts{Context: ctx})
	if err != nil {
		return false, fmt.Errorf("failed to get game type: %w", err)
	}

	rootClaim, err := game.RootClaim(&bind.CallOpts{Context: ctx})
	if err != nil {
		return false, fmt.Errorf("failed to get root claim: %w", err)
	}

	extraData, err := game.ExtraData(&bind.CallOpts{Context: ctx})
	if err != nil {
		return false, fmt.Errorf("failed to get extra data: %w", err)
	}

	factoryResult, err := factory.Games(&bind.CallOpts{Context: ctx}, gameType, rootClaim, extraData)
	if err != nil {
		return false, fmt.Errorf("failed to verify game with factory: %w", err)
	}

	return factoryResult.Proxy == addr && factoryResult.Timestamp > 0, nil
}

// ParamValidationFunc is a type for parameter validation functions
type ParamValidationFunc func(params map[string]interface{}) error

// ParamValidations maps check types to their parameter validation functions
var ParamValidations = map[CheckType]ParamValidationFunc{
	ExactMatchCheck:  ValidateCheckExactMatch,
	DisputeGameCheck: ValidateCheckDisputeGame,
}

// ValidateCheckExactMatch validates the parameters for the exact match check
func ValidateCheckExactMatch(params map[string]interface{}) error {
	_, ok := params["match"].(string)
	if !ok {
		return fmt.Errorf("match parameter not found or invalid")
	}
	return nil
}

// ValidateCheckDisputeGame validates the parameters for the dispute game check
func ValidateCheckDisputeGame(params map[string]interface{}) error {
	_, ok := params["disputeGameFactory"].(string)
	if !ok {
		return fmt.Errorf("disputeGameFactory parameter not found or invalid")
	}
	return nil
}
