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
	// provenWithdrawalsGames holds the dispute game address the portal reports, per call. The
	// zero address means the proof record read empty. The last entry answers every later call,
	// so a test can make a later read of the same record disagree with the first.
	provenWithdrawalsGames []common.Address
	// proofSubmitters is the append-only submitter list the portal reports.
	proofSubmitters []common.Address
	// missingPinnedBlock makes calls pinned to a block hash fail, as they do on a node that does
	// not hold that block.
	missingPinnedBlock bool
}

// fakeL1 serves the few JSON-RPC methods the validator needs, so a test can control exactly what
// the portal reports for a withdrawal that has a WithdrawalProvenExtension1 event.
type fakeL1 struct {
	t *testing.T
	fakeConfig

	portalABI *abi.ABI
	gameABI   *abi.ABI

	provenWithdrawalsCalls int
}

func newFakeL1(t *testing.T, config fakeConfig) *httptest.Server {
	t.Helper()
	portalABI, err := l1.OptimismPortal2MetaData.GetAbi()
	require.NoError(t, err)
	gameABI, err := dispute.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)

	f := &fakeL1{t: t, fakeConfig: config, portalABI: portalABI, gameABI: gameABI}
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

		var nodeErr rpcError
		if errors.As(err, &nodeErr) {
			responses = append(responses, json.RawMessage(fmt.Sprintf(
				`{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":%q}}`, req.ID, nodeErr.message)))
			continue
		}
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
			require.Equal(f.t, testHeadBlockHash(f.t), *target.BlockHash)
		}

		return f.call(call.To, call.Data)
	case "eth_blockNumber":
		return hexutil.Uint64(testProveBlock), nil
	case "eth_getBlockByNumber":
		return testHeadHeader(), nil
	default:
		return nil, fmt.Errorf("unexpected rpc method %s", req.Method)
	}
}

// testHeadHeader is the header the fake reports as the head of the chain.
func testHeadHeader() *types.Header {
	return &types.Header{
		Number:     new(big.Int).SetUint64(testProveBlock),
		Difficulty: new(big.Int),
		GasLimit:   30_000_000,
		Time:       testGameCreatedAt,
		Extra:      []byte("fake head"),
	}
}

func testHeadBlockHash(t *testing.T) common.Hash {
	t.Helper()
	return testHeadHeader().Hash()
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

// nextProvenWithdrawalsGame returns the game address for this read of the proof record.
func (f *fakeL1) nextProvenWithdrawalsGame() common.Address {
	index := f.provenWithdrawalsCalls
	if index >= len(f.provenWithdrawalsGames) {
		index = len(f.provenWithdrawalsGames) - 1
	}
	f.provenWithdrawalsCalls++
	return f.provenWithdrawalsGames[index]
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
		values = []any{f.nextProvenWithdrawalsGame(), uint64(testGameCreatedAt + 1)}
	case "numProofSubmitters":
		values = []any{big.NewInt(int64(len(f.proofSubmitters)))}
	case "proofSubmitters":
		index, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			return nil, fmt.Errorf("failed to unpack proofSubmitters inputs: %w", err)
		}
		values = []any{f.proofSubmitters[index[1].(*big.Int).Int64()]}
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

var (
	// recordDeleted is a proof record that reads empty on every read.
	recordDeleted = []common.Address{{}}
	// recordLive is a proof record that holds a dispute game.
	recordLive = []common.Address{testGameAddress}
	// submitterRecorded is the append-only submitter list after the withdrawal was proven.
	submitterRecorded = []common.Address{testProofSubmitter}
	// submitterUnknown is the submitter list a node reports before it has the prove transaction.
	submitterUnknown []common.Address
)

// TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof pins the fix for the monitor stall.
// Anyone can delete a proof record once its game resolved in favor of the challenger or was
// blacklisted. The prove events stay in the logs forever, so the monitor must skip the event
// instead of returning an error, which would stop it from advancing its L1 cursor.
func TestGetEnrichedWithdrawalsEventsMapSkipsDeletedProof(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordDeleted,
		proofSubmitters:        submitterRecorded,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.NoError(t, err)
	require.Empty(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapFailsOnUnknownSubmitter pins the limit of the skip. A
// lagging node, a failover between nodes, or a reorg of the prove transaction all report an
// empty record and an empty submitter list. The monitor must fail the block range and retry it,
// because skipping would drop a proof that may still be live.
func TestGetEnrichedWithdrawalsEventsMapFailsOnUnknownSubmitter(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordDeleted,
		proofSubmitters:        submitterUnknown,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.ErrorIs(t, err, ErrWithdrawalProofMissing)
	require.Nil(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapFailsOnReappearingRecord covers a source node that moves
// backwards: the first read at chain head is empty, but the pinned read holds a record. The
// monitor must retry rather than skip.
func TestGetEnrichedWithdrawalsEventsMapFailsOnReappearingRecord(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: []common.Address{{}, testGameAddress},
		proofSubmitters:        submitterRecorded,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.ErrorIs(t, err, ErrWithdrawalProofMissing)
	require.Nil(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapFailsOnMissingPinnedBlock covers a node that does not hold
// the block the reads are pinned to. The call fails, so the monitor retries the range instead of
// deciding from a mix of states.
func TestGetEnrichedWithdrawalsEventsMapFailsOnMissingPinnedBlock(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordDeleted,
		proofSubmitters:        submitterRecorded,
		missingPinnedBlock:     true,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEventsMap(testProveBlock, &stop)

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrWithdrawalProofDeleted)
	require.Nil(t, events)
}

// TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof is the control for the tests above: an event
// with a proof record still in state must be enriched, not skipped.
func TestGetEnrichedWithdrawalsEventsMapKeepsLiveProof(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordLive,
		proofSubmitters:        submitterRecorded,
	})
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
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordDeleted,
		proofSubmitters:        submitterRecorded,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEvents(testProveBlock, &stop)

	require.NoError(t, err)
	require.Empty(t, events)
}

// TestGetEnrichedWithdrawalsEventsFailsOnUnknownSubmitter covers the slice variant of the
// unexplained empty record.
func TestGetEnrichedWithdrawalsEventsFailsOnUnknownSubmitter(t *testing.T) {
	validator := newTestValidator(t, fakeConfig{
		provenWithdrawalsGames: recordDeleted,
		proofSubmitters:        submitterUnknown,
	})
	stop := testProveBlock

	events, err := validator.GetEnrichedWithdrawalsEvents(testProveBlock, &stop)

	require.ErrorIs(t, err, ErrWithdrawalProofMissing)
	require.Nil(t, events)
}

// TestProofDeletionIsDeleted documents the pair of reads the skip depends on.
func TestProofDeletionIsDeleted(t *testing.T) {
	require.True(t, ProofDeletion{RecordEmpty: true, SubmitterKnown: true}.IsDeleted())
	require.False(t, ProofDeletion{RecordEmpty: true}.IsDeleted())
	require.False(t, ProofDeletion{SubmitterKnown: true}.IsDeleted())
}
