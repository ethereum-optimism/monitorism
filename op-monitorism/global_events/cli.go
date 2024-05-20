package global_events

import (
	// "fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"

	NicknameFlagName              = "nickname"
	OptimismPortalAddressFlagName = "optimismportal.address"
	SafeAddressFlagName           = "safe.address"
	OnePassVaultFlagName          = "op.vault"
)

type CLIConfig struct {
	L1NodeURL             string
	Nickname              string
	OptimismPortalAddress common.Address

	// Optional
	SafeAddress  *common.Address
	OnePassVault *string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL: ctx.String(L1NodeURLFlagName),
		Nickname:  ctx.String(NicknameFlagName),
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
	}
}
