
package fault

import (
	// "fmt"
	// "strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	//"github.com/ethereum/go-ethereum/common"

	"github.com/urfave/cli/v2"
)

const (

	l1RpcProvider        = "l1RpcProvider"
  l2RpcProvider        = "l2RpcProvider"
	startOutputIndex = "startOutputIndex"
	optimismPortalAddress       = "optimismPortalAddress"
)

func ReadCLIFlags(ctx *cli.Context) (Config, error) {
	cfg := Config{
		l1RpcProvider:        ctx.String(l1RpcProvider),
		l2RpcProvider:        ctx.String(l2RpcProvider),
		startOutputIndex: ctx.Uint64(startOutputIndex),
		optimismPortalAddress: ctx.String(optimismPortalAddress),
	}
	return cfg, nil
}

// Previous Type Option
// type Options = {
//   l1RpcProvider: Provider
//   l2RpcProvider: Provider
//   startOutputIndex: number
//   optimismPortalAddress?: string
// }


func ClIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    l1RpcProvider,
			Usage:   "l1RpcProvider node url.",
      Value:   "http://127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "L1_RPC_PROVIDER"),
		},
		&cli.StringFlag{
			Name:    l2RpcProvider,
			Usage:   "l2RpcProvider node url.",
      Value:   "http://127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "L2_RPC_PROVIDER"),
		},

		&cli.Uint64Flag{
			Name:    startOutputIndex,
			Usage:   "Start block Index",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "START_OUTPUT_INDEX"),
		},
		&cli.StringFlag{
			Name:    optimismPortalAddress,
			Usage:   "The address of the OP portal.",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "OPTIMISM_PORTAL_ADDRESS"),
		},
	}
}
