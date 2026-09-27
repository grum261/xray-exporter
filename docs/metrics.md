# Metrics

Every metric the exporter emits is listed below, grouped by the collector that
produces it. Each collector can be turned off; see
[Configuration](configuration.md).

Each scrape of `/metrics` fetches Xray's `/debug/vars` once, and all
collectors share that one response. Nothing is cached between scrapes. If the
fetch fails, the Xray collectors emit nothing, so there are no stale series.
Only `xray_scrape_success` (then `0`) and `xray_scrape_duration_seconds`
are reported.

## Scrape health

Collector `scrape`.

| Metric | Type | Description |
|--------|------|-------------|
| `xray_scrape_success` | gauge | `1` if the last fetch of `/debug/vars` succeeded, `0` otherwise. |
| `xray_scrape_duration_seconds` | gauge | Duration of the last fetch of `/debug/vars`. |

## Traffic

Collector `traffic`. The values come from Xray's stats counters. They exist
only if `stats` and the matching `policy` switches are enabled in Xray (see
[Xray setup](xray-setup.md#stats-and-policy)).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `xray_inbound_uplink_bytes_total` | counter | `tag` | Bytes received from clients on the inbound. |
| `xray_inbound_downlink_bytes_total` | counter | `tag` | Bytes sent to clients on the inbound. |
| `xray_outbound_uplink_bytes_total` | counter | `tag` | Bytes sent through the outbound. |
| `xray_outbound_downlink_bytes_total` | counter | `tag` | Bytes received through the outbound. |
| `xray_user_uplink_bytes_total` | counter | `user` | Bytes uploaded by the user. |
| `xray_user_downlink_bytes_total` | counter | `user` | Bytes downloaded by the user. |

- `tag` is the inbound or outbound tag from your Xray config. `user` is the
  client's `email`. Clients without an `email` produce no user series.
- A series appears only after the first byte passes through its tag. A
  freshly restarted Xray with no traffic reports nothing yet.
- All counters reset to zero when Xray restarts. Query them with `rate()` or
  `increase()`, which handle counter resets. Don't graph the raw values.

## Observatory

Collector `observatory`. Xray has two observatory types, and each produces a
different set of metrics. See [Observatory](xray-setup.md#observatory-optional)
to choose one.

Both types emit these:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `xray_observatory_alive` | gauge | `tag` | `1` if the outbound's last probe succeeded, `0` if it failed. |
| `xray_observatory_probe_delay_milliseconds` | gauge | `tag` | Latency of the last probe, in milliseconds. |

Only the plain `observatory` emits these:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `xray_observatory_probe_last_seen_timestamp_seconds` | gauge | `tag` | Unix time of the last successful probe. |
| `xray_observatory_probe_last_try_timestamp_seconds` | gauge | `tag` | Unix time of the last probe attempt. |

Only `burstObservatory` emits these. The values cover the probes in its
sliding window (`health_ping`):

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `xray_burst_observatory_samples` | gauge | `tag` | Number of probes in the window. |
| `xray_burst_observatory_failures` | gauge | `tag` | Number of failed probes in the window. |
| `xray_burst_observatory_rtt_average_seconds` | gauge | `tag` | Mean round-trip time. |
| `xray_burst_observatory_rtt_min_seconds` | gauge | `tag` | Minimum round-trip time. |
| `xray_burst_observatory_rtt_max_seconds` | gauge | `tag` | Maximum round-trip time. |
| `xray_burst_observatory_rtt_deviation_seconds` | gauge | `tag` | Standard deviation of the round-trip time (jitter). |

`tag` is the outbound tag. Only outbounds that match the observatory's
`subjectSelector` are reported.

## Xray runtime

Collector `memory`. These are Go runtime memory stats of the **Xray process**,
taken from `memstats` in `/debug/vars`.

| Metric | Type | Description |
|--------|------|-------------|
| `xray_memstats_alloc_bytes` | gauge | Bytes of allocated heap objects. |
| `xray_memstats_sys_bytes` | gauge | Total bytes of memory obtained from the OS. |
| `xray_memstats_heap_alloc_bytes` | gauge | Bytes of allocated heap objects. |
| `xray_memstats_heap_inuse_bytes` | gauge | Bytes in in-use heap spans. |
| `xray_memstats_heap_idle_bytes` | gauge | Bytes in idle heap spans. |
| `xray_memstats_heap_released_bytes` | gauge | Bytes of physical memory returned to the OS. |
| `xray_memstats_next_gc_bytes` | gauge | Heap size at which the next GC cycle starts. |
| `xray_memstats_gc_cpu_fraction` | gauge | Fraction of CPU time spent in GC since Xray started. |
| `xray_memstats_num_gc_total` | counter | Completed GC cycles. |
| `xray_memstats_gc_pause_seconds_total` | counter | Total GC stop-the-world pause time. |
| `xray_memstats_mallocs_total` | counter | Heap objects allocated. |
| `xray_memstats_frees_total` | counter | Heap objects freed. |

## Exporter self-metrics

These are always on. They describe the **exporter** process, not Xray.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `xray_exporter_build_info` | gauge | `version`, `revision`, `goversion`, `goos`, `goarch` | Always `1`. The labels describe the build of the exporter. |
| `go_*` | — | — | Go runtime metrics of the exporter (goroutines, GC, heap). |
| `process_*` | — | — | Process metrics of the exporter (CPU, RSS, file descriptors, start time). |

Don't confuse `go_*` with `xray_memstats_*`. The first describes the
exporter's runtime, the second describes Xray's.

## Example queries

Throughput per outbound, in bits per second:

```promql
sum by (tag) (rate(xray_outbound_downlink_bytes_total[5m])) * 8
```

Traffic per user over the last 24 hours, upload and download combined:

```promql
sum by (user) (
    increase(xray_user_uplink_bytes_total[24h])
  + increase(xray_user_downlink_bytes_total[24h])
)
```

Outbounds that are down right now:

```promql
xray_observatory_alive == 0
```

Share of failed probes per outbound (burst observatory):

```promql
xray_burst_observatory_failures / xray_burst_observatory_samples
```

## Example alerting rules

```yaml
groups:
  - name: xray
    rules:
      - alert: XrayScrapeFailing
        expr: xray_scrape_success == 0
        for: 5m
        annotations:
          summary: "xray-exporter on {{ $labels.instance }} can't reach Xray's /debug/vars"

      - alert: XrayOutboundDown
        expr: xray_observatory_alive == 0
        for: 5m
        annotations:
          summary: "Outbound {{ $labels.tag }} on {{ $labels.instance }} fails its observatory probes"
```
