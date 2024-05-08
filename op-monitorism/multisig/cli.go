package multisig

import (
	"fmt"

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

	portalAddress := ctx.String(OptimismPortalAddressFlagName)
	if !common.IsHexAddress(portalAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", OptimismPortalAddressFlagName)
	}
	cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)

	safeAddress := ctx.String(SafeAddressFlagName)
	if len(safeAddress) > 0 {
		if !common.IsHexAddress(safeAddress) {
			return cfg, fmt.Errorf("--%s is not a hex-encoded address", SafeAddressFlagName)
		}
		addr := common.HexToAddress(safeAddress)
		cfg.SafeAddress = &addr
	}

	onePassVault := ctx.String(OnePassVaultFlagName)
	if len(onePassVault) > 0 {
		cfg.OnePassVault = &onePassVault
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
		&cli.StringFlag{
			Name:     OptimismPortalAddressFlagName,
			Usage:    "Address of the OptimismPortal contract",
			EnvVars:  opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     NicknameFlagName,
			Usage:    "Nickname of chain being monitored",
			EnvVars:  opservice.PrefixEnvVar(envVar, "NICKNAME"),
			Required: true,
		},
		&cli.StringFlag{
			Name:    SafeAddressFlagName,
			Usage:   "Address of the Safe contract",
			EnvVars: opservice.PrefixEnvVar(envVar, "SAFE"),
		},
		&cli.StringFlag{
			Name:    OnePassVaultFlagName,
			Usage:   "1Pass vault name storing presigned safe txs following a 'ready-<nonce>.json' item name format",
			EnvVars: opservice.PrefixEnvVar(envVar, "1PASS_VAULT_NAME"),
		},
	}
}
