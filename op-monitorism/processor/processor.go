package processor

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ProcessingFunc is the type for transaction processing functions
type ProcessingFunc func(block *types.Block, tx *types.Transaction, client *ethclient.Client) error

// BlockProcessor handles the monitoring and processing of Ethereum blocks
type BlockProcessor struct {
	client        *ethclient.Client
	processFunc   ProcessingFunc
	interval      time.Duration
	lastProcessed *big.Int
	ctx           context.Context
	cancel        context.CancelFunc
}

// Config holds the configuration for the processor
type Config struct {
	StartBlock *big.Int      // Optional: starting block number
	Interval   time.Duration // Optional: polling interval
}

// NewBlockProcessor creates a new processor instance
func NewBlockProcessor(rpcURL string, processFunc ProcessingFunc, config *Config) (*BlockProcessor, error) {
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

	p := &BlockProcessor{
		client:      client,
		processFunc: processFunc,
		interval:    config.Interval,
		ctx:         ctx,
		cancel:      cancel,
	}

	// If starting block is specified, use it; otherwise will start from latest block
	p.lastProcessed = config.StartBlock

	return p, nil
}

// Start begins the processing loop
func (p *BlockProcessor) Start() error {
	// If no starting block was specified, get the latest finalized block
	if p.lastProcessed == nil {
		block, err := p.getLatestFinalizedBlock()
		if err != nil {
			return err
		}
        if block == nil {
            p.lastProcessed = big.NewInt(0)
        } else {
		    p.lastProcessed = block.Number()
        }
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.processNewBlocks(); err != nil {
				// We don't want to stop the processor if we encounter an error, simply log it and keep
				// looping. Since the processor won't increment the lastProcessed block number, it will
				// keep trying to process the same block over and over again if necessary.
				fmt.Printf("error processing blocks: %v\n", err)
			}
		}
	}
}

// Stop halts the processing loop
func (p *BlockProcessor) Stop() {
	p.cancel()
}

func (p *BlockProcessor) processNewBlocks() error {
	latestBlock, err := p.getLatestFinalizedBlock()
	
	if err != nil {
		return err
	}

	// Process blocks one at a time, updating lastProcessed after each
	nextBlock := new(big.Int).Add(p.lastProcessed, common.Big1)
	for nextBlock.Cmp(latestBlock.Number()) <= 0 {
		block, err := p.client.BlockByNumber(p.ctx, nextBlock)
		if err != nil {
			return fmt.Errorf("failed to get block %s: %w", nextBlock.String(), err)
		}

		// Process each transaction in the block
		for _, tx := range block.Transactions() {
			if err := p.processFunc(block, tx, p.client); err != nil {
				return fmt.Errorf("failed to process transaction %s: %w", tx.Hash().String(), err)
			}
		}

		// Update lastProcessed after each successful block
		p.lastProcessed = nextBlock
		nextBlock.Add(nextBlock, common.Big1)
	}

	return nil
}

func (p *BlockProcessor) getLatestFinalizedBlock() (*types.Block, error) {
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
