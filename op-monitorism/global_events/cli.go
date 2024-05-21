package global_events

import (
	// "fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	// "github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

// args in CLI have to be standardized and clean.
const (
	L1NodeURLFlagName     = "l1.node.url"
	NicknameFlagName      = "nickname"
	PathYamlRulesFlagName = "PathYamlRules"
)

type CLIConfig struct {
	L1NodeURL     string
	Nickname      string
	PathYamlRules string
	// Optional
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:     ctx.String(L1NodeURLFlagName),
		Nickname:      ctx.String(NicknameFlagName),
		PathYamlRules: ctx.String(PathYamlRulesFlagName),
	}

	return cfg, nil
}

func CLIFlags(envVar string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    L1NodeURLFlagName,
			Usage:   "Node URL of L1 peer",
			Value:   "http://127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envVar, "L1_NODE_URL"),
		},
		&cli.StringFlag{
			Name:     NicknameFlagName,
			Usage:    "Nickname of chain being monitored",
			EnvVars:  opservice.PrefixEnvVar(envVar, "NICKNAME"), //need to change the name to BLOCKCHAIN_NAME
			Required: true,
		},
		&cli.StringFlag{
			Name:     PathYamlRulesFlagName,
			Usage:    "Path to the yaml file containing the events to monitor",
			EnvVars:  opservice.PrefixEnvVar(envVar, "PATH_YAML"), //need to change the name to BLOCKCHAIN_NAME
			Required: true,
		},
	}
}
