// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bindings

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TypesWithdrawalTransaction is an auto generated low-level Go binding around an user-defined struct.
type TypesWithdrawalTransaction struct {
	Nonce    *big.Int
	Sender   common.Address
	Target   common.Address
	Value    *big.Int
	GasLimit *big.Int
	Data     []byte
}

// OptimismPortal2MetaData contains all meta data concerning the OptimismPortal2 contract.
var OptimismPortal2MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_proofMaturityDelaySeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_disputeGameFinalityDelaySeconds\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"stateMutability\":\"payable\",\"type\":\"receive\"},{\"inputs\":[],\"name\":\"balance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIDisputeGame\",\"name\":\"_disputeGame\",\"type\":\"address\"}],\"name\":\"blacklistDisputeGame\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_withdrawalHash\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"_proofSubmitter\",\"type\":\"address\"}],\"name\":\"checkWithdrawal\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_mint\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositERC20Transaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"uint64\",\"name\":\"_gasLimit\",\"type\":\"uint64\"},{\"internalType\":\"bool\",\"name\":\"_isCreation\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"_data\",\"type\":\"bytes\"}],\"name\":\"depositTransaction\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIDisputeGame\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"disputeGameBlacklist\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"disputeGameFactory\",\"outputs\":[{\"internalType\":\"contractIDisputeGameFactory\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"disputeGameFinalityDelaySeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"donateETH\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.WithdrawalTransaction\",\"name\":\"_tx\",\"type\":\"tuple\"}],\"name\":\"finalizeWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.WithdrawalTransaction\",\"name\":\"_tx\",\"type\":\"tuple\"},{\"internalType\":\"address\",\"name\":\"_proofSubmitter\",\"type\":\"address\"}],\"name\":\"finalizeWithdrawalTransactionExternalProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"finalizedWithdrawals\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"guardian\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIDisputeGameFactory\",\"name\":\"_disputeGameFactory\",\"type\":\"address\"},{\"internalType\":\"contractISystemConfig\",\"name\":\"_systemConfig\",\"type\":\"address\"},{\"internalType\":\"contractISuperchainConfig\",\"name\":\"_superchainConfig\",\"type\":\"address\"},{\"internalType\":\"GameType\",\"name\":\"_initialRespectedGameType\",\"type\":\"uint32\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l2Sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint64\",\"name\":\"_byteCount\",\"type\":\"uint64\"}],\"name\":\"minimumGasLimit\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_withdrawalHash\",\"type\":\"bytes32\"}],\"name\":\"numProofSubmitters\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"params\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"prevBaseFee\",\"type\":\"uint128\"},{\"internalType\":\"uint64\",\"name\":\"prevBoughtGas\",\"type\":\"uint64\"},{\"internalType\":\"uint64\",\"name\":\"prevBlockNum\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"proofMaturityDelaySeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"proofSubmitters\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"gasLimit\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structTypes.WithdrawalTransaction\",\"name\":\"_tx\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_disputeGameIndex\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"version\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"stateRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"messagePasserStorageRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"latestBlockhash\",\"type\":\"bytes32\"}],\"internalType\":\"structTypes.OutputRootProof\",\"name\":\"_outputRootProof\",\"type\":\"tuple\"},{\"internalType\":\"bytes[]\",\"name\":\"_withdrawalProof\",\"type\":\"bytes[]\"}],\"name\":\"proveWithdrawalTransaction\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"provenWithdrawals\",\"outputs\":[{\"internalType\":\"contractIDisputeGame\",\"name\":\"disputeGameProxy\",\"type\":\"address\"},{\"internalType\":\"uint64\",\"name\":\"timestamp\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"respectedGameType\",\"outputs\":[{\"internalType\":\"GameType\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"respectedGameTypeUpdatedAt\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"_decimals\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"_name\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_symbol\",\"type\":\"bytes32\"}],\"name\":\"setGasPayingToken\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"GameType\",\"name\":\"_gameType\",\"type\":\"uint32\"}],\"name\":\"setRespectedGameType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"superchainConfig\",\"outputs\":[{\"internalType\":\"contractISuperchainConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemConfig\",\"outputs\":[{\"internalType\":\"contractISystemConfig\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"version\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"contractIDisputeGame\",\"name\":\"disputeGame\",\"type\":\"address\"}],\"name\":\"DisputeGameBlacklisted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"GameType\",\"name\":\"newGameType\",\"type\":\"uint32\"},{\"indexed\":true,\"internalType\":\"Timestamp\",\"name\":\"updatedAt\",\"type\":\"uint64\"}],\"name\":\"RespectedGameTypeSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"version\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"opaqueData\",\"type\":\"bytes\"}],\"name\":\"TransactionDeposited\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"withdrawalHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"name\":\"WithdrawalFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"withdrawalHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"}],\"name\":\"WithdrawalProven\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"withdrawalHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"proofSubmitter\",\"type\":\"address\"}],\"name\":\"WithdrawalProvenExtension1\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"AlreadyFinalized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"BadTarget\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Blacklisted\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"CallPaused\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ContentLengthMismatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyItem\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"GasEstimation\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDataRemainder\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidDisputeGame\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidGameType\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidHeader\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidMerkleProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LargeCalldata\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NoValue\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NonReentrant\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OnlyCustomGasToken\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"OutOfGas\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ProposalNotValidated\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"SmallGasLimit\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"TransferFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unauthorized\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnexpectedList\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"UnexpectedString\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"Unproven\",\"type\":\"error\"}]",
}

// OptimismPortal2ABI is the input ABI used to generate the binding from.
// Deprecated: Use OptimismPortal2MetaData.ABI instead.
var OptimismPortal2ABI = OptimismPortal2MetaData.ABI

// OptimismPortal2 is an auto generated Go binding around an Ethereum contract.
type OptimismPortal2 struct {
	OptimismPortal2Caller     // Read-only binding to the contract
	OptimismPortal2Transactor // Write-only binding to the contract
	OptimismPortal2Filterer   // Log filterer for contract events
}

