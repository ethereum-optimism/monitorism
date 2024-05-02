package monitorism

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/httputil"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

const (
	LoopIntervalMsecFlagName = "loop.interval.msec"
)

type Monitor interface {
	Run(context.Context)
	Close(context.Context) error
}

type cliApp struct {
	log     log.Logger
	stopped atomic.Bool

	loopIntervalMs uint64
	worker         *clock.LoopFn

	monitor Monitor

	registry   *prometheus.Registry
	metricsCfg opmetrics.CLIConfig
	metricsSrv *httputil.HTTPServer
}

func NewCliApp(ctx *cli.Context, log log.Logger, registry *prometheus.Registry, monitor Monitor) (cliapp.Lifecycle, error) {
	loopIntervalMs := ctx.Uint64(LoopIntervalMsecFlagName)
	if loopIntervalMs == 0 {
		return nil, errors.New("zero loop interval configured")
	}

	return &cliApp{
		log:            log,
		loopIntervalMs: loopIntervalMs,
		monitor:        monitor,
		registry:       registry,
		metricsCfg:     opmetrics.ReadCLIConfig(ctx),
	}, nil
}

func DefaultCLIFlags(envVarPrefix string) []cli.Flag {
	defaultFlags := append(oplog.CLIFlags(envVarPrefix), opmetrics.CLIFlags(envVarPrefix)...)
	return append(defaultFlags, &cli.Uint64Flag{
		Name:    LoopIntervalMsecFlagName,
		Usage:   "Loop interval of the monitor in milliseconds",
		Value:   60_000,
		EnvVars: opservice.PrefixEnvVar(envVarPrefix, "LOOP_INTERVAL_MSEC"),
	})
}

func (app *cliApp) Start(ctx context.Context) error {
	if app.worker != nil {
		return errors.New("monitor already started")
	}

	app.log.Info("starting metrics server", "host", app.metricsCfg.ListenAddr, "port", app.metricsCfg.ListenPort)
	srv, err := opmetrics.StartServer(app.registry, app.metricsCfg.ListenAddr, app.metricsCfg.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	app.log.Info("starting monitor...", "loop_interval_ms", app.loopIntervalMs)

	// Tick to avoid having to wait a full interval on startup
	app.monitor.Run(ctx)

	app.worker = clock.NewLoopFn(clock.SystemClock, app.monitor.Run, nil, time.Millisecond*time.Duration(app.loopIntervalMs))
	app.metricsSrv = srv
	return nil
}

func (app *cliApp) Stop(ctx context.Context) error {
	if app.stopped.Load() {
		return errors.New("monitor already closed")
	}

	app.log.Info("closing monitor...")
	if err := app.worker.Close(); err != nil {
		app.log.Error("error stopping worker loop", "err", err)
	}
	if err := app.monitor.Close(ctx); err != nil {
		app.log.Error("error closing monitor", "err", err)
	}
	if err := app.metricsSrv.Close(); err != nil {
		app.log.Error("error closing metrics server", "err", err)
	}

	app.stopped.Store(true)
	return nil
}

func (app *cliApp) Stopped() bool {
	return app.stopped.Load()
}
