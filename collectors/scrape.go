package collectors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/logger"
)

// ScrapeCollector emits health metrics about the upstream xray scrape:
// scrape_success and scrape_duration_seconds. It triggers a Fetch on
// every Collect (coalesced by CachedClient with peers), then reads the
// duration/error recorded by the cached client.
type ScrapeCollector struct {
	client       *expvar.CachedClient
	logger       logger.Logger
	fetchTimeout time.Duration

	success  *prometheus.Desc
	duration *prometheus.Desc
}

// NewScrapeCollector returns a prometheus.Collector that exposes
// xray_scrape_success and xray_scrape_duration_seconds.
func NewScrapeCollector(client *expvar.CachedClient, opts ...Option) (prometheus.Collector, error) {
	if client == nil {
		return nil, ErrClientRequired
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return &ScrapeCollector{
		client:       client,
		logger:       cfg.logger,
		fetchTimeout: cfg.fetchTimeout,
		success:      newDesc("", "scrape_success", "Whether the last xray /debug/vars scrape succeeded (1=success, 0=failure).", nil),
		duration:     newDesc("", "scrape_duration_seconds", "Duration of the last xray scrape in seconds.", nil),
	}, nil
}

// Describe implements prometheus.Collector.
func (c *ScrapeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.success
	ch <- c.duration
}

// Collect implements prometheus.Collector.
func (c *ScrapeCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.fetchTimeout)
	defer cancel()

	if _, err := c.client.Fetch(ctx); err != nil {
		c.logger.WarnContext(ctx, "xray scrape failed", "error", err)
	}

	duration, ok, err := c.client.LastFetch()
	if !ok {
		// No fetch recorded yet — emit success=0 to avoid silence.
		ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.duration, prometheus.GaugeValue, 0)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.duration, prometheus.GaugeValue, duration.Seconds())
	success := 1.0
	if err != nil {
		success = 0
	}
	ch <- prometheus.MustNewConstMetric(c.success, prometheus.GaugeValue, success)
}
