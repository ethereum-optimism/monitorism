package faultproof_withdrawals

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
