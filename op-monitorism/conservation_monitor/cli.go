package conservation_monitor

import (
	"time"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName         = "node.url"
	StartBlockFlagName      = "start.block"
	PollingIntervalFlagName = "poll.interval"
)

type CLIConfig struct {
	NodeUrl         string        `yaml:"node_url"`
	StartBlock      uint64        `yaml:"start_block"`
	PollingInterval time.Duration `yaml:"poll_interval"`
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		NodeUrl:         ctx.String(NodeURLFlagName),
		StartBlock:      ctx.Uint64(StartBlockFlagName),
		PollingInterval: ctx.Duration(PollingIntervalFlagName),
	}
	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL",
			Value:   "http://localhost:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.Uint64Flag{
			Name:    StartBlockFlagName,
			Usage:   "Starting block number (0 for latest)",
			Value:   0,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "START_BLOCK"),
		},
		&cli.DurationFlag{
			Name:    PollingIntervalFlagName,
			Usage:   "The polling interval (should be less than blocktime for safety) in seconds",
			Value:   12 * time.Second,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "POLL_INTERVAL"),
		},
	}
}
