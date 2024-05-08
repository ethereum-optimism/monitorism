package balances

import (
	"fmt"
	"strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName  = "node.url"
	AccountsFlagName = "accounts"
)

type CLIConfig struct {
	NodeUrl  string
	Accounts []Account
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{NodeUrl: ctx.String(NodeURLFlagName)}
	accounts := ctx.StringSlice(AccountsFlagName)
	if len(accounts) == 0 {
		return cfg, fmt.Errorf("--%s must have at least one account", AccountsFlagName)
	}

	for _, account := range accounts {
		split := strings.Split(account, ":")
		if len(split) != 2 {
			return cfg, fmt.Errorf("failed to parse `address:nickname`: %s", account)
		}

		addr, nickname := split[0], split[1]
		if !common.IsHexAddress(addr) {
			return cfg, fmt.Errorf("address is not a hex-encoded address: %s", addr)
		}
		if len(nickname) == 0 {
			return cfg, fmt.Errorf("nickname for %s not set", addr)
		}

		cfg.Accounts = append(cfg.Accounts, Account{common.HexToAddress(addr), nickname})
	}

	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL of a peer",
			Value:   "127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.StringSliceFlag{
			Name:     AccountsFlagName,
			Usage:    "One or accounts formatted via `address:nickname`",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "ACCOUNTS"),
			Required: true,
		},
	}
}
