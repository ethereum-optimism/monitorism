package bindings

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// superFaultDisputeGameABI is the minimal ABI needed to read a super game's
// per-chain output root. Super game types (SUPER_CANNON, SUPER_PERMISSIONED_CANNON,
// SUPER_ASTERISC_KONA, SUPER_CANNON_KONA) commit to a super root covering many
// chains; the portal extracts this chain's output root via rootClaimByChainId.
const superFaultDisputeGameABI = `[{"inputs":[{"internalType":"uint256","name":"_chainId","type":"uint256"}],"name":"rootClaimByChainId","outputs":[{"internalType":"Claim","name":"rootClaim_","type":"bytes32"}],"stateMutability":"pure","type":"function"}]`

// SuperFaultDisputeGame is a thin caller over the super-root dispute game.
type SuperFaultDisputeGame struct {
	contract *bind.BoundContract
}

// NewSuperFaultDisputeGame binds the super dispute game at the given address.
func NewSuperFaultDisputeGame(address common.Address, backend bind.ContractBackend) (*SuperFaultDisputeGame, error) {
	parsed, err := abi.JSON(strings.NewReader(superFaultDisputeGameABI))
	if err != nil {
		return nil, err
	}
	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &SuperFaultDisputeGame{contract: contract}, nil
}

// RootClaimByChainId returns the output root this super game commits to for the
// given L2 chain id.
func (g *SuperFaultDisputeGame) RootClaimByChainId(opts *bind.CallOpts, chainID *big.Int) ([32]byte, error) {
	var out []interface{}
	if err := g.contract.Call(opts, &out, "rootClaimByChainId", chainID); err != nil {
		return [32]byte{}, err
	}
	return *abi.ConvertType(out[0], new([32]byte)).(*[32]byte), nil
}
