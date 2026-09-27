package collectors

import (
	"time"

	"github.com/grum261/xray-exporter/logger"
	loggerzap "github.com/grum261/xray-exporter/logger/zap"
	"github.com/grum261/xray-exporter/option"
)

// defaultFetchTimeout caps a single upstream scrape. Must be shorter than
// Prometheus' scrape_timeout but long enough for cold connections.
const defaultFetchTimeout = 8 * time.Second

// config is the resolved collector configuration assembled from Options.
// Lives in the same package as the collector constructors and never leaves it.
type config struct {
	logger       logger.Logger
	fetchTimeout time.Duration
}

func defaultConfig() *config {
	return &config{
		logger:       loggerzap.Default(),
		fetchTimeout: defaultFetchTimeout,
	}
}

// Option configures a collector. Option is parameterized over the
// unexported config, so only the WithXxx constructors in this package can
// produce values that satisfy it.
type Option = option.Option[config]

// WithLogger overrides the logger used by the collector to report scrape
// failures. Defaults to a production zap logger.
func WithLogger(l logger.Logger) Option {
	return option.New(func(c *config) {
		if l != nil {
			c.logger = l
		}
	})
}

// WithFetchTimeout overrides the per-scrape upstream timeout each collector
// applies when fetching /debug/vars. Must be shorter than Prometheus'
// scrape_timeout but long enough for cold connections. Defaults to 8s.
func WithFetchTimeout(d time.Duration) Option {
	return option.New(func(c *config) {
		if d > 0 {
			c.fetchTimeout = d
		}
	})
}
