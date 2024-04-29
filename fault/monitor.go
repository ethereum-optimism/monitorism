package fault

import (
	"context"
	"errors"
	// "math/big"
	"sync/atomic"
	"time"
  "fmt"
  "encoding/hex"
	// "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
   "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
  // "log"
	// "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/prometheus/client_golang/prometheus"
)
 // go run . fault --l1RpcProvider consensus.X.X --l2RpcProvider http://localhost:8545

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
	l1RpcProviderClient  *ethclient.Client
	l2RpcProviderClient *rpc.Client
	startOutputIndex uint64
  optimismPortalAddress string
}

func NewMonitor(ctx context.Context, log log.Logger, cfg Config, m metrics.Factory) (*Monitor, error) {
	log.Info("Creating the fault monitor.")
	l1RpcProvider, err := ethclient.Dial(cfg.l1RpcProvider)
	if err != nil {
		return nil, err
	}

	l2RpcProvider, err := rpc.DialContext(ctx, cfg.l2RpcProvider)
	if err != nil {
		return nil, err
	}
  // create a new rpc client for l1 and l2
  

  

	return &Monitor{
		log:            log,
		l1RpcProviderClient: l1RpcProvider,
    l2RpcProviderClient: l2RpcProvider,
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

  fmt.Println("Starting the Tick")
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

// Proof holds the structure of the response from eth_getProof
type Proof struct {
    AccountProof []string               `json:"accountProof"`
    StorageHash  map[string]interface{} `json:"storageHash"`
    Balance      string                 `json:"balance"`
    CodeHash     string                 `json:"codeHash"`
    Nonce        string                 `json:"nonce"`
    StorageProof []StorageProof         `json:"storageProof"`
}

type StorageProof struct {
    Key   string   `json:"key"`
    Value string   `json:"value"`
    Proof []string `json:"proof"`
}

// FetchProof uses the underlying client's `CallContext` method to make a generic JSON-RPC API call.
func FetchProof(client *rpc.Client, accountAddress string, keys []string, blockNumber int64) (*Proof, error) {
    var proof Proof
    args := map[string]interface{}{
        "address":     accountAddress,
        "storageKeys": keys,
        "blockNumber": fmt.Sprintf("0x%x", blockNumber),
    }

    err := client.CallContext(context.Background(), &proof, "eth_getProof", args)
    if err != nil {
        return nil, err
    }
    return &proof, nil
}


func (b *Monitor) tick(ctx context.Context) {
	 b.log.Info("Checking if the submitted outputRoot is correct....")

   address := common.HexToAddress("0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F")

   l2outputOracle, err := bindings.NewL2OutputOracle(address, b.l1RpcProviderClient)
   if err != nil {
     b.log.Crit("Failed to instantiate a NewL2OutputOracle contract: %v", err)
     b.Stop(ctx)
   }

	 l2ooBlockNumber, err := l2outputOracle.LatestBlockNumber(&bind.CallOpts{})
   println("the current block is:",l2ooBlockNumber.Int64())
 


	committedL2Output, err := l2outputOracle.GetL2OutputAfter(&bind.CallOpts{}, l2ooBlockNumber) //Get the OutputRoot from the L1.
  if err != nil {
     b.log.Crit("`GetL2Output()` failed details here ->", err)
     b.Stop(ctx)
  }
  emptyStringArray := []string{}
  hexOutputRoot := hex.EncodeToString(committedL2Output.OutputRoot[:]) // L2OutputRoot from the L1.
  b.log.Info(fmt.Sprintf("OutputRoot retrieve for `l2outputOracle` in (L1): %s", hexOutputRoot))
  value, err := FetchProof(b.l2RpcProviderClient, "0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F",emptyStringArray,l2ooBlockNumber.Int64()) // Make the fetch proof from the L2 the function is panicking somehow (I am stuck there for now).
  if err != nil {
     b.log.Crit("`FetchProof` failed details here ->", err)
     b.Stop(ctx)
  }
  OutputRootFromEthProof := ""  // this has the be the reconstructed with the outputRoot from the FetchProof (eth_getProof).
  fmt.Println(value) // Just to know if the code is executed until here. For Now I am stuck into the FetchProof that return an error (about the size of the mapping).
  // 1. Now we first reconstruct the outputRoot from the proof.
  // Using this -> keccak256(0,value.stateRoot, value.storagehash, blockhash) block hash is not present in current object `proof` and need to be modified.
  

  // Then we compare the outputRoot from the proof with the outputRoot from the L1.
  if OutputRootFromEthProof == hexOutputRoot {
    b.log.Info("The outputRoot is correct.")
    // log with prometheus the matched state state.
    b.isCurrentlyMismatched.WithLabelValues("0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F", "l2outputOracle").Set(0)
  } else {
    b.log.Crit("The outputRoot is incorrect.")
    // log with prometheus the mismatched state state.
    b.isCurrentlyMismatched.WithLabelValues("0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F", "l2outputOracle").Set(1)
  }
  
  }

