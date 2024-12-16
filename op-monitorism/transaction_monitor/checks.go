package transaction_monitor

import (
	"context"
	"fmt"

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
	// First check if there's any code at the address
	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get code at address: %w", err)
	}
	if len(code) == 0 {
		return false, nil
	}

	// Get factory address from params
	factoryAddr, ok := params["disputeGameFactory"].(string)
	if !ok {
		return false, fmt.Errorf("disputeGameFactory parameter not found or invalid")
	}
	factory := common.HexToAddress(factoryAddr)

	// Get contract ABIs
	disputeGameABI, err := DisputeGameMetaData.GetAbi()
	if err != nil {
		return false, fmt.Errorf("failed to get dispute game ABI: %w", err)
	}

	disputeGameFactoryABI, err := DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		return false, fmt.Errorf("failed to get dispute game factory ABI: %w", err)
	}

	// Create contract bindings
	game := bind.NewBoundContract(addr, *disputeGameABI, client, client, client)
	factoryContract := bind.NewBoundContract(factory, *disputeGameFactoryABI, client, client, client)

	// Get game parameters
	var gameTypeResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &gameTypeResult, "gameType"); err != nil {
		return false, fmt.Errorf("failed to get game type: %w", err)
	}
	if len(gameTypeResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for gameType")
	}
	gameType, ok := gameTypeResult[0].(uint32)
	if !ok {
		return false, fmt.Errorf("invalid game type returned")
	}

	var rootClaimResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &rootClaimResult, "rootClaim"); err != nil {
		return false, fmt.Errorf("failed to get root claim: %w", err)
	}
	if len(rootClaimResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for rootClaim")
	}
	rootClaim, ok := rootClaimResult[0].([32]byte)
	if !ok {
		return false, fmt.Errorf("invalid root claim returned")
	}

	var extraDataResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &extraDataResult, "extraData"); err != nil {
		return false, fmt.Errorf("failed to get extra data: %w", err)
	}
	if len(extraDataResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for extraData")
	}
	extraData, ok := extraDataResult[0].([]byte)
	if !ok {
		return false, fmt.Errorf("invalid extra data returned")
	}

	// Verify with factory
	var factoryResult []interface{}
	if err := factoryContract.Call(&bind.CallOpts{Context: ctx}, &factoryResult, "games", gameType, rootClaim, extraData); err != nil {
		return false, fmt.Errorf("failed to verify game with factory: %w", err)
	}
	if len(factoryResult) != 2 {
		return false, fmt.Errorf("unexpected number of return values from factory")
	}

	proxy, ok := factoryResult[0].(common.Address)
	if !ok {
		return false, fmt.Errorf("invalid proxy address returned")
	}

	timestamp, ok := factoryResult[1].(uint64)
	if !ok {
		return false, fmt.Errorf("invalid timestamp returned")
	}

	return proxy == addr && timestamp > 0, nil
}
