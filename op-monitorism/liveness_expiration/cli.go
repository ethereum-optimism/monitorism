package liveness_expiration

import (
	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName             = "l1.node.url"
	EventBlockRangeFlagName       = "event.block.range"
	StartingL1BlockHeightFlagName = "start.block.height"

	SafeAddressFlagName           = "safe.address"
	LivenessModuleAddressFlagName = "livenessmodule.address"
	LivenessGuardAddressFlagName  = "livenessguard.address"
)

type CLIConfig struct {
	L1NodeURL             string
	EventBlockRange       uint64
	StartingL1BlockHeight uint64

	LivenessModuleAddress common.Address
	LivenessGuardAddress  common.Address
	SafeAddress           common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:             ctx.String(L1NodeURLFlagName),
		EventBlockRange:       ctx.Uint64(EventBlockRangeFlagName),
		StartingL1BlockHeight: ctx.Uint64(StartingL1BlockHeightFlagName),
		SafeAddress:           common.HexToAddress(ctx.String(SafeAddressFlagName)),
		LivenessModuleAddress: common.HexToAddress(ctx.String(LivenessModuleAddressFlagName)),
		LivenessGuardAddress:  common.HexToAddress(ctx.String(LivenessGuardAddressFlagName)),
	}

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
		&cli.Uint64Flag{
			Name:     StartingL1BlockHeightFlagName,
			Usage:    "Starting height to scan for events (still not implemented for now.. The monitoring will start at the last block number)",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_BLOCK_HEIGHT"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     LivenessModuleAddressFlagName,
			Usage:    "Address of the LivenessModuleAddress contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "LIVENESS_MODULE_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     LivenessGuardAddressFlagName,
			Usage:    "Address of the LivenessGuardAddress contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "LIVENESS_GUARD_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     SafeAddressFlagName,
			Usage:    "Address of the safe contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "SAFE_ADDRESS"),
			Required: true,
		},
	}
}
