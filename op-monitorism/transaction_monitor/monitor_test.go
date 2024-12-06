package transaction_monitor

import (
	"context"
	"math/big"
	"testing"
    "crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testutils/anvil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var (
	// Anvil test accounts
	watchedAddress    = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	allowedAddress    = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
	unauthorizedAddr  = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
	
	// Private keys
	allowedKey, _     = crypto.HexToECDSA("59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")
	unauthorizedKey, _ = crypto.HexToECDSA("5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a")
)

func setupAnvil(t *testing.T) (*anvil.Runner, *ethclient.Client, string) {
	anvil.Test(t)

	ctx := context.Background()
	logger := log.New()
	anvilRunner, err := anvil.New("http://localhost:8545", logger)
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

func TestTransactionMonitoring(t *testing.T) {
	ctx := context.Background()
	_, client, rpc := setupAnvil(t)

	threshold := big.NewInt(params.Ether)
	cfg := CLIConfig{
		L1NodeUrl: rpc,
		WatchConfigs: []WatchConfig{{
			Address:    watchedAddress,
			AllowList:  []common.Address{allowedAddress},
			Thresholds: map[string]*big.Int{allowedAddress.Hex(): threshold},
		}},
	}

	registry := metrics.NewRegistry()
	metricsFactory := metrics.With(registry)

	monitor, err := NewMonitor(ctx, log.New(), metricsFactory, cfg)
	require.NoError(t, err)
	defer monitor.Close(ctx)

	sendTx := func(key *ecdsa.PrivateKey, to common.Address, value *big.Int) {
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

	t.Run("allowed address below threshold", func(t *testing.T) {
		sendTx(allowedKey, watchedAddress, big.NewInt(params.Ether/2))
		monitor.Run(ctx)
		require.Equal(t, float64(1), getCounterValue(t, monitor.transactions, watchedAddress.Hex(), allowedAddress.Hex(), watchedAddress.Hex(), "processed"))
		require.Equal(t, float64(0), getCounterValue(t, monitor.thresholdExceededTx, watchedAddress.Hex(), allowedAddress.Hex(), threshold.String()))
	})

	t.Run("allowed address above threshold", func(t *testing.T) {
		sendTx(allowedKey, watchedAddress, big.NewInt(params.Ether*2))
		monitor.Run(ctx)
		require.Equal(t, float64(2), getCounterValue(t, monitor.transactions, watchedAddress.Hex(), allowedAddress.Hex(), watchedAddress.Hex(), "processed"))
		require.Equal(t, float64(1), getCounterValue(t, monitor.thresholdExceededTx, watchedAddress.Hex(), allowedAddress.Hex(), threshold.String()))
	})

	t.Run("unauthorized address", func(t *testing.T) {
		sendTx(unauthorizedKey, watchedAddress, big.NewInt(params.Ether/2))
		monitor.Run(ctx)
		require.Equal(t, float64(1), getCounterValue(t, monitor.unauthorizedTx, watchedAddress.Hex(), unauthorizedAddr.Hex()))
	})
}

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labelValues ...string) float64 {
	m, err := counter.GetMetricWithLabelValues(labelValues...)
	require.NoError(t, err)
	return testutil.ToFloat64(m)
}
