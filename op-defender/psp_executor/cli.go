package psp_executor

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName         = "rpc.url"
	PrivateKeyFlagName      = "privatekey"
	PortAPIFlagName         = "port.api"
	ReceiverAddressFlagName = "receiver.address"
	DataFlagName            = "data"
)

type CLIConfig struct {
	NodeURL         string
	privatekeyflag  string
	PortAPI         string
	ReceiverAddress string
	HexString       string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{NodeURL: ctx.String(NodeURLFlagName)}
	if len(PrivateKeyFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PrivateKeyFlagName set to execute the pause on mainnet")
	}
	cfg.privatekeyflag = ctx.String(PrivateKeyFlagName)
	if len(PortAPIFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PortAPIFlagName set to execute the pause on mainnet")
	}
	cfg.PortAPI = ctx.String(PortAPIFlagName)
	if len(ReceiverAddressFlagName) == 0 {
		return cfg, fmt.Errorf("must have a ReceiverAddressFlagName set to receive the pause on mainnet.")
	}
	cfg.ReceiverAddress = ctx.String(ReceiverAddressFlagName)
	if len(DataFlagName) == 0 {
		return cfg, fmt.Errorf("must have a `data` set to execute the calldata on mainnet.")
	}
	cfg.HexString = ctx.String(DataFlagName)
	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL of a peer",
			Value:   "http://127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.StringFlag{
			Name:     PrivateKeyFlagName,
			Usage:    "Private key of the account that will issue the pause ()",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PRIVATE_KEY"),
			Required: true,
		},

		&cli.StringFlag{
			Name:     ReceiverAddressFlagName,
			Usage:    "The receiver address of the pause request.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "RECEIVER_ADDRESS"),
			Required: true,
		},

		&cli.StringFlag{
			Name:     PortAPIFlagName,
			Value:    "8080",
			Usage:    "Port of the API server you want to listen on (e.g. 8080).",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PORT_API"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     DataFlagName,
			Value:    "",
			Usage:    "calldata to execute the pause on mainnet with the signatures.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "CALLDATA"),
			Required: false,
		},
	}
}
