package transaction_monitor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testutils/anvil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var (
	// Anvil test accounts
	watchedAddress   = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	allowedAddress   = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	unauthorizedAddr = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
	factoryAddress   = common.HexToAddress("0x90F79bf6EB2c4f870365E785982E1f101E93b906")

	// Private keys
	watchedKey, _      = crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	allowedKey, _      = crypto.HexToECDSA("59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")
	unauthorizedKey, _ = crypto.HexToECDSA("5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a")
)

func setupAnvil(t *testing.T) (*anvil.Runner, *ethclient.Client, string) {
	anvil.Test(t)

	ctx := context.Background()
	logger := log.New()

    anvilRunner, err := anvil.New("http://127.0.1:8545", logger)
	require.NoError(t, err)

	err = anvilRunner.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = anvilRunner.Stop()
	})

	client, err := ethclient.Dial(anvilRunner.RPCUrl())
	require.NoError(t, err)

	return anvilRunner, client, anvilRunner.RPCUrl()
}

func sendTx(t *testing.T, ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, to common.Address, value *big.Int) {
	nonce, err := client.PendingNonceAt(ctx, crypto.PubkeyToAddress(key.PublicKey))
	require.NoError(t, err)

	gasPrice, err := client.SuggestGasPrice(ctx)
	require.NoError(t, err)

	tx := types.NewTransaction(nonce, to, value, 21000, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(31337)), key)
	require.NoError(t, err)

	err = client.SendTransaction(ctx, signedTx)
	require.NoError(t, err)

	// Wait for receipt
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
		if err == nil {
			require.Equal(t, uint64(1), receipt.Status, "transaction failed")
			return
		}
	}
	t.Fatal("timeout waiting for transaction receipt")
}

func TestTransactionMonitoring(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, client, rpc := setupAnvil(t)

	factory := factoryAddress

	cfg := CLIConfig{
		NodeUrl:     rpc,
		StartBlock:  0,
		WatchConfigs: []WatchConfig{{
			Address: watchedAddress,
			Filters: []CheckConfig{
				{
					Type: ExactMatchCheck,
					Params: map[string]interface{}{
						"match": allowedAddress.Hex(),
					},
				},
				{
					Type: DisputeGameCheck,
					Params: map[string]interface{}{
						"disputeGameFactory": factory.Hex(),
					},
				},
			},
		}},
	}

	registry := opmetrics.NewRegistry()
	monitor, err := NewMonitor(ctx, log.New(), opmetrics.With(registry), cfg)
	require.NoError(t, err)

	// Start monitor in background
	go monitor.Run(ctx)
	defer monitor.Close(ctx)

	t.Run("allowed address transaction", func(t *testing.T) {
		// Send transaction to allowed address
		sendTx(t, ctx, client, watchedKey, allowedAddress, big.NewInt(params.Ether))
		time.Sleep(2 * time.Second)

		// Check metrics
		require.Equal(t, float64(1), getCounterValue(t, monitor.transactions, watchedAddress.Hex(), allowedAddress.Hex(), "processed"))
		require.Equal(t, float64(1.0), getCounterValue(t, monitor.ethSpent, watchedAddress.Hex()))
		require.Equal(t, float64(0), getCounterValue(t, monitor.unauthorizedTx, watchedAddress.Hex()))
	})

	t.Run("unauthorized address", func(t *testing.T) {
		sendTx(t, ctx, client, watchedKey, unauthorizedAddr, big.NewInt(params.Ether/2))
		time.Sleep(2 * time.Second)

		require.Equal(t, float64(1), getCounterValue(t, monitor.unauthorizedTx, watchedAddress.Hex()))
		require.Equal(t, float64(1.5), getCounterValue(t, monitor.ethSpent, watchedAddress.Hex()))
	})

	t.Run("multiple blocks processed", func(t *testing.T) {
		// Send multiple transactions to generate blocks
		for i := 0; i < 3; i++ {
			sendTx(t, ctx, client, watchedKey, allowedAddress, big.NewInt(params.Ether/10))
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)

		// Get the latest block number
		blockNum, err := client.BlockNumber(ctx)
		require.NoError(t, err)

		// Check if blocks were processed
		total := float64(0)
		for i := uint64(0); i <= blockNum; i++ {
			total += getCounterValue(t, monitor.blocksProcessed, fmt.Sprint(i))
		}
		require.Greater(t, total, float64(0), "should have processed some blocks")
	})

	t.Run("RPC errors handling", func(t *testing.T) {
		// Create a new registry for the bad monitor
		badRegistry := opmetrics.NewRegistry()
		
		// Create a monitor with invalid RPC URL to trigger errors
		badCfg := cfg
		badCfg.NodeUrl = "http://nonexistent:8545"
		
		badMonitor, err := NewMonitor(ctx, log.New(), opmetrics.With(badRegistry), badCfg)
		require.NoError(t, err)
		
		go badMonitor.Run(ctx)
		time.Sleep(2 * time.Second)
		
		// Should have recorded some RPC errors
		require.Greater(t, 
			getCounterValue(t, badMonitor.unexpectedRpcErrors, "monitor", "blockNumber"),
			float64(0),
			"should have recorded RPC errors")
		
		badMonitor.Close(ctx)
	})

	t.Run("multiple unauthorized transactions", func(t *testing.T) {
		// Send multiple unauthorized transactions
		for i := 0; i < 3; i++ {
			sendTx(t, ctx, client, watchedKey, unauthorizedAddr, big.NewInt(params.Ether/4))
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)

		require.Equal(t, float64(4), getCounterValue(t, monitor.unauthorizedTx, watchedAddress.Hex()))
		require.Equal(t, float64(2.55), getCounterValue(t, monitor.ethSpent, watchedAddress.Hex()))
	})
}

func TestDisputeGameVerifier(t *testing.T) {
	ctx := context.Background()
	_, _, rpc := setupAnvil(t)

	factory := factoryAddress

	cfg := CLIConfig{
		NodeUrl:     rpc,
		StartBlock:  0,
		WatchConfigs: []WatchConfig{{
			Address: watchedAddress,
			Filters: []CheckConfig{{
				Type: DisputeGameCheck,
				Params: map[string]interface{}{
					"disputeGameFactory": factory.Hex(),
				},
			}},
		}},
	}

	registry := opmetrics.NewRegistry()
	monitor, err := NewMonitor(ctx, log.New(), opmetrics.With(registry), cfg)
	require.NoError(t, err)

    go monitor.Run(ctx)
	defer monitor.Close(ctx)

	t.Run("verifier setup", func(t *testing.T) {
		// Verify the verifier is set up correctly
		require.Contains(t, monitor.gameVerifiers, factory)
		require.NotNil(t, monitor.gameVerifiers[factory].cache)
		require.Equal(t, factory, monitor.gameVerifiers[factory].factory)
	})
}

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labelValues ...string) float64 {
	m, err := counter.GetMetricWithLabelValues(labelValues...)
	require.NoError(t, err)
	return testutil.ToFloat64(m)
}
