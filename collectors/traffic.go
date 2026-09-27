package collectors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/logger"
)

// TrafficCollector exposes per-inbound, per-outbound, and per-user
// uplink/downlink byte counters reported by xray's stats subsystem.
type TrafficCollector struct {
	client       expvar.Snapshotter
	logger       logger.Logger
	fetchTimeout time.Duration

	inboundUp    *prometheus.Desc
	inboundDown  *prometheus.Desc
	outboundUp   *prometheus.Desc
	outboundDown *prometheus.Desc
	userUp       *prometheus.Desc
	userDown     *prometheus.Desc
}

// NewTrafficCollector returns a prometheus.Collector for xray traffic counters.
func NewTrafficCollector(client expvar.Snapshotter, opts ...Option) (prometheus.Collector, error) {
	if client == nil {
		return nil, ErrClientRequired
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	tagLabels := []string{tagLabel}
	userLabels := []string{userLabel}
	return &TrafficCollector{
		client:       client,
		logger:       cfg.logger,
		fetchTimeout: cfg.fetchTimeout,
		inboundUp:    newDesc("inbound", "uplink_bytes_total", "Total uplink bytes per inbound tag.", tagLabels),
		inboundDown:  newDesc("inbound", "downlink_bytes_total", "Total downlink bytes per inbound tag.", tagLabels),
		outboundUp:   newDesc("outbound", "uplink_bytes_total", "Total uplink bytes per outbound tag.", tagLabels),
		outboundDown: newDesc("outbound", "downlink_bytes_total", "Total downlink bytes per outbound tag.", tagLabels),
		userUp:       newDesc("user", "uplink_bytes_total", "Total uplink bytes per user.", userLabels),
		userDown:     newDesc("user", "downlink_bytes_total", "Total downlink bytes per user.", userLabels),
	}, nil
}

// Describe implements prometheus.Collector.
func (c *TrafficCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.inboundUp
	ch <- c.inboundDown
	ch <- c.outboundUp
	ch <- c.outboundDown
	ch <- c.userUp
	ch <- c.userDown
}

// Collect implements prometheus.Collector.
func (c *TrafficCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.fetchTimeout)
	defer cancel()

	snap, err := c.client.Fetch(ctx)
	if err != nil {
		c.logger.WarnContext(ctx, "traffic collector: scrape failed", "error", err)
		return
	}

	for tag, tc := range snap.Stats.Inbound {
		ch <- prometheus.MustNewConstMetric(c.inboundUp, prometheus.CounterValue, float64(tc.Uplink), tag)
		ch <- prometheus.MustNewConstMetric(c.inboundDown, prometheus.CounterValue, float64(tc.Downlink), tag)
	}
	for tag, tc := range snap.Stats.Outbound {
		ch <- prometheus.MustNewConstMetric(c.outboundUp, prometheus.CounterValue, float64(tc.Uplink), tag)
		ch <- prometheus.MustNewConstMetric(c.outboundDown, prometheus.CounterValue, float64(tc.Downlink), tag)
	}
	for user, tc := range snap.Stats.User {
		ch <- prometheus.MustNewConstMetric(c.userUp, prometheus.CounterValue, float64(tc.Uplink), user)
		ch <- prometheus.MustNewConstMetric(c.userDown, prometheus.CounterValue, float64(tc.Downlink), user)
	}
}
