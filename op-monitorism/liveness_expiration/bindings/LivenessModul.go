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

// LivenessModuleMetaData contains all meta data concerning the LivenessModule contract.
var LivenessModuleMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_safe\",\"type\":\"address\",\"internalType\":\"contractGnosisSafe\"},{\"name\":\"_livenessGuard\",\"type\":\"address\",\"internalType\":\"contractLivenessGuard\"},{\"name\":\"_livenessInterval\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_minOwners\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_thresholdPercentage\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_fallbackOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"canRemove\",\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"canRemove_\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"fallbackOwner\",\"inputs\":[],\"outputs\":[{\"name\":\"fallbackOwner_\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getRequiredThreshold\",\"inputs\":[{\"name\":\"_numOwners\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"threshold_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"livenessGuard\",\"inputs\":[],\"outputs\":[{\"name\":\"livenessGuard_\",\"type\":\"address\",\"internalType\":\"contractLivenessGuard\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"livenessInterval\",\"inputs\":[],\"outputs\":[{\"name\":\"livenessInterval_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"minOwners\",\"inputs\":[],\"outputs\":[{\"name\":\"minOwners_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"ownershipTransferredToFallback\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"removeOwners\",\"inputs\":[{\"name\":\"_previousOwners\",\"type\":\"address[]\",\"internalType\":\"address[]\"},{\"name\":\"_ownersToRemove\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"safe\",\"inputs\":[],\"outputs\":[{\"name\":\"safe_\",\"type\":\"address\",\"internalType\":\"contractGnosisSafe\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"thresholdPercentage\",\"inputs\":[],\"outputs\":[{\"name\":\"thresholdPercentage_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"OwnershipTransferredToFallback\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"RemovedOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"OwnerRemovalFailed\",\"inputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}]}]",
}

// LivenessModuleABI is the input ABI used to generate the binding from.
// Deprecated: Use LivenessModuleMetaData.ABI instead.
var LivenessModuleABI = LivenessModuleMetaData.ABI

// LivenessModule is an auto generated Go binding around an Ethereum contract.
type LivenessModule struct {
	LivenessModuleCaller     // Read-only binding to the contract
	LivenessModuleTransactor // Write-only binding to the contract
	LivenessModuleFilterer   // Log filterer for contract events
}

