package withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals/bindings"
	opservice "github.com/ethereum-optimism/optimism/op-service"

	"log"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"
	L2NodeURLFlagName = "l2.node.url"

	EventBlockRangeFlagName       = "event.block.range"
	StartingL1BlockHeightFlagName = "start.block.height"

	OptimismPortalAddressFlagName = "optimismportal.address"
)

type CLIConfig struct {
	L1NodeURL string
	L2NodeURL string

	EventBlockRange       uint64
	StartingL1BlockHeight uint64

	OptimismPortalAddress common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:             ctx.String(L1NodeURLFlagName),
		L2NodeURL:             ctx.String(L2NodeURLFlagName),
		EventBlockRange:       ctx.Uint64(EventBlockRangeFlagName),
		StartingL1BlockHeight: ctx.Uint64(StartingL1BlockHeightFlagName),
	}

	// Parse the portal address first
	portalAddress := ctx.String(OptimismPortalAddressFlagName)
	if !common.IsHexAddress(portalAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", OptimismPortalAddressFlagName)
	}
	cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)

	if cfg.StartingL1BlockHeight == 0 {
		startingBlock, err := findContractDeploymentBlock(context.Background(), cfg.L1NodeURL, cfg.OptimismPortalAddress)
		if err != nil {
			log.Printf("WARNING: failed to find contract deployment block: %v", err)
			cfg.StartingL1BlockHeight = 0
		} else {
			cfg.StartingL1BlockHeight = startingBlock
		}
	}

	return cfg, nil
}

func CLIFlags(envVar string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    L1NodeURLFlagName,
			Usage:   "Node URL of L1 peer Geth node",
			EnvVars: opservice.PrefixEnvVar(envVar, "L1_NODE_URL"),
		},
		&cli.StringFlag{
			Name:    L2NodeURLFlagName,
			Usage:   "Node URL of L2 peer Op-Geth node",
			EnvVars: opservice.PrefixEnvVar(envVar, "L2_NODE_URL"),
		},
		&cli.Uint64Flag{
			Name:    EventBlockRangeFlagName,
			Usage:   "Max block range when scanning for events",
			Value:   1000,
			EnvVars: opservice.PrefixEnvVar(envVar, "EVENT_BLOCK_RANGE"),
		},
		&cli.Uint64Flag{
			Name:     StartingL1BlockHeightFlagName,
			Usage:    "Starting height to scan for events",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_BLOCK_HEIGHT"),
			Value:    0,
			Required: true,
		},
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
	}
}

func findContractDeploymentBlock(ctx context.Context, l1NodeURL string, contractAddress common.Address) (uint64, error) {
	client, err := ethclient.Dial(l1NodeURL)
	if err != nil {
		return 0, fmt.Errorf("failed to dial L1 node: %w", err)
	}
	defer client.Close()

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block number: %w", err)
	}

	start := uint64(0)
	end := latestBlock

	for start < end {
		block := (start + end) / 2

		callOpts := &bind.CallOpts{
			BlockNumber: big.NewInt(int64(block)),
		}

		contract, err := bindings.NewOptimismPortalCaller(contractAddress, client)
		if err != nil {
			return 0, fmt.Errorf("failed to create contract caller: %w", err)
		}

		_, err = contract.Version(callOpts)
		if err != nil {
			start = block + 1
		} else {
			end = block
		}
	}

	return end, nil
}
