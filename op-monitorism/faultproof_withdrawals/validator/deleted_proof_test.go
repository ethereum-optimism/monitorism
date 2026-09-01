package validator

import (
	"context"
	"encoding/json"
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

// fakeL1 serves the few JSON-RPC methods the validator needs, so a test can control exactly
// what provenWithdrawals returns for a withdrawal that has a WithdrawalProvenExtension1 event.
type fakeL1 struct {
	t *testing.T
	// provenWithdrawalsGame is the dispute game address the portal reports. The zero address
	// means the proof record was deleted.
	provenWithdrawalsGame common.Address
	portalABI             *abi.ABI
	gameABI               *abi.ABI
}

func newFakeL1(t *testing.T, provenWithdrawalsGame common.Address) *httptest.Server {
	t.Helper()
	portalABI, err := l1.OptimismPortal2MetaData.GetAbi()
	require.NoError(t, err)
	gameABI, err := dispute.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)

	f := &fakeL1{t: t, provenWithdrawalsGame: provenWithdrawalsGame, portalABI: portalABI, gameABI: gameABI}
	server := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(server.Close)
	return server
}

type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func (f *fakeL1) serve(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&raw))

	var batch []rpcRequest
	if err := json.Unmarshal(raw, &batch); err != nil {
		var single rpcRequest
		require.NoError(f.t, json.Unmarshal(raw, &single))
		batch = []rpcRequest{single}
	}

	responses := make([]json.RawMessage, 0, len(batch))
	for _, req := range batch {
		result, err := f.handle(req)
		require.NoError(f.t, err)
		encoded, err := json.Marshal(result)
		require.NoError(f.t, err)
		responses = append(responses, json.RawMessage(fmt.Sprintf(
			`{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, encoded)))
	}

	w.Header().Set("Content-Type", "application/json")
	if len(batch) == 1 {
		_, err := w.Write(responses[0])
		require.NoError(f.t, err)
		return
	}
	out, err := json.Marshal(responses)
	require.NoError(f.t, err)
	_, err = w.Write(out)
	require.NoError(f.t, err)
}

func (f *fakeL1) handle(req rpcRequest) (any, error) {
	switch req.Method {
	case "eth_getLogs":
		return f.logs(), nil
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
		return f.call(call.To, call.Data)
	case "eth_blockNumber":
		return hexutil.Uint64(testProveBlock), nil
	default:
		return nil, fmt.Errorf("unexpected rpc method %s", req.Method)
	}
}

// logs returns a single WithdrawalProvenExtension1 event for the test withdrawal.
func (f *fakeL1) logs() []map[string]any {
	event := f.portalABI.Events["WithdrawalProvenExtension1"]
	return []map[string]any{{
		"address": testPortalAddress,
		"topics": []common.Hash{
			event.ID,
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
	}}
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

func newTestValidator(t *testing.T, provenWithdrawalsGame common.Address) *ProvenWithdrawalValidator {
	t.Helper()
	server := newFakeL1(t, provenWithdrawalsGame)
	ctx := context.Background()
	l1Proxy, err := NewL1Proxy(ctx, server.URL, testPortalAddress)
	require.NoError(t, err)
	return &ProvenWithdrawalValidator{L1Proxy: l1Proxy, ctx: ctx, log: log.New()}
}

// TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof pins the fix for the monitor stall.
// Anyone can delete a proof record once its game resolved in favor of the challenger or was
// blacklisted. The prove events stay in the logs forever, so the monitor must skip the event
// instead of returning an error, which would stop it from advancing its L1 cursor.
func TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof(t *testing.T) {
	validator := newTestValidator(t, common.Address{})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.NoError(t, err)
	require.Empty(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof is the control for the test above: an event
// with a proof record still in state must be enriched, not skipped.
func TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof(t *testing.T) {
	validator := newTestValidator(t, testGameAddress)
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.NoError(t, err)
	require.Len(t, events, 1)
	event := events[testProveTxHash]
	require.NotNil(t, event)
	require.Equal(t, testGameAddress, event.DisputeGame.DisputeGameData.ProxyAddress)
	require.Equal(t, testWithdrawalHash, common.BytesToHash(event.Event.WithdrawalHash[:]))
}

// TestGetEnrichedWithdrawalsEventsSkipsDeletedProof covers the slice variant of the same loop.
func TestGetEnrichedWithdrawalsEventsSkipsDeletedProof(t *testing.T) {
	validator := newTestValidator(t, common.Address{})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEvents(testProveBlock, &stop)

	require.NoError(t, err)
	require.Empty(t, events)
}

// TestSubmittedProofDataIsDeleted documents the on-chain signal the skip depends on: the portal
// zeroes the dispute game address when a proof record is deleted.
func TestSubmittedProofDataIsDeleted(t *testing.T) {
	deleted := SubmittedProofData{disputeGameProxyAddress: common.Address{}}
	live := SubmittedProofData{disputeGameProxyAddress: testGameAddress}

	require.True(t, deleted.IsDeleted())
	require.False(t, live.IsDeleted())
}
