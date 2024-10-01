package psp_executor

import (
	opservice "github.com/ethereum-optimism/optimism/op-service"
	optls "github.com/ethereum-optimism/optimism/op-service/tls"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	NodeURLFlagName                 = "rpc.url"
	PrivateKeyFlagName              = "privatekey"
	PortAPIFlagName                 = "port.api"
	SuperChainConfigAddressFlagName = "superchainconfig.address"
	SafeAddressFlagName             = "safe.address"
	PathFlagName                    = "path"
	ChainIDFlagName                 = "chainid"
	BlockDurationFlagName           = "blockduration"
)

type CLIConfig struct {
	NodeURL                 string
	PortAPI                 string
	Path                    string
	BlockDuration           uint64
	privatekeyflag          string
	SuperChainConfigAddress common.Address
	SafeAddress             common.Address
	ChainID                 uint64
	TLSConfig               optls.CLIConfig
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		NodeURL:                 ctx.String(NodeURLFlagName),
		PortAPI:                 ctx.String(PortAPIFlagName),
		Path:                    ctx.String(PathFlagName),
		privatekeyflag:          ctx.String(PrivateKeyFlagName),
		SuperChainConfigAddress: common.HexToAddress(ctx.String(SuperChainConfigAddressFlagName)),
		SafeAddress:             common.HexToAddress(ctx.String(SafeAddressFlagName)),
		ChainID:                 ctx.Uint64(ChainIDFlagName),
		BlockDuration:           ctx.Uint64(BlockDurationFlagName),
		TLSConfig:               optls.ReadCLIConfig(ctx),
	}

	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    NodeURLFlagName,
			Usage:   "Node URL of a peer",
			Value:   "http://127.0.0.1:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "NODE_URL"),
		},
		&cli.StringFlag{
			Name:     PrivateKeyFlagName,
			Usage:    "Privatekey of the account that will issue the pause transaction",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PRIVATE_KEY"),
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     PortAPIFlagName,
			Value:    8080,
			Usage:    "Port of the API server you want to listen on (e.g. 8080)",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PORT_API"),
			Required: false,
		},
		&cli.StringFlag{
			Name:     SuperChainConfigAddressFlagName,
			Usage:    "SuperChainConfig address to know the current status of the superchainconfig",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "SUPERCHAINCONFIG_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     SafeAddressFlagName,
			Usage:    "Safe address that will execute the PSPs",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "SAFE_ADDRESS"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     PathFlagName,
			Usage:    "Absolute path to the JSON file containing the PSPs",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "PATH_TO_PSPS"),
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     BlockDurationFlagName,
			Usage:    "Block duration of the current chain that op-defender is running on",
			Value:    12,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "BLOCK_DURATION"),
			Required: false,
		},
		&cli.Uint64Flag{
			Name:     ChainIDFlagName,
			Usage:    "ChainID of the current chain that op-defender is running on",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "CHAIN_ID"),
			Required: true,
		},
	}
	// Add mtls flags
	flags = append(flags, []cli.Flag{
		&cli.StringFlag{
			Name:    optls.TLSCaCertFlagName,
			Usage:   "tls ca cert path",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "TLS_CA"),
		},
		&cli.StringFlag{
			Name:    optls.TLSCertFlagName,
			Usage:   "tls cert path",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "TLS_CERT"),
		},
		&cli.StringFlag{
			Name:    optls.TLSKeyFlagName,
			Usage:   "tls key",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "TLS_KEY"),
		},
	}...)
	return flags
}

func (c CLIConfig) Check() error {

	if err := c.TLSConfig.Check(); err != nil {
		return err
	}
	return nil
}