// OptimismPortal2Caller is an auto generated read-only Go binding around an Ethereum contract.
type OptimismPortal2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortal2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type OptimismPortal2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortal2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OptimismPortal2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OptimismPortal2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OptimismPortal2Session struct {
	Contract     *OptimismPortal2  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OptimismPortal2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OptimismPortal2CallerSession struct {
	Contract *OptimismPortal2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// OptimismPortal2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OptimismPortal2TransactorSession struct {
	Contract     *OptimismPortal2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// OptimismPortal2Raw is an auto generated low-level Go binding around an Ethereum contract.
type OptimismPortal2Raw struct {
	Contract *OptimismPortal2 // Generic contract binding to access the raw methods on
}

// OptimismPortal2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OptimismPortal2CallerRaw struct {
	Contract *OptimismPortal2Caller // Generic read-only contract binding to access the raw methods on
}

// OptimismPortal2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OptimismPortal2TransactorRaw struct {
	Contract *OptimismPortal2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewOptimismPortal2 creates a new instance of OptimismPortal2, bound to a specific deployed contract.
func NewOptimismPortal2(address common.Address, backend bind.ContractBackend) (*OptimismPortal2, error) {
	contract, err := bindOptimismPortal2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2{OptimismPortal2Caller: OptimismPortal2Caller{contract: contract}, OptimismPortal2Transactor: OptimismPortal2Transactor{contract: contract}, OptimismPortal2Filterer: OptimismPortal2Filterer{contract: contract}}, nil
}

// NewOptimismPortal2Caller creates a new read-only instance of OptimismPortal2, bound to a specific deployed contract.
func NewOptimismPortal2Caller(address common.Address, caller bind.ContractCaller) (*OptimismPortal2Caller, error) {
	contract, err := bindOptimismPortal2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2Caller{contract: contract}, nil
}

// NewOptimismPortal2Transactor creates a new write-only instance of OptimismPortal2, bound to a specific deployed contract.
func NewOptimismPortal2Transactor(address common.Address, transactor bind.ContractTransactor) (*OptimismPortal2Transactor, error) {
	contract, err := bindOptimismPortal2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2Transactor{contract: contract}, nil
}

// NewOptimismPortal2Filterer creates a new log filterer instance of OptimismPortal2, bound to a specific deployed contract.
func NewOptimismPortal2Filterer(address common.Address, filterer bind.ContractFilterer) (*OptimismPortal2Filterer, error) {
	contract, err := bindOptimismPortal2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2Filterer{contract: contract}, nil
}

// bindOptimismPortal2 binds a generic wrapper to an already deployed contract.
func bindOptimismPortal2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OptimismPortal2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal2 *OptimismPortal2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal2.Contract.OptimismPortal2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal2 *OptimismPortal2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.OptimismPortal2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal2 *OptimismPortal2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.OptimismPortal2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OptimismPortal2 *OptimismPortal2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OptimismPortal2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OptimismPortal2 *OptimismPortal2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OptimismPortal2 *OptimismPortal2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.contract.Transact(opts, method, params...)
}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Caller) Balance(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "balance")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Session) Balance() (*big.Int, error) {
	return _OptimismPortal2.Contract.Balance(&_OptimismPortal2.CallOpts)
}

// Balance is a free data retrieval call binding the contract method 0xb69ef8a8.
//
// Solidity: function balance() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2CallerSession) Balance() (*big.Int, error) {
	return _OptimismPortal2.Contract.Balance(&_OptimismPortal2.CallOpts)
}

// CheckWithdrawal is a free data retrieval call binding the contract method 0x71c1566e.
//
// Solidity: function checkWithdrawal(bytes32 _withdrawalHash, address _proofSubmitter) view returns()
func (_OptimismPortal2 *OptimismPortal2Caller) CheckWithdrawal(opts *bind.CallOpts, _withdrawalHash [32]byte, _proofSubmitter common.Address) error {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "checkWithdrawal", _withdrawalHash, _proofSubmitter)

	if err != nil {
		return err
	}

	return err

}

// CheckWithdrawal is a free data retrieval call binding the contract method 0x71c1566e.
//
// Solidity: function checkWithdrawal(bytes32 _withdrawalHash, address _proofSubmitter) view returns()
func (_OptimismPortal2 *OptimismPortal2Session) CheckWithdrawal(_withdrawalHash [32]byte, _proofSubmitter common.Address) error {
	return _OptimismPortal2.Contract.CheckWithdrawal(&_OptimismPortal2.CallOpts, _withdrawalHash, _proofSubmitter)
}

// CheckWithdrawal is a free data retrieval call binding the contract method 0x71c1566e.
//
// Solidity: function checkWithdrawal(bytes32 _withdrawalHash, address _proofSubmitter) view returns()
func (_OptimismPortal2 *OptimismPortal2CallerSession) CheckWithdrawal(_withdrawalHash [32]byte, _proofSubmitter common.Address) error {
	return _OptimismPortal2.Contract.CheckWithdrawal(&_OptimismPortal2.CallOpts, _withdrawalHash, _proofSubmitter)
}

// DisputeGameBlacklist is a free data retrieval call binding the contract method 0x45884d32.
//
// Solidity: function disputeGameBlacklist(address ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Caller) DisputeGameBlacklist(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "disputeGameBlacklist", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// DisputeGameBlacklist is a free data retrieval call binding the contract method 0x45884d32.
//
// Solidity: function disputeGameBlacklist(address ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Session) DisputeGameBlacklist(arg0 common.Address) (bool, error) {
	return _OptimismPortal2.Contract.DisputeGameBlacklist(&_OptimismPortal2.CallOpts, arg0)
}

// DisputeGameBlacklist is a free data retrieval call binding the contract method 0x45884d32.
//
// Solidity: function disputeGameBlacklist(address ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2CallerSession) DisputeGameBlacklist(arg0 common.Address) (bool, error) {
	return _OptimismPortal2.Contract.DisputeGameBlacklist(&_OptimismPortal2.CallOpts, arg0)
}

// DisputeGameFactory is a free data retrieval call binding the contract method 0xf2b4e617.
//
// Solidity: function disputeGameFactory() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) DisputeGameFactory(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "disputeGameFactory")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DisputeGameFactory is a free data retrieval call binding the contract method 0xf2b4e617.
//
// Solidity: function disputeGameFactory() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) DisputeGameFactory() (common.Address, error) {
	return _OptimismPortal2.Contract.DisputeGameFactory(&_OptimismPortal2.CallOpts)
}

// DisputeGameFactory is a free data retrieval call binding the contract method 0xf2b4e617.
//
// Solidity: function disputeGameFactory() view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) DisputeGameFactory() (common.Address, error) {
	return _OptimismPortal2.Contract.DisputeGameFactory(&_OptimismPortal2.CallOpts)
}

