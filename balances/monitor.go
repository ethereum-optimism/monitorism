package balances

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "monitorism"
)

type Account struct {
	Address  common.Address
	Nickname string
}

type Config struct {
	NodeUrl        string
	LoopIntervalMs uint64
	Accounts       []Account
}

type Monitor struct {
	log log.Logger

	loopIntervalMs uint64
	worker         *clock.LoopFn
	stopped        atomic.Bool

	balances *prometheus.GaugeVec

	rpc      client.RPC
	accounts []Account
}

func NewMonitor(ctx context.Context, log log.Logger, cfg Config, m metrics.Factory) (*Monitor, error) {
	log.Info("creating monitor")
	rpc, err := client.NewRPC(ctx, log, cfg.NodeUrl)
	if err != nil {
		return nil, err
	}

	if len(cfg.Accounts) == 0 {
		return nil, errors.New("no accounts configured")
	}
	for _, account := range cfg.Accounts {
		log.Info("configured account", "address", account.Address, "nickname", account.Nickname)
	}

	return &Monitor{
		log:            log,
		loopIntervalMs: cfg.LoopIntervalMs,

		balances: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "balance",
			Help:      "Balances held by accounts registered with the monitor",
		}, []string{"address", "nickname"}),

		rpc:      rpc,
		accounts: cfg.Accounts,
	}, nil
}

func (b *Monitor) Start(ctx context.Context) error {
	if b.worker != nil {
		return errors.New("balance monitor already started")
	}

	b.log.Info("starting balance monitor...", "loop_interval_ms", b.loopIntervalMs)
	b.tick(ctx)
	b.worker = clock.NewLoopFn(clock.SystemClock, b.tick, nil, time.Millisecond*time.Duration(b.loopIntervalMs))
	return nil
}

func (b *Monitor) Stop(_ context.Context) error {
	b.log.Info("stopping balance monitor...")
	err := b.worker.Close()
	if err == nil {
		b.stopped.Store(true)
	}

	return err
}

func (b *Monitor) Stopped() bool {
	return b.stopped.Load()
}

func (b *Monitor) tick(ctx context.Context) {
	b.log.Info("querying balances...")
	batchElems := make([]rpc.BatchElem, len(b.accounts))
	for i := 0; i < len(b.accounts); i++ {
		batchElems[i] = rpc.BatchElem{
			Method: "eth_getBalance",
			Args:   []interface{}{b.accounts[i].Address, "latest"},
			Result: new(hexutil.Big),
		}
	}
	if err := b.rpc.BatchCallContext(ctx, batchElems); err != nil {
		b.log.Error("failed getBalance batch request", "err", err)
		return
	}

	// TODO: Metric for client errors

	for i := 0; i < len(b.accounts); i++ {
		account := b.accounts[i]
		if batchElems[i].Error != nil {
			b.log.Error("failed to query account balance", "address", account.Address, "nickname", account.Nickname, "err", batchElems[i].Error)
			continue
		}

		ethBalance := weiToEther((batchElems[i].Result).(*hexutil.Big).ToInt())
		b.balances.WithLabelValues(account.Address.String(), account.Nickname).Set(ethBalance)
		b.log.Info("set balance", "address", account.Address, "nickname", account.Nickname, "balance", ethBalance)
	}
}

func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
