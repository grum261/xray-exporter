package collectors

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap"

	"github.com/grum261/xray-exporter/expvar"
	"github.com/grum261/xray-exporter/internal/version"
	"github.com/grum261/xray-exporter/logger"
	loggerzap "github.com/grum261/xray-exporter/logger/zap"
)

// stubClient lets tests inject snapshots or errors.
type stubClient struct {
	snap expvar.Snapshot
	err  error
}

func (s *stubClient) Fetch(_ context.Context) (expvar.Snapshot, error) {
	return s.snap, s.err
}

func discardLogger() logger.Logger {
	return loggerzap.New(zap.NewNop())
}

func newCachedStub(t *testing.T, snap expvar.Snapshot, err error) *expvar.CachedClient {
	t.Helper()
	c, cerr := expvar.NewCachedClient(&stubClient{snap: snap, err: err})
	if cerr != nil {
		t.Fatalf("NewCachedClient: %v", cerr)
	}
	return c
}

func TestTrafficCollector_Success(t *testing.T) {
	t.Parallel()

	snap := expvar.Snapshot{
		Stats: expvar.Stats{
			Inbound:  map[string]expvar.TrafficCounters{"transparent": {Uplink: 100, Downlink: 200}},
			Outbound: map[string]expvar.TrafficCounters{"proxy": {Uplink: 1000, Downlink: 2000}, "direct": {Uplink: 50, Downlink: 75}},
		},
	}
	coll, err := NewTrafficCollector(&stubClient{snap: snap}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewTrafficCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_outbound_downlink_bytes_total Total downlink bytes per outbound tag.
# TYPE xray_outbound_downlink_bytes_total counter
xray_outbound_downlink_bytes_total{tag="direct"} 75
xray_outbound_downlink_bytes_total{tag="proxy"} 2000
# HELP xray_outbound_uplink_bytes_total Total uplink bytes per outbound tag.
# TYPE xray_outbound_uplink_bytes_total counter
xray_outbound_uplink_bytes_total{tag="direct"} 50
xray_outbound_uplink_bytes_total{tag="proxy"} 1000
`
	err = testutil.GatherAndCompare(reg, strings.NewReader(want),
		"xray_outbound_downlink_bytes_total",
		"xray_outbound_uplink_bytes_total",
	)
	if err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}

func TestTrafficCollector_ScrapeFailure(t *testing.T) {
	t.Parallel()

	coll, err := NewTrafficCollector(&stubClient{err: errors.New("network down")}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewTrafficCollector: %v", err)
	}

	count := testutil.CollectAndCount(coll, "xray_outbound_uplink_bytes_total")
	if count != 0 {
		t.Errorf("outbound metrics emitted on failure: got %d, want 0", count)
	}
}

func TestObservatoryCollector_Success(t *testing.T) {
	t.Parallel()

	snap := expvar.Snapshot{
		Observatory: map[string]expvar.ProbeResult{"proxy": {Alive: true, Delay: 234, LastSeenTime: 1000}},
	}
	coll, err := NewObservatoryCollector(&stubClient{snap: snap}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewObservatoryCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_observatory_alive Whether the observed outbound is reachable (1=alive, 0=dead).
# TYPE xray_observatory_alive gauge
xray_observatory_alive{tag="proxy"} 1
# HELP xray_observatory_probe_delay_milliseconds Probe round-trip time in milliseconds.
# TYPE xray_observatory_probe_delay_milliseconds gauge
xray_observatory_probe_delay_milliseconds{tag="proxy"} 234
`
	err = testutil.GatherAndCompare(reg, strings.NewReader(want),
		"xray_observatory_alive",
		"xray_observatory_probe_delay_milliseconds",
	)
	if err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}

func TestObservatoryCollector_Burst(t *testing.T) {
	t.Parallel()

	// Burst observatory: no last_seen/last_try, but a health_ping block.
	snap := expvar.Snapshot{
		Observatory: map[string]expvar.ProbeResult{
			"ams": {
				Alive:       true,
				Delay:       214,
				OutboundTag: "ams",
				HealthPing: &expvar.HealthPing{
					All:       4,
					Fail:      1,
					Deviation: 8710715,
					Average:   214854167,
					Max:       225698626,
					Min:       201426041,
				},
			},
		},
	}
	coll, err := NewObservatoryCollector(&stubClient{snap: snap}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewObservatoryCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_burst_observatory_samples Number of probes in the burst observatory's current sliding window.
# TYPE xray_burst_observatory_samples gauge
xray_burst_observatory_samples{tag="ams"} 4
# HELP xray_burst_observatory_failures Number of failed probes in the burst observatory's current sliding window.
# TYPE xray_burst_observatory_failures gauge
xray_burst_observatory_failures{tag="ams"} 1
# HELP xray_burst_observatory_rtt_average_seconds Average probe round-trip time over the sliding window.
# TYPE xray_burst_observatory_rtt_average_seconds gauge
xray_burst_observatory_rtt_average_seconds{tag="ams"} 0.214854167
# HELP xray_burst_observatory_rtt_max_seconds Maximum probe round-trip time over the sliding window.
# TYPE xray_burst_observatory_rtt_max_seconds gauge
xray_burst_observatory_rtt_max_seconds{tag="ams"} 0.225698626
# HELP xray_burst_observatory_rtt_min_seconds Minimum probe round-trip time over the sliding window.
# TYPE xray_burst_observatory_rtt_min_seconds gauge
xray_burst_observatory_rtt_min_seconds{tag="ams"} 0.201426041
# HELP xray_burst_observatory_rtt_deviation_seconds Standard deviation of probe round-trip time over the sliding window.
# TYPE xray_burst_observatory_rtt_deviation_seconds gauge
xray_burst_observatory_rtt_deviation_seconds{tag="ams"} 0.008710715
`
	err = testutil.GatherAndCompare(reg, strings.NewReader(want),
		"xray_burst_observatory_samples",
		"xray_burst_observatory_failures",
		"xray_burst_observatory_rtt_average_seconds",
		"xray_burst_observatory_rtt_max_seconds",
		"xray_burst_observatory_rtt_min_seconds",
		"xray_burst_observatory_rtt_deviation_seconds",
	)
	if err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}

	// last_seen/last_try must be absent when the burst observatory omits them.
	for _, name := range []string{
		"xray_observatory_probe_last_seen_timestamp_seconds",
		"xray_observatory_probe_last_try_timestamp_seconds",
	} {
		if n := testutil.CollectAndCount(coll, name); n != 0 {
			t.Errorf("%s: got %d samples, want 0 for burst observatory", name, n)
		}
	}
}

func TestMemoryCollector_Success(t *testing.T) {
	t.Parallel()

	snap := expvar.Snapshot{
		MemStats: expvar.MemStats{HeapInuse: 5000, NumGC: 7, PauseTotalNs: 2_000_000_000},
	}
	coll, err := NewMemoryCollector(&stubClient{snap: snap}, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewMemoryCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_memstats_gc_pause_seconds_total Cumulative GC stop-the-world pause time in seconds.
# TYPE xray_memstats_gc_pause_seconds_total counter
xray_memstats_gc_pause_seconds_total 2
# HELP xray_memstats_heap_inuse_bytes Bytes in in-use spans.
# TYPE xray_memstats_heap_inuse_bytes gauge
xray_memstats_heap_inuse_bytes 5000
# HELP xray_memstats_num_gc_total Number of completed GC cycles.
# TYPE xray_memstats_num_gc_total counter
xray_memstats_num_gc_total 7
`
	err = testutil.GatherAndCompare(reg, strings.NewReader(want),
		"xray_memstats_gc_pause_seconds_total",
		"xray_memstats_heap_inuse_bytes",
		"xray_memstats_num_gc_total",
	)
	if err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}

func TestScrapeCollector_Success(t *testing.T) {
	t.Parallel()

	cached := newCachedStub(t, expvar.Snapshot{}, nil)
	coll, err := NewScrapeCollector(cached, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewScrapeCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_scrape_success Whether the last xray /debug/vars scrape succeeded (1=success, 0=failure).
# TYPE xray_scrape_success gauge
xray_scrape_success 1
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want), "xray_scrape_success"); err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}

func TestScrapeCollector_Failure(t *testing.T) {
	t.Parallel()

	cached := newCachedStub(t, expvar.Snapshot{}, errors.New("network down"))
	coll, err := NewScrapeCollector(cached, WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("NewScrapeCollector: %v", err)
	}

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(coll); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_scrape_success Whether the last xray /debug/vars scrape succeeded (1=success, 0=failure).
# TYPE xray_scrape_success gauge
xray_scrape_success 0
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want), "xray_scrape_success"); err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}

