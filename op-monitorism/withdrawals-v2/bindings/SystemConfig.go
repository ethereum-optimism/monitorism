package bindings

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// systemConfigABI is the minimal ABI needed to read the L2 chain id, used to
// mirror the portal's super-game output-root predicate
// (rootClaimByChainId(systemConfig.l2ChainId())).
const systemConfigABI = `[{"inputs":[],"name":"l2ChainId","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`

// SystemConfig is a thin caller over the SystemConfig contract.
type SystemConfig struct {
	contract *bind.BoundContract
}

// NewSystemConfig binds the SystemConfig at the given address.
func NewSystemConfig(address common.Address, backend bind.ContractBackend) (*SystemConfig, error) {
	parsed, err := abi.JSON(strings.NewReader(systemConfigABI))
	if err != nil {
		return nil, err
	}
	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &SystemConfig{contract: contract}, nil
}

// L2ChainId returns the chain id of the L2 this SystemConfig governs.
func (s *SystemConfig) L2ChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	if err := s.contract.Call(opts, &out, "l2ChainId"); err != nil {
		return nil, err
	}
	return *abi.ConvertType(out[0], new(*big.Int)).(**big.Int), nil
}
