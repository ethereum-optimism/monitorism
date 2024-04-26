package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/balances"
  "github.com/ethereum-optimism/monitorism/fault"

	"github.com/ethereum-optimism/monitorism/metrics"

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
				Name:        "balances",
				Description: "Monitors the specified account balances",
				Flags:       append(defaultFlags, balances.ClIFlags(EnvVarPrefix)...),
				Action:      cliapp.LifecycleCmd(BalanceMain),
			},
 			{
				Name:        "fault",
				Description: "Monitors the faults than can occured during the OutputRoot Published between L1 and L2.",
				Flags:       append(defaultFlags, fault.ClIFlags(EnvVarPrefix)...),
				Action:      cliapp.LifecycleCmd(FaultMain),
			},

			{
				Name:        "version",
				Description: "print version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
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

func FaultMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	metricsRegistry := opmetrics.NewRegistry()
	metricsConfig := opmetrics.ReadCLIConfig(ctx)

	cfg, err := fault.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fault config from flags: %w", err)
	}

	app, err := fault.NewMonitor(ctx.Context, log, cfg, opmetrics.With(metricsRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to create fault monitor: %w", err)
	}

	return metrics.WithMetricsServer(log, app, metricsRegistry, metricsConfig), nil
}

