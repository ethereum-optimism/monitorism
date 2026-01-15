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
	L2OOAddressFlagName           = "l2oo.address"
	StartOutputIndexFlagName      = "start.output.index"
)

type CLIConfig struct {
	L1NodeURL string
	L2NodeURL string

	OptimismPortalAddress common.Address
	L2OOAddress           common.Address
	StartOutputIndex      int64
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:        ctx.String(L1NodeURLFlagName),
		L2NodeURL:        ctx.String(L2NodeURLFlagName),
		StartOutputIndex: ctx.Int64(StartOutputIndexFlagName),
	}

	// Check if L2OO address is provided directly
	l2OOAddress := ctx.String(L2OOAddressFlagName)
	portalAddress := ctx.String(OptimismPortalAddressFlagName)

	// Validate that at least one address is provided
	if l2OOAddress == "" && portalAddress == "" {
		return cfg, fmt.Errorf("either --%s or --%s must be provided", L2OOAddressFlagName, OptimismPortalAddressFlagName)
	}

	// Validate that both are not provided (to avoid confusion)
	if l2OOAddress != "" && portalAddress != "" {
		return cfg, fmt.Errorf("cannot provide both --%s and --%s, choose one", L2OOAddressFlagName, OptimismPortalAddressFlagName)
	}

	if l2OOAddress != "" {
		if !common.IsHexAddress(l2OOAddress) {
			return cfg, fmt.Errorf("--%s is not a hex-encoded address", L2OOAddressFlagName)
		}
		cfg.L2OOAddress = common.HexToAddress(l2OOAddress)
	} else {
		if !common.IsHexAddress(portalAddress) {
			return cfg, fmt.Errorf("--%s is not a hex-encoded address", OptimismPortalAddressFlagName)
		}
		cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)
	}

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
			Name:    OptimismPortalAddressFlagName,
			Usage:   "Address of the OptimismPortal contract (alternative to l2oo.address)",
			EnvVars: opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
		},
		&cli.StringFlag{
			Name:    L2OOAddressFlagName,
			Usage:   "Address of the L2OutputOracle contract (alternative to optimismportal.address)",
			EnvVars: opservice.PrefixEnvVar(envVar, "L2OO_ADDRESS"),
		},
	}
}
