package main

import (
	"context"
	"fmt"

	monitorism "github.com/ethereum-optimism/monitorism/op-monitorism"
	"github.com/ethereum-optimism/monitorism/op-monitorism/balances"
	"github.com/ethereum-optimism/monitorism/op-monitorism/drippie"
	"github.com/ethereum-optimism/monitorism/op-monitorism/fault"
	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals"
	"github.com/ethereum-optimism/monitorism/op-monitorism/global_events"
	"github.com/ethereum-optimism/monitorism/op-monitorism/liveness_expiration"
	"github.com/ethereum-optimism/monitorism/op-monitorism/multisig"
	"github.com/ethereum-optimism/monitorism/op-monitorism/secrets"
	"github.com/ethereum-optimism/monitorism/op-monitorism/transaction_monitor"
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
				Name:        "drippie",
				Usage:       "Monitors Drippie contract",
				Description: "Monitors Drippie contract",
				Flags:       append(drippie.CLIFlags("DRIPPIE_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(DrippieMain),
			},
			{
				Name:        "secrets",
				Usage:       "Monitors secrets revealed in the CheckSecrets dripcheck",
				Description: "Monitors secrets revealed in the CheckSecrets dripcheck",
				Flags:       append(secrets.CLIFlags("SECRETS_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(SecretsMain),
			},
			{
				Name:        "global_events",
				Usage:       "Monitors global events with YAML configuration",
				Description: "Monitors global events with YAML configuration",
				Flags:       append(global_events.CLIFlags("GLOBAL_EVENT_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(GlobalEventMain),
			},
			{
				Name:        "liveness_expiration",
				Usage:       "Monitor the liveness expiration on Gnosis Safe.",
				Description: "Monitor the liveness expiration on Gnosis Safe.",
				Flags:       append(liveness_expiration.CLIFlags("LIVENESS_EXPIRATION_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(LivenessExpirationMain),
			},
			{
				Name:        "faultproof_withdrawals",
				Usage:       "Monitors withdrawals on the OptimismPortal in order to detect forgery. Note: Requires chains with Fault Proofs.",
				Description: "Monitors withdrawals on the OptimismPortal in order to detect forgery. Note: Requires chains with Fault Proofs.",
				Flags:       append(faultproof_withdrawals.CLIFlags("FAULTPROOF_WITHDRAWAL_MON"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(FaultproofWithdrawalsMain),
			},
			{
				Name:        "transaction_monitor",
				Usage:       "Monitors transactions from specified addresses and alerts above a certain threshold",
				Description: "Monitors transactions from specified addresses and alerts above a certain threshold",
				Flags:       append(transaction_monitor.CLIFlags("TRANSACTION_MONITOR"), defaultFlags...),
				Action:      cliapp.LifecycleCmd(TransactionMonitorMain),
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

func LivenessExpirationMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := liveness_expiration.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LivenessExpiration config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := liveness_expiration.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LivenessExpiration monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}
func GlobalEventMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := global_events.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse global_events config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := global_events.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create global_events monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
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
		return nil, fmt.Errorf("failed to create withdrawals monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func FaultproofWithdrawalsMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := faultproof_withdrawals.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse faultproof withdrawals config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := faultproof_withdrawals.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create faultproof withdrawals monitor: %w", err)
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

func DrippieMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := drippie.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse drippie config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := drippie.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create drippie monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func SecretsMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := secrets.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secrets config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := secrets.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}

func TransactionMonitorMain(ctx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
	log := oplog.NewLogger(oplog.AppOut(ctx), oplog.ReadCLIConfig(ctx))
	cfg, err := transaction_monitor.ReadCLIFlags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction monitor config from flags: %w", err)
	}

	metricsRegistry := opmetrics.NewRegistry()
	monitor, err := transaction_monitor.NewMonitor(ctx.Context, log, opmetrics.With(metricsRegistry), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction monitor: %w", err)
	}

	return monitorism.NewCliApp(ctx, log, metricsRegistry, monitor)
}
