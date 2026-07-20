package processor

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filterRPCAPI struct {
	handle func(context.Context, testFilterCriteria) ([]types.Log, error)
}

type testFilterCriteria struct {
	FromBlock rpc.BlockNumber  `json:"fromBlock"`
	ToBlock   rpc.BlockNumber  `json:"toBlock"`
	Addresses []common.Address `json:"address"`
	Topics    [][]common.Hash  `json:"topics"`
}

func (a *filterRPCAPI) GetLogs(ctx context.Context, criteria testFilterCriteria) ([]types.Log, error) {
	return a.handle(ctx, criteria)
}

func newRPCClient(
	t *testing.T,
	handle func(context.Context, testFilterCriteria) ([]types.Log, error),
) (*ethclient.Client, func()) {
	t.Helper()
	// Keep the ethclient boundary in the test: FilterCriteria is decoded by the
	// JSON-RPC server exactly as it is by a real node, while DialInProc avoids a
	// listening socket and keeps cancellation deterministic.
	server := rpc.NewServer()
	require.NoError(t, server.RegisterName("eth", &filterRPCAPI{handle: handle}))
	rpcClient := rpc.DialInProc(server)
	client := ethclient.NewClient(rpcClient)
	return client, func() {
		client.Close()
		server.Stop()
	}
}

func testProcessor(client *ethclient.Client, process LogProcessingFunc) *BlockProcessor {
	ctx, cancel := context.WithCancel(context.Background())
	return &BlockProcessor{
		client:         client,
		logProcessFunc: process,
		ctx:            ctx,
		cancel:         cancel,
		log:            log.New(),
		retryDelay:     time.Millisecond,
		metrics: Metrics{
			processingErrors: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_processing_errors_total"}),
		},
	}
}

func TestFilteredLogsQueryOrderingDispatchAndRetry(t *testing.T) {
	address := common.HexToAddress("0x1000000000000000000000000000000000000001")
	topic := common.HexToHash("0x1234")
	logs := []types.Log{
		{Address: address, Topics: []common.Hash{topic}, BlockNumber: 42, TxIndex: 2, Index: 9},
		{Address: address, Topics: []common.Hash{topic}, BlockNumber: 42, TxIndex: 0, Index: 2},
		{Address: address, Topics: []common.Hash{topic}, BlockNumber: 42, TxIndex: 1, Index: 5},
	}

	var attempts atomic.Int32
	var query testFilterCriteria
	client, closeClient := newRPCClient(t, func(_ context.Context, criteria testFilterCriteria) ([]types.Log, error) {
		query = criteria
		if attempts.Add(1) == 1 {
			return nil, errors.New("temporary failure")
		}
		return logs, nil
	})
	defer closeClient()

	var dispatched []uint
	processor := testProcessor(client, func(_ *types.Block, lg types.Log, _ *ethclient.Client) error {
		dispatched = append(dispatched, lg.Index)
		return nil
	})
	processor.logFilterAddresses = []common.Address{address}
	processor.logFilterTopics = [][]common.Hash{{topic}}
	block := types.NewBlockWithHeader(&types.Header{Number: big.NewInt(42)})

	require.NoError(t, processor.processFilteredLogs(block))
	assert.Equal(t, int32(2), attempts.Load(), "a transient eth_getLogs failure is retried")
	assert.Equal(t, rpc.BlockNumber(42), query.FromBlock)
	assert.Equal(t, rpc.BlockNumber(42), query.ToBlock)
	assert.Equal(t, []common.Address{address}, query.Addresses)
	assert.Equal(t, [][]common.Hash{{topic}}, query.Topics)
	assert.Equal(t, []uint{2, 5, 9}, dispatched, "callbacks run in canonical block-log order")
}

func TestFilteredLogsRetryCancellation(t *testing.T) {
	attempted := make(chan struct{}, 1)
	client, closeClient := newRPCClient(t, func(_ context.Context, _ testFilterCriteria) ([]types.Log, error) {
		attempted <- struct{}{}
		return nil, errors.New("still unavailable")
	})
	defer closeClient()

	processor := testProcessor(client, nil)
	processor.retryDelay = time.Hour
	processor.logFilterAddresses = []common.Address{common.HexToAddress("0x1")}
	done := make(chan error, 1)
	go func() {
		_, err := processor.getFilteredLogsWithRetry(big.NewInt(42))
		done <- err
	}()

	select {
	case <-attempted:
	case <-time.After(time.Second):
		t.Fatal("eth_getLogs was not attempted")
	}
	processor.cancel()
	select {
	case err := <-done:
		assert.True(t, errors.Is(err, context.Canceled), "got %v", err)
	case <-time.After(time.Second):
		t.Fatal("filtered-log retry did not stop promptly on cancellation")
	}
}
