package validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	testPortalAddress   = common.HexToAddress("0x1000000000000000000000000000000000000001")
	testGameAddress     = common.HexToAddress("0x2000000000000000000000000000000000000002")
	testProofSubmitter  = common.HexToAddress("0x3000000000000000000000000000000000000003")
	testWithdrawalHash  = common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	testProveTxHash     = common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	testRootClaim       = common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666")
	testProveBlock      = uint64(100)
	testL2BlockNumber   = big.NewInt(4242)
	testL2ChainID       = big.NewInt(10)
	testGameCreatedAt   = uint64(1700000000)
	testGameResolvedAt  = uint64(1700000600)
	testGameStatusValue = uint8(CHALLENGER_WINS)
)

// fakeConfig describes the L1 state the fake reports.
type fakeConfig struct {
	// provenWithdrawalsGame is the dispute game address at the captured block.
	provenWithdrawalsGame common.Address
	// missingPinnedBlock makes calls pinned to a block hash fail, as they do on a node that does
	// not hold that block.
	missingPinnedBlock bool
	// headBlockNumber is the header number returned for the proof classification.
	headBlockNumber uint64
}

// fakeL1 serves the few JSON-RPC methods the validator needs, so a test can control exactly what
// the portal reports for a withdrawal that has a WithdrawalProvenExtension1 event.
type fakeL1 struct {
	fakeConfig

	portalABI *abi.ABI
	gameABI   *abi.ABI
}

func newFakeL1(t *testing.T, config fakeConfig) *httptest.Server {
	t.Helper()
	if config.headBlockNumber == 0 {
		config.headBlockNumber = testProveBlock
	}
	portalABI, err := l1.OptimismPortal2MetaData.GetAbi()
	require.NoError(t, err)
	gameABI, err := dispute.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)

	f := &fakeL1{fakeConfig: config, portalABI: portalABI, gameABI: gameABI}
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	return server
}

type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// rpcError makes the fake answer a request with a JSON-RPC error, the way a node answers a call
// against a block it does not have.
type rpcError struct{ message string }

func (e rpcError) Error() string { return e.message }

