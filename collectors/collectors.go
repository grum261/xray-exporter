// Package collectors implements per-domain prometheus.Collector
// implementations that scrape xray's /debug/vars endpoint on demand.
//
// Each collector is constructed via its New* function and returns a
// prometheus.Collector. Collectors share an *expvar.CachedClient so
// that concurrent Collect calls coalesce into a single upstream HTTP
// request per scrape cycle.
package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "xray"

	tagLabel  = "tag"
	userLabel = "user"
)

// newDesc builds a prometheus.Desc with the xray namespace.
func newDesc(subsystem, name, help string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystem, name),
		help,
		labels,
		nil,
	)
}
