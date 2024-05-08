package main

import (
	"context"
	"fmt"

	monitorism "github.com/ethereum-optimism/monitorism/op-monitorism"
	"github.com/ethereum-optimism/monitorism/op-monitorism/balances"
	"github.com/ethereum-optimism/monitorism/op-monitorism/fault"
	"github.com/ethereum-optimism/monitorism/op-monitorism/multisig"
	"github.com/ethereum-optimism/monitorism/op-monitorism/withdrawals"

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
	defaultFlags := monitorism.DefaultCLIFlags("MONITORISM")
	return &cli.App{
		Name:                 "Monitorism",
		Usage:                "OP Stack Monitoring",
		Description:          "OP Stack Monitoring",
		EnableBashCompletion: true,
		Version:              params.VersionWithCommit(GitCommit, GitDate),
		Commands: []*cli.Command{
			{
				Name:        "multisig",
				Usage:       "Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password",
				Description: "Monitors OptimismPortal pause status, Safe nonce, and Pre-Signed nonce stored in 1Password",
				Flags:       append(multisig.CLIFlags("MULTISIG_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(MultisigMain),
			},
			{
				Name:        "fault",
				Usage:       "Monitors output roots posted on L1 against L2",
				Description: "Monitors output roots posted on L1 against L2",
				Flags:       append(fault.CLIFlags("FAULT_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(FaultMain),
			},
			{
				Name:        "withdrawals",
				Usage:       "Monitors proven withdrawals on L1 against L2",
				Description: "Monitors proven withdrawals on L1 against L2",
				Flags:       append(withdrawals.CLIFlags("WITHDRAWAL_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(WithdrawalsMain),
			},
			{
				Name:        "balances",
				Usage:       "Monitors account balances",
				Description: "Monitors account balances",
				Flags:       append(balances.CLIFlags("BALANCE_MON"), defaultFlags...),
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

func MultisigMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := multisig.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multisig config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := multisig.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create multisig monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func FaultMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := fault.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fault config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := fault.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create fault monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func WithdrawalsMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := withdrawals.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse withdrawals config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := withdrawals.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create withdrawal monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func BalanceMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := balances.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse balances config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := balances.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}
