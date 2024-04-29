package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/balances"
	"github.com/ethereum-optimism/monitorism/metrics"
	"github.com/ethereum-optimism/monitorism/withdrawals"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/params"

	"github.com/urfave/cli/v2"
)

const (
	EnvVarPrefix = "MONITORISM"
)

func newCli(GitCommit string, GitDate string) *cli.App {
	defaultFlags := oplog.CLIFlags(EnvVarPrefix)
	defaultFlags = append(defaultFlags, opmetrics.CLIFlags(EnvVarPrefix)...)

	return &cli.App{
		Name:                 "Monitorism",
		Description:          "OP Stack Monitoring",
		EnableBashCompletion: true,
		Version:              params.VersionWithCommit(GitCommit, GitDate),
		Commands: []*cli.Command{
			{
				Name:        "withdrawals",
				Usage:       "Monitors proven withdrawals on L1 against L2",
				Description: "Monitors proven withdrawals on L1 against L2",
				Flags:       append(defaultFlags, withdrawals.CLIFlags(EnvVarPrefix)...),
				Action:      cliapp.LifecycleCmd(WithdrawalsMain),
			},
			{
				Name:        "balances",
				Usage:       "Monitors account balances",
				Description: "Monitors account balances",
				Flags:       append(defaultFlags, balances.CLIFlags(EnvVarPrefix)...),
				Action:      cliapp.LifecycleCmd(BalanceMain),
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

func WithdrawalsMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	metricsRegistry := opmetrics.NewRegistry()
	metricsConfig := opmetrics.ReadCLIConfig(ctx)

	cfg, err := withdrawals.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse withdrawals config from flags: %w", err)
	}

	app, err := withdrawals.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal monitor: %w", err)
	}

	return metrics.WithMetricsServer(log, app, metricsRegistry, metricsConfig), nil
}

func BalanceMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	metricsRegistry := opmetrics.NewRegistry()
	metricsConfig := opmetrics.ReadCLIConfig(ctx)

	cfg, err := balances.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balances config from flags: %w", err)
	}

	app, err := balances.NewMonitor(ctx.Context, log, cfg, opmetrics.With(metricsRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to create balance monitor: %w", err)
	}

	return metrics.WithMetricsServer(log, app, metricsRegistry, metricsConfig), nil
}
