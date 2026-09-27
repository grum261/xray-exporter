package collectors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/logger"
)

// ObservatoryCollector exposes xray's observatory probe results
// (per-outbound liveness, RTT, last-seen/last-try timestamps).
type ObservatoryCollector struct {
	client       expvar.Snapshotter
	logger       logger.Logger
	fetchTimeout time.Duration

	alive *prometheus.Desc
	delay *prometheus.Desc
	seen  *prometheus.Desc
	try   *prometheus.Desc

	// Burst observatory health-ping statistics (nil fields for the plain
	// observatory, which does not report them).
	samples   *prometheus.Desc
	failures  *prometheus.Desc
	rttAvg    *prometheus.Desc
	rttMax    *prometheus.Desc
	rttMin    *prometheus.Desc
	rttStdDev *prometheus.Desc
}

// NewObservatoryCollector returns a prometheus.Collector for xray observatory probes.
func NewObservatoryCollector(client expvar.Snapshotter, opts ...Option) (prometheus.Collector, error) {
	if client == nil {
		return nil, ErrClientRequired
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	tagLabels := []string{tagLabel}
	return &ObservatoryCollector{
		client:       client,
		logger:       cfg.logger,
		fetchTimeout: cfg.fetchTimeout,
		alive:        newDesc("observatory", "alive", "Whether the observed outbound is reachable (1=alive, 0=dead).", tagLabels),
		delay:        newDesc("observatory", "probe_delay_milliseconds", "Probe round-trip time in milliseconds.", tagLabels),
		seen:         newDesc("observatory", "probe_last_seen_timestamp_seconds", "Unix timestamp of the last successful probe.", tagLabels),
		try:          newDesc("observatory", "probe_last_try_timestamp_seconds", "Unix timestamp of the last probe attempt.", tagLabels),
		samples:      newDesc("burst_observatory", "samples", "Number of probes in the burst observatory's current sliding window.", tagLabels),
		failures:     newDesc("burst_observatory", "failures", "Number of failed probes in the burst observatory's current sliding window.", tagLabels),
		rttAvg:       newDesc("burst_observatory", "rtt_average_seconds", "Average probe round-trip time over the sliding window.", tagLabels),
		rttMax:       newDesc("burst_observatory", "rtt_max_seconds", "Maximum probe round-trip time over the sliding window.", tagLabels),
		rttMin:       newDesc("burst_observatory", "rtt_min_seconds", "Minimum probe round-trip time over the sliding window.", tagLabels),
		rttStdDev:    newDesc("burst_observatory", "rtt_deviation_seconds", "Standard deviation of probe round-trip time over the sliding window.", tagLabels),
	}, nil
}

// Describe implements prometheus.Collector.
func (c *ObservatoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.alive
	ch <- c.delay
	ch <- c.seen
	ch <- c.try
	ch <- c.samples
	ch <- c.failures
	ch <- c.rttAvg
	ch <- c.rttMax
	ch <- c.rttMin
	ch <- c.rttStdDev
}

// Collect implements prometheus.Collector.
func (c *ObservatoryCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.fetchTimeout)
	defer cancel()

	snap, err := c.client.Fetch(ctx)
	if err != nil {
		c.logger.WarnContext(ctx, "observatory collector: scrape failed", "error", err)
		return
	}

	for tag, p := range snap.Observatory {
		alive := 0.0
		if p.Alive {
			alive = 1.0
		}
		ch <- prometheus.MustNewConstMetric(c.alive, prometheus.GaugeValue, alive, tag)
		ch <- prometheus.MustNewConstMetric(c.delay, prometheus.GaugeValue, float64(p.Delay), tag)

		// The burst observatory omits these timestamps; only emit them when
		// present so we don't report a bogus 1970 epoch.
		if p.LastSeenTime > 0 {
			ch <- prometheus.MustNewConstMetric(c.seen, prometheus.GaugeValue, float64(p.LastSeenTime), tag)
		}
		if p.LastTryTime > 0 {
			ch <- prometheus.MustNewConstMetric(c.try, prometheus.GaugeValue, float64(p.LastTryTime), tag)
		}

		// Health-ping statistics are only present for the burst observatory.
		if hp := p.HealthPing; hp != nil {
			ch <- prometheus.MustNewConstMetric(c.samples, prometheus.GaugeValue, float64(hp.All), tag)
			ch <- prometheus.MustNewConstMetric(c.failures, prometheus.GaugeValue, float64(hp.Fail), tag)
			ch <- prometheus.MustNewConstMetric(c.rttAvg, prometheus.GaugeValue, float64(hp.Average)/nsPerSecond, tag)
			ch <- prometheus.MustNewConstMetric(c.rttMax, prometheus.GaugeValue, float64(hp.Max)/nsPerSecond, tag)
			ch <- prometheus.MustNewConstMetric(c.rttMin, prometheus.GaugeValue, float64(hp.Min)/nsPerSecond, tag)
			ch <- prometheus.MustNewConstMetric(c.rttStdDev, prometheus.GaugeValue, float64(hp.Deviation)/nsPerSecond, tag)
		}
	}
}
