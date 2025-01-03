package fault

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"
	L2NodeURLFlagName = "l2.node.url"

	OptimismPortalAddressFlagName = "optimismportal.address"
	StartOutputIndexFlagName      = "start.output.index"
)

type CLIConfig struct {
	L1NodeURL string
	L2NodeURL string

	OptimismPortalAddress common.Address
	StartOutputIndex      int64
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:        ctx.String(L1NodeURLFlagName),
		L2NodeURL:        ctx.String(L2NodeURLFlagName),
		StartOutputIndex: ctx.Int64(StartOutputIndexFlagName),
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
			Usage:   "Node URL of L1 peer Geth node",
			EnvVars: opservice.PrefixEnvVar(envVar, "L1_NODE_URL"),
		},
		&cli.StringFlag{
			Name:    L2NodeURLFlagName,
			Usage:   "Node URL of L2 peer Op-Geth node",
			EnvVars: opservice.PrefixEnvVar(envVar, "L2_NODE_URL"),
		},
		&cli.Int64Flag{
			Name:    StartOutputIndexFlagName,
			Usage:   "Output index to start from. -1 to find first unfinalized index",
			Value:   -1,
			EnvVars: opservice.PrefixEnvVar(envVar, "START_OUTPUT_INDEX"),
		},
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
	}
}
