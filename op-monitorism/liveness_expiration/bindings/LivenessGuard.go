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

// LivenessGuardMetaData contains all meta data concerning the LivenessGuard contract.
var LivenessGuardMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_safe\",\"type\":\"address\",\"internalType\":\"contractGnosisSafe\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"checkAfterExecution\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"checkTransaction\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"data\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"operation\",\"type\":\"uint8\",\"internalType\":\"enumEnum.Operation\"},{\"name\":\"safeTxGas\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"baseGas\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"gasPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"gasToken\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"refundReceiver\",\"type\":\"address\",\"internalType\":\"addresspayable\"},{\"name\":\"signatures\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"msgSender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"lastLive\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"safe\",\"inputs\":[],\"outputs\":[{\"name\":\"safe_\",\"type\":\"address\",\"internalType\":\"contractGnosisSafe\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"showLiveness\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"version\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"OwnerRecorded\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false}]",
}

// LivenessGuardABI is the input ABI used to generate the binding from.
// Deprecated: Use LivenessGuardMetaData.ABI instead.
var LivenessGuardABI = LivenessGuardMetaData.ABI

// LivenessGuard is an auto generated Go binding around an Ethereum contract.
type LivenessGuard struct {
	LivenessGuardCaller     // Read-only binding to the contract
	LivenessGuardTransactor // Write-only binding to the contract
	LivenessGuardFilterer   // Log filterer for contract events
}

