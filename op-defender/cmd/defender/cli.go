package main

import (
	"context"
	"fmt"

	defender "github.com/ethereum-optimism/monitorism/op-defender"
	"github.com/ethereum-optimism/monitorism/op-defender/psp_executor"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	executor "github.com/ethereum-optimism/monitorism/op-defender/psp_executor"
	"github.com/ethereum/go-ethereum/params"

	"github.com/urfave/cli/v2"
)

const (
	EnvVarPrefix = "DEFENDER"
)

func newCli(GitCommit string, GitDate string) *cli.App {
	defaultFlags := defender.DefaultCLIFlags("DEFENDER")
	return &cli.App{
		Name:                 "Defender",
		Usage:                "OP Stack Automated Defense",
		Description:          "OP Stack Automated Defense",
		EnableBashCompletion: true,
		Version:              params.VersionWithCommit(GitCommit, GitDate),
		Commands: []*cli.Command{
			{
				Name:        "psp_executor",
				Usage:       "Service to execute PSPs through API.",
				Description: "Service to execute PSPs through API.",
				Flags:       append(psp_executor.CLIFlags("PSPEXECUTOR"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(PSPExecutorMain),
			},
			{
				Name:        "version",
				Usage:       "Show version",
				Description: "Show version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}

// PSPExecutorMain() is a the entrypoint for the PSPExecutor API HTTP server.
func PSPExecutorMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := psp_executor.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse psp_executor config from flags: %w", err)
	}
	if err := cfg.Check(); err != nil {
		return nil, err
	}

	metricsRegistry := opmetrics.NewRegistry()
	executor := &executor.DefenderExecutor{}
	defender_thread, err := psp_executor.NewDefender(ctx.Context, log, opmetrics.With(metricsRegistry), cfg, executor)
	if err != nil {
		return nil, fmt.Errorf("Failed to create psp_executor HTTP API service: %w", err)
	}
	return defender.NewCliApp(ctx, log, metricsRegistry, defender_thread)
}
