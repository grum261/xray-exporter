// Package expvar fetches and parses xray's /debug/vars endpoint.
package expvar

import "time"

// Snapshot is a parsed /debug/vars response.
type Snapshot struct {
	Stats       Stats                  `json:"stats"`
	Observatory map[string]ProbeResult `json:"observatory"`
	MemStats    MemStats               `json:"memstats"`
	ScrapedAt   time.Time              `json:"-"`
}

// Stats groups inbound, outbound, and per-user traffic counters.
type Stats struct {
	Inbound  map[string]TrafficCounters `json:"inbound"`
	Outbound map[string]TrafficCounters `json:"outbound"`
	User     map[string]TrafficCounters `json:"user"`
}

// TrafficCounters is a directional byte counter pair.
type TrafficCounters struct {
	Uplink   uint64 `json:"uplink"`
	Downlink uint64 `json:"downlink"`
}

// ProbeResult mirrors xray's observatory OutboundStatus.
// Delay is in milliseconds; LastSeenTime and LastTryTime are unix seconds.
//
// The plain observatory populates LastSeenTime/LastTryTime; the burst
// observatory omits them and instead reports sliding-window statistics in
// HealthPing. HealthPing is nil for the plain observatory.
type ProbeResult struct {
	Alive        bool        `json:"alive"`
	Delay        int64       `json:"delay"`
	OutboundTag  string      `json:"outbound_tag"`
	LastSeenTime int64       `json:"last_seen_time"`
	LastTryTime  int64       `json:"last_try_time"`
	HealthPing   *HealthPing `json:"health_ping,omitempty"`
}

// HealthPing holds the burst observatory's sliding-window probe statistics.
// All/Fail are probe counts in the window; Deviation/Average/Max/Min are
// round-trip times in nanoseconds.
type HealthPing struct {
	All       int64 `json:"all"`
	Fail      int64 `json:"fail"`
	Deviation int64 `json:"deviation"`
	Average   int64 `json:"average"`
	Max       int64 `json:"max"`
	Min       int64 `json:"min"`
}

// MemStats is the subset of runtime.MemStats we surface from expvar.
// Field names match Go runtime PascalCase since that's how expvar serializes them.
type MemStats struct {
	Alloc         uint64  `json:"Alloc"`
	Sys           uint64  `json:"Sys"`
	HeapAlloc     uint64  `json:"HeapAlloc"`
	HeapInuse     uint64  `json:"HeapInuse"`
	HeapIdle      uint64  `json:"HeapIdle"`
	HeapReleased  uint64  `json:"HeapReleased"`
	NumGC         uint32  `json:"NumGC"`
	PauseTotalNs  uint64  `json:"PauseTotalNs"`
	GCCPUFraction float64 `json:"GCCPUFraction"`
	NextGC        uint64  `json:"NextGC"`
	Mallocs       uint64  `json:"Mallocs"`
	Frees         uint64  `json:"Frees"`
}
