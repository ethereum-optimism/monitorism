package multisig

import (
	"fmt"

	opservice "github.com/ethereum-optimism/optimism/op-service"

	"github.com/urfave/cli/v2"
)

const (
	L1NodeURLFlagName = "l1.node.url"
	NicknameFlagName  = "nickname"

	// Notion flags
	NotionDatabaseIDFlagName = "notion.database.id"
	NotionTokenFlagName      = "notion.token"
)

type CLIConfig struct {
	L1NodeURL string
	Nickname  string

	// Notion configuration (required)
	NotionDatabaseID string
	NotionToken      string
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:        ctx.String(L1NodeURLFlagName),
		Nickname:         ctx.String(NicknameFlagName),
		NotionDatabaseID: ctx.String(NotionDatabaseIDFlagName),
		NotionToken:      ctx.String(NotionTokenFlagName),
	}

	// Notion validation
	if cfg.NotionDatabaseID == "" {
		return cfg, fmt.Errorf("--%s is required", NotionDatabaseIDFlagName)
	}
	if cfg.NotionToken == "" {
		return cfg, fmt.Errorf("--%s is required", NotionTokenFlagName)
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
			Name:     NicknameFlagName,
			Usage:    "Nickname of chain being monitored",
			EnvVars:  opservice.PrefixEnvVar(envVar, "NICKNAME"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     NotionDatabaseIDFlagName,
			Usage:    "Notion database ID containing Safe records",
			EnvVars:  opservice.PrefixEnvVar(envVar, "NOTION_DATABASE_ID"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     NotionTokenFlagName,
			Usage:    "Notion integration token (API key)",
			EnvVars:  opservice.PrefixEnvVar(envVar, "NOTION_TOKEN"),
			Required: true,
		},
	}
}
