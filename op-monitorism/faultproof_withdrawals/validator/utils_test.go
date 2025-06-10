package validator

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

func TestRetryBlockNumber(t *testing.T) {
	tests := []struct {
		name          string
		blockNumber   *big.Int
		maxRetries    int
		mockResponses []struct {
			block *types.Block
			err   error
		}
		expectedBlock *types.Block
		expectedErr   error
	}{
		{
			name:        "success on first try",
			blockNumber: big.NewInt(100),
			maxRetries:  3,
			mockResponses: []struct {
				block *types.Block
				err   error
			}{
				{&types.Block{}, nil},
			},
			expectedBlock: &types.Block{},
			expectedErr:   nil,
		},
		{
			name:        "success after retries",
			blockNumber: big.NewInt(100),
			maxRetries:  3,
			mockResponses: []struct {
				block *types.Block
				err   error
			}{
				{nil, errors.New("first error")},
				{nil, errors.New("second error")},
				{&types.Block{}, nil},
			},
			expectedBlock: &types.Block{},
			expectedErr:   nil,
		},
		{
			name:        "max retries exceeded",
			blockNumber: big.NewInt(100),
			maxRetries:  2,
			mockResponses: []struct {
				block *types.Block
				err   error
			}{
				{nil, errors.New("first error")},
				{nil, errors.New("second error")},
				{nil, errors.New("third error")},
			},
			expectedBlock: nil,
			expectedErr:   errors.New("failed to get block after all retries"),
		},
		{
			name:        "infinite retries (maxRetries < 1)",
			blockNumber: big.NewInt(100),
			maxRetries:  0,
			mockResponses: []struct {
				block *types.Block
				err   error
			}{
				{nil, errors.New("first error")},
				{nil, errors.New("second error")},
				{nil, errors.New("third error")},
			},
			expectedBlock: nil,
			expectedErr:   errors.New("context cancelled during block fetch retry"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockEthClient)
			logger := log.New()

			// Set up mock responses in sequence
			for i, response := range tt.mockResponses {
				call := mockClient.On("BlockByNumber", mock.Anything, tt.blockNumber)
				if i == len(tt.mockResponses)-1 {
					call.Return(response.block, response.err)
				} else {
					call.Return(response.block, response.err).Once()
				}
			}

			// Use a timeout context for infinite retry test
			ctx := context.Background()
			if tt.maxRetries < 1 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
			}

			block, err := RetryBlockNumber(ctx, mockClient, logger, tt.blockNumber, tt.maxRetries)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBlock, block)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestRetryBlockNumberContextCancellation(t *testing.T) {
	mockClient := new(MockEthClient)
	logger := log.New()
	blockNumber := big.NewInt(100)

	// Set up mock to return error
	mockClient.On("BlockByNumber", mock.Anything, blockNumber).Return(nil, errors.New("test error"))

	// Create a context that will be cancelled after a short delay
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	block, err := RetryBlockNumber(ctx, mockClient, logger, blockNumber, 3)

	assert.Error(t, err)
	assert.Equal(t, "context cancelled during block fetch retry", err.Error())
	assert.Nil(t, block)
}

func TestRetryLatestBlock(t *testing.T) {
	mockClient := new(MockEthClient)
	logger := log.New()

	// Set up mock to return a block
	expectedBlock := &types.Block{}
	mockClient.On("BlockByNumber", mock.Anything, (*big.Int)(nil)).Return(expectedBlock, nil)

	ctx := context.Background()
	block, err := RetryLatestBlock(ctx, mockClient, logger, 3)

	assert.NoError(t, err)
	assert.Equal(t, expectedBlock, block)
	mockClient.AssertExpectations(t)
}
