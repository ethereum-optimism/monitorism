package liveness_expiration

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"
	L2NodeURLFlagName = "l2.node.url"

	EventBlockRangeFlagName       = "event.block.range"
	StartingL1BlockHeightFlagName = "start.block.height"

	SafeAddressFlagName      = "safe.address"
	LoopIntervalMsecFlagName = "loop.interval.msec"
)

type CLIConfig struct {
	L1NodeURL string
	L2NodeURL string

	EventBlockRange       uint64
	LoopIntervalMsec      uint64
	StartingL1BlockHeight uint64

	SafeAddress common.Address
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:             ctx.String(L1NodeURLFlagName),
		L2NodeURL:             ctx.String(L2NodeURLFlagName),
		EventBlockRange:       ctx.Uint64(EventBlockRangeFlagName),
		StartingL1BlockHeight: ctx.Uint64(StartingL1BlockHeightFlagName),
		LoopIntervalMsec:      ctx.Uint64(LoopIntervalMsecFlagName),
		SafeAddress:           common.HexToAddress(ctx.String(SafeAddressFlagName)),
	}

	if cfg.LoopIntervalMsec == 0 {
		return cfg, errors.New("no loop interval configured")
	}

	portalAddress := ctx.String(SafeAddressFlagName)
	if !common.IsHexAddress(portalAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", SafeAddressFlagName)
	}
	// cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)

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
			Usage:    "Starting height to scan for events",
			EnvVars:  opservice.PrefixEnvVar(envVar, "START_BLOCK_HEIGHT"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     SafeAddressFlagName,
			Usage:    "Address of the safe contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
	}
}