// DisputeGameFinalityDelaySeconds is a free data retrieval call binding the contract method 0x952b2797.
//
// Solidity: function disputeGameFinalityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Caller) DisputeGameFinalityDelaySeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "disputeGameFinalityDelaySeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DisputeGameFinalityDelaySeconds is a free data retrieval call binding the contract method 0x952b2797.
//
// Solidity: function disputeGameFinalityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Session) DisputeGameFinalityDelaySeconds() (*big.Int, error) {
	return _OptimismPortal2.Contract.DisputeGameFinalityDelaySeconds(&_OptimismPortal2.CallOpts)
}

// DisputeGameFinalityDelaySeconds is a free data retrieval call binding the contract method 0x952b2797.
//
// Solidity: function disputeGameFinalityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2CallerSession) DisputeGameFinalityDelaySeconds() (*big.Int, error) {
	return _OptimismPortal2.Contract.DisputeGameFinalityDelaySeconds(&_OptimismPortal2.CallOpts)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Caller) FinalizedWithdrawals(opts *bind.CallOpts, arg0 [32]byte) (bool, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "finalizedWithdrawals", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Session) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal2.Contract.FinalizedWithdrawals(&_OptimismPortal2.CallOpts, arg0)
}

// FinalizedWithdrawals is a free data retrieval call binding the contract method 0xa14238e7.
//
// Solidity: function finalizedWithdrawals(bytes32 ) view returns(bool)
func (_OptimismPortal2 *OptimismPortal2CallerSession) FinalizedWithdrawals(arg0 [32]byte) (bool, error) {
	return _OptimismPortal2.Contract.FinalizedWithdrawals(&_OptimismPortal2.CallOpts, arg0)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) Guardian(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "guardian")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) Guardian() (common.Address, error) {
	return _OptimismPortal2.Contract.Guardian(&_OptimismPortal2.CallOpts)
}

// Guardian is a free data retrieval call binding the contract method 0x452a9320.
//
// Solidity: function guardian() view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) Guardian() (common.Address, error) {
	return _OptimismPortal2.Contract.Guardian(&_OptimismPortal2.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) L2Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "l2Sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) L2Sender() (common.Address, error) {
	return _OptimismPortal2.Contract.L2Sender(&_OptimismPortal2.CallOpts)
}

// L2Sender is a free data retrieval call binding the contract method 0x9bf62d82.
//
// Solidity: function l2Sender() view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) L2Sender() (common.Address, error) {
	return _OptimismPortal2.Contract.L2Sender(&_OptimismPortal2.CallOpts)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal2 *OptimismPortal2Caller) MinimumGasLimit(opts *bind.CallOpts, _byteCount uint64) (uint64, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "minimumGasLimit", _byteCount)

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal2 *OptimismPortal2Session) MinimumGasLimit(_byteCount uint64) (uint64, error) {
	return _OptimismPortal2.Contract.MinimumGasLimit(&_OptimismPortal2.CallOpts, _byteCount)
}

// MinimumGasLimit is a free data retrieval call binding the contract method 0xa35d99df.
//
// Solidity: function minimumGasLimit(uint64 _byteCount) pure returns(uint64)
func (_OptimismPortal2 *OptimismPortal2CallerSession) MinimumGasLimit(_byteCount uint64) (uint64, error) {
	return _OptimismPortal2.Contract.MinimumGasLimit(&_OptimismPortal2.CallOpts, _byteCount)
}

// NumProofSubmitters is a free data retrieval call binding the contract method 0x513747ab.
//
// Solidity: function numProofSubmitters(bytes32 _withdrawalHash) view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Caller) NumProofSubmitters(opts *bind.CallOpts, _withdrawalHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "numProofSubmitters", _withdrawalHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumProofSubmitters is a free data retrieval call binding the contract method 0x513747ab.
//
// Solidity: function numProofSubmitters(bytes32 _withdrawalHash) view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Session) NumProofSubmitters(_withdrawalHash [32]byte) (*big.Int, error) {
	return _OptimismPortal2.Contract.NumProofSubmitters(&_OptimismPortal2.CallOpts, _withdrawalHash)
}

// NumProofSubmitters is a free data retrieval call binding the contract method 0x513747ab.
//
// Solidity: function numProofSubmitters(bytes32 _withdrawalHash) view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2CallerSession) NumProofSubmitters(_withdrawalHash [32]byte) (*big.Int, error) {
	return _OptimismPortal2.Contract.NumProofSubmitters(&_OptimismPortal2.CallOpts, _withdrawalHash)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal2 *OptimismPortal2Caller) Params(opts *bind.CallOpts) (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "params")

	outstruct := new(struct {
		PrevBaseFee   *big.Int
		PrevBoughtGas uint64
		PrevBlockNum  uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.PrevBaseFee = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.PrevBoughtGas = *abi.ConvertType(out[1], new(uint64)).(*uint64)
	outstruct.PrevBlockNum = *abi.ConvertType(out[2], new(uint64)).(*uint64)

	return *outstruct, err

}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal2 *OptimismPortal2Session) Params() (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	return _OptimismPortal2.Contract.Params(&_OptimismPortal2.CallOpts)
}

// Params is a free data retrieval call binding the contract method 0xcff0ab96.
//
// Solidity: function params() view returns(uint128 prevBaseFee, uint64 prevBoughtGas, uint64 prevBlockNum)
func (_OptimismPortal2 *OptimismPortal2CallerSession) Params() (struct {
	PrevBaseFee   *big.Int
	PrevBoughtGas uint64
	PrevBlockNum  uint64
}, error) {
	return _OptimismPortal2.Contract.Params(&_OptimismPortal2.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Caller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OptimismPortal2 *OptimismPortal2Session) Paused() (bool, error) {
	return _OptimismPortal2.Contract.Paused(&_OptimismPortal2.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_OptimismPortal2 *OptimismPortal2CallerSession) Paused() (bool, error) {
	return _OptimismPortal2.Contract.Paused(&_OptimismPortal2.CallOpts)
}

// ProofMaturityDelaySeconds is a free data retrieval call binding the contract method 0xbf653a5c.
//
// Solidity: function proofMaturityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Caller) ProofMaturityDelaySeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "proofMaturityDelaySeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ProofMaturityDelaySeconds is a free data retrieval call binding the contract method 0xbf653a5c.
//
// Solidity: function proofMaturityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2Session) ProofMaturityDelaySeconds() (*big.Int, error) {
	return _OptimismPortal2.Contract.ProofMaturityDelaySeconds(&_OptimismPortal2.CallOpts)
}

// ProofMaturityDelaySeconds is a free data retrieval call binding the contract method 0xbf653a5c.
//
// Solidity: function proofMaturityDelaySeconds() view returns(uint256)
func (_OptimismPortal2 *OptimismPortal2CallerSession) ProofMaturityDelaySeconds() (*big.Int, error) {
	return _OptimismPortal2.Contract.ProofMaturityDelaySeconds(&_OptimismPortal2.CallOpts)
}

// ProofSubmitters is a free data retrieval call binding the contract method 0xa3860f48.
//
// Solidity: function proofSubmitters(bytes32 , uint256 ) view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) ProofSubmitters(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "proofSubmitters", arg0, arg1)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ProofSubmitters is a free data retrieval call binding the contract method 0xa3860f48.
//
// Solidity: function proofSubmitters(bytes32 , uint256 ) view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) ProofSubmitters(arg0 [32]byte, arg1 *big.Int) (common.Address, error) {
	return _OptimismPortal2.Contract.ProofSubmitters(&_OptimismPortal2.CallOpts, arg0, arg1)
}

