package collectors

import (
	"context"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/logger"
)

// nsPerSecond converts nanoseconds to seconds for memstats pause times.
const nsPerSecond = 1e9

// MemoryCollector exposes the runtime/MemStats fields surfaced by xray's
// /debug/vars endpoint (heap, GC, allocator counters).
type MemoryCollector struct {
	client       expvar.Snapshotter
	logger       logger.Logger
	fetchTimeout time.Duration

	alloc        *prometheus.Desc
	sys          *prometheus.Desc
	heapAlloc    *prometheus.Desc
	heapInuse    *prometheus.Desc
	heapIdle     *prometheus.Desc
	heapReleased *prometheus.Desc
	nextGC       *prometheus.Desc
	gcFraction   *prometheus.Desc
	numGC        *prometheus.Desc
	gcPause      *prometheus.Desc
	mallocs      *prometheus.Desc
	frees        *prometheus.Desc
}

// NewMemoryCollector returns a prometheus.Collector for xray memory metrics.
func NewMemoryCollector(client expvar.Snapshotter, opts ...Option) (prometheus.Collector, error) {
	if client == nil {
		return nil, ErrClientRequired
	}
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return &MemoryCollector{
		client:       client,
		logger:       cfg.logger,
		fetchTimeout: cfg.fetchTimeout,
		alloc:        newDesc("memstats", "alloc_bytes", "Bytes of allocated heap objects.", nil),
		sys:          newDesc("memstats", "sys_bytes", "Total bytes of memory obtained from the OS.", nil),
		heapAlloc:    newDesc("memstats", "heap_alloc_bytes", "Bytes of allocated heap objects.", nil),
		heapInuse:    newDesc("memstats", "heap_inuse_bytes", "Bytes in in-use spans.", nil),
		heapIdle:     newDesc("memstats", "heap_idle_bytes", "Bytes in idle (unused) spans.", nil),
		heapReleased: newDesc("memstats", "heap_released_bytes", "Bytes of physical memory returned to the OS.", nil),
		nextGC:       newDesc("memstats", "next_gc_bytes", "Target heap size for next GC cycle.", nil),
		gcFraction:   newDesc("memstats", "gc_cpu_fraction", "Fraction of CPU time used by GC since program start.", nil),
		numGC:        newDesc("memstats", "num_gc_total", "Number of completed GC cycles.", nil),
		gcPause:      newDesc("memstats", "gc_pause_seconds_total", "Cumulative GC stop-the-world pause time in seconds.", nil),
		mallocs:      newDesc("memstats", "mallocs_total", "Cumulative count of heap objects allocated.", nil),
		frees:        newDesc("memstats", "frees_total", "Cumulative count of heap objects freed.", nil),
	}, nil
}

// Describe implements prometheus.Collector.
func (c *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.alloc
	ch <- c.sys
	ch <- c.heapAlloc
	ch <- c.heapInuse
	ch <- c.heapIdle
	ch <- c.heapReleased
	ch <- c.nextGC
	ch <- c.gcFraction
	ch <- c.numGC
	ch <- c.gcPause
	ch <- c.mallocs
	ch <- c.frees
}

// Collect implements prometheus.Collector.
func (c *MemoryCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.fetchTimeout)
	defer cancel()

	snap, err := c.client.Fetch(ctx)
	if err != nil {
		c.logger.WarnContext(ctx, "memory collector: scrape failed", "error", err)
		return
	}

	m := snap.MemStats

	gauges := []struct {
		desc  *prometheus.Desc
		value uint64
	}{
		{c.alloc, m.Alloc},
		{c.sys, m.Sys},
		{c.heapAlloc, m.HeapAlloc},
		{c.heapInuse, m.HeapInuse},
		{c.heapIdle, m.HeapIdle},
		{c.heapReleased, m.HeapReleased},
		{c.nextGC, m.NextGC},
	}
	for _, g := range gauges {
		ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, float64(g.value))
	}

	ch <- prometheus.MustNewConstMetric(c.gcFraction, prometheus.GaugeValue, m.GCCPUFraction)

	// GC pause exposed in seconds (PauseTotalNs / 1e9).
	pauseSeconds := float64(m.PauseTotalNs) / nsPerSecond
	if math.IsInf(pauseSeconds, 0) || math.IsNaN(pauseSeconds) {
		pauseSeconds = 0
	}

	counters := []struct {
		desc  *prometheus.Desc
		value float64
	}{
		{c.numGC, float64(m.NumGC)},
		{c.gcPause, pauseSeconds},
		{c.mallocs, float64(m.Mallocs)},
		{c.frees, float64(m.Frees)},
	}
	for _, ct := range counters {
		ch <- prometheus.MustNewConstMetric(ct.desc, prometheus.CounterValue, ct.value)
	}
}
