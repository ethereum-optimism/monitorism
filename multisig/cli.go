package multisig

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"

	OptimismPortalAddressFlagName = "optimismportal.address"
	SafeAddressFlagName           = "safe.address"

	OnePassServiceTokenFlagName = "op.service.token"
	OnePassVaultFlagName        = "op.vault"
)

type Account struct {
	Nickname              string
	SafeAddress           string
	OptimismPortalAddress string
	Vault                 string
}

type CLIConfig struct {
	L1NodeURL             string
	OptimismPortalAddress common.Address
	SafeAddress           common.Address

	OnePassServiceToken string
	OnePassVault        string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:           ctx.String(L1NodeURLFlagName),
		OnePassServiceToken: ctx.String(OnePassServiceTokenFlagName),
		OnePassVault:        ctx.String(OnePassVaultFlagName),
	}

	portalAddress := ctx.String(OptimismPortalAddressFlagName)
	if !common.IsHexAddress(portalAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", OptimismPortalAddressFlagName)
	}
	cfg.OptimismPortalAddress = common.HexToAddress(portalAddress)

	safeAddress := ctx.String(SafeAddressFlagName)
	if !common.IsHexAddress(safeAddress) {
		return cfg, fmt.Errorf("--%s is not a hex-encoded address", SafeAddressFlagName)
	}
	cfg.SafeAddress = common.HexToAddress(safeAddress)

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
			Name:    OptimismPortalAddressFlagName,
			Usage:   "Address of the OptimismPortal contract",
			EnvVars: opservice.PrefixEnvVar(envVar, "OPTIMISM_PORTAL"),
		},
		&cli.StringFlag{
			Name:    SafeAddressFlagName,
			Usage:   "Address of the Safe contract",
			EnvVars: opservice.PrefixEnvVar(envVar, "SAFE"),
		},
		&cli.StringFlag{
			Name:    OnePassServiceTokenFlagName,
			Usage:   "1Pass Service Token",
			EnvVars: opservice.PrefixEnvVar(envVar, "1PASS_SERVICE_TOKEN"),
		},
		&cli.StringFlag{
			Name:    OnePassVaultFlagName,
			Usage:   "1Pass Vault name",
			EnvVars: opservice.PrefixEnvVar(envVar, "1PASS_VAULT_NAME"),
		},
	}
}
