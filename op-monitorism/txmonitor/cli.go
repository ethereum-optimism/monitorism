package txmonitor

import (
	"fmt"
    "os"
    "math/big"
	"encoding/json"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"
	ConfigFileFlagName = "config.file"
)

type WatchConfig struct {
	Address    common.Address         `json:"address"`
	AllowList  []common.Address      `json:"allow_list"`
	Thresholds map[string]*big.Int   `json:"thresholds"`
}

type CLIConfig struct {
	L1NodeUrl     string
	WatchConfigs  []WatchConfig
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeUrl: ctx.String(L1NodeURLFlagName),
	}

	// Read and parse config file
	configFile := ctx.String(ConfigFileFlagName)
	if configFile == "" {
		return cfg, fmt.Errorf("config file must be specified")
	}

	var rawConfigs []struct {
		Address    string            `json:"address"`
		AllowList  []string         `json:"allow_list"`
		Thresholds map[string]string `json:"thresholds"`
	}

	if err := ReadJSONFile(configFile, &rawConfigs); err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse and validate each watch config
	for _, rawConfig := range rawConfigs {
		watchConfig := WatchConfig{
			Thresholds: make(map[string]*big.Int),
		}

		// Validate watch address
		if !common.IsHexAddress(rawConfig.Address) {
			return cfg, fmt.Errorf("invalid watch address: %s", rawConfig.Address)
		}
		watchConfig.Address = common.HexToAddress(rawConfig.Address)

		// Validate allowlist
		for _, addr := range rawConfig.AllowList {
			if !common.IsHexAddress(addr) {
				return cfg, fmt.Errorf("invalid allowlist address: %s", addr)
			}
			watchConfig.AllowList = append(watchConfig.AllowList, common.HexToAddress(addr))
		}

		// Parse thresholds
		for addr, thresholdStr := range rawConfig.Thresholds {
			if !common.IsHexAddress(addr) {
				return cfg, fmt.Errorf("invalid threshold address: %s", addr)
			}

			threshold, ok := new(big.Int).SetString(thresholdStr, 10)
			if !ok {
				return cfg, fmt.Errorf("invalid threshold value for address %s: %s", addr, thresholdStr)
			}

			watchConfig.Thresholds[addr] = threshold
		}

		// Validate that all allowlist addresses have thresholds
		for _, addr := range watchConfig.AllowList {
			if _, exists := watchConfig.Thresholds[addr.Hex()]; !exists {
				return cfg, fmt.Errorf("allowlist address %s missing threshold", addr.Hex())
			}
		}

		cfg.WatchConfigs = append(cfg.WatchConfigs, watchConfig)
	}

	if len(cfg.WatchConfigs) == 0 {
		return cfg, fmt.Errorf("at least one watch config must be specified")
	}

	return cfg, nil
}

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    L1NodeURLFlagName,
			Usage:   "L1 Node URL",
			Value:   "http://localhost:8545",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "L1_NODE_URL"),
		},
		&cli.StringFlag{
			Name:     ConfigFileFlagName,
			Usage:    "Path to JSON config file containing watch addresses, allowlists, and thresholds",
			Required: true,
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "CONFIG_FILE"),
		},
	}
}

func ReadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
