package withdrawalsv2

import (
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName             = "l1.node.url"
	StartBlockFlagName            = "start.block"
	PollingIntervalFlagName       = "poll.interval"
	OptimismPortalAddressFlagName = "optimism.portal.address"
	UseLatestFlagName             = "use.latest"
)

type CLIConfig struct {
	L1NodeURL             string
	OptimismPortalAddress string
	StartBlock            uint64
	PollingInterval       time.Duration
	UseLatest             bool
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:             ctx.String(L1NodeURLFlagName),
		OptimismPortalAddress: ctx.String(OptimismPortalAddressFlagName),
		StartBlock:            ctx.Uint64(StartBlockFlagName),
		PollingInterval:       ctx.Duration(PollingIntervalFlagName),
		UseLatest:             ctx.Bool(UseLatestFlagName),
	}

	return cfg, nil
}

func CLIFlags(envVar string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     L1NodeURLFlagName,
			Usage:    "Node URL of L1 archive+trace Geth node (must serve debug_traceTransaction)",
			EnvVars:  opservice.PrefixEnvVar(envVar, "L1_NODE_URL"),
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     StartBlockFlagName,
			Usage:    "Starting L1 block number to scan",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_BLOCK"),
			Required: true,
		},
		&cli.DurationFlag{
			Name:    PollingIntervalFlagName,
			Usage:   "Polling interval for scanning L1 blocks",
			EnvVars: opservice.PrefixEnvVar(envVar, "POLL_INTERVAL"),
			Value:   time.Second,
		},
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal2 contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
		&cli.BoolFlag{
			Name:    UseLatestFlagName,
			Usage:   "Scan from the 'latest' block instead of 'finalized' (not reorg-safe; useful for local/anvil nodes that lack a finalized block)",
			EnvVars: opservice.PrefixEnvVar(envVar, "USE_LATEST"),
			Value:   false,
		},
	}
}