func TestConstructors_NilClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func() (prometheus.Collector, error)
	}{
		{"scrape", func() (prometheus.Collector, error) { return NewScrapeCollector(nil) }},
		{"memory", func() (prometheus.Collector, error) { return NewMemoryCollector(nil) }},
		{"traffic", func() (prometheus.Collector, error) { return NewTrafficCollector(nil) }},
		{"observatory", func() (prometheus.Collector, error) { return NewObservatoryCollector(nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.fn(); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestWithLogger_NilNoOp(t *testing.T) {
	t.Parallel()

	// Passing nil to WithLogger must not crash — the option silently
	// keeps the default logger.
	_, err := NewMemoryCollector(&stubClient{}, WithLogger(nil))
	if err != nil {
		t.Fatalf("NewMemoryCollector with WithLogger(nil): %v", err)
	}
}

func TestBuildInfoCollector(t *testing.T) {
	t.Parallel()

	info := version.Info{
		Version:   "1.2.3",
		Revision:  "abc123",
		GoVersion: "go1.26.2",
		GOOS:      "linux",
		GOARCH:    "arm64",
	}
	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(NewBuildInfoCollector(info)); err != nil {
		t.Fatalf("Register: %v", err)
	}

	want := `
# HELP xray_exporter_build_info A metric with a constant '1' value labeled by version, revision, goversion, goos and goarch from which xray-exporter was built.
# TYPE xray_exporter_build_info gauge
xray_exporter_build_info{goarch="arm64",goos="linux",goversion="go1.26.2",revision="abc123",version="1.2.3"} 1
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want), "xray_exporter_build_info"); err != nil {
		t.Errorf("GatherAndCompare: %v", err)
	}
}
