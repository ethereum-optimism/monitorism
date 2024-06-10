package drippie

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName      = "l1.node.url"
	DrippieAddressFlagName = "drippie.address"
)

type CLIConfig struct {
	L1NodeURL      string
	DrippieAddress common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL: ctx.String(L1NodeURLFlagName),
	}

	drippieAddress := ctx.String(DrippieAddressFlagName)
	if !common.IsHexAddress(drippieAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", DrippieAddressFlagName)
	}
	cfg.DrippieAddress = common.HexToAddress(drippieAddress)

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
			Name:     DrippieAddressFlagName,
			Usage:    "Address of the Drippie contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "DRIPPIE"),
			Required: true,
		},
	}
}
