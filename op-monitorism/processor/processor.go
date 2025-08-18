package processor

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
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

// LogProcessingFunc is the type for log processing functions (invoked per log)
type LogProcessingFunc func(block *types.Block, lg types.Log, client *ethclient.Client) error

type Metrics struct {
	highestBlockSeen      prometheus.Gauge
	highestBlockProcessed prometheus.Gauge
	processingErrors      prometheus.Counter
	currentBackoffDelay   prometheus.Gauge
	backoffIncreases      prometheus.Counter
	backoffDecreases      prometheus.Counter
}

// BlockProcessor handles the monitoring and processing of Ethereum blocks
type BlockProcessor struct {
	client           *ethclient.Client
	txProcessFunc    TxProcessingFunc
	blockProcessFunc BlockProcessingFunc
	logProcessFunc   LogProcessingFunc
	interval         time.Duration
	lastProcessed    *big.Int
	log              log.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	metrics          Metrics
	useLatest        bool

	// dynamic backoff state
	currentDelay      time.Duration
	stableCleanBlocks int
	errorsThisBlock   int

	// dynamic backoff config
	minDelay       time.Duration
	maxDelay       time.Duration
	decreaseStep   time.Duration
	stableWindow   int
	jitterFraction float64
	jitterRng      *rand.Rand
}

// Config holds the configuration for the processor
type Config struct {
	StartBlock *big.Int      // Optional: starting block number
	Interval   time.Duration // Optional: polling interval
	UseLatest  bool          // Optional: use latest block instead of finalized block, not reorg safe

	// Dynamic backoff configuration (optional)
	MinDelay       time.Duration // Minimum per-block delay
	MaxDelay       time.Duration // Maximum per-block delay
	DecreaseStep   time.Duration // How much to decrease after stability
	StableWindow   int           // Clean blocks before decreasing delay
	JitterFraction float64       // +/- fraction jitter to avoid lockstep (e.g., 0.1 = +/-10%)
}

// NewBlockProcessor creates a new processor instance
func NewBlockProcessor(
	m metrics.Factory,
	log log.Logger,
	rpcURL string,
	txProcessFunc TxProcessingFunc,
	blockProcessFunc BlockProcessingFunc,
	logProcessFunc LogProcessingFunc,
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

	// Local RNG for jitter
	seed := time.Now().UnixNano()
	_ = seed

	ctx, cancel := context.WithCancel(context.Background())

	metrics := Metrics{
		highestBlockSeen:      m.NewGauge(prometheus.GaugeOpts{Name: "highest_block_seen"}),
		highestBlockProcessed: m.NewGauge(prometheus.GaugeOpts{Name: "highest_block_processed"}),
		processingErrors:      m.NewCounter(prometheus.CounterOpts{Name: "processing_errors_total"}),
		currentBackoffDelay:   m.NewGauge(prometheus.GaugeOpts{Name: "current_backoff_delay_seconds"}),
		backoffIncreases:      m.NewCounter(prometheus.CounterOpts{Name: "backoff_increases_total"}),
		backoffDecreases:      m.NewCounter(prometheus.CounterOpts{Name: "backoff_decreases_total"}),
	}

	p := &BlockProcessor{
		client:           client,
		txProcessFunc:    txProcessFunc,
		blockProcessFunc: blockProcessFunc,
		logProcessFunc:   logProcessFunc,
		interval:         config.Interval,
		ctx:              ctx,
		cancel:           cancel,
		metrics:          metrics,
		log:              log,
		useLatest:        config.UseLatest,
	}

	// Initialize RNG for jitter
	p.jitterRng = rand.New(rand.NewSource(time.Now().UnixNano()))

	// Resolve dynamic backoff config
	if config.MaxDelay == 0 {
		config.MaxDelay = 5 * time.Second
	}
	if config.DecreaseStep == 0 {
		config.DecreaseStep = 100 * time.Millisecond
	}
	if config.StableWindow == 0 {
		config.StableWindow = 5
	}
	if config.JitterFraction == 0 {
		config.JitterFraction = 0.1
	}

	// Resolve dynamic backoff config.
	p.minDelay = config.MinDelay
	p.maxDelay = config.MaxDelay
	p.decreaseStep = config.DecreaseStep
	p.stableWindow = config.StableWindow
	p.jitterFraction = config.JitterFraction
	p.currentDelay = p.minDelay

	// If starting block is specified, use it; otherwise will start from latest block
	p.lastProcessed = config.StartBlock

	return p, nil
}

// Start begins the processing loop
func (p *BlockProcessor) Start() error {
	// If no starting block was specified, get the latest finalized block
	if p.lastProcessed == nil || p.lastProcessed.Cmp(big.NewInt(0)) == 0 {
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

				// Update backoff even if we had an error.
				p.updateBackoff()
			}
		}
	}
}

// Stop halts the processing loop
func (p *BlockProcessor) Stop() {
	p.cancel()
}

func (p *BlockProcessor) processNewBlocks() error {
	// Reset the current error count if nonzero.
	p.errorsThisBlock = 0

	// Grab the latest block.
	latestBlock, err := p.getLatestBlock()
	if err != nil {
		p.errorsThisBlock++
		return err
	}

	// Update highest seen block metric.
	p.metrics.highestBlockSeen.Set(float64(latestBlock.Number().Int64()))

	// Process blocks one at a time, updating lastProcessed after each
	nextBlock := new(big.Int).Add(p.lastProcessed, common.Big1)
	for nextBlock.Cmp(latestBlock.Number()) <= 0 {
		// Process the block.
		if err := p.processBlock(nextBlock); err != nil {
			p.errorsThisBlock++
			return err
		}

		// Update backoff after a successful block.
		p.updateBackoff()

		// Move on to the next block.
		nextBlock.Add(nextBlock, common.Big1)
	}

	return nil
}

