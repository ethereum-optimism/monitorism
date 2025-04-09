package faultproof_withdrawals

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	L1GethURLFlagName        = "l1.geth.url"
	L2NodeURLFlagName        = "l2.node.url" // Deprecated
	L2GethURLFlagName        = "l2.geth.url"
	L2GethBackupURLsFlagName = "l2.geth.backup.urls"

	EventBlockRangeFlagName           = "event.block.range"
	StartingL1BlockHeightFlagName     = "start.block.height"
	HoursInThePastToStartFromFlagName = "start.block.hours.ago"

	OptimismPortalAddressFlagName = "optimismportal.address"
)

type CLIConfig struct {
	L1GethURL        string
	L2OpGethURL      string
	L2OpNodeURL      string
	L2GethBackupURLs []string

	EventBlockRange           uint64
	StartingL1BlockHeight     int64
	HoursInThePastToStartFrom uint64

	OptimismPortalAddress common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1GethURL:                 ctx.String(L1GethURLFlagName),
		L2OpGethURL:               ctx.String(L2GethURLFlagName),
		L2GethBackupURLs:          ctx.StringSlice(L2GethBackupURLsFlagName),
		L2OpNodeURL:               "", // Ignored since deprecated
		EventBlockRange:           ctx.Uint64(EventBlockRangeFlagName),
		StartingL1BlockHeight:     ctx.Int64(StartingL1BlockHeightFlagName),
		HoursInThePastToStartFrom: ctx.Uint64(HoursInThePastToStartFromFlagName),
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
			Name:     L1GethURLFlagName,
			Usage:    "L1 execution layer node URL",
			EnvVars:  opservice.PrefixEnvVar(envVar, "L1_GETH_URL"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     L2NodeURLFlagName,
			Usage:    "[DEPRECATED] L2 rollup node consensus layer (op-node) URL - this flag is ignored",
			EnvVars:  opservice.PrefixEnvVar(envVar, "L2_OP_NODE_URL"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     L2GethURLFlagName,
			Usage:    "L2 OP Stack execution layer client(op-geth) URL",
			EnvVars:  opservice.PrefixEnvVar(envVar, "L2_OP_GETH_URL"),
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:     L2GethBackupURLsFlagName,
			Usage:    "Backup L2 OP Stack execution layer client URLs (format: name=url,name2=url2)",
			EnvVars:  opservice.PrefixEnvVar(envVar, "L2_OP_GETH_BACKUP_URLS"),
			Required: false,
		},
		&cli.Uint64Flag{
			Name:     EventBlockRangeFlagName,
			Usage:    "Max block range when scanning for events",
			Value:    1000,
			EnvVars:  opservice.PrefixEnvVar(envVar, "EVENT_BLOCK_RANGE"),
			Required: false,
		},
		&cli.Int64Flag{
			Name:     StartingL1BlockHeightFlagName,
			Usage:    "Starting height to scan for events. This will take precedence if set.",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_BLOCK_HEIGHT"),
			Required: false,
			Value:    -1,
		},
		&cli.Uint64Flag{
			Name:     HoursInThePastToStartFromFlagName,
			Usage:    "How many hours in the past to start to check for forgery. Default will be 336 (14 days) days if not set. The real block to start from will be found within the hour precision.",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_HOURS_IN_THE_PAST"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
	}
}
