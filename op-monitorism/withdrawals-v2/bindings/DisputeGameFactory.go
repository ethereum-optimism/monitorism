package bindings

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// disputeGameFactoryABI is the minimal ABI needed to resolve a game by its
// factory index. gameAtIndex returns the immutable (gameType, timestamp, proxy)
// tuple recorded when the game was created, so reading it at latest L1 state
// still yields the game the withdrawal was proven against.
const disputeGameFactoryABI = `[{"inputs":[{"internalType":"uint256","name":"_index","type":"uint256"}],"name":"gameAtIndex","outputs":[{"internalType":"GameType","name":"gameType_","type":"uint32"},{"internalType":"Timestamp","name":"timestamp_","type":"uint64"},{"internalType":"contract IDisputeGame","name":"proxy_","type":"address"}],"stateMutability":"view","type":"function"}]`

// DisputeGameFactory is a thin caller over the DisputeGameFactory contract.
type DisputeGameFactory struct {
	contract *bind.BoundContract
}

// NewDisputeGameFactory binds the DisputeGameFactory at the given address.
func NewDisputeGameFactory(address common.Address, backend bind.ContractBackend) (*DisputeGameFactory, error) {
	parsed, err := abi.JSON(strings.NewReader(disputeGameFactoryABI))
	if err != nil {
		return nil, err
	}
	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &DisputeGameFactory{contract: contract}, nil
}

// GameAtIndex returns the game created at the given factory index.
func (f *DisputeGameFactory) GameAtIndex(opts *bind.CallOpts, index *big.Int) (struct {
	GameType  uint32
	Timestamp uint64
	Proxy     common.Address
}, error) {
	var out []interface{}
	err := f.contract.Call(opts, &out, "gameAtIndex", index)
	outstruct := new(struct {
		GameType  uint32
		Timestamp uint64
		Proxy     common.Address
	})
	if err != nil {
		return *outstruct, err
	}
	outstruct.GameType = *abi.ConvertType(out[0], new(uint32)).(*uint32)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.Proxy = *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	return *outstruct, nil
}
