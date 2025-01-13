package balances

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "balance_mon"
)

type Account struct {
	Address  common.Address
	Nickname string
}

type Monitor struct {
	log log.Logger

	rpc      client.RPC
	accounts []Account

	// metrics
	balances            *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating balance monitor")
	rpc, err := client.NewRPC(ctx, log, cfg.NodeUrl)
	if err != nil {
		return nil, err
	}

	for _, account := range cfg.Accounts {
		log.Info("configured account", "address", account.Address, "nickname", account.Nickname)
	}

	return &Monitor{
		log:      log,
		rpc:      rpc,
		accounts: cfg.Accounts,

		balances: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "balances",
			Help:      "balances held by accounts registered with the monitor",
		}, []string{"address", "nickname"}),
		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpectedRpcErrors",
			Help:      "number of unexpected rpc errors",
		}, []string{"section", "name"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	m.log.Info("querying balances...")
	batchElems := make([]rpc.BatchElem, len(m.accounts))
	for i := 0; i < len(m.accounts); i++ {
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBalance",
			Args:   []interface{}{m.accounts[i].Address, "latest"},
			Result: new(hexutil.Big),
		}
	}
	if err := m.rpc.BatchCallContext(ctx, batchElems); err != nil {
		m.log.Error("failed getBalance batch request", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("balances", "batched_getBalance").Inc()
		return
	}

	for i := 0; i < len(m.accounts); i++ {
		account := m.accounts[i]
		if batchElems[i].Error != nil {
			m.log.Error("failed to query account balance", "address", account.Address, "nickname", account.Nickname, "err", batchElems[i].Error)
			m.unexpectedRpcErrors.WithLabelValues("balances", "getBalance").Inc()
			continue
		}

		ethBalance := weiToEther((batchElems[i].Result).(*hexutil.Big).ToInt())
		m.balances.WithLabelValues(account.Address.String(), account.Nickname).Set(ethBalance)
		m.log.Info("set balance", "address", account.Address, "nickname", account.Nickname, "balance", ethBalance)
	}
}

func (m *Monitor) Close(_ context.Context) error {
	m.rpc.Close()
	return nil
}

func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
