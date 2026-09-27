// Command xray-exporter exposes xray-core /debug/vars as Prometheus metrics.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/collectors"
	"github.com/grum261/xray-exporter/config"
	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/internal/version"
	"github.com/grum261/xray-exporter/logger"
	"github.com/grum261/xray-exporter/server"
)

// exitUsageError is the conventional exit code for a command-line usage
// error, matching the behavior of the standard flag package.
const exitUsageError = 2

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(exitUsageError)
	}
	if cfg.ShowVersion {
		fmt.Println(version.Get())
		return
	}
	log := cfg.NewLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("exporter terminated with error", "error", err)
		os.Exit(1)
	}
}

// run is the composition root: it builds the expvar client, the
// configured collectors, and the HTTP server, then blocks on Run.
func run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	info := version.Get()
	log.Info("starting xray-exporter", "version", info.Version, "revision", info.Revision, "go", info.GoVersion)

	cached, err := newCachedClient(cfg)
	if err != nil {
		return err
	}

	cs, err := buildCollectors(cached, log, cfg, info)
	if err != nil {
		return fmt.Errorf("build collectors: %w", err)
	}

	srv, err := server.New(serverOptions(cfg, log, cs)...)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	return srv.Run(ctx)
}

func newCachedClient(cfg *config.Config) (*expvar.CachedClient, error) {
	var clientOpts []expvar.Option
	if cfg.XrayEndpoint != "" {
		clientOpts = append(clientOpts, expvar.WithEndpoint(cfg.XrayEndpoint))
	}
	if cfg.ScrapeTimeout > 0 {
		clientOpts = append(clientOpts, expvar.WithTimeout(cfg.ScrapeTimeout))
	}

	client, err := expvar.NewClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create expvar client: %w", err)
	}
	cached, err := expvar.NewCachedClient(client)
	if err != nil {
		return nil, fmt.Errorf("create cached client: %w", err)
	}
	return cached, nil
}

func serverOptions(cfg *config.Config, log logger.Logger, cs []prometheus.Collector) []server.Option {
	opts := []server.Option{
		server.WithLogger(log),
		server.WithCollectors(cs...),
	}
	if cfg.ListenAddr != "" {
		opts = append(opts, server.WithListenAddr(cfg.ListenAddr))
	}
	if cfg.MetricsPath != "" {
		opts = append(opts, server.WithMetricsPath(cfg.MetricsPath))
	}
	return opts
}

// buildCollectors assembles the registered collectors via a uniform table:
// the exporter's own self-metrics (go, process, build_info) are always on,
// followed by the xray collectors (scrape, memory, traffic, observatory) in
// that order, each gated by cfg.
func buildCollectors(cached *expvar.CachedClient, log logger.Logger, cfg *config.Config, info version.Info) ([]prometheus.Collector, error) {
	withLogger := collectors.WithLogger(log)

	type entry struct {
		name    string
		enabled bool
		build   func() (prometheus.Collector, error)
	}

	entries := []entry{
		// Exporter self-metrics — describe the exporter process itself, not
		// xray. Always enabled.
		{"go", true, func() (prometheus.Collector, error) {
			return collectors.NewGoCollector(), nil
		}},
		{"process", true, func() (prometheus.Collector, error) {
			return collectors.NewProcessCollector(), nil
		}},
		{"build_info", true, func() (prometheus.Collector, error) {
			return collectors.NewBuildInfoCollector(info), nil
		}},
		{"scrape", cfg.EnableScrape, func() (prometheus.Collector, error) {
			return collectors.NewScrapeCollector(cached, withLogger)
		}},
		{"memory", cfg.EnableMemory, func() (prometheus.Collector, error) {
			return collectors.NewMemoryCollector(cached, withLogger)
		}},
		{"traffic", cfg.EnableTraffic, func() (prometheus.Collector, error) {
			return collectors.NewTrafficCollector(cached, withLogger)
		}},
		{"observatory", cfg.EnableObservatory, func() (prometheus.Collector, error) {
			return collectors.NewObservatoryCollector(cached, withLogger)
		}},
	}

	var out []prometheus.Collector
	for _, e := range entries {
		if !e.enabled {
			log.Info("collector disabled", "name", e.name)
			continue
		}
		c, err := e.build()
		if err != nil {
			return nil, fmt.Errorf("build %s collector: %w", e.name, err)
		}
		log.Info("collector enabled", "name", e.name)
		out = append(out, c)
	}
	return out, nil
}