func (f *fakeL1) serve(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var batch []rpcRequest
	if err := json.Unmarshal(raw, &batch); err != nil {
		var single rpcRequest
		if err := json.Unmarshal(raw, &single); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		batch = []rpcRequest{single}
	}

	responses := make([]json.RawMessage, 0, len(batch))
	for _, req := range batch {
		result, err := f.handle(req)

		var nodeErr rpcError
		if errors.As(err, &nodeErr) {
			responses = append(responses, json.RawMessage(fmt.Sprintf(
				`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":%q}}`, req.ID, nodeErr.message)))
			continue
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		encoded, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		responses = append(responses, json.RawMessage(fmt.Sprintf(
			`{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, encoded)))
	}

	w.Header().Set("Content-Type", "application/json")
	if len(batch) == 1 {
		_, _ = w.Write(responses[0])
		return
	}
	out, err := json.Marshal(responses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(out)
}

func (f *fakeL1) handle(req rpcRequest) (any, error) {
	switch req.Method {
	case "eth_getLogs":
		return f.logs(req)
	case "eth_call":
		var call struct {
			To    common.Address `json:"to"`
			Data  hexutil.Bytes  `json:"data"`
			Input hexutil.Bytes  `json:"input"`
		}
		if err := json.Unmarshal(req.Params[0], &call); err != nil {
			return nil, err
		}
		if len(call.Data) == 0 {
			call.Data = call.Input
		}

		// A call pinned to a block hash carries an object rather than a block tag. A node that
		// does not hold that block answers with an error.
		var target struct {
			BlockHash *common.Hash `json:"blockHash"`
		}
		if len(req.Params) > 1 {
			_ = json.Unmarshal(req.Params[1], &target)
		}
		if target.BlockHash != nil {
			if f.missingPinnedBlock {
				return nil, rpcError{message: "header not found"}
			}
			expected := testHeadHeader(f.headBlockNumber).Hash()
			if *target.BlockHash != expected {
				return nil, fmt.Errorf("unexpected block hash %s, expected %s", *target.BlockHash, expected)
			}
		}

		return f.call(call.To, call.Data)
	case "eth_blockNumber":
		return hexutil.Uint64(testProveBlock), nil
	case "eth_getBlockByNumber":
		var blockTag string
		if err := json.Unmarshal(req.Params[0], &blockTag); err != nil {
			return nil, err
		}
		if blockTag == "latest" {
			return testHeadHeader(f.headBlockNumber), nil
		}
		number, err := hexutil.DecodeUint64(blockTag)
		if err != nil {
			return nil, err
		}
		return testHeadHeader(number), nil
	default:
		return nil, fmt.Errorf("unexpected rpc method %s", req.Method)
	}
}

// testHeadHeader is the header the fake reports as the head of the chain.
func testHeadHeader(number uint64) *types.Header {
	return &types.Header{
		Number:     new(big.Int).SetUint64(number),
		Difficulty: new(big.Int),
		GasLimit:   30_000_000,
		Time:       testGameCreatedAt,
		Extra:      []byte("fake head"),
	}
}

// logs returns a single WithdrawalProvenExtension1 event for the test withdrawal.
func (f *fakeL1) logs(_ rpcRequest) ([]map[string]any, error) {
	return []map[string]any{{
		"address": testPortalAddress,
		"topics": []common.Hash{
			f.portalABI.Events["WithdrawalProvenExtension1"].ID,
			testWithdrawalHash,
			common.BytesToHash(testProofSubmitter.Bytes()),
		},
		"data":             "0x",
		"blockNumber":      hexutil.Uint64(testProveBlock),
		"blockHash":        common.Hash{},
		"transactionHash":  testProveTxHash,
		"transactionIndex": hexutil.Uint64(0),
		"logIndex":         hexutil.Uint64(0),
		"removed":          false,
	}}, nil
}

func (f *fakeL1) call(to common.Address, data []byte) (hexutil.Bytes, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("call data too short")
	}
	contractABI := f.gameABI
	if to == testPortalAddress {
		contractABI = f.portalABI
	}
	method, err := contractABI.MethodById(data[:4])
	if err != nil {
		return nil, fmt.Errorf("unexpected call to %s: %w", to, err)
	}

	var values []any
	switch method.Name {
	case "provenWithdrawals":
		values = []any{f.provenWithdrawalsGame, uint64(testGameCreatedAt + 1)}
	case "rootClaim":
		values = []any{[32]byte(testRootClaim)}
	case "l2BlockNumber":
		values = []any{testL2BlockNumber}
	case "l2ChainId":
		values = []any{testL2ChainID}
	case "status":
		values = []any{testGameStatusValue}
	case "createdAt":
		values = []any{testGameCreatedAt}
	case "resolvedAt":
		values = []any{testGameResolvedAt}
	default:
		return nil, fmt.Errorf("unexpected call to method %s", method.Name)
	}

	out, err := method.Outputs.Pack(values...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack outputs for %s: %w", method.Name, err)
	}
	return out, nil
}

func newTestValidator(t *testing.T, config fakeConfig) *ProvenWithdrawalValidator {
	t.Helper()
	server := newFakeL1(t, config)
	ctx := context.Background()
	l1Proxy, err := NewL1Proxy(ctx, server.URL, testPortalAddress)
	require.NoError(t, err)
	return &ProvenWithdrawalValidator{L1Proxy: l1Proxy, ctx: ctx, log: log.New()}
}

// TestGetScanHeaderUsesLatestTag ensures the scan captures the unsafe latest header.
func TestGetScanHeaderUsesLatestTag(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{})

	header, err := validator.GetLatestL1Header()

	require.NoError(t, err)
	require.Equal(t, testProveBlock, header.Number.Uint64())
}

// TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof covers an empty proof at its event block.
func TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{})
	stop := testProveBlock
	header := testHeadHeader(testProveBlock)

	events, err := validator.GetEnrichedWithdrawalsEventsMap(
		testProveBlock,
		&stop,
		header.Number.Uint64(),
		header.Hash(),
	)

	require.NoError(t, err)
	require.Empty(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapRejectsHeaderBeforeEvent rejects unsafe classification.
func TestGetEnrichedWithdrawalsEventsMapRejectsHeaderBeforeEvent(t *testing.T) {
	headerNumber := testProveBlock - 1
	validator := newTestValidator(t, fakeConfig{headBlockNumber: headerNumber})
	stop := testProveBlock
	header := testHeadHeader(headerNumber)

	events, err := validator.GetEnrichedWithdrawalsEventsMap(
		testProveBlock,
		&stop,
		header.Number.Uint64(),
		header.Hash(),
	)

	require.ErrorIs(t, err, ErrWithdrawalProofMissing)
	require.Nil(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapFailsOnMissingPinnedBlock covers an unavailable captured block.
func TestGetEnrichedWithdrawalsEventsMapFailsOnMissingPinnedBlock(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{missingPinnedBlock: true})
	stop := testProveBlock
	header := testHeadHeader(testProveBlock)

	events, err := validator.GetEnrichedWithdrawalsEventsMap(
		testProveBlock,
		&stop,
		header.Number.Uint64(),
		header.Hash(),
	)

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrWithdrawalProofDeleted)
	require.Nil(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof covers a proof at the captured block.
func TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{provenWithdrawalsGame: testGameAddress})
	stop := testProveBlock
	header := testHeadHeader(testProveBlock)

	events, err := validator.GetEnrichedWithdrawalsEventsMap(
		testProveBlock,
		&stop,
		header.Number.Uint64(),
		header.Hash(),
	)

	require.NoError(t, err)
	require.Len(t, events, 1)
	event := events[testProveTxHash]
	require.NotNil(t, event)
	require.Equal(t, testGameAddress, event.DisputeGame.DisputeGameData.ProxyAddress)
	require.Equal(t, testWithdrawalHash, common.BytesToHash(event.Event.WithdrawalHash[:]))
}

// TestGetEnrichedWithdrawalsEventsSkipsDeletedProof covers the slice API.
func TestGetEnrichedWithdrawalsEventsSkipsDeletedProof(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{})
	stop := testProveBlock
	header := testHeadHeader(testProveBlock)

	events, err := validator.GetEnrichedWithdrawalsEvents(
		testProveBlock,
		&stop,
		header.Number.Uint64(),
		header.Hash(),
	)

	require.NoError(t, err)
	require.Empty(t, events)
}