// ProofSubmitters is a free data retrieval call binding the contract method 0xa3860f48.
//
// Solidity: function proofSubmitters(bytes32 , uint256 ) view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) ProofSubmitters(arg0 [32]byte, arg1 *big.Int) (common.Address, error) {
	return _OptimismPortal2.Contract.ProofSubmitters(&_OptimismPortal2.CallOpts, arg0, arg1)
}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xbb2c727e.
//
// Solidity: function provenWithdrawals(bytes32 , address ) view returns(address disputeGameProxy, uint64 timestamp)
func (_OptimismPortal2 *OptimismPortal2Caller) ProvenWithdrawals(opts *bind.CallOpts, arg0 [32]byte, arg1 common.Address) (struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "provenWithdrawals", arg0, arg1)

	outstruct := new(struct {
		DisputeGameProxy common.Address
		Timestamp        uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.DisputeGameProxy = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Timestamp = *abi.ConvertType(out[1], new(uint64)).(*uint64)

	return *outstruct, err

}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xbb2c727e.
//
// Solidity: function provenWithdrawals(bytes32 , address ) view returns(address disputeGameProxy, uint64 timestamp)
func (_OptimismPortal2 *OptimismPortal2Session) ProvenWithdrawals(arg0 [32]byte, arg1 common.Address) (struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}, error) {
	return _OptimismPortal2.Contract.ProvenWithdrawals(&_OptimismPortal2.CallOpts, arg0, arg1)
}

// ProvenWithdrawals is a free data retrieval call binding the contract method 0xbb2c727e.
//
// Solidity: function provenWithdrawals(bytes32 , address ) view returns(address disputeGameProxy, uint64 timestamp)
func (_OptimismPortal2 *OptimismPortal2CallerSession) ProvenWithdrawals(arg0 [32]byte, arg1 common.Address) (struct {
	DisputeGameProxy common.Address
	Timestamp        uint64
}, error) {
	return _OptimismPortal2.Contract.ProvenWithdrawals(&_OptimismPortal2.CallOpts, arg0, arg1)
}

// RespectedGameType is a free data retrieval call binding the contract method 0x3c9f397c.
//
// Solidity: function respectedGameType() view returns(uint32)
func (_OptimismPortal2 *OptimismPortal2Caller) RespectedGameType(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "respectedGameType")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// RespectedGameType is a free data retrieval call binding the contract method 0x3c9f397c.
//
// Solidity: function respectedGameType() view returns(uint32)
func (_OptimismPortal2 *OptimismPortal2Session) RespectedGameType() (uint32, error) {
	return _OptimismPortal2.Contract.RespectedGameType(&_OptimismPortal2.CallOpts)
}

// RespectedGameType is a free data retrieval call binding the contract method 0x3c9f397c.
//
// Solidity: function respectedGameType() view returns(uint32)
func (_OptimismPortal2 *OptimismPortal2CallerSession) RespectedGameType() (uint32, error) {
	return _OptimismPortal2.Contract.RespectedGameType(&_OptimismPortal2.CallOpts)
}

// RespectedGameTypeUpdatedAt is a free data retrieval call binding the contract method 0x4fd0434c.
//
// Solidity: function respectedGameTypeUpdatedAt() view returns(uint64)
func (_OptimismPortal2 *OptimismPortal2Caller) RespectedGameTypeUpdatedAt(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "respectedGameTypeUpdatedAt")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// RespectedGameTypeUpdatedAt is a free data retrieval call binding the contract method 0x4fd0434c.
//
// Solidity: function respectedGameTypeUpdatedAt() view returns(uint64)
func (_OptimismPortal2 *OptimismPortal2Session) RespectedGameTypeUpdatedAt() (uint64, error) {
	return _OptimismPortal2.Contract.RespectedGameTypeUpdatedAt(&_OptimismPortal2.CallOpts)
}

// RespectedGameTypeUpdatedAt is a free data retrieval call binding the contract method 0x4fd0434c.
//
// Solidity: function respectedGameTypeUpdatedAt() view returns(uint64)
func (_OptimismPortal2 *OptimismPortal2CallerSession) RespectedGameTypeUpdatedAt() (uint64, error) {
	return _OptimismPortal2.Contract.RespectedGameTypeUpdatedAt(&_OptimismPortal2.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) SuperchainConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "superchainConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) SuperchainConfig() (common.Address, error) {
	return _OptimismPortal2.Contract.SuperchainConfig(&_OptimismPortal2.CallOpts)
}

// SuperchainConfig is a free data retrieval call binding the contract method 0x35e80ab3.
//
// Solidity: function superchainConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) SuperchainConfig() (common.Address, error) {
	return _OptimismPortal2.Contract.SuperchainConfig(&_OptimismPortal2.CallOpts)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Caller) SystemConfig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "systemConfig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2Session) SystemConfig() (common.Address, error) {
	return _OptimismPortal2.Contract.SystemConfig(&_OptimismPortal2.CallOpts)
}

