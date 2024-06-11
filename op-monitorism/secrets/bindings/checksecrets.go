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
)

// CheckSecretsParams is an auto generated low-level Go binding around an user-defined struct.
type CheckSecretsParams struct {
	Delay                  *big.Int
	SecretHashMustExist    [32]byte
	SecretHashMustNotExist [32]byte
}

// CheckSecretsMetaData contains all meta data concerning the CheckSecrets contract.
var CheckSecretsMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_params\",\"type\":\"bytes\"}],\"name\":\"check\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"execute_\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_secret\",\"type\":\"bytes\"}],\"name\":\"reveal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"revealedSecrets\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"secretHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"secret\",\"type\":\"bytes\"}],\"name\":\"SecretRevealed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"delay\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"secretHashMustExist\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"secretHashMustNotExist\",\"type\":\"bytes32\"}],\"indexed\":false,\"internalType\":\"structCheckSecrets.Params\",\"name\":\"params\",\"type\":\"tuple\"}],\"name\":\"_EventToExposeStructInABI__Params\",\"type\":\"event\"}]",
}

// CheckSecretsABI is the input ABI used to generate the binding from.
// Deprecated: Use CheckSecretsMetaData.ABI instead.
var CheckSecretsABI = CheckSecretsMetaData.ABI

// CheckSecrets is an auto generated Go binding around an Ethereum contract.
type CheckSecrets struct {
	CheckSecretsCaller     // Read-only binding to the contract
	CheckSecretsTransactor // Write-only binding to the contract
	CheckSecretsFilterer   // Log filterer for contract events
}

