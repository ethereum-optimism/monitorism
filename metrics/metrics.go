package metrics

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

type cliApp struct {
	log log.Logger

	underlying cliapp.Lifecycle

	cfg        opmetrics.CLIConfig
	registry   *prometheus.Registry
	metricsSrv *httputil.HTTPServer
}

func WithMetricsServer(log log.Logger, app cliapp.Lifecycle, r *prometheus.Registry, cfg opmetrics.CLIConfig) cliapp.Lifecycle {
	return &cliApp{
		log:        log,
		underlying: app,
		cfg:        cfg,
		registry:   r,
	}
}

func (app *cliApp) Start(ctx context.Context) error {
	if app.metricsSrv != nil {
		return errors.New("metrics server already started")
	}

	app.log.Info("starting metrics server", "host", app.cfg.ListenAddr, "port", app.cfg.ListenPort)
	srv, err := opmetrics.StartServer(app.registry, app.cfg.ListenAddr, app.cfg.ListenPort)
	if err != nil {
		return fmt.Errorf("unable to start metrics server: %w", err)
	}

	app.metricsSrv = srv
	return app.underlying.Start(ctx)
}

func (app *cliApp) Stop(ctx context.Context) error {
	if app.metricsSrv == nil {
		return nil
	}

	app.log.Info("stopping metrics server...")
	if err := errors.Join(app.underlying.Stop(ctx), app.metricsSrv.Close()); err != nil {
		return fmt.Errorf("error on close: %w", err)
	}
	return nil
}

func (app *cliApp) Stopped() bool {
	return app.metricsSrv.Closed() && app.underlying.Stopped()
}