// SystemConfig is a free data retrieval call binding the contract method 0x33d7e2bd.
//
// Solidity: function systemConfig() view returns(address)
func (_OptimismPortal2 *OptimismPortal2CallerSession) SystemConfig() (common.Address, error) {
	return _OptimismPortal2.Contract.SystemConfig(&_OptimismPortal2.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OptimismPortal2 *OptimismPortal2Caller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _OptimismPortal2.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OptimismPortal2 *OptimismPortal2Session) Version() (string, error) {
	return _OptimismPortal2.Contract.Version(&_OptimismPortal2.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() pure returns(string)
func (_OptimismPortal2 *OptimismPortal2CallerSession) Version() (string, error) {
	return _OptimismPortal2.Contract.Version(&_OptimismPortal2.CallOpts)
}

// BlacklistDisputeGame is a paid mutator transaction binding the contract method 0x7d6be8dc.
//
// Solidity: function blacklistDisputeGame(address _disputeGame) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) BlacklistDisputeGame(opts *bind.TransactOpts, _disputeGame common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "blacklistDisputeGame", _disputeGame)
}

// BlacklistDisputeGame is a paid mutator transaction binding the contract method 0x7d6be8dc.
//
// Solidity: function blacklistDisputeGame(address _disputeGame) returns()
func (_OptimismPortal2 *OptimismPortal2Session) BlacklistDisputeGame(_disputeGame common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.BlacklistDisputeGame(&_OptimismPortal2.TransactOpts, _disputeGame)
}

// BlacklistDisputeGame is a paid mutator transaction binding the contract method 0x7d6be8dc.
//
// Solidity: function blacklistDisputeGame(address _disputeGame) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) BlacklistDisputeGame(_disputeGame common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.BlacklistDisputeGame(&_OptimismPortal2.TransactOpts, _disputeGame)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) DepositERC20Transaction(opts *bind.TransactOpts, _to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "depositERC20Transaction", _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal2 *OptimismPortal2Session) DepositERC20Transaction(_to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DepositERC20Transaction(&_OptimismPortal2.TransactOpts, _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositERC20Transaction is a paid mutator transaction binding the contract method 0x149f2f22.
//
// Solidity: function depositERC20Transaction(address _to, uint256 _mint, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) DepositERC20Transaction(_to common.Address, _mint *big.Int, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DepositERC20Transaction(&_OptimismPortal2.TransactOpts, _to, _mint, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) DepositTransaction(opts *bind.TransactOpts, _to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "depositTransaction", _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal2 *OptimismPortal2Session) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DepositTransaction(&_OptimismPortal2.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DepositTransaction is a paid mutator transaction binding the contract method 0xe9e05c42.
//
// Solidity: function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes _data) payable returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) DepositTransaction(_to common.Address, _value *big.Int, _gasLimit uint64, _isCreation bool, _data []byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DepositTransaction(&_OptimismPortal2.TransactOpts, _to, _value, _gasLimit, _isCreation, _data)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) DonateETH(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "donateETH")
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal2 *OptimismPortal2Session) DonateETH() (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DonateETH(&_OptimismPortal2.TransactOpts)
}

// DonateETH is a paid mutator transaction binding the contract method 0x8b4c40b0.
//
// Solidity: function donateETH() payable returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) DonateETH() (*types.Transaction, error) {
	return _OptimismPortal2.Contract.DonateETH(&_OptimismPortal2.TransactOpts)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) FinalizeWithdrawalTransaction(opts *bind.TransactOpts, _tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "finalizeWithdrawalTransaction", _tx)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal2 *OptimismPortal2Session) FinalizeWithdrawalTransaction(_tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal2.TransactOpts, _tx)
}

// FinalizeWithdrawalTransaction is a paid mutator transaction binding the contract method 0x8c3152e9.
//
// Solidity: function finalizeWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) FinalizeWithdrawalTransaction(_tx TypesWithdrawalTransaction) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.FinalizeWithdrawalTransaction(&_OptimismPortal2.TransactOpts, _tx)
}

// FinalizeWithdrawalTransactionExternalProof is a paid mutator transaction binding the contract method 0x43ca1c50.
//
// Solidity: function finalizeWithdrawalTransactionExternalProof((uint256,address,address,uint256,uint256,bytes) _tx, address _proofSubmitter) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) FinalizeWithdrawalTransactionExternalProof(opts *bind.TransactOpts, _tx TypesWithdrawalTransaction, _proofSubmitter common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "finalizeWithdrawalTransactionExternalProof", _tx, _proofSubmitter)
}

// FinalizeWithdrawalTransactionExternalProof is a paid mutator transaction binding the contract method 0x43ca1c50.
//
// Solidity: function finalizeWithdrawalTransactionExternalProof((uint256,address,address,uint256,uint256,bytes) _tx, address _proofSubmitter) returns()
func (_OptimismPortal2 *OptimismPortal2Session) FinalizeWithdrawalTransactionExternalProof(_tx TypesWithdrawalTransaction, _proofSubmitter common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.FinalizeWithdrawalTransactionExternalProof(&_OptimismPortal2.TransactOpts, _tx, _proofSubmitter)
}

// FinalizeWithdrawalTransactionExternalProof is a paid mutator transaction binding the contract method 0x43ca1c50.
//
// Solidity: function finalizeWithdrawalTransactionExternalProof((uint256,address,address,uint256,uint256,bytes) _tx, address _proofSubmitter) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) FinalizeWithdrawalTransactionExternalProof(_tx TypesWithdrawalTransaction, _proofSubmitter common.Address) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.FinalizeWithdrawalTransactionExternalProof(&_OptimismPortal2.TransactOpts, _tx, _proofSubmitter)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e819e54.
//
// Solidity: function initialize(address _disputeGameFactory, address _systemConfig, address _superchainConfig, uint32 _initialRespectedGameType) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) Initialize(opts *bind.TransactOpts, _disputeGameFactory common.Address, _systemConfig common.Address, _superchainConfig common.Address, _initialRespectedGameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "initialize", _disputeGameFactory, _systemConfig, _superchainConfig, _initialRespectedGameType)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e819e54.
//
// Solidity: function initialize(address _disputeGameFactory, address _systemConfig, address _superchainConfig, uint32 _initialRespectedGameType) returns()
func (_OptimismPortal2 *OptimismPortal2Session) Initialize(_disputeGameFactory common.Address, _systemConfig common.Address, _superchainConfig common.Address, _initialRespectedGameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.Initialize(&_OptimismPortal2.TransactOpts, _disputeGameFactory, _systemConfig, _superchainConfig, _initialRespectedGameType)
}