// CheckSecretsCaller is an auto generated read-only Go binding around an Ethereum contract.
type CheckSecretsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CheckSecretsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CheckSecretsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CheckSecretsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CheckSecretsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CheckSecretsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CheckSecretsSession struct {
	Contract     *CheckSecrets     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CheckSecretsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CheckSecretsCallerSession struct {
	Contract *CheckSecretsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// CheckSecretsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CheckSecretsTransactorSession struct {
	Contract     *CheckSecretsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// CheckSecretsRaw is an auto generated low-level Go binding around an Ethereum contract.
type CheckSecretsRaw struct {
	Contract *CheckSecrets // Generic contract binding to access the raw methods on
}

// CheckSecretsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CheckSecretsCallerRaw struct {
	Contract *CheckSecretsCaller // Generic read-only contract binding to access the raw methods on
}

// CheckSecretsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CheckSecretsTransactorRaw struct {
	Contract *CheckSecretsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCheckSecrets creates a new instance of CheckSecrets, bound to a specific deployed contract.
func NewCheckSecrets(address common.Address, backend bind.ContractBackend) (*CheckSecrets, error) {
	contract, err := bindCheckSecrets(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &CheckSecrets{CheckSecretsCaller: CheckSecretsCaller{contract: contract}, CheckSecretsTransactor: CheckSecretsTransactor{contract: contract}, CheckSecretsFilterer: CheckSecretsFilterer{contract: contract}}, nil
}

// NewCheckSecretsCaller creates a new read-only instance of CheckSecrets, bound to a specific deployed contract.
func NewCheckSecretsCaller(address common.Address, caller bind.ContractCaller) (*CheckSecretsCaller, error) {
	contract, err := bindCheckSecrets(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CheckSecretsCaller{contract: contract}, nil
}

// NewCheckSecretsTransactor creates a new write-only instance of CheckSecrets, bound to a specific deployed contract.
func NewCheckSecretsTransactor(address common.Address, transactor bind.ContractTransactor) (*CheckSecretsTransactor, error) {
	contract, err := bindCheckSecrets(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CheckSecretsTransactor{contract: contract}, nil
}

// NewCheckSecretsFilterer creates a new log filterer instance of CheckSecrets, bound to a specific deployed contract.
func NewCheckSecretsFilterer(address common.Address, filterer bind.ContractFilterer) (*CheckSecretsFilterer, error) {
	contract, err := bindCheckSecrets(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CheckSecretsFilterer{contract: contract}, nil
}

// bindCheckSecrets binds a generic wrapper to an already deployed contract.
func bindCheckSecrets(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CheckSecretsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CheckSecrets *CheckSecretsRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CheckSecrets.Contract.CheckSecretsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CheckSecrets *CheckSecretsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CheckSecrets.Contract.CheckSecretsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CheckSecrets *CheckSecretsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CheckSecrets.Contract.CheckSecretsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_CheckSecrets *CheckSecretsCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _CheckSecrets.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_CheckSecrets *CheckSecretsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _CheckSecrets.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_CheckSecrets *CheckSecretsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _CheckSecrets.Contract.contract.Transact(opts, method, params...)
}

// Check is a free data retrieval call binding the contract method 0xc64b3bb5.
//
// Solidity: function check(bytes _params) view returns(bool execute_)
func (_CheckSecrets *CheckSecretsCaller) Check(opts *bind.CallOpts, _params []byte) (bool, error) {
	var out []interface{}
	err := _CheckSecrets.contract.Call(opts, &out, "check", _params)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Check is a free data retrieval call binding the contract method 0xc64b3bb5.
//
// Solidity: function check(bytes _params) view returns(bool execute_)
func (_CheckSecrets *CheckSecretsSession) Check(_params []byte) (bool, error) {
	return _CheckSecrets.Contract.Check(&_CheckSecrets.CallOpts, _params)
}

// Check is a free data retrieval call binding the contract method 0xc64b3bb5.
//
// Solidity: function check(bytes _params) view returns(bool execute_)
func (_CheckSecrets *CheckSecretsCallerSession) Check(_params []byte) (bool, error) {
	return _CheckSecrets.Contract.Check(&_CheckSecrets.CallOpts, _params)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CheckSecrets *CheckSecretsCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _CheckSecrets.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CheckSecrets *CheckSecretsSession) Name() (string, error) {
	return _CheckSecrets.Contract.Name(&_CheckSecrets.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_CheckSecrets *CheckSecretsCallerSession) Name() (string, error) {
	return _CheckSecrets.Contract.Name(&_CheckSecrets.CallOpts)
}

// RevealedSecrets is a free data retrieval call binding the contract method 0x246167bc.
//
// Solidity: function revealedSecrets(bytes32 ) view returns(uint256)
func (_CheckSecrets *CheckSecretsCaller) RevealedSecrets(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _CheckSecrets.contract.Call(opts, &out, "revealedSecrets", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RevealedSecrets is a free data retrieval call binding the contract method 0x246167bc.
//
// Solidity: function revealedSecrets(bytes32 ) view returns(uint256)
func (_CheckSecrets *CheckSecretsSession) RevealedSecrets(arg0 [32]byte) (*big.Int, error) {
	return _CheckSecrets.Contract.RevealedSecrets(&_CheckSecrets.CallOpts, arg0)
}

// RevealedSecrets is a free data retrieval call binding the contract method 0x246167bc.
//
// Solidity: function revealedSecrets(bytes32 ) view returns(uint256)
func (_CheckSecrets *CheckSecretsCallerSession) RevealedSecrets(arg0 [32]byte) (*big.Int, error) {
	return _CheckSecrets.Contract.RevealedSecrets(&_CheckSecrets.CallOpts, arg0)
}

// Reveal is a paid mutator transaction binding the contract method 0x72f12a5d.
//
// Solidity: function reveal(bytes _secret) returns()
func (_CheckSecrets *CheckSecretsTransactor) Reveal(opts *bind.TransactOpts, _secret []byte) (*types.Transaction, error) {
	return _CheckSecrets.contract.Transact(opts, "reveal", _secret)
}

// Reveal is a paid mutator transaction binding the contract method 0x72f12a5d.
//
// Solidity: function reveal(bytes _secret) returns()
func (_CheckSecrets *CheckSecretsSession) Reveal(_secret []byte) (*types.Transaction, error) {
	return _CheckSecrets.Contract.Reveal(&_CheckSecrets.TransactOpts, _secret)
}

// Reveal is a paid mutator transaction binding the contract method 0x72f12a5d.
//
// Solidity: function reveal(bytes _secret) returns()
func (_CheckSecrets *CheckSecretsTransactorSession) Reveal(_secret []byte) (*types.Transaction, error) {
	return _CheckSecrets.Contract.Reveal(&_CheckSecrets.TransactOpts, _secret)
}

// CheckSecretsSecretRevealedIterator is returned from FilterSecretRevealed and is used to iterate over the raw logs and unpacked data for SecretRevealed events raised by the CheckSecrets contract.
type CheckSecretsSecretRevealedIterator struct {
	Event *CheckSecretsSecretRevealed // Event containing the contract specifics and raw log

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
func (it *CheckSecretsSecretRevealedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CheckSecretsSecretRevealed)
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
		it.Event = new(CheckSecretsSecretRevealed)
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
func (it *CheckSecretsSecretRevealedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CheckSecretsSecretRevealedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CheckSecretsSecretRevealed represents a SecretRevealed event raised by the CheckSecrets contract.
type CheckSecretsSecretRevealed struct {
	SecretHash [32]byte
	Secret     []byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterSecretRevealed is a free log retrieval operation binding the contract event 0xbab2b812958b05e36be1f0553f496fa5d27441155d6be0469e1c3fe1e51ad858.
//
// Solidity: event SecretRevealed(bytes32 indexed secretHash, bytes secret)
func (_CheckSecrets *CheckSecretsFilterer) FilterSecretRevealed(opts *bind.FilterOpts, secretHash [][32]byte) (*CheckSecretsSecretRevealedIterator, error) {

	var secretHashRule []interface{}
	for _, secretHashItem := range secretHash {
		secretHashRule = append(secretHashRule, secretHashItem)
	}

	logs, sub, err := _CheckSecrets.contract.FilterLogs(opts, "SecretRevealed", secretHashRule)
	if err != nil {
		return nil, err
	}
	return &CheckSecretsSecretRevealedIterator{contract: _CheckSecrets.contract, event: "SecretRevealed", logs: logs, sub: sub}, nil
}

// WatchSecretRevealed is a free log subscription operation binding the contract event 0xbab2b812958b05e36be1f0553f496fa5d27441155d6be0469e1c3fe1e51ad858.
//
// Solidity: event SecretRevealed(bytes32 indexed secretHash, bytes secret)
func (_CheckSecrets *CheckSecretsFilterer) WatchSecretRevealed(opts *bind.WatchOpts, sink chan<- *CheckSecretsSecretRevealed, secretHash [][32]byte) (event.Subscription, error) {

	var secretHashRule []interface{}
	for _, secretHashItem := range secretHash {
		secretHashRule = append(secretHashRule, secretHashItem)
	}

	logs, sub, err := _CheckSecrets.contract.WatchLogs(opts, "SecretRevealed", secretHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CheckSecretsSecretRevealed)
				if err := _CheckSecrets.contract.UnpackLog(event, "SecretRevealed", log); err != nil {
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

// ParseSecretRevealed is a log parse operation binding the contract event 0xbab2b812958b05e36be1f0553f496fa5d27441155d6be0469e1c3fe1e51ad858.
//
// Solidity: event SecretRevealed(bytes32 indexed secretHash, bytes secret)
func (_CheckSecrets *CheckSecretsFilterer) ParseSecretRevealed(log types.Log) (*CheckSecretsSecretRevealed, error) {
	event := new(CheckSecretsSecretRevealed)
	if err := _CheckSecrets.contract.UnpackLog(event, "SecretRevealed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CheckSecretsEventToExposeStructInABIParamsIterator is returned from FilterEventToExposeStructInABIParams and is used to iterate over the raw logs and unpacked data for EventToExposeStructInABIParams events raised by the CheckSecrets contract.
type CheckSecretsEventToExposeStructInABIParamsIterator struct {
	Event *CheckSecretsEventToExposeStructInABIParams // Event containing the contract specifics and raw log

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
func (it *CheckSecretsEventToExposeStructInABIParamsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CheckSecretsEventToExposeStructInABIParams)
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
		it.Event = new(CheckSecretsEventToExposeStructInABIParams)
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
func (it *CheckSecretsEventToExposeStructInABIParamsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CheckSecretsEventToExposeStructInABIParamsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CheckSecretsEventToExposeStructInABIParams represents a EventToExposeStructInABIParams event raised by the CheckSecrets contract.
type CheckSecretsEventToExposeStructInABIParams struct {
	Params CheckSecretsParams
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterEventToExposeStructInABIParams is a free log retrieval operation binding the contract event 0xe696aefa5d73185c96c67a87c9597f9ef8d746736183999616f1e6f5a839b282.
//
// Solidity: event _EventToExposeStructInABI__Params((uint256,bytes32,bytes32) params)
func (_CheckSecrets *CheckSecretsFilterer) FilterEventToExposeStructInABIParams(opts *bind.FilterOpts) (*CheckSecretsEventToExposeStructInABIParamsIterator, error) {

	logs, sub, err := _CheckSecrets.contract.FilterLogs(opts, "_EventToExposeStructInABI__Params")
	if err != nil {
		return nil, err
	}
	return &CheckSecretsEventToExposeStructInABIParamsIterator{contract: _CheckSecrets.contract, event: "_EventToExposeStructInABI__Params", logs: logs, sub: sub}, nil
}

// WatchEventToExposeStructInABIParams is a free log subscription operation binding the contract event 0xe696aefa5d73185c96c67a87c9597f9ef8d746736183999616f1e6f5a839b282.
//
// Solidity: event _EventToExposeStructInABI__Params((uint256,bytes32,bytes32) params)
func (_CheckSecrets *CheckSecretsFilterer) WatchEventToExposeStructInABIParams(opts *bind.WatchOpts, sink chan<- *CheckSecretsEventToExposeStructInABIParams) (event.Subscription, error) {

	logs, sub, err := _CheckSecrets.contract.WatchLogs(opts, "_EventToExposeStructInABI__Params")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CheckSecretsEventToExposeStructInABIParams)
				if err := _CheckSecrets.contract.UnpackLog(event, "_EventToExposeStructInABI__Params", log); err != nil {
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

// ParseEventToExposeStructInABIParams is a log parse operation binding the contract event 0xe696aefa5d73185c96c67a87c9597f9ef8d746736183999616f1e6f5a839b282.
//
// Solidity: event _EventToExposeStructInABI__Params((uint256,bytes32,bytes32) params)
func (_CheckSecrets *CheckSecretsFilterer) ParseEventToExposeStructInABIParams(log types.Log) (*CheckSecretsEventToExposeStructInABIParams, error) {
	event := new(CheckSecretsEventToExposeStructInABIParams)
	if err := _CheckSecrets.contract.UnpackLog(event, "_EventToExposeStructInABI__Params", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