// LivenessModuleCaller is an auto generated read-only Go binding around an Ethereum contract.
type LivenessModuleCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessModuleTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LivenessModuleTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessModuleFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LivenessModuleFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessModuleSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LivenessModuleSession struct {
	Contract     *LivenessModule   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LivenessModuleCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LivenessModuleCallerSession struct {
	Contract *LivenessModuleCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// LivenessModuleTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LivenessModuleTransactorSession struct {
	Contract     *LivenessModuleTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// LivenessModuleRaw is an auto generated low-level Go binding around an Ethereum contract.
type LivenessModuleRaw struct {
	Contract *LivenessModule // Generic contract binding to access the raw methods on
}

// LivenessModuleCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LivenessModuleCallerRaw struct {
	Contract *LivenessModuleCaller // Generic read-only contract binding to access the raw methods on
}

// LivenessModuleTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LivenessModuleTransactorRaw struct {
	Contract *LivenessModuleTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLivenessModule creates a new instance of LivenessModule, bound to a specific deployed contract.
func NewLivenessModule(address common.Address, backend bind.ContractBackend) (*LivenessModule, error) {
	contract, err := bindLivenessModule(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LivenessModule{LivenessModuleCaller: LivenessModuleCaller{contract: contract}, LivenessModuleTransactor: LivenessModuleTransactor{contract: contract}, LivenessModuleFilterer: LivenessModuleFilterer{contract: contract}}, nil
}

// NewLivenessModuleCaller creates a new read-only instance of LivenessModule, bound to a specific deployed contract.
func NewLivenessModuleCaller(address common.Address, caller bind.ContractCaller) (*LivenessModuleCaller, error) {
	contract, err := bindLivenessModule(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LivenessModuleCaller{contract: contract}, nil
}

// NewLivenessModuleTransactor creates a new write-only instance of LivenessModule, bound to a specific deployed contract.
func NewLivenessModuleTransactor(address common.Address, transactor bind.ContractTransactor) (*LivenessModuleTransactor, error) {
	contract, err := bindLivenessModule(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LivenessModuleTransactor{contract: contract}, nil
}

// NewLivenessModuleFilterer creates a new log filterer instance of LivenessModule, bound to a specific deployed contract.
func NewLivenessModuleFilterer(address common.Address, filterer bind.ContractFilterer) (*LivenessModuleFilterer, error) {
	contract, err := bindLivenessModule(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LivenessModuleFilterer{contract: contract}, nil
}

// bindLivenessModule binds a generic wrapper to an already deployed contract.
func bindLivenessModule(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LivenessModuleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LivenessModule *LivenessModuleRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LivenessModule.Contract.LivenessModuleCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LivenessModule *LivenessModuleRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LivenessModule.Contract.LivenessModuleTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LivenessModule *LivenessModuleRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LivenessModule.Contract.LivenessModuleTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LivenessModule *LivenessModuleCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LivenessModule.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LivenessModule *LivenessModuleTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LivenessModule.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LivenessModule *LivenessModuleTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LivenessModule.Contract.contract.Transact(opts, method, params...)
}

// CanRemove is a free data retrieval call binding the contract method 0xd45996f1.
//
// Solidity: function canRemove(address _owner) view returns(bool canRemove_)
func (_LivenessModule *LivenessModuleCaller) CanRemove(opts *bind.CallOpts, _owner common.Address) (bool, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "canRemove", _owner)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// CanRemove is a free data retrieval call binding the contract method 0xd45996f1.
//
// Solidity: function canRemove(address _owner) view returns(bool canRemove_)
func (_LivenessModule *LivenessModuleSession) CanRemove(_owner common.Address) (bool, error) {
	return _LivenessModule.Contract.CanRemove(&_LivenessModule.CallOpts, _owner)
}

// CanRemove is a free data retrieval call binding the contract method 0xd45996f1.
//
// Solidity: function canRemove(address _owner) view returns(bool canRemove_)
func (_LivenessModule *LivenessModuleCallerSession) CanRemove(_owner common.Address) (bool, error) {
	return _LivenessModule.Contract.CanRemove(&_LivenessModule.CallOpts, _owner)
}

// FallbackOwner is a free data retrieval call binding the contract method 0x602b263b.
//
// Solidity: function fallbackOwner() view returns(address fallbackOwner_)
func (_LivenessModule *LivenessModuleCaller) FallbackOwner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "fallbackOwner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FallbackOwner is a free data retrieval call binding the contract method 0x602b263b.
//
// Solidity: function fallbackOwner() view returns(address fallbackOwner_)
func (_LivenessModule *LivenessModuleSession) FallbackOwner() (common.Address, error) {
	return _LivenessModule.Contract.FallbackOwner(&_LivenessModule.CallOpts)
}

// FallbackOwner is a free data retrieval call binding the contract method 0x602b263b.
//
// Solidity: function fallbackOwner() view returns(address fallbackOwner_)
func (_LivenessModule *LivenessModuleCallerSession) FallbackOwner() (common.Address, error) {
	return _LivenessModule.Contract.FallbackOwner(&_LivenessModule.CallOpts)
}

// GetRequiredThreshold is a free data retrieval call binding the contract method 0x86644d1e.
//
// Solidity: function getRequiredThreshold(uint256 _numOwners) view returns(uint256 threshold_)
func (_LivenessModule *LivenessModuleCaller) GetRequiredThreshold(opts *bind.CallOpts, _numOwners *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "getRequiredThreshold", _numOwners)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRequiredThreshold is a free data retrieval call binding the contract method 0x86644d1e.
//
// Solidity: function getRequiredThreshold(uint256 _numOwners) view returns(uint256 threshold_)
func (_LivenessModule *LivenessModuleSession) GetRequiredThreshold(_numOwners *big.Int) (*big.Int, error) {
	return _LivenessModule.Contract.GetRequiredThreshold(&_LivenessModule.CallOpts, _numOwners)
}

// GetRequiredThreshold is a free data retrieval call binding the contract method 0x86644d1e.
//
// Solidity: function getRequiredThreshold(uint256 _numOwners) view returns(uint256 threshold_)
func (_LivenessModule *LivenessModuleCallerSession) GetRequiredThreshold(_numOwners *big.Int) (*big.Int, error) {
	return _LivenessModule.Contract.GetRequiredThreshold(&_LivenessModule.CallOpts, _numOwners)
}

// LivenessGuard is a free data retrieval call binding the contract method 0xbe81c694.
//
// Solidity: function livenessGuard() view returns(address livenessGuard_)
func (_LivenessModule *LivenessModuleCaller) LivenessGuard(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "livenessGuard")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// LivenessGuard is a free data retrieval call binding the contract method 0xbe81c694.
//
// Solidity: function livenessGuard() view returns(address livenessGuard_)
func (_LivenessModule *LivenessModuleSession) LivenessGuard() (common.Address, error) {
	return _LivenessModule.Contract.LivenessGuard(&_LivenessModule.CallOpts)
}

// LivenessGuard is a free data retrieval call binding the contract method 0xbe81c694.
//
// Solidity: function livenessGuard() view returns(address livenessGuard_)
func (_LivenessModule *LivenessModuleCallerSession) LivenessGuard() (common.Address, error) {
	return _LivenessModule.Contract.LivenessGuard(&_LivenessModule.CallOpts)
}

// LivenessInterval is a free data retrieval call binding the contract method 0x38af7c5c.
//
// Solidity: function livenessInterval() view returns(uint256 livenessInterval_)
func (_LivenessModule *LivenessModuleCaller) LivenessInterval(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "livenessInterval")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LivenessInterval is a free data retrieval call binding the contract method 0x38af7c5c.
//
// Solidity: function livenessInterval() view returns(uint256 livenessInterval_)
func (_LivenessModule *LivenessModuleSession) LivenessInterval() (*big.Int, error) {
	return _LivenessModule.Contract.LivenessInterval(&_LivenessModule.CallOpts)
}

// LivenessInterval is a free data retrieval call binding the contract method 0x38af7c5c.
//
// Solidity: function livenessInterval() view returns(uint256 livenessInterval_)
func (_LivenessModule *LivenessModuleCallerSession) LivenessInterval() (*big.Int, error) {
	return _LivenessModule.Contract.LivenessInterval(&_LivenessModule.CallOpts)
}

// MinOwners is a free data retrieval call binding the contract method 0x4b810d3e.
//
// Solidity: function minOwners() view returns(uint256 minOwners_)
func (_LivenessModule *LivenessModuleCaller) MinOwners(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "minOwners")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinOwners is a free data retrieval call binding the contract method 0x4b810d3e.
//
// Solidity: function minOwners() view returns(uint256 minOwners_)
func (_LivenessModule *LivenessModuleSession) MinOwners() (*big.Int, error) {
	return _LivenessModule.Contract.MinOwners(&_LivenessModule.CallOpts)
}

// MinOwners is a free data retrieval call binding the contract method 0x4b810d3e.
//
// Solidity: function minOwners() view returns(uint256 minOwners_)
func (_LivenessModule *LivenessModuleCallerSession) MinOwners() (*big.Int, error) {
	return _LivenessModule.Contract.MinOwners(&_LivenessModule.CallOpts)
}

// OwnershipTransferredToFallback is a free data retrieval call binding the contract method 0x5b2e9c03.
//
// Solidity: function ownershipTransferredToFallback() view returns(bool)
func (_LivenessModule *LivenessModuleCaller) OwnershipTransferredToFallback(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "ownershipTransferredToFallback")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// OwnershipTransferredToFallback is a free data retrieval call binding the contract method 0x5b2e9c03.
//
// Solidity: function ownershipTransferredToFallback() view returns(bool)
func (_LivenessModule *LivenessModuleSession) OwnershipTransferredToFallback() (bool, error) {
	return _LivenessModule.Contract.OwnershipTransferredToFallback(&_LivenessModule.CallOpts)
}

// OwnershipTransferredToFallback is a free data retrieval call binding the contract method 0x5b2e9c03.
//
// Solidity: function ownershipTransferredToFallback() view returns(bool)
func (_LivenessModule *LivenessModuleCallerSession) OwnershipTransferredToFallback() (bool, error) {
	return _LivenessModule.Contract.OwnershipTransferredToFallback(&_LivenessModule.CallOpts)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessModule *LivenessModuleCaller) Safe(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "safe")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessModule *LivenessModuleSession) Safe() (common.Address, error) {
	return _LivenessModule.Contract.Safe(&_LivenessModule.CallOpts)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessModule *LivenessModuleCallerSession) Safe() (common.Address, error) {
	return _LivenessModule.Contract.Safe(&_LivenessModule.CallOpts)
}

// ThresholdPercentage is a free data retrieval call binding the contract method 0x4ed2859e.
//
// Solidity: function thresholdPercentage() view returns(uint256 thresholdPercentage_)
func (_LivenessModule *LivenessModuleCaller) ThresholdPercentage(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "thresholdPercentage")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ThresholdPercentage is a free data retrieval call binding the contract method 0x4ed2859e.
//
// Solidity: function thresholdPercentage() view returns(uint256 thresholdPercentage_)
func (_LivenessModule *LivenessModuleSession) ThresholdPercentage() (*big.Int, error) {
	return _LivenessModule.Contract.ThresholdPercentage(&_LivenessModule.CallOpts)
}

// ThresholdPercentage is a free data retrieval call binding the contract method 0x4ed2859e.
//
// Solidity: function thresholdPercentage() view returns(uint256 thresholdPercentage_)
func (_LivenessModule *LivenessModuleCallerSession) ThresholdPercentage() (*big.Int, error) {
	return _LivenessModule.Contract.ThresholdPercentage(&_LivenessModule.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessModule *LivenessModuleCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _LivenessModule.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessModule *LivenessModuleSession) Version() (string, error) {
	return _LivenessModule.Contract.Version(&_LivenessModule.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessModule *LivenessModuleCallerSession) Version() (string, error) {
	return _LivenessModule.Contract.Version(&_LivenessModule.CallOpts)
}

// RemoveOwners is a paid mutator transaction binding the contract method 0xf6ac15c4.
//
// Solidity: function removeOwners(address[] _previousOwners, address[] _ownersToRemove) returns()
func (_LivenessModule *LivenessModuleTransactor) RemoveOwners(opts *bind.TransactOpts, _previousOwners []common.Address, _ownersToRemove []common.Address) (*types.Transaction, error) {
	return _LivenessModule.contract.Transact(opts, "removeOwners", _previousOwners, _ownersToRemove)
}

// RemoveOwners is a paid mutator transaction binding the contract method 0xf6ac15c4.
//
// Solidity: function removeOwners(address[] _previousOwners, address[] _ownersToRemove) returns()
func (_LivenessModule *LivenessModuleSession) RemoveOwners(_previousOwners []common.Address, _ownersToRemove []common.Address) (*types.Transaction, error) {
	return _LivenessModule.Contract.RemoveOwners(&_LivenessModule.TransactOpts, _previousOwners, _ownersToRemove)
}

// RemoveOwners is a paid mutator transaction binding the contract method 0xf6ac15c4.
//
// Solidity: function removeOwners(address[] _previousOwners, address[] _ownersToRemove) returns()
func (_LivenessModule *LivenessModuleTransactorSession) RemoveOwners(_previousOwners []common.Address, _ownersToRemove []common.Address) (*types.Transaction, error) {
	return _LivenessModule.Contract.RemoveOwners(&_LivenessModule.TransactOpts, _previousOwners, _ownersToRemove)
}

// LivenessModuleOwnershipTransferredToFallbackIterator is returned from FilterOwnershipTransferredToFallback and is used to iterate over the raw logs and unpacked data for OwnershipTransferredToFallback events raised by the LivenessModule contract.
type LivenessModuleOwnershipTransferredToFallbackIterator struct {
	Event *LivenessModuleOwnershipTransferredToFallback // Event containing the contract specifics and raw log

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
func (it *LivenessModuleOwnershipTransferredToFallbackIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LivenessModuleOwnershipTransferredToFallback)
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
		it.Event = new(LivenessModuleOwnershipTransferredToFallback)
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
func (it *LivenessModuleOwnershipTransferredToFallbackIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LivenessModuleOwnershipTransferredToFallbackIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LivenessModuleOwnershipTransferredToFallback represents a OwnershipTransferredToFallback event raised by the LivenessModule contract.
type LivenessModuleOwnershipTransferredToFallback struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferredToFallback is a free log retrieval operation binding the contract event 0x903af33ae084e750e45a10a53f61a69a4c5444f13a59954f5b3cf96e2862b5f4.
//
// Solidity: event OwnershipTransferredToFallback()
func (_LivenessModule *LivenessModuleFilterer) FilterOwnershipTransferredToFallback(opts *bind.FilterOpts) (*LivenessModuleOwnershipTransferredToFallbackIterator, error) {

	logs, sub, err := _LivenessModule.contract.FilterLogs(opts, "OwnershipTransferredToFallback")
	if err != nil {
		return nil, err
	}
	return &LivenessModuleOwnershipTransferredToFallbackIterator{contract: _LivenessModule.contract, event: "OwnershipTransferredToFallback", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferredToFallback is a free log subscription operation binding the contract event 0x903af33ae084e750e45a10a53f61a69a4c5444f13a59954f5b3cf96e2862b5f4.
//
// Solidity: event OwnershipTransferredToFallback()
func (_LivenessModule *LivenessModuleFilterer) WatchOwnershipTransferredToFallback(opts *bind.WatchOpts, sink chan<- *LivenessModuleOwnershipTransferredToFallback) (event.Subscription, error) {

	logs, sub, err := _LivenessModule.contract.WatchLogs(opts, "OwnershipTransferredToFallback")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LivenessModuleOwnershipTransferredToFallback)
				if err := _LivenessModule.contract.UnpackLog(event, "OwnershipTransferredToFallback", log); err != nil {
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

// ParseOwnershipTransferredToFallback is a log parse operation binding the contract event 0x903af33ae084e750e45a10a53f61a69a4c5444f13a59954f5b3cf96e2862b5f4.
//
// Solidity: event OwnershipTransferredToFallback()
func (_LivenessModule *LivenessModuleFilterer) ParseOwnershipTransferredToFallback(log types.Log) (*LivenessModuleOwnershipTransferredToFallback, error) {
	event := new(LivenessModuleOwnershipTransferredToFallback)
	if err := _LivenessModule.contract.UnpackLog(event, "OwnershipTransferredToFallback", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LivenessModuleRemovedOwnerIterator is returned from FilterRemovedOwner and is used to iterate over the raw logs and unpacked data for RemovedOwner events raised by the LivenessModule contract.
type LivenessModuleRemovedOwnerIterator struct {
	Event *LivenessModuleRemovedOwner // Event containing the contract specifics and raw log

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
func (it *LivenessModuleRemovedOwnerIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LivenessModuleRemovedOwner)
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
		it.Event = new(LivenessModuleRemovedOwner)
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
func (it *LivenessModuleRemovedOwnerIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LivenessModuleRemovedOwnerIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LivenessModuleRemovedOwner represents a RemovedOwner event raised by the LivenessModule contract.
type LivenessModuleRemovedOwner struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterRemovedOwner is a free log retrieval operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_LivenessModule *LivenessModuleFilterer) FilterRemovedOwner(opts *bind.FilterOpts, owner []common.Address) (*LivenessModuleRemovedOwnerIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _LivenessModule.contract.FilterLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return &LivenessModuleRemovedOwnerIterator{contract: _LivenessModule.contract, event: "RemovedOwner", logs: logs, sub: sub}, nil
}

// WatchRemovedOwner is a free log subscription operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_LivenessModule *LivenessModuleFilterer) WatchRemovedOwner(opts *bind.WatchOpts, sink chan<- *LivenessModuleRemovedOwner, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _LivenessModule.contract.WatchLogs(opts, "RemovedOwner", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LivenessModuleRemovedOwner)
				if err := _LivenessModule.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
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

// ParseRemovedOwner is a log parse operation binding the contract event 0xf8d49fc529812e9a7c5c50e69c20f0dccc0db8fa95c98bc58cc9a4f1c1299eaf.
//
// Solidity: event RemovedOwner(address indexed owner)
func (_LivenessModule *LivenessModuleFilterer) ParseRemovedOwner(log types.Log) (*LivenessModuleRemovedOwner, error) {
	event := new(LivenessModuleRemovedOwner)
	if err := _LivenessModule.contract.UnpackLog(event, "RemovedOwner", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
