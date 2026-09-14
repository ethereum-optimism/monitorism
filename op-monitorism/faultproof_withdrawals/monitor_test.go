package faultproof_withdrawals

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Block), args.Error(1)
}

func (m *MockEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Header), args.Error(1)
}

func createTestBlock(timestamp uint64) *types.Block {
	header := &types.Header{
		Time:     timestamp,
		Number:   big.NewInt(0),
		GasLimit: 1000000,
		GasUsed:  0,
		BaseFee:  big.NewInt(0),
	}
	return types.NewBlockWithHeader(header)
}

type dynamicMock struct {
	targetTime time.Time
}

func (d *dynamicMock) BlockByNumber(ctx context.Context, num *big.Int) (*types.Block, error) {
	blockNum := num.Int64()
	timestamp := d.targetTime.Unix() + (blockNum-625)*3600 // 1 hour per block
	return createTestBlock(uint64(timestamp)), nil
}

func (d *dynamicMock) HeaderByNumber(ctx context.Context, num *big.Int) (*types.Header, error) {
	block, err := d.BlockByNumber(ctx, num)
	if err != nil {
		return nil, err
	}
	return block.Header(), nil
}

func TestGetBlockAtApproximateTimeBinarySearch(t *testing.T) {
	targetTime := time.Now().Add(-24 * time.Hour)

	t.Run("successful block finding", func(t *testing.T) {
		logger := log.New()
		mockClient := &dynamicMock{targetTime: targetTime}

		monitor := &Monitor{
			log: logger,
		}

		ctx := context.Background()
		block, err := monitor.getBlockAtApproximateTimeBinarySearch(ctx, mockClient, big.NewInt(1000), big.NewInt(24))

		assert.NoError(t, err)
		assert.Equal(t, "624", block.String())
	})

	tests := []struct {
		name          string
		latestBlock   *big.Int
		hoursInPast   *big.Int
		mockBlocks    map[string]*types.Block
		expectedBlock *big.Int
		expectedErr   error
	}{
		{
			name:        "block not found error",
			latestBlock: big.NewInt(1000),
			hoursInPast: big.NewInt(24),
			mockBlocks: map[string]*types.Block{
				"500": createTestBlock(uint64(targetTime.Add(12 * time.Hour).Unix())),
			},
			expectedBlock: nil,
			expectedErr:   errors.New("failed to get block after all retries"),
		},
		{
			name:        "exact match found",
			latestBlock: big.NewInt(1000),
			hoursInPast: big.NewInt(24),
			mockBlocks: map[string]*types.Block{
				"500": createTestBlock(uint64(targetTime.Unix())),
			},
			expectedBlock: big.NewInt(500),
			expectedErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockEthClient)
			logger := log.New()

			for blockNum, block := range tt.mockBlocks {
				num := new(big.Int)
				num.SetString(blockNum, 10)
				mockClient.On("BlockByNumber", mock.Anything, num).Return(block, nil).Maybe()
				mockClient.On("HeaderByNumber", mock.Anything, num).Return(block.Header(), nil).Maybe()
			}

			mockClient.On("BlockByNumber", mock.Anything, mock.MatchedBy(func(n *big.Int) bool {
				numStr := n.String()
				_, ok := tt.mockBlocks[numStr]
				return !ok
			})).Return(nil, errors.New("failed to get block after all retries")).Maybe()
			mockClient.On("HeaderByNumber", mock.Anything, mock.MatchedBy(func(n *big.Int) bool {
				numStr := n.String()
				_, ok := tt.mockBlocks[numStr]
				return !ok
			})).Return(nil, errors.New("failed to get block after all retries")).Maybe()

			monitor := &Monitor{
				log: logger,
			}

			ctx := context.Background()
			block, err := monitor.getBlockAtApproximateTimeBinarySearch(ctx, mockClient, tt.latestBlock, tt.hoursInPast)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBlock.String(), block.String())
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestGetBlockAtApproximateTimeBinarySearchContextCancellation(t *testing.T) {
	mockClient := new(MockEthClient)
	logger := log.New()

	mockClient.On("HeaderByNumber", mock.Anything, mock.Anything).Return(
		nil,
		errors.New("block not found"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	monitor := &Monitor{
		log: logger,
	}

	block, err := monitor.getBlockAtApproximateTimeBinarySearch(ctx, mockClient, big.NewInt(1000), big.NewInt(24))

	assert.Error(t, err)
	assert.Equal(t, "block not found", err.Error())
	assert.Nil(t, block)
	mockClient.AssertExpectations(t)
}

type monitorRunFilterCriteria struct {
	FromBlock rpc.BlockNumber `json:"fromBlock"`
	ToBlock   rpc.BlockNumber `json:"toBlock"`
}

type monitorRunCallArgs struct {
	To    common.Address `json:"to"`
	Data  hexutil.Bytes  `json:"data"`
	Input hexutil.Bytes  `json:"input"`
}

type monitorRunRPCAPI struct {
	header                    *types.Header
	startHeight               uint64
	portalAddress             common.Address
	event                     types.Log
	emptyProof                hexutil.Bytes
	provenWithdrawalsSelector []byte
	proofCalls                atomic.Uint64
}

func (a *monitorRunRPCAPI) ChainId() hexutil.Uint64 {
	return hexutil.Uint64(10)
}

func (a *monitorRunRPCAPI) BlockNumber() hexutil.Uint64 {
	return hexutil.Uint64(a.header.Number.Uint64())
}

func (a *monitorRunRPCAPI) GetBlockByNumber(
	_ context.Context,
	number rpc.BlockNumber,
	fullTransactions bool,
) (*types.Header, error) {
	if number != rpc.LatestBlockNumber {
		return nil, fmt.Errorf("expected latest block tag, got %d", number)
	}
	if fullTransactions {
		return nil, errors.New("expected header-only block request")
	}
	return a.header, nil
}

func (a *monitorRunRPCAPI) GetLogs(
	_ context.Context,
	criteria monitorRunFilterCriteria,
) ([]types.Log, error) {
	if criteria.FromBlock != rpc.BlockNumber(a.startHeight) {
		return nil, fmt.Errorf("expected scan start %d, got %d", a.startHeight, criteria.FromBlock)
	}
	capturedHeight := a.header.Number.Uint64()
	if criteria.ToBlock != rpc.BlockNumber(capturedHeight) {
		return nil, fmt.Errorf("expected scan end %d, got %d", capturedHeight, criteria.ToBlock)
	}
	return []types.Log{a.event}, nil
}

func (a *monitorRunRPCAPI) Call(
	_ context.Context,
	args monitorRunCallArgs,
	target rpc.BlockNumberOrHash,
) (hexutil.Bytes, error) {
	if args.To != a.portalAddress {
		return nil, fmt.Errorf("expected portal call to %s, got %s", a.portalAddress, args.To)
	}
	data := args.Data
	if len(data) == 0 {
		data = args.Input
	}
	if len(data) < 4 || !bytes.Equal(data[:4], a.provenWithdrawalsSelector) {
		return nil, errors.New("expected provenWithdrawals call")
	}
	if target.BlockHash == nil {
		return nil, errors.New("expected block-hash-pinned call")
	}
	if *target.BlockHash != a.header.Hash() {
		return nil, fmt.Errorf("expected captured hash %s, got %s", a.header.Hash(), *target.BlockHash)
	}
	a.proofCalls.Add(1)
	return a.emptyProof, nil
}

func TestMonitorRunAdvancesCursorAfterDeletedProof(t *testing.T) {
	const (
		startHeight    = uint64(100)
		capturedHeight = uint64(105)
	)

	portalAddress := common.HexToAddress("0x1000000000000000000000000000000000000001")
	proofSubmitter := common.HexToAddress("0x2000000000000000000000000000000000000002")
	withdrawalHash := common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	transactionHash := common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	capturedHeader := &types.Header{
		Number:     new(big.Int).SetUint64(capturedHeight),
		Difficulty: new(big.Int),
		GasLimit:   30_000_000,
		Time:       1_700_000_000,
		Extra:      []byte("captured head"),
	}

	portalABI, err := l1.OptimismPortal2MetaData.GetAbi()
	require.NoError(t, err)
	provenWithdrawals := portalABI.Methods["provenWithdrawals"]
	emptyProof, err := provenWithdrawals.Outputs.Pack(common.Address{}, uint64(0))
	require.NoError(t, err)

	api := &monitorRunRPCAPI{
		header:        capturedHeader,
		startHeight:   startHeight,
		portalAddress: portalAddress,
		event: types.Log{
			Address: portalAddress,
			Topics: []common.Hash{
				portalABI.Events["WithdrawalProvenExtension1"].ID,
				withdrawalHash,
				common.BytesToHash(proofSubmitter.Bytes()),
			},
			BlockNumber: startHeight,
			TxHash:      transactionHash,
		},
		emptyProof:                hexutil.Bytes(emptyProof),
		provenWithdrawalsSelector: provenWithdrawals.ID,
	}
	rpcServer := rpc.NewServer()
	require.NoError(t, rpcServer.RegisterName("eth", api))
	server := httptest.NewServer(rpcServer)
	t.Cleanup(func() {
		server.Close()
		rpcServer.Stop()
	})

	ctx := context.Background()
	registry := opmetrics.NewRegistry()
	logger := testlog.Logger(t, log.LevelDebug)
	monitor, err := NewMonitor(ctx, logger, opmetrics.With(registry), CLIConfig{
		L1GethURL:             server.URL,
		L2OpGethURL:           server.URL,
		EventBlockRange:       20,
		StartingL1BlockHeight: int64(startHeight),
		OptimismPortalAddress: portalAddress,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = monitor.Close(ctx)
	})

	monitor.Run(ctx)

	require.Equal(t, uint64(1), api.proofCalls.Load())
	require.Zero(t, monitor.state.eventsProcessed)
	require.Zero(t, monitor.state.withdrawalsProcessed)
	require.Equal(t, capturedHeight, monitor.state.nextL1Height)
}
