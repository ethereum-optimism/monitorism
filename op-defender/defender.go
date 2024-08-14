package defender

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/httputil"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

type Defender interface {
	Run(context.Context)
	Close(context.Context) error
}
type cliApp struct {
	log            log.Logger
	stopped        atomic.Bool
	loopIntervalMs uint64 // loop interval of the main thread.
	worker         *clock.LoopFn
	defender       Defender
	registry       *prometheus.Registry
	metricsCfg     opmetrics.CLIConfig
	metricsSrv     *httputil.HTTPServer
}

func NewCliApp(ctx *cli.Context, log log.Logger, registry *prometheus.Registry, defender Defender) (cliapp.Lifecycle, error) {

	return &cliApp{
		log:        log,
		defender:   defender,
		registry:   registry,
		metricsCfg: opmetrics.ReadCLIConfig(ctx),
	}, nil
}

func DefaultCLIFlags(envVarPrefix string) []cli.Flag {
	defaultFlags := append(oplog.CLIFlags(envVarPrefix), opmetrics.CLIFlags(envVarPrefix)...)
	return defaultFlags
}

func (app *cliApp) Start(ctx context.Context) error {
	if app.worker != nil {
		return errors.New("Defender service already running..")
	}

	app.log.Info("starting metrics server", "host", app.metricsCfg.ListenAddr, "port", app.metricsCfg.ListenPort)
	srv, err := opmetrics.StartServer(app.registry, app.metricsCfg.ListenAddr, app.metricsCfg.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	app.log.Info("Start defender service", "loop_interval_ms", app.loopIntervalMs)

	// Tick to avoid having to wait a full interval on startup
	app.defender.Run(ctx)

	app.metricsSrv = srv
	return nil
}

func (app *cliApp) Stop(ctx context.Context) error {
	if app.stopped.Load() {
		return errors.New("defender already closed")
	}

	app.log.Info("closing defender...")
	if err := app.worker.Close(); err != nil {
		app.log.Error("error stopping worker loop", "err", err)
	}
	if err := app.defender.Close(ctx); err != nil {
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