// Initialize is a paid mutator transaction binding the contract method 0x8e819e54.
//
// Solidity: function initialize(address _disputeGameFactory, address _systemConfig, address _superchainConfig, uint32 _initialRespectedGameType) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) Initialize(_disputeGameFactory common.Address, _systemConfig common.Address, _superchainConfig common.Address, _initialRespectedGameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.Initialize(&_OptimismPortal2.TransactOpts, _disputeGameFactory, _systemConfig, _superchainConfig, _initialRespectedGameType)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _disputeGameIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) ProveWithdrawalTransaction(opts *bind.TransactOpts, _tx TypesWithdrawalTransaction, _disputeGameIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "proveWithdrawalTransaction", _tx, _disputeGameIndex, _outputRootProof, _withdrawalProof)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _disputeGameIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal2 *OptimismPortal2Session) ProveWithdrawalTransaction(_tx TypesWithdrawalTransaction, _disputeGameIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.ProveWithdrawalTransaction(&_OptimismPortal2.TransactOpts, _tx, _disputeGameIndex, _outputRootProof, _withdrawalProof)
}

// ProveWithdrawalTransaction is a paid mutator transaction binding the contract method 0x4870496f.
//
// Solidity: function proveWithdrawalTransaction((uint256,address,address,uint256,uint256,bytes) _tx, uint256 _disputeGameIndex, (bytes32,bytes32,bytes32,bytes32) _outputRootProof, bytes[] _withdrawalProof) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) ProveWithdrawalTransaction(_tx TypesWithdrawalTransaction, _disputeGameIndex *big.Int, _outputRootProof TypesOutputRootProof, _withdrawalProof [][]byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.ProveWithdrawalTransaction(&_OptimismPortal2.TransactOpts, _tx, _disputeGameIndex, _outputRootProof, _withdrawalProof)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) SetGasPayingToken(opts *bind.TransactOpts, _token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "setGasPayingToken", _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal2 *OptimismPortal2Session) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.SetGasPayingToken(&_OptimismPortal2.TransactOpts, _token, _decimals, _name, _symbol)
}

// SetGasPayingToken is a paid mutator transaction binding the contract method 0x71cfaa3f.
//
// Solidity: function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) SetGasPayingToken(_token common.Address, _decimals uint8, _name [32]byte, _symbol [32]byte) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.SetGasPayingToken(&_OptimismPortal2.TransactOpts, _token, _decimals, _name, _symbol)
}

// SetRespectedGameType is a paid mutator transaction binding the contract method 0x7fc48504.
//
// Solidity: function setRespectedGameType(uint32 _gameType) returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) SetRespectedGameType(opts *bind.TransactOpts, _gameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.contract.Transact(opts, "setRespectedGameType", _gameType)
}

// SetRespectedGameType is a paid mutator transaction binding the contract method 0x7fc48504.
//
// Solidity: function setRespectedGameType(uint32 _gameType) returns()
func (_OptimismPortal2 *OptimismPortal2Session) SetRespectedGameType(_gameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.SetRespectedGameType(&_OptimismPortal2.TransactOpts, _gameType)
}

