package transaction_monitor

import (
	"fmt"
	"os"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

const (
	NodeURLFlagName  = "node.url"
	ConfigFileFlagName = "config.file"
	StartBlockFlagName = "start.block"
)

type CLIConfig struct {
	NodeUrl    string        `yaml:"node_url"`
	StartBlock   uint64        `yaml:"start_block"`
	WatchConfigs []WatchConfig `yaml:"watch_configs"`
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		NodeUrl:  ctx.String(NodeURLFlagName),
		StartBlock: ctx.Uint64(StartBlockFlagName),
	}

	// Read and parse config file
	configFile := ctx.String(ConfigFileFlagName)
	if configFile == "" {
		return cfg, fmt.Errorf("config file must be specified")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.WatchConfigs) == 0 {
		return cfg, fmt.Errorf("at least one watch config must be specified")
	}

	// Validate filters
	for _, config := range cfg.WatchConfigs {
		for _, filter := range config.Filters {
			switch filter.Type {
			case ExactMatchCheck:
				if _, ok := filter.Params["match"].(string); !ok {
					return cfg, fmt.Errorf("exact match check requires 'match' parameter")
				}
			case DisputeGameCheck:
				if _, ok := filter.Params["disputeGameFactory"].(string); !ok {
					return cfg, fmt.Errorf("dispute game check requires 'disputeGameFactory' parameter")
				}
			default:
				return cfg, fmt.Errorf("unknown check type: %s", filter.Type)
			}
		}
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
		&cli.StringFlag{
			Name:     ConfigFileFlagName,
			Usage:    "Path to YAML config file containing watch addresses and filters",
			Required: true,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "TX_CONFIG_FILE"),
		},
		&cli.Uint64Flag{
			Name:    StartBlockFlagName,
			Usage:   "Starting block number (0 for latest)",
			Value:   0,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "START_BLOCK"),
		},
	}
}
