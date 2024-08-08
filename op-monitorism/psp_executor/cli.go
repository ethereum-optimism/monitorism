package psp_executor

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName    = "node.url"
	PrivateKeyFlagName = "privatekey"
	PortAPIFlagName    = "port.api"
)

type CLIConfig struct {
	NodeUrl        string
	privatekeyflag string
	portapi        string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{NodeUrl: ctx.String(NodeURLFlagName)}
	if len(PrivateKeyFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PrivateKeyFlagName set to execute the pause on mainnet")
	}
	cfg.privatekeyflag = ctx.String(PrivateKeyFlagName)
	if len(PortAPIFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PortAPIFlagName set to execute the pause on mainnet")
	}
	cfg.portapi = ctx.String(PortAPIFlagName)
	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL of a peer",
			Value:   "127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.StringFlag{
			Name:     PrivateKeyFlagName,
			Usage:    "Private key of the account that will issue the pause ()",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PRIVATE_KEY"),
			Required: true,
		},

		&cli.StringFlag{
			Name:     PortAPIFlagName,
			Value:    "8080",
			Usage:    "Port of the API server you want to listen on (e.g. 8080).",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PORT_API"),
			Required: false,
		},
	}
}
