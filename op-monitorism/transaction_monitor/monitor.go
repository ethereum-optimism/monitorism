package transaction_monitor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "tx_mon"
)

type CheckType string

const (
	ExactMatchCheck  CheckType = "exact_match"
	DisputeGameCheck CheckType = "dispute_game"
)

type CheckConfig struct {
	Type   CheckType              `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}

type WatchConfig struct {
	Address    common.Address      `yaml:"address"`
	Filters    []CheckConfig       `yaml:"filters"`
}

type DisputeGameVerifier struct {
	factory    common.Address
	factoryABI abi.ABI
	gameABI    abi.ABI
	cache      map[common.Address]bool
	mu         sync.RWMutex
}

type Monitor struct {
	log           log.Logger
	client        *ethclient.Client
	watchConfigs  map[common.Address]WatchConfig
	allowedAddrs  map[common.Address]bool
	gameVerifiers map[common.Address]*DisputeGameVerifier
	startBlock    uint64
	mu            sync.RWMutex

	// metrics
	transactions        *prometheus.CounterVec
	unauthorizedTx      *prometheus.CounterVec
	ethSpent           *prometheus.CounterVec
    blocksProcessed    *prometheus.CounterVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// DisputeGameFactoryABI contains just the games() function we need
const DisputeGameFactoryABI = `[{
    "inputs": [
        {"internalType": "uint32", "name": "_gameType", "type": "uint32"},
        {"internalType": "bytes32", "name": "_rootClaim", "type": "bytes32"},
        {"internalType": "bytes", "name": "_extraData", "type": "bytes"}
    ],
    "name": "games",
    "outputs": [
        {"internalType": "address", "name": "proxy_", "type": "address"},
        {"internalType": "uint64", "name": "timestamp_", "type": "uint64"}
    ],
    "stateMutability": "view",
    "type": "function"
}]`

// DisputeGameABI contains just the functions we need from the game contract
const DisputeGameABI = `[{
    "inputs": [],
    "name": "gameType",
    "outputs": [{"internalType": "uint32", "name": "", "type": "uint32"}],
    "stateMutability": "view",
    "type": "function"
}, {
    "inputs": [],
    "name": "rootClaim",
    "outputs": [{"internalType": "bytes32", "name": "", "type": "bytes32"}],
    "stateMutability": "pure",
    "type": "function"
}, {
    "inputs": [],
    "name": "extraData",
    "outputs": [{"internalType": "bytes", "name": "", "type": "bytes"}],
    "stateMutability": "pure",
    "type": "function"
}]`

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	client, err := ethclient.Dial(cfg.NodeUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial node: %w", err)
	}

	factoryABI, err := abi.JSON(strings.NewReader(DisputeGameFactoryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse factory ABI: %w", err)
	}

	gameABI, err := abi.JSON(strings.NewReader(DisputeGameABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse game ABI: %w", err)
	}

	mon := &Monitor{
		log:           log,
		client:        client,
		watchConfigs:  make(map[common.Address]WatchConfig),
		allowedAddrs:  make(map[common.Address]bool),
		gameVerifiers: make(map[common.Address]*DisputeGameVerifier),
		startBlock:    cfg.StartBlock,

		transactions: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "transactions_total",
				Help:      "Total number of transactions",
			},
			[]string{"from"},
		),

		unauthorizedTx: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "unauthorized_transactions_total",
				Help:      "Number of transactions from unauthorized addresses",
			},
			[]string{"from"},
		),

		ethSpent: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "eth_spent_total",
				Help:      "Cumulative ETH spent by address",
			},
			[]string{"address"},
		),

        blocksProcessed: m.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: MetricsNamespace,
                Name:      "blocks_processed_total",
                Help:      "Number of blocks processed",
            },
            []string{"number"},
        ),

		unexpectedRpcErrors: m.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: MetricsNamespace,
				Name:      "unexpected_rpc_errors_total",
				Help:      "Number of unexpected RPC errors",
			},
			[]string{"section", "name"},
		),
	}

	// Initialize filters and verifiers
	for _, config := range cfg.WatchConfigs {
		mon.watchConfigs[config.Address] = config

		for _, filter := range config.Filters {
			switch filter.Type {
			case ExactMatchCheck:
				if match, ok := filter.Params["match"].(string); ok {
					mon.allowedAddrs[common.HexToAddress(match)] = true
				}
			case DisputeGameCheck:
				if factory, ok := filter.Params["disputeGameFactory"].(string); ok {
					factoryAddr := common.HexToAddress(factory)
					verifier := &DisputeGameVerifier{
						factory:    factoryAddr,
						factoryABI: factoryABI,
						gameABI:    gameABI,
						cache:      make(map[common.Address]bool),
					}
					mon.gameVerifiers[factoryAddr] = verifier
				}
			}
		}
	}

	return mon, nil
}

func (v *DisputeGameVerifier) verifyGame(ctx context.Context, client *ethclient.Client, gameAddr common.Address) (bool, error) {
	v.mu.RLock()
	if cached, exists := v.cache[gameAddr]; exists {
		v.mu.RUnlock()
		return cached, nil
	}
	v.mu.RUnlock()

	// Create contract bindings
	game := bind.NewBoundContract(gameAddr, v.gameABI, client, client, client)
	factory := bind.NewBoundContract(v.factory, v.factoryABI, client, client, client)

	// Get game parameters
	var gameTypeResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &gameTypeResult, "gameType"); err != nil {
		return false, fmt.Errorf("failed to get game type: %w", err)
	}
	if len(gameTypeResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for gameType")
	}
	gameType, ok := gameTypeResult[0].(uint32)
	if !ok {
		return false, fmt.Errorf("invalid game type returned")
	}

	var rootClaimResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &rootClaimResult, "rootClaim"); err != nil {
		return false, fmt.Errorf("failed to get root claim: %w", err)
	}
	if len(rootClaimResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for rootClaim")
	}
	rootClaim, ok := rootClaimResult[0].([32]byte)
	if !ok {
		return false, fmt.Errorf("invalid root claim returned")
	}

	var extraDataResult []interface{}
	if err := game.Call(&bind.CallOpts{Context: ctx}, &extraDataResult, "extraData"); err != nil {
		return false, fmt.Errorf("failed to get extra data: %w", err)
	}
	if len(extraDataResult) != 1 {
		return false, fmt.Errorf("unexpected number of return values for extraData")
	}
	extraData, ok := extraDataResult[0].([]byte)
	if !ok {
		return false, fmt.Errorf("invalid extra data returned")
	}

	// Verify with factory
	var factoryResult []interface{}
	if err := factory.Call(&bind.CallOpts{Context: ctx}, &factoryResult, "games", gameType, rootClaim, extraData); err != nil {
		return false, fmt.Errorf("failed to verify game with factory: %w", err)
	}
	if len(factoryResult) != 2 {
		return false, fmt.Errorf("unexpected number of return values from factory")
	}

	proxy, ok := factoryResult[0].(common.Address)
	if !ok {
		return false, fmt.Errorf("invalid proxy address returned")
	}

	timestamp, ok := factoryResult[1].(uint64)
	if !ok {
		return false, fmt.Errorf("invalid timestamp returned")
	}

	isValid := proxy == gameAddr && timestamp > 0

	// Cache the result
	v.mu.Lock()
	v.cache[gameAddr] = isValid
	v.mu.Unlock()

	return isValid, nil
}

func (m *Monitor) Run(ctx context.Context) {
	// Initialize starting block
	currentBlock := m.startBlock
	if currentBlock == 0 {
		var err error
		currentBlock, err = m.client.BlockNumber(ctx)
		if err != nil {
			m.log.Error("failed to get initial block number", "err", err)
			return
		}
		m.log.Info("starting from latest block", "number", currentBlock)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info("monitor stopping")
			return
		case <-ticker.C:
			// Get latest block number
			latestBlock, err := m.client.BlockNumber(ctx)
			if err != nil {
				m.log.Error("failed to get latest block number", "err", err)
				continue
			}

			// Process blocks until we catch up to head
			for currentBlock <= latestBlock {
				if err := m.processBlock(ctx, currentBlock); err != nil {
					m.log.Error("failed to process block", "number", currentBlock, "err", err)
					if !errors.Is(err, ethereum.NotFound) {
						m.unexpectedRpcErrors.WithLabelValues("monitor", "getBlock").Inc()
					}
					break
				}
				currentBlock++
			}
		}
	}
}

func (m *Monitor) processBlock(ctx context.Context, blockNum uint64) error {
	block, err := m.client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

    m.blocksProcessed.WithLabelValues(fmt.Sprint(blockNum)).Inc()

	for _, tx := range block.Transactions() {
        from, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	    if err != nil {
		    return fmt.Errorf("failed to find tx sender: %w", err)
    	}

		if config, exists := m.watchConfigs[from]; exists {
			m.processTx(ctx, tx, config)
		}
	}

	return nil
}

func (m *Monitor) isAddressAllowed(ctx context.Context, addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.allowedAddrs[addr] {
		return true
	}

	// Check if address is an allowed dispute game
	for _, verifier := range m.gameVerifiers {
		isValid, err := verifier.verifyGame(ctx, m.client, addr)
		if err != nil {
			m.log.Error("failed to verify dispute game", "addr", addr, "err", err)
			continue
		}
		if isValid {
			return true
		}
	}

	return false
}

func (m *Monitor) processTx(ctx context.Context, tx *types.Transaction, config WatchConfig) {
	watchAddr, err := types.Sender(types.NewLondonSigner(tx.ChainId()), tx)
	if err != nil {
		m.log.Error("failed to get transaction sender", "err", err)
		return
	}

	toAddr := *tx.To()

	// Track cumulative ETH spent
	weiValue := new(big.Float).SetInt(tx.Value())
	ethValue := new(big.Float).Quo(weiValue, big.NewFloat(1e18))
	ethFloat, _ := ethValue.Float64()
	m.ethSpent.WithLabelValues(watchAddr.String()).Add(ethFloat)

	// Check if sender is authorized
	if !m.isAddressAllowed(ctx, toAddr) {
		m.unauthorizedTx.WithLabelValues(
			watchAddr.String()).Inc()
		return
	}

	m.transactions.WithLabelValues(
		watchAddr.String(),
		tx.To().String(),
		"processed").Inc()
}

func (m *Monitor) Close(ctx context.Context) error {
	m.client.Close()
	return nil
}
