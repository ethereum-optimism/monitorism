package fault

import (
	"context"
	"errors"
	// "math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	// "github.com/ethereum/go-ethereum/params"
	// "github.com/ethereum/go-ethereum/rpc"

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
	l1RpcProvider        string
	l2RpcProvider        string
	startOutputIndex uint64
	optimismPortalAddress       string
}

type Monitor struct {
	log log.Logger

	worker         *clock.LoopFn
	stopped        atomic.Bool

	highestOutputIndex *prometheus.GaugeVec

	isCurrentlyMismatched *prometheus.GaugeVec
	nodeConnectionFailures *prometheus.GaugeVec
	l1RpcProvider      client.RPC
	l2RpcProvider      client.RPC
	startOutputIndex uint64
  optimismPortalAddress string
}

func NewMonitor(ctx context.Context, log log.Logger, cfg Config, m metrics.Factory) (*Monitor, error) {
	log.Info("Creating the fault monitor.")
	l1RpcProvider, err := client.NewRPC(ctx, log, cfg.l1RpcProvider)
	if err != nil {
		return nil, err
	}

	l2RpcProvider, err := client.NewRPC(ctx, log, cfg.l2RpcProvider)
	if err != nil {
		return nil, err
	}
	return &Monitor{
		log:            log,
		l1RpcProvider: l1RpcProvider,
    l2RpcProvider: l2RpcProvider,
		highestOutputIndex: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestOutputIndex",
			Help:      "Highest output indices (checked and known)",
		}, []string{"address", "nickname"}),
		isCurrentlyMismatched: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "isCurrentlyMismatched",
			Help:      "0 if state is ok, 1 if state is mismatched",
		}, []string{"address", "nickname"}),

		nodeConnectionFailures: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "nodeConnectionFailures",
			Help:      "Number of times node connection has failed", // Probably need to use the labels: ['layer', 'section'],

		}, []string{"address", "nickname"}),
		startOutputIndex:      cfg.startOutputIndex,
		optimismPortalAddress: cfg.optimismPortalAddress,
	}, nil
}

func (b *Monitor) Start(ctx context.Context) error {
	if b.worker != nil {
		return errors.New("fault monitor already started")
	}

	b.log.Info("starting fault monitor...", "optimismPortalAddress", b.optimismPortalAddress)
	b.tick(ctx)
  b.worker = clock.NewLoopFn(clock.SystemClock, b.tick, nil, time.Millisecond*time.Duration(1000)) //TODO: hardcode 1000 here but should be different in the future.
	return nil
}

func (b *Monitor) Stop(_ context.Context) error {
	b.log.Info("stopping fault monitor...")
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
	 b.log.Info("Checking if the submitted outputRoot is correct....")
	// batchElems := make([]rpc.BatchElem, len(b.accounts))
	// for i := 0; i < len(b.accounts); i++ {
	// 	batchElems[i] = rpc.BatchElem{
	// 		Method: "eth_getBalance",
	// 		Args:   []interface{}{b.accounts[i].Address, "latest"},
	// 		Result: new(hexutil.Big),
	// 	}
	// }
	// if err := b.rpc.BatchCallContext(ctx, batchElems); err != nil {
	// 	b.log.Error("failed getBalance batch request", "err", err)
	// 	return
	// }
	//
	// // TODO: Metric for client errors
	//
	// for i := 0; i < len(b.accounts); i++ {
	// 	account := b.accounts[i]
	// 	if batchElems[i].Error != nil {
	// 		b.log.Error("failed to query account balance", "address", account.Address, "nickname", account.Nickname, "err", batchElems[i].Error)
	// 		continue
	// 	}
	//
	// 	ethBalance := weiToEther((batchElems[i].Result).(*hexutil.Big).ToInt())
	// 	b.balances.WithLabelValues(account.Address.String(), account.Nickname).Set(ethBalance)
	// 	b.log.Info("set balance", "address", account.Address, "nickname", account.Nickname, "balance", ethBalance)
	// }
}

