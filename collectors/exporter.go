package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/grum261/xray-exporter/internal/version"
)

// The collectors in this file describe the xray-exporter process itself rather
// than the xray instance it scrapes. They wrap client_golang's stock
// collectors so the composition root can build them through the same
// collectors.New* surface as the xray collectors, instead of reaching into
// prometheus packages directly.

// NewGoCollector returns a collector exposing the exporter's own Go runtime
// metrics (go_*): goroutines, GC, heap, and friends.
func NewGoCollector() prometheus.Collector {
	return promcollectors.NewGoCollector()
}

// NewProcessCollector returns a collector exposing the exporter's own process
// metrics (process_*): CPU, resident memory, open file descriptors, start time.
func NewProcessCollector() prometheus.Collector {
	return promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{})
}

// NewBuildInfoCollector returns a collector exposing xray_exporter_build_info,
// a constant 1 gauge labeled with the given build metadata.
func NewBuildInfoCollector(info version.Info) prometheus.Collector {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "exporter",
		Name:      "build_info",
		Help:      "A metric with a constant '1' value labeled by version, revision, goversion, goos and goarch from which xray-exporter was built.",
		ConstLabels: prometheus.Labels{
			"version":   info.Version,
			"revision":  info.Revision,
			"goversion": info.GoVersion,
			"goos":      info.GOOS,
			"goarch":    info.GOARCH,
		},
	})
	g.Set(1)
	return g
}
