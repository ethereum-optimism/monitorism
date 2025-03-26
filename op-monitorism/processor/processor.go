package processor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
)

// TxProcessingFunc is the type for transaction processing functions
type TxProcessingFunc func(block *types.Block, tx *types.Transaction, client *ethclient.Client) error

// TxProcessingFunc is the type for block processing functions
type BlockProcessingFunc func(block *types.Block, client *ethclient.Client) error

type Metrics struct {
	highestBlockSeen      prometheus.Gauge
	highestBlockProcessed prometheus.Gauge
	processingErrors      prometheus.Counter
}

// BlockProcessor handles the monitoring and processing of Ethereum blocks
type BlockProcessor struct {
	client           *ethclient.Client
	txProcessFunc    TxProcessingFunc
	blockProcessFunc BlockProcessingFunc
	interval         time.Duration
	lastProcessed    *big.Int
	log              log.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	metrics          Metrics
}

// Config holds the configuration for the processor
type Config struct {
	StartBlock *big.Int      // Optional: starting block number
	Interval   time.Duration // Optional: polling interval
}

// NewBlockProcessor creates a new processor instance
func NewBlockProcessor(
	m metrics.Factory,
	log log.Logger,
	rpcURL string,
	txProcessFunc TxProcessingFunc,
	blockProcessFunc BlockProcessingFunc,
	config *Config,
) (*BlockProcessor, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	// Set defaults if config is nil
	if config == nil {
		config = &Config{
			Interval: 12 * time.Second,
		}
	}

	// Set default interval if not specified
	if config.Interval == 0 {
		config.Interval = 12 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	metrics := Metrics{
		highestBlockSeen:      m.NewGauge(prometheus.GaugeOpts{Name: "highest_block_seen"}),
		highestBlockProcessed: m.NewGauge(prometheus.GaugeOpts{Name: "highest_block_processed"}),
		processingErrors:      m.NewCounter(prometheus.CounterOpts{Name: "processing_errors_total"}),
	}

	p := &BlockProcessor{
		client:           client,
		txProcessFunc:    txProcessFunc,
		blockProcessFunc: blockProcessFunc,
		interval:         config.Interval,
		ctx:              ctx,
		cancel:           cancel,
		metrics:          metrics,
		log:              log,
	}

	// If starting block is specified, use it; otherwise will start from latest block
	p.lastProcessed = config.StartBlock

	return p, nil
}

// Start begins the processing loop
func (p *BlockProcessor) Start() error {
	// If no starting block was specified, get the latest finalized block
	if p.lastProcessed.Cmp(big.NewInt(0)) == 0 {
		block, err := p.getLatestBlock()
		if err != nil {
			return err
		}
		p.lastProcessed = block.Number()
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case <-ticker.C:
			if err := p.processNewBlocks(); err != nil {
				// We don't want to stop the processor if we encounter an error, simply log it and keep
				// looping. Since the processor won't increment the lastProcessed block number, it will
				// keep trying to process the same block over and over again if necessary.
				p.log.Error("error processing blocks", "err", err)
			}
		}
	}
}

// Stop halts the processing loop
func (p *BlockProcessor) Stop() {
	p.cancel()
}

func (p *BlockProcessor) processNewBlocks() error {
	latestBlock, err := p.getLatestBlock()
	if err != nil {
		return err
	}

	// Update highest seen block metric.
	p.metrics.highestBlockSeen.Set(float64(latestBlock.Number().Int64()))

	// Process blocks one at a time, updating lastProcessed after each
	nextBlock := new(big.Int).Add(p.lastProcessed, common.Big1)
	for nextBlock.Cmp(latestBlock.Number()) <= 0 {
		p.log.Info("processing block", "block", nextBlock.String())

		block, err := p.client.BlockByNumber(p.ctx, nextBlock)
		if err != nil {
			return fmt.Errorf("failed to get block %s: %w", nextBlock.String(), err)
		}

		// Process each transaction in the block
		if p.txProcessFunc != nil {
			for _, tx := range block.Transactions() {
				if err := p.processTransactionWithRetry(block, tx); err != nil {
					return err
				}
			}
		}

		// Process the full block
		if p.blockProcessFunc != nil {
			if err := p.processBlockWithRetry(block); err != nil {
				return err
			}
		}

		// Update lastProcessed after each successful block
		p.lastProcessed = new(big.Int).Set(nextBlock)
		nextBlock.Add(nextBlock, common.Big1)

		// Update highest processed block metric.
		p.metrics.highestBlockProcessed.Set(float64(p.lastProcessed.Int64()))
	}

	return nil
}

func (p *BlockProcessor) processTransactionWithRetry(block *types.Block, tx *types.Transaction) error {
	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
			if err := p.txProcessFunc(block, tx, p.client); err == nil {
				return nil
			} else {
				p.log.Error("error processing transaction", "tx", tx.Hash().String(), "err", err)
				p.metrics.processingErrors.Inc()
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (p *BlockProcessor) processBlockWithRetry(block *types.Block) error {
	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
			if err := p.blockProcessFunc(block, p.client); err == nil {
				return nil
			} else {
				p.log.Error("error processing block", "block", block.Hash().String(), "err", err)
				p.metrics.processingErrors.Inc()
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (p *BlockProcessor) getLatestBlock() (*types.Block, error) {
	var header *types.Header
	err := p.client.Client().CallContext(p.ctx, &header, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return nil, fmt.Errorf("failed to get finalized header: %w", err)
	}

	block, err := p.client.BlockByNumber(p.ctx, header.Number)
	if err != nil {
		return nil, fmt.Errorf("failed to get finalized block %s: %w", header.Number.String(), err)
	}
	if block == nil {
		return nil, fmt.Errorf("finalized block not found")
	}

	return block, nil
}
