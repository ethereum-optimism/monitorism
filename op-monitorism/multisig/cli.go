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

	// Webhook flags
	WebhookURLFlagName = "webhook.url"

	// Risk threshold flags
	HighValueThresholdFlagName = "high.value.threshold.usd"
)

type CLIConfig struct {
	L1NodeURL string
	Nickname  string

	// Notion configuration (required)
	NotionDatabaseID string
	NotionToken      string

	// Webhook configuration (optional)
	WebhookURL string

	// Risk threshold configuration
	HighValueThresholdUSD int
}

func ReadCLIFlags(ctx *cli.Context) (CLIConfig, error) {
	cfg := CLIConfig{
		L1NodeURL:             ctx.String(L1NodeURLFlagName),
		Nickname:              ctx.String(NicknameFlagName),
		NotionDatabaseID:      ctx.String(NotionDatabaseIDFlagName),
		NotionToken:           ctx.String(NotionTokenFlagName),
		WebhookURL:            ctx.String(WebhookURLFlagName),
		HighValueThresholdUSD: ctx.Int(HighValueThresholdFlagName),
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
		}, &cli.StringFlag{
			Name:    WebhookURLFlagName,
			Usage:   "Webhook URL for sending alerts (optional)",
			EnvVars: opservice.PrefixEnvVar(envVar, "WEBHOOK_URL"),
		},
		&cli.IntFlag{
			Name:    HighValueThresholdFlagName,
			Usage:   "USD threshold for high-value Safe validation (e.g., 1000000 for $1M)",
			Value:   1000000, // Default to $1M
			EnvVars: opservice.PrefixEnvVar(envVar, "HIGH_VALUE_THRESHOLD_USD"),
		},
	}
}
