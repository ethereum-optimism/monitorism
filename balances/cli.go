package balances

import (
	"fmt"
	"strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName        = "node.url"
	LoopIntervalMsFlagName = "loop.interval"
	AccountsFlagName       = "accounts"
)

func ReadCLIFlags(ctx *cli.Context) (Config, error) {
	cfg := Config{
		NodeUrl:        ctx.String(NodeURLFlagName),
		LoopIntervalMs: ctx.Uint64(LoopIntervalMsFlagName),
	}

	if cfg.LoopIntervalMs == 0 {
		return cfg, fmt.Errorf("loop interval set to zero")
	}

	accounts := ctx.StringSlice(AccountsFlagName)
	for _, account := range accounts {
		split := strings.Split(account, ":")
		if len(split) != 2 {
			return cfg, fmt.Errorf("failed to parse `address:nickname`: %s", account)
		}

		addr, nickname := split[0], split[1]
		if !common.IsHexAddress(addr) {
			return cfg, fmt.Errorf("incorrect address: %s", addr)
		}
		if len(nickname) == 0 {
			return cfg, fmt.Errorf("nickname for %s not set", addr)
		}

		cfg.Accounts = append(cfg.Accounts, Account{common.HexToAddress(addr), nickname})
	}

	return cfg, nil
}

func ClIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL of a peer",
			Value:   "127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.Uint64Flag{
			Name:    LoopIntervalMsFlagName,
			Usage:   "Loop interval in milliseconds",
			Value:   60_000,
			EnvVars: opservice.PrefixEnvVar(envPrefix, "LOOP_INTERVAL_MS"),
		},
		&cli.StringSliceFlag{
			Name:    AccountsFlagName,
			Usage:   "One or accounts formatted via `account1:nickname1,account2:nickname2`",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "ACCOUNTS"),
		},
	}
}
