package faultproof_withdrawals

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	L1GethURLFlagName = "l1.geth.url"
	L2NodeURLFlagName = "l2.node.url"
	L2GethURLFlagName = "l2.geth.url"

	EventBlockRangeFlagName       = "event.block.range"
	StartingL1BlockHeightFlagName = "start.block.height"

	OptimismPortalAddressFlagName = "optimismportal.address"
)

type CLIConfig struct {
	L1GethURL   string
	L2OpGethURL string
	L2OpNodeURL string

	EventBlockRange       uint64
	StartingL1BlockHeight uint64

	OptimismPortalAddress common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1GethURL:             ctx.String(L1GethURLFlagName),
		L2OpGethURL:           ctx.String(L2GethURLFlagName),
		L2OpNodeURL:           ctx.String(L2NodeURLFlagName),
		EventBlockRange:       ctx.Uint64(EventBlockRangeFlagName),
		StartingL1BlockHeight: ctx.Uint64(StartingL1BlockHeightFlagName),
	}

	portalAddress := ctx.String(OptimismPortalAddressFlagName)
	if !common.IsHexAddress(portalAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", OptimismPortalAddressFlagName)
	}
	cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)

	return cfg, nil
}

func CLIFlags(envVar string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    L1GethURLFlagName,
			Usage:   "L1 execution layer node URL",
			EnvVars: opservice.PrefixEnvVar(envVar, "L1_GETH_URL"),
		},
		&cli.StringFlag{
			Name:    L2NodeURLFlagName,
			Usage:   "L2 rollup node consensus layer (op-node) URL",
			EnvVars: opservice.PrefixEnvVar(envVar, "L2_OP_NODE_URL"),
		},
		&cli.StringFlag{
			Name:    L2GethURLFlagName,
			Usage:   "L2 OP Stack execution layer client(op-geth) URL",
			EnvVars: opservice.PrefixEnvVar(envVar, "L2_OP_GETH_URL"),
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
			Required: false,
		},
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
	}
}
