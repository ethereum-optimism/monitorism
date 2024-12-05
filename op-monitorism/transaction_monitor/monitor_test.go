package transaction_monitor

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

    "github.com/ethereum-optimism/optimism/op-service/metrics"
    "github.com/prometheus/client_golang/prometheus/testutil"
)

type testAccount struct {
	key     *ecdsa.PrivateKey
	address common.Address
}

func setupTestAccounts(t *testing.T) []testAccount {
	accounts := make([]testAccount, 3)
	for i := range accounts {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		accounts[i] = testAccount{
			key:     key,
			address: crypto.PubkeyToAddress(key.PublicKey),
		}
	}
	return accounts
}

func setupTestNode(t *testing.T, accounts []testAccount) (*node.Node, *ethclient.Client) {
	alloc := make(core.GenesisAlloc)
	fundAmount := new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))
	for _, acc := range accounts {
		alloc[acc.address] = core.GenesisAccount{Balance: fundAmount}
	}

	genesis := &core.Genesis{
		Config:    params.AllEthashProtocolChanges,
		Alloc:     alloc,
		ExtraData: []byte("test genesis"),
		Timestamp: uint64(time.Now().Unix()),
		BaseFee:   big.NewInt(params.InitialBaseFee),
	}

	nodeConfig := &node.Config{
		Name:    "test-node",
		P2P:     p2p.Config{NoDiscovery: true},
		HTTPHost: "127.0.0.1",
		HTTPPort: 0,
	}

	n, err := node.New(nodeConfig)
	require.NoError(t, err)

	ethConfig := &ethconfig.Config{
		Genesis: genesis,
		RPCGasCap: 1000000,
	}

	ethservice, err := eth.New(n, ethConfig)
	require.NoError(t, err)

	err = n.Start()
	require.NoError(t, err)

	client, err := ethclient.Dial(n.HTTPEndpoint())
	require.NoError(t, err)

	// Wait for transaction indexing to complete
	for {
		progress, err := ethservice.BlockChain().TxIndexProgress()
		if err == nil && progress.Done() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return n, client
}

func TestTransactionMonitoring(t *testing.T) {
	ctx := context.Background()
	accounts := setupTestAccounts(t)
	watchedAddr := accounts[0].address
	allowedAddr := accounts[1].address
	unauthorizedAddr := accounts[2].address

	node, client := setupTestNode(t, accounts)
	defer node.Close()

	threshold := big.NewInt(params.Ether)
	cfg := CLIConfig{
		L1NodeUrl: node.HTTPEndpoint(),
		WatchConfigs: []WatchConfig{{
			Address:    watchedAddr,
			AllowList:  []common.Address{allowedAddr},
			Thresholds: map[string]*big.Int{allowedAddr.Hex(): threshold},
		}},
	}

    registry := metrics.NewRegistry()
    metricsFactory := metrics.With(registry)

	monitor, err := NewMonitor(ctx, log.New(), metricsFactory, cfg)
	require.NoError(t, err)
	
    defer monitor.Close(ctx)

	sendTx := func(from *ecdsa.PrivateKey, to common.Address, value *big.Int) {
		nonce, err := client.PendingNonceAt(ctx, crypto.PubkeyToAddress(from.PublicKey))
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(ctx)
		require.NoError(t, err)

		tx := types.NewTransaction(nonce, to, value, 21000, gasPrice, nil)
		signedTx, err := types.SignTx(tx, types.NewLondonSigner(big.NewInt(1337)), from)
		require.NoError(t, err)

		err = client.SendTransaction(ctx, signedTx)
		require.NoError(t, err)

		receipt, err := client.TransactionReceipt(ctx, signedTx.Hash())
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)
	}

	t.Run("allowed address below threshold", func(t *testing.T) {
		sendTx(accounts[1].key, watchedAddr, big.NewInt(params.Ether/2))
		monitor.Run(ctx)
		require.Equal(t, float64(1), getCounterValue(t, monitor.transactions, watchedAddr.Hex(), allowedAddr.Hex(), watchedAddr.Hex(), "processed"))
		require.Equal(t, float64(0), getCounterValue(t, monitor.thresholdExceededTx, watchedAddr.Hex(), allowedAddr.Hex(), threshold.String()))
	})

	t.Run("allowed address above threshold", func(t *testing.T) {
		sendTx(accounts[1].key, watchedAddr, big.NewInt(params.Ether*2))
		monitor.Run(ctx)
		require.Equal(t, float64(2), getCounterValue(t, monitor.transactions, watchedAddr.Hex(), allowedAddr.Hex(), watchedAddr.Hex(), "processed"))
		require.Equal(t, float64(1), getCounterValue(t, monitor.thresholdExceededTx, watchedAddr.Hex(), allowedAddr.Hex(), threshold.String()))
	})

	t.Run("unauthorized address", func(t *testing.T) {
		sendTx(accounts[2].key, watchedAddr, big.NewInt(params.Ether/2))
		monitor.Run(ctx)
		require.Equal(t, float64(1), getCounterValue(t, monitor.unauthorizedTx, watchedAddr.Hex(), unauthorizedAddr.Hex()))
	})
}

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labelValues ...string) float64 {
	m, err := counter.GetMetricWithLabelValues(labelValues...)
	require.NoError(t, err)
	return testutil.ToFloat64(m)
}
