package withdrawals

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"

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
			Name:    L1NodeURLFlagName,
			Usage:   "Node URL of L1 peer",
			Value:   "127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envVar, "L1_NODE_URL"),
		},
		&cli.StringFlag{
			Name:    L2NodeURLFlagName,
			Usage:   "Node URL of L2 peer",
			Value:   "127.0.0.1:9545",
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