// LivenessGuardCaller is an auto generated read-only Go binding around an Ethereum contract.
type LivenessGuardCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessGuardTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LivenessGuardTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessGuardFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LivenessGuardFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LivenessGuardSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LivenessGuardSession struct {
	Contract     *LivenessGuard    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LivenessGuardCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LivenessGuardCallerSession struct {
	Contract *LivenessGuardCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// LivenessGuardTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LivenessGuardTransactorSession struct {
	Contract     *LivenessGuardTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// LivenessGuardRaw is an auto generated low-level Go binding around an Ethereum contract.
type LivenessGuardRaw struct {
	Contract *LivenessGuard // Generic contract binding to access the raw methods on
}

// LivenessGuardCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LivenessGuardCallerRaw struct {
	Contract *LivenessGuardCaller // Generic read-only contract binding to access the raw methods on
}

// LivenessGuardTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LivenessGuardTransactorRaw struct {
	Contract *LivenessGuardTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLivenessGuard creates a new instance of LivenessGuard, bound to a specific deployed contract.
func NewLivenessGuard(address common.Address, backend bind.ContractBackend) (*LivenessGuard, error) {
	contract, err := bindLivenessGuard(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LivenessGuard{LivenessGuardCaller: LivenessGuardCaller{contract: contract}, LivenessGuardTransactor: LivenessGuardTransactor{contract: contract}, LivenessGuardFilterer: LivenessGuardFilterer{contract: contract}}, nil
}

// NewLivenessGuardCaller creates a new read-only instance of LivenessGuard, bound to a specific deployed contract.
func NewLivenessGuardCaller(address common.Address, caller bind.ContractCaller) (*LivenessGuardCaller, error) {
	contract, err := bindLivenessGuard(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LivenessGuardCaller{contract: contract}, nil
}

// NewLivenessGuardTransactor creates a new write-only instance of LivenessGuard, bound to a specific deployed contract.
func NewLivenessGuardTransactor(address common.Address, transactor bind.ContractTransactor) (*LivenessGuardTransactor, error) {
	contract, err := bindLivenessGuard(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LivenessGuardTransactor{contract: contract}, nil
}

// NewLivenessGuardFilterer creates a new log filterer instance of LivenessGuard, bound to a specific deployed contract.
func NewLivenessGuardFilterer(address common.Address, filterer bind.ContractFilterer) (*LivenessGuardFilterer, error) {
	contract, err := bindLivenessGuard(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LivenessGuardFilterer{contract: contract}, nil
}

// bindLivenessGuard binds a generic wrapper to an already deployed contract.
func bindLivenessGuard(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LivenessGuardMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LivenessGuard *LivenessGuardRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LivenessGuard.Contract.LivenessGuardCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LivenessGuard *LivenessGuardRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LivenessGuard.Contract.LivenessGuardTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LivenessGuard *LivenessGuardRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LivenessGuard.Contract.LivenessGuardTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LivenessGuard *LivenessGuardCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LivenessGuard.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LivenessGuard *LivenessGuardTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LivenessGuard.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LivenessGuard *LivenessGuardTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LivenessGuard.Contract.contract.Transact(opts, method, params...)
}

// LastLive is a free data retrieval call binding the contract method 0xe458779b.
//
// Solidity: function lastLive(address ) view returns(uint256)
func (_LivenessGuard *LivenessGuardCaller) LastLive(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _LivenessGuard.contract.Call(opts, &out, "lastLive", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastLive is a free data retrieval call binding the contract method 0xe458779b.
//
// Solidity: function lastLive(address ) view returns(uint256)
func (_LivenessGuard *LivenessGuardSession) LastLive(arg0 common.Address) (*big.Int, error) {
	return _LivenessGuard.Contract.LastLive(&_LivenessGuard.CallOpts, arg0)
}

// LastLive is a free data retrieval call binding the contract method 0xe458779b.
//
// Solidity: function lastLive(address ) view returns(uint256)
func (_LivenessGuard *LivenessGuardCallerSession) LastLive(arg0 common.Address) (*big.Int, error) {
	return _LivenessGuard.Contract.LastLive(&_LivenessGuard.CallOpts, arg0)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessGuard *LivenessGuardCaller) Safe(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LivenessGuard.contract.Call(opts, &out, "safe")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessGuard *LivenessGuardSession) Safe() (common.Address, error) {
	return _LivenessGuard.Contract.Safe(&_LivenessGuard.CallOpts)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address safe_)
func (_LivenessGuard *LivenessGuardCallerSession) Safe() (common.Address, error) {
	return _LivenessGuard.Contract.Safe(&_LivenessGuard.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessGuard *LivenessGuardCaller) Version(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _LivenessGuard.contract.Call(opts, &out, "version")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessGuard *LivenessGuardSession) Version() (string, error) {
	return _LivenessGuard.Contract.Version(&_LivenessGuard.CallOpts)
}

// Version is a free data retrieval call binding the contract method 0x54fd4d50.
//
// Solidity: function version() view returns(string)
func (_LivenessGuard *LivenessGuardCallerSession) Version() (string, error) {
	return _LivenessGuard.Contract.Version(&_LivenessGuard.CallOpts)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 , bool ) returns()
func (_LivenessGuard *LivenessGuardTransactor) CheckAfterExecution(opts *bind.TransactOpts, arg0 [32]byte, arg1 bool) (*types.Transaction, error) {
	return _LivenessGuard.contract.Transact(opts, "checkAfterExecution", arg0, arg1)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 , bool ) returns()
func (_LivenessGuard *LivenessGuardSession) CheckAfterExecution(arg0 [32]byte, arg1 bool) (*types.Transaction, error) {
	return _LivenessGuard.Contract.CheckAfterExecution(&_LivenessGuard.TransactOpts, arg0, arg1)
}

// CheckAfterExecution is a paid mutator transaction binding the contract method 0x93271368.
//
// Solidity: function checkAfterExecution(bytes32 , bool ) returns()
func (_LivenessGuard *LivenessGuardTransactorSession) CheckAfterExecution(arg0 [32]byte, arg1 bool) (*types.Transaction, error) {
	return _LivenessGuard.Contract.CheckAfterExecution(&_LivenessGuard.TransactOpts, arg0, arg1)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_LivenessGuard *LivenessGuardTransactor) CheckTransaction(opts *bind.TransactOpts, to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _LivenessGuard.contract.Transact(opts, "checkTransaction", to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_LivenessGuard *LivenessGuardSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _LivenessGuard.Contract.CheckTransaction(&_LivenessGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// CheckTransaction is a paid mutator transaction binding the contract method 0x75f0bb52.
//
// Solidity: function checkTransaction(address to, uint256 value, bytes data, uint8 operation, uint256 safeTxGas, uint256 baseGas, uint256 gasPrice, address gasToken, address refundReceiver, bytes signatures, address msgSender) returns()
func (_LivenessGuard *LivenessGuardTransactorSession) CheckTransaction(to common.Address, value *big.Int, data []byte, operation uint8, safeTxGas *big.Int, baseGas *big.Int, gasPrice *big.Int, gasToken common.Address, refundReceiver common.Address, signatures []byte, msgSender common.Address) (*types.Transaction, error) {
	return _LivenessGuard.Contract.CheckTransaction(&_LivenessGuard.TransactOpts, to, value, data, operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, signatures, msgSender)
}

// ShowLiveness is a paid mutator transaction binding the contract method 0x4c205d0d.
//
// Solidity: function showLiveness() returns()
func (_LivenessGuard *LivenessGuardTransactor) ShowLiveness(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LivenessGuard.contract.Transact(opts, "showLiveness")
}

// ShowLiveness is a paid mutator transaction binding the contract method 0x4c205d0d.
//
// Solidity: function showLiveness() returns()
func (_LivenessGuard *LivenessGuardSession) ShowLiveness() (*types.Transaction, error) {
	return _LivenessGuard.Contract.ShowLiveness(&_LivenessGuard.TransactOpts)
}

// ShowLiveness is a paid mutator transaction binding the contract method 0x4c205d0d.
//
// Solidity: function showLiveness() returns()
func (_LivenessGuard *LivenessGuardTransactorSession) ShowLiveness() (*types.Transaction, error) {
	return _LivenessGuard.Contract.ShowLiveness(&_LivenessGuard.TransactOpts)
}

// LivenessGuardOwnerRecordedIterator is returned from FilterOwnerRecorded and is used to iterate over the raw logs and unpacked data for OwnerRecorded events raised by the LivenessGuard contract.
type LivenessGuardOwnerRecordedIterator struct {
	Event *LivenessGuardOwnerRecorded // Event containing the contract specifics and raw log

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
func (it *LivenessGuardOwnerRecordedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LivenessGuardOwnerRecorded)
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
		it.Event = new(LivenessGuardOwnerRecorded)
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
func (it *LivenessGuardOwnerRecordedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LivenessGuardOwnerRecordedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LivenessGuardOwnerRecorded represents a OwnerRecorded event raised by the LivenessGuard contract.
type LivenessGuardOwnerRecorded struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterOwnerRecorded is a free log retrieval operation binding the contract event 0x833bc129023866d52116d61e94b791eb8be46f05709362e0bcf1fe7c1a8c225c.
//
// Solidity: event OwnerRecorded(address owner)
func (_LivenessGuard *LivenessGuardFilterer) FilterOwnerRecorded(opts *bind.FilterOpts) (*LivenessGuardOwnerRecordedIterator, error) {

	logs, sub, err := _LivenessGuard.contract.FilterLogs(opts, "OwnerRecorded")
	if err != nil {
		return nil, err
	}
	return &LivenessGuardOwnerRecordedIterator{contract: _LivenessGuard.contract, event: "OwnerRecorded", logs: logs, sub: sub}, nil
}

// WatchOwnerRecorded is a free log subscription operation binding the contract event 0x833bc129023866d52116d61e94b791eb8be46f05709362e0bcf1fe7c1a8c225c.
//
// Solidity: event OwnerRecorded(address owner)
func (_LivenessGuard *LivenessGuardFilterer) WatchOwnerRecorded(opts *bind.WatchOpts, sink chan<- *LivenessGuardOwnerRecorded) (event.Subscription, error) {

	logs, sub, err := _LivenessGuard.contract.WatchLogs(opts, "OwnerRecorded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LivenessGuardOwnerRecorded)
				if err := _LivenessGuard.contract.UnpackLog(event, "OwnerRecorded", log); err != nil {
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

// ParseOwnerRecorded is a log parse operation binding the contract event 0x833bc129023866d52116d61e94b791eb8be46f05709362e0bcf1fe7c1a8c225c.
//
// Solidity: event OwnerRecorded(address owner)
func (_LivenessGuard *LivenessGuardFilterer) ParseOwnerRecorded(log types.Log) (*LivenessGuardOwnerRecorded, error) {
	event := new(LivenessGuardOwnerRecorded)
	if err := _LivenessGuard.contract.UnpackLog(event, "OwnerRecorded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
