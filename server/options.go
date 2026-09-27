package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/logger"
	loggerzap "github.com/grum261/xray-exporter/logger/zap"
	"github.com/grum261/xray-exporter/option"
)

const (
	defaultListenAddr        = ":9356"
	defaultMetricsPath       = "/metrics"
	defaultReadHeaderTimeout = 5 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

// config is the resolved server configuration assembled from Options.
type config struct {
	listenAddr        string
	metricsPath       string
	readHeaderTimeout time.Duration
	shutdownTimeout   time.Duration
	logger            logger.Logger
	collectors        []prometheus.Collector
}

func defaultConfig() *config {
	return &config{
		listenAddr:        defaultListenAddr,
		metricsPath:       defaultMetricsPath,
		readHeaderTimeout: defaultReadHeaderTimeout,
		shutdownTimeout:   defaultShutdownTimeout,
		logger:            loggerzap.Default(),
	}
}

// Option configures a Server. Option is parameterized over the unexported
// config, so only the WithXxx constructors in this package can produce
// values that satisfy it.
type Option = option.Option[config]

// WithListenAddr overrides the address the HTTP server listens on.
func WithListenAddr(addr string) Option {
	return option.New(func(c *config) {
		if addr != "" {
			c.listenAddr = addr
		}
	})
}

// WithMetricsPath overrides the path under which metrics are exposed.
func WithMetricsPath(path string) Option {
	return option.New(func(c *config) {
		if path != "" {
			c.metricsPath = path
		}
	})
}

// WithReadHeaderTimeout overrides the HTTP server's ReadHeaderTimeout.
func WithReadHeaderTimeout(d time.Duration) Option {
	return option.New(func(c *config) {
		if d > 0 {
			c.readHeaderTimeout = d
		}
	})
}

// WithShutdownTimeout overrides the graceful-shutdown deadline.
func WithShutdownTimeout(d time.Duration) Option {
	return option.New(func(c *config) {
		if d > 0 {
			c.shutdownTimeout = d
		}
	})
}

// WithLogger overrides the logger. Defaults to a production zap logger.
func WithLogger(l logger.Logger) Option {
	return option.New(func(c *config) {
		if l != nil {
			c.logger = l
		}
	})
}

// WithCollectors appends prometheus.Collectors to be registered when
// the Server is constructed. May be called multiple times — collectors
// accumulate across calls.
func WithCollectors(cs ...prometheus.Collector) Option {
	return option.New(func(c *config) {
		c.collectors = append(c.collectors, cs...)
	})
}
