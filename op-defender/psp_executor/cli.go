package psp_executor

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName                 = "rpc.url"
	PrivateKeyFlagName              = "privatekey"
	PortAPIFlagName                 = "port.api"
	ReceiverAddressFlagName         = "receiver.address"
	DataFlagName                    = "data"
	SuperChainConfigAddressFlagName = "superchainconfig.address"
	SafeAddressFlagName             = "safe.address"
	PathFlagName                    = "path"
)

type CLIConfig struct {
	NodeURL                 string
	privatekeyflag          string
	PortAPI                 string
	ReceiverAddress         string
	HexString               string //@todo: remove this is not necessary
	SuperChainConfigAddress string
	SafeAddress             common.Address
	Path                    string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{NodeURL: ctx.String(NodeURLFlagName)}
	if len(PrivateKeyFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PrivateKeyFlagName set to execute the pause.")
	}
	cfg.privatekeyflag = ctx.String(PrivateKeyFlagName)

	if len(PortAPIFlagName) == 0 {
		return cfg, fmt.Errorf("must have a PortAPIFlagName set to execute the pause.")
	}
	cfg.PortAPI = ctx.String(PortAPIFlagName)

	if len(ReceiverAddressFlagName) == 0 {
		return cfg, fmt.Errorf("must have a ReceiverAddressFlagName set to receive the pause.")
	}
	cfg.ReceiverAddress = ctx.String(ReceiverAddressFlagName)

	if len(DataFlagName) == 0 {
		return cfg, fmt.Errorf("must have a `data` set to execute the calldata.")
	}
	cfg.HexString = ctx.String(DataFlagName)

	if len(SuperChainConfigAddressFlagName) == 0 {
		return cfg, fmt.Errorf("must have a `SuperChainConfigAddress` to know the current status of the superchainconfig.")
	}
	cfg.SuperChainConfigAddress = ctx.String(SuperChainConfigAddressFlagName)

	if len(PathFlagName) == 0 {
		return cfg, fmt.Errorf("must have a `PathFlagName` to know where the PSPs are stored.")
	}
	cfg.Path = ctx.String(PathFlagName)

	if len(SafeAddressFlagName) == 0 {
		return cfg, fmt.Errorf("must have a `SafeAddress` to know the current status of the safe.")
	}
	cfg.SafeAddress = common.HexToAddress(ctx.String(SafeAddressFlagName))
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
		&cli.StringFlag{
			Name:     SuperChainConfigAddressFlagName,
			Usage:    "SuperChainConfig address to know the current status of the superchainconfig.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "SUPERCHAINCONFIG_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     SafeAddressFlagName,
			Usage:    "Safe address that will execute the PSPs.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "SAFE_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     PathFlagName,
			Usage:    "Path to the PSPs filename.",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PATH_TO_PSPS"),
			Required: true,
		},
	}
}