// processBlock processes a single block and handles all errors
func (p *BlockProcessor) processBlock(blockNumber *big.Int) error {
	p.log.Info("processing block", "block", blockNumber.String())

	// Get the block with retry
	block, err := p.getBlockWithRetry(blockNumber)
	if err != nil {
		return err // Context cancellation or unrecoverable error
	}

	// Process each transaction in the block
	if p.txProcessFunc != nil {
		for _, tx := range block.Transactions() {
			if err := p.processTransactionWithRetry(block, tx); err != nil {
				return err // Context cancellation
			}
		}
	}

	// Process the full block
	if p.blockProcessFunc != nil {
		if err := p.processBlockWithRetry(block); err != nil {
			return err // Context cancellation
		}
	}

	// Process logs for this block via receipts
	if p.logProcessFunc != nil {
		receipts, err := p.getBlockReceiptsWithRetry(block)
		if err != nil {
			return err // Context cancellation or unrecoverable error
		}
		for _, rcpt := range receipts {
			for _, lg := range rcpt.Logs {
				if err := p.processLogWithRetry(block, *lg); err != nil {
					return err // Context cancellation
				}
			}
		}
	}

	// Update lastProcessed after successful block
	p.lastProcessed = new(big.Int).Set(blockNumber)
	p.metrics.highestBlockProcessed.Set(float64(p.lastProcessed.Int64()))

	return nil
}

// updateBackoff adjusts the processing delay based on retry count
func (p *BlockProcessor) updateBackoff() {
	// Figure out if we had errors, reset error counter.
	hadErrors := p.errorsThisBlock > 0
	p.errorsThisBlock = 0

	// If we had errors, increase the delay multiplicatively.
	if hadErrors {
		// If we had no delay, set it to the decrease step, otherwise double it.
		if p.currentDelay == 0 {
			p.currentDelay = p.decreaseStep
		} else {
			p.currentDelay *= 2
		}

		// Clamp the delay to the max delay.
		if p.currentDelay > p.maxDelay {
			p.currentDelay = p.maxDelay
		}

		// Reset the stable clean blocks counter.
		p.stableCleanBlocks = 0
		p.metrics.backoffIncreases.Inc()
	} else {
		// If we had no errors, increment the stable clean blocks counter.
		p.stableCleanBlocks++

		// If we've had no errors for the stable window, decrease the delay.
		if p.stableCleanBlocks >= p.stableWindow && p.currentDelay > p.minDelay {
			// Decrease the delay by the decrease step, but don't go below the min delay.
			if p.currentDelay > p.decreaseStep+p.minDelay {
				p.currentDelay -= p.decreaseStep
			} else {
				p.currentDelay = p.minDelay
			}

			// Reset the stable clean blocks counter.
			p.stableCleanBlocks = 0
			p.metrics.backoffDecreases.Inc()
		}
	}

	// Update the current backoff delay metric.
	p.metrics.currentBackoffDelay.Set(p.currentDelay.Seconds())
	log.Info("current delay", "delay", p.currentDelay.Seconds(), "errors", p.errorsThisBlock, "stable", p.stableCleanBlocks)

	// Sleep with jitter to control processing rate.
	if p.currentDelay > 0 {
		jitter := p.jitterFraction
		if jitter > 0 {
			delta := (p.jitterRng.Float64()*2 - 1) * jitter
			sleepDur := max(time.Duration(float64(p.currentDelay)*(1+delta)), 0)
			time.Sleep(sleepDur)
		} else {
			time.Sleep(p.currentDelay)
		}
	}
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
				p.errorsThisBlock++
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
				p.errorsThisBlock++
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (p *BlockProcessor) processLogWithRetry(block *types.Block, lg types.Log) error {
	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
			if err := p.logProcessFunc(block, lg, p.client); err == nil {
				return nil
			} else {
				p.log.Error("error processing log", "block", block.Hash().String(), "logIndex", lg.Index, "txHash", lg.TxHash.Hex(), "err", err)
				p.metrics.processingErrors.Inc()
				p.errorsThisBlock++
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// getBlockWithRetry gets a block by number with retry logic
func (p *BlockProcessor) getBlockWithRetry(blockNumber *big.Int) (*types.Block, error) {
	for {
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		default:
			block, err := p.client.BlockByNumber(p.ctx, blockNumber)
			if err == nil {
				return block, nil
			} else {
				p.log.Error("error getting block", "block", blockNumber.String(), "err", err)
				p.metrics.processingErrors.Inc()
				p.errorsThisBlock++
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// getBlockReceiptsWithRetry gets block receipts with retry logic
func (p *BlockProcessor) getBlockReceiptsWithRetry(block *types.Block) ([]*types.Receipt, error) {
	for {
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		default:
			var receipts []*types.Receipt
			err := p.client.Client().CallContext(p.ctx, &receipts, "eth_getBlockReceipts", block.Hash())
			if err == nil {
				return receipts, nil
			} else {
				p.log.Error("error getting block receipts", "block", block.Hash().String(), "err", err)
				p.metrics.processingErrors.Inc()
				p.errorsThisBlock++
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (p *BlockProcessor) getLatestBlock() (*types.Block, error) {
	var tag string
	if p.useLatest {
		tag = "latest"
	} else {
		tag = "finalized"
	}

	var header *types.Header
	err := p.client.Client().CallContext(p.ctx, &header, "eth_getBlockByNumber", tag, false)
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