// SetRespectedGameType is a paid mutator transaction binding the contract method 0x7fc48504.
//
// Solidity: function setRespectedGameType(uint32 _gameType) returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) SetRespectedGameType(_gameType uint32) (*types.Transaction, error) {
	return _OptimismPortal2.Contract.SetRespectedGameType(&_OptimismPortal2.TransactOpts, _gameType)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal2 *OptimismPortal2Transactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OptimismPortal2.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal2 *OptimismPortal2Session) Receive() (*types.Transaction, error) {
	return _OptimismPortal2.Contract.Receive(&_OptimismPortal2.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_OptimismPortal2 *OptimismPortal2TransactorSession) Receive() (*types.Transaction, error) {
	return _OptimismPortal2.Contract.Receive(&_OptimismPortal2.TransactOpts)
}

// OptimismPortal2DisputeGameBlacklistedIterator is returned from FilterDisputeGameBlacklisted and is used to iterate over the raw logs and unpacked data for DisputeGameBlacklisted events raised by the OptimismPortal2 contract.
type OptimismPortal2DisputeGameBlacklistedIterator struct {
	Event *OptimismPortal2DisputeGameBlacklisted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2DisputeGameBlacklistedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2DisputeGameBlacklisted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2DisputeGameBlacklisted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2DisputeGameBlacklistedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2DisputeGameBlacklistedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2DisputeGameBlacklisted represents a DisputeGameBlacklisted event raised by the OptimismPortal2 contract.
type OptimismPortal2DisputeGameBlacklisted struct {
	DisputeGame common.Address
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterDisputeGameBlacklisted is a free log retrieval operation binding the contract event 0x192c289026d59a41a27f5aea08f3969b57931b0589202d14f4368cded95d3cda.
//
// Solidity: event DisputeGameBlacklisted(address indexed disputeGame)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterDisputeGameBlacklisted(opts *bind.FilterOpts, disputeGame []common.Address) (*OptimismPortal2DisputeGameBlacklistedIterator, error) {

	var disputeGameRule []interface{}
	for _, disputeGameItem := range disputeGame {
		disputeGameRule = append(disputeGameRule, disputeGameItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "DisputeGameBlacklisted", disputeGameRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2DisputeGameBlacklistedIterator{contract: _OptimismPortal2.contract, event: "DisputeGameBlacklisted", logs: logs, sub: sub}, nil
}

// WatchDisputeGameBlacklisted is a free log subscription operation binding the contract event 0x192c289026d59a41a27f5aea08f3969b57931b0589202d14f4368cded95d3cda.
//
// Solidity: event DisputeGameBlacklisted(address indexed disputeGame)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchDisputeGameBlacklisted(opts *bind.WatchOpts, sink chan<- *OptimismPortal2DisputeGameBlacklisted, disputeGame []common.Address) (event.Subscription, error) {

	var disputeGameRule []interface{}
	for _, disputeGameItem := range disputeGame {
		disputeGameRule = append(disputeGameRule, disputeGameItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "DisputeGameBlacklisted", disputeGameRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2DisputeGameBlacklisted)
				if err := _OptimismPortal2.contract.UnpackLog(event, "DisputeGameBlacklisted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDisputeGameBlacklisted is a log parse operation binding the contract event 0x192c289026d59a41a27f5aea08f3969b57931b0589202d14f4368cded95d3cda.
//
// Solidity: event DisputeGameBlacklisted(address indexed disputeGame)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseDisputeGameBlacklisted(log types.Log) (*OptimismPortal2DisputeGameBlacklisted, error) {
	event := new(OptimismPortal2DisputeGameBlacklisted)
	if err := _OptimismPortal2.contract.UnpackLog(event, "DisputeGameBlacklisted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2InitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OptimismPortal2 contract.
type OptimismPortal2InitializedIterator struct {
	Event *OptimismPortal2Initialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2InitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2Initialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2Initialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2InitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2InitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2Initialized represents a Initialized event raised by the OptimismPortal2 contract.
type OptimismPortal2Initialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterInitialized(opts *bind.FilterOpts) (*OptimismPortal2InitializedIterator, error) {

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2InitializedIterator{contract: _OptimismPortal2.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OptimismPortal2Initialized) (event.Subscription, error) {

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2Initialized)
				if err := _OptimismPortal2.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseInitialized(log types.Log) (*OptimismPortal2Initialized, error) {
	event := new(OptimismPortal2Initialized)
	if err := _OptimismPortal2.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2RespectedGameTypeSetIterator is returned from FilterRespectedGameTypeSet and is used to iterate over the raw logs and unpacked data for RespectedGameTypeSet events raised by the OptimismPortal2 contract.
type OptimismPortal2RespectedGameTypeSetIterator struct {
	Event *OptimismPortal2RespectedGameTypeSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2RespectedGameTypeSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2RespectedGameTypeSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2RespectedGameTypeSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2RespectedGameTypeSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2RespectedGameTypeSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2RespectedGameTypeSet represents a RespectedGameTypeSet event raised by the OptimismPortal2 contract.
type OptimismPortal2RespectedGameTypeSet struct {
	NewGameType uint32
	UpdatedAt   uint64
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterRespectedGameTypeSet is a free log retrieval operation binding the contract event 0x049fe9dd413cdf037cce27011cc1790c753118272f3630e6e8bdfa5e82081760.
//
// Solidity: event RespectedGameTypeSet(uint32 indexed newGameType, uint64 indexed updatedAt)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterRespectedGameTypeSet(opts *bind.FilterOpts, newGameType []uint32, updatedAt []uint64) (*OptimismPortal2RespectedGameTypeSetIterator, error) {

	var newGameTypeRule []interface{}
	for _, newGameTypeItem := range newGameType {
		newGameTypeRule = append(newGameTypeRule, newGameTypeItem)
	}
	var updatedAtRule []interface{}
	for _, updatedAtItem := range updatedAt {
		updatedAtRule = append(updatedAtRule, updatedAtItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "RespectedGameTypeSet", newGameTypeRule, updatedAtRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2RespectedGameTypeSetIterator{contract: _OptimismPortal2.contract, event: "RespectedGameTypeSet", logs: logs, sub: sub}, nil
}

// WatchRespectedGameTypeSet is a free log subscription operation binding the contract event 0x049fe9dd413cdf037cce27011cc1790c753118272f3630e6e8bdfa5e82081760.
//
// Solidity: event RespectedGameTypeSet(uint32 indexed newGameType, uint64 indexed updatedAt)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchRespectedGameTypeSet(opts *bind.WatchOpts, sink chan<- *OptimismPortal2RespectedGameTypeSet, newGameType []uint32, updatedAt []uint64) (event.Subscription, error) {

	var newGameTypeRule []interface{}
	for _, newGameTypeItem := range newGameType {
		newGameTypeRule = append(newGameTypeRule, newGameTypeItem)
	}
	var updatedAtRule []interface{}
	for _, updatedAtItem := range updatedAt {
		updatedAtRule = append(updatedAtRule, updatedAtItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "RespectedGameTypeSet", newGameTypeRule, updatedAtRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2RespectedGameTypeSet)
				if err := _OptimismPortal2.contract.UnpackLog(event, "RespectedGameTypeSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRespectedGameTypeSet is a log parse operation binding the contract event 0x049fe9dd413cdf037cce27011cc1790c753118272f3630e6e8bdfa5e82081760.
//
// Solidity: event RespectedGameTypeSet(uint32 indexed newGameType, uint64 indexed updatedAt)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseRespectedGameTypeSet(log types.Log) (*OptimismPortal2RespectedGameTypeSet, error) {
	event := new(OptimismPortal2RespectedGameTypeSet)
	if err := _OptimismPortal2.contract.UnpackLog(event, "RespectedGameTypeSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2TransactionDepositedIterator is returned from FilterTransactionDeposited and is used to iterate over the raw logs and unpacked data for TransactionDeposited events raised by the OptimismPortal2 contract.
type OptimismPortal2TransactionDepositedIterator struct {
	Event *OptimismPortal2TransactionDeposited // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2TransactionDepositedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2TransactionDeposited)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2TransactionDeposited)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2TransactionDepositedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2TransactionDepositedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2TransactionDeposited represents a TransactionDeposited event raised by the OptimismPortal2 contract.
type OptimismPortal2TransactionDeposited struct {
	From       common.Address
	To         common.Address
	Version    *big.Int
	OpaqueData []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterTransactionDeposited is a free log retrieval operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterTransactionDeposited(opts *bind.FilterOpts, from []common.Address, to []common.Address, version []*big.Int) (*OptimismPortal2TransactionDepositedIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "TransactionDeposited", fromRule, toRule, versionRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2TransactionDepositedIterator{contract: _OptimismPortal2.contract, event: "TransactionDeposited", logs: logs, sub: sub}, nil
}

// WatchTransactionDeposited is a free log subscription operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchTransactionDeposited(opts *bind.WatchOpts, sink chan<- *OptimismPortal2TransactionDeposited, from []common.Address, to []common.Address, version []*big.Int) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}
	var versionRule []interface{}
	for _, versionItem := range version {
		versionRule = append(versionRule, versionItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "TransactionDeposited", fromRule, toRule, versionRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2TransactionDeposited)
				if err := _OptimismPortal2.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransactionDeposited is a log parse operation binding the contract event 0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32.
//
// Solidity: event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseTransactionDeposited(log types.Log) (*OptimismPortal2TransactionDeposited, error) {
	event := new(OptimismPortal2TransactionDeposited)
	if err := _OptimismPortal2.contract.UnpackLog(event, "TransactionDeposited", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2WithdrawalFinalizedIterator is returned from FilterWithdrawalFinalized and is used to iterate over the raw logs and unpacked data for WithdrawalFinalized events raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalFinalizedIterator struct {
	Event *OptimismPortal2WithdrawalFinalized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2WithdrawalFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2WithdrawalFinalized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2WithdrawalFinalized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2WithdrawalFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2WithdrawalFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2WithdrawalFinalized represents a WithdrawalFinalized event raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalFinalized struct {
	WithdrawalHash [32]byte
	Success        bool
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalFinalized is a free log retrieval operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterWithdrawalFinalized(opts *bind.FilterOpts, withdrawalHash [][32]byte) (*OptimismPortal2WithdrawalFinalizedIterator, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "WithdrawalFinalized", withdrawalHashRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2WithdrawalFinalizedIterator{contract: _OptimismPortal2.contract, event: "WithdrawalFinalized", logs: logs, sub: sub}, nil
}

// WatchWithdrawalFinalized is a free log subscription operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchWithdrawalFinalized(opts *bind.WatchOpts, sink chan<- *OptimismPortal2WithdrawalFinalized, withdrawalHash [][32]byte) (event.Subscription, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "WithdrawalFinalized", withdrawalHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2WithdrawalFinalized)
				if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalFinalized is a log parse operation binding the contract event 0xdb5c7652857aa163daadd670e116628fb42e869d8ac4251ef8971d9e5727df1b.
//
// Solidity: event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseWithdrawalFinalized(log types.Log) (*OptimismPortal2WithdrawalFinalized, error) {
	event := new(OptimismPortal2WithdrawalFinalized)
	if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2WithdrawalProvenIterator is returned from FilterWithdrawalProven and is used to iterate over the raw logs and unpacked data for WithdrawalProven events raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalProvenIterator struct {
	Event *OptimismPortal2WithdrawalProven // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2WithdrawalProvenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2WithdrawalProven)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2WithdrawalProven)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2WithdrawalProvenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2WithdrawalProvenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2WithdrawalProven represents a WithdrawalProven event raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalProven struct {
	WithdrawalHash [32]byte
	From           common.Address
	To             common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalProven is a free log retrieval operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterWithdrawalProven(opts *bind.FilterOpts, withdrawalHash [][32]byte, from []common.Address, to []common.Address) (*OptimismPortal2WithdrawalProvenIterator, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "WithdrawalProven", withdrawalHashRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2WithdrawalProvenIterator{contract: _OptimismPortal2.contract, event: "WithdrawalProven", logs: logs, sub: sub}, nil
}

// WatchWithdrawalProven is a free log subscription operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchWithdrawalProven(opts *bind.WatchOpts, sink chan<- *OptimismPortal2WithdrawalProven, withdrawalHash [][32]byte, from []common.Address, to []common.Address) (event.Subscription, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "WithdrawalProven", withdrawalHashRule, fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2WithdrawalProven)
				if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalProven", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalProven is a log parse operation binding the contract event 0x67a6208cfcc0801d50f6cbe764733f4fddf66ac0b04442061a8a8c0cb6b63f62.
//
// Solidity: event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseWithdrawalProven(log types.Log) (*OptimismPortal2WithdrawalProven, error) {
	event := new(OptimismPortal2WithdrawalProven)
	if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalProven", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OptimismPortal2WithdrawalProvenExtension1Iterator is returned from FilterWithdrawalProvenExtension1 and is used to iterate over the raw logs and unpacked data for WithdrawalProvenExtension1 events raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalProvenExtension1Iterator struct {
	Event *OptimismPortal2WithdrawalProvenExtension1 // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *OptimismPortal2WithdrawalProvenExtension1Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OptimismPortal2WithdrawalProvenExtension1)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(OptimismPortal2WithdrawalProvenExtension1)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *OptimismPortal2WithdrawalProvenExtension1Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OptimismPortal2WithdrawalProvenExtension1Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OptimismPortal2WithdrawalProvenExtension1 represents a WithdrawalProvenExtension1 event raised by the OptimismPortal2 contract.
type OptimismPortal2WithdrawalProvenExtension1 struct {
	WithdrawalHash [32]byte
	ProofSubmitter common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalProvenExtension1 is a free log retrieval operation binding the contract event 0x798f9f13695f8f045aa5f80ed8efebb695f3c7fe65da381969f2f28bf3c60b97.
//
// Solidity: event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter)
func (_OptimismPortal2 *OptimismPortal2Filterer) FilterWithdrawalProvenExtension1(opts *bind.FilterOpts, withdrawalHash [][32]byte, proofSubmitter []common.Address) (*OptimismPortal2WithdrawalProvenExtension1Iterator, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var proofSubmitterRule []interface{}
	for _, proofSubmitterItem := range proofSubmitter {
		proofSubmitterRule = append(proofSubmitterRule, proofSubmitterItem)
	}

	logs, sub, err := _OptimismPortal2.contract.FilterLogs(opts, "WithdrawalProvenExtension1", withdrawalHashRule, proofSubmitterRule)
	if err != nil {
		return nil, err
	}
	return &OptimismPortal2WithdrawalProvenExtension1Iterator{contract: _OptimismPortal2.contract, event: "WithdrawalProvenExtension1", logs: logs, sub: sub}, nil
}

// WatchWithdrawalProvenExtension1 is a free log subscription operation binding the contract event 0x798f9f13695f8f045aa5f80ed8efebb695f3c7fe65da381969f2f28bf3c60b97.
//
// Solidity: event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter)
func (_OptimismPortal2 *OptimismPortal2Filterer) WatchWithdrawalProvenExtension1(opts *bind.WatchOpts, sink chan<- *OptimismPortal2WithdrawalProvenExtension1, withdrawalHash [][32]byte, proofSubmitter []common.Address) (event.Subscription, error) {

	var withdrawalHashRule []interface{}
	for _, withdrawalHashItem := range withdrawalHash {
		withdrawalHashRule = append(withdrawalHashRule, withdrawalHashItem)
	}
	var proofSubmitterRule []interface{}
	for _, proofSubmitterItem := range proofSubmitter {
		proofSubmitterRule = append(proofSubmitterRule, proofSubmitterItem)
	}

	logs, sub, err := _OptimismPortal2.contract.WatchLogs(opts, "WithdrawalProvenExtension1", withdrawalHashRule, proofSubmitterRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OptimismPortal2WithdrawalProvenExtension1)
				if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalProvenExtension1", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalProvenExtension1 is a log parse operation binding the contract event 0x798f9f13695f8f045aa5f80ed8efebb695f3c7fe65da381969f2f28bf3c60b97.
//
// Solidity: event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter)
func (_OptimismPortal2 *OptimismPortal2Filterer) ParseWithdrawalProvenExtension1(log types.Log) (*OptimismPortal2WithdrawalProvenExtension1, error) {
	event := new(OptimismPortal2WithdrawalProvenExtension1)
	if err := _OptimismPortal2.contract.UnpackLog(event, "WithdrawalProvenExtension1", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
