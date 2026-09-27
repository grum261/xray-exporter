# Configuration

xray-exporter is configured with command-line flags. Each flag also has an
environment variable. The flag wins if both are set. The exporter has no
config file.

## Options

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--web.listen-address` | `LISTEN_ADDR` | `:9356` | Address the HTTP server listens on. Use `127.0.0.1:9356` if Prometheus runs on the same host. |
| `--web.telemetry-path` | `TELEMETRY_PATH` | `/metrics` | Path of the metrics endpoint. |
| `--xray.endpoint` | `XRAY_ENDPOINT` | `http://127.0.0.1:11111/debug/vars` | URL of Xray's `/debug/vars`. See [Xray setup](xray-setup.md). |
| `--xray.timeout` | `XRAY_TIMEOUT` | `5s` | Timeout of a single request to Xray, as a [Go duration](https://pkg.go.dev/time#ParseDuration) (`500ms`, `5s`, `1m`). |
| `--log.level` | `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn` or `error`. An unknown value falls back to `info`. |
| `--collector.scrape` | `COLLECTOR_SCRAPE` | `true` | Emit `xray_scrape_*`. |
| `--collector.memory` | `COLLECTOR_MEMORY` | `true` | Emit `xray_memstats_*`. |
| `--collector.traffic` | `COLLECTOR_TRAFFIC` | `true` | Emit the inbound, outbound and user traffic counters. |
| `--collector.observatory` | `COLLECTOR_OBSERVATORY` | `true` | Emit `xray_observatory_*` and `xray_burst_observatory_*`. |
| `--version` | — | — | Print the version and exit. |
| `-h`, `--help` | — | — | Print usage and exit. |

Notes:

- To turn off a boolean flag, give it an explicit value:
  `--collector.memory=false`. Go's flag parser has no `--no-…` form.
- Environment variables are parsed the same way as flags. Booleans accept
  `true`/`false`, `1`/`0` and `t`/`f`. An empty variable counts as unset.
- A value that can't be parsed, such as `XRAY_TIMEOUT=5` without a unit,
  makes the exporter exit with code 2. This applies to both flags and
  environment variables.
- `xray-exporter -h` lists the environment variable next to each flag.
- The exporter always emits its own metrics (`xray_exporter_build_info`,
  `go_*`, `process_*`). They can't be turned off.
- Keep `--xray.timeout` below the Prometheus `scrape_timeout`, which defaults
  to 10s. Otherwise Prometheus gives up on the scrape before the exporter
  reports that Xray didn't respond.

Example:

```bash
xray-exporter \
  --web.listen-address=127.0.0.1:9356 \
  --xray.endpoint=http://127.0.0.1:11111/debug/vars \
  --xray.timeout=3s \
  --collector.memory=false
```

The same configuration with environment variables:

```bash
LISTEN_ADDR=127.0.0.1:9356 \
XRAY_ENDPOINT=http://127.0.0.1:11111/debug/vars \
XRAY_TIMEOUT=3s \
COLLECTOR_MEMORY=false \
xray-exporter
```

## HTTP endpoints

| Path | Description |
|------|-------------|
| `/metrics` (or `--web.telemetry-path`) | Metrics in the Prometheus text format. Each request fetches `/debug/vars` from Xray once. |
| `/-/healthy` | Returns `200 ok` while the exporter process is up. It doesn't check Xray; use `xray_scrape_success` for that. |
| `/` | A landing page with links to the two endpoints above. |

## Logging

Logs are JSON lines on stderr. At `info` the exporter logs startup, the
collectors it enabled and shutdown. At `warn` it also logs failed Xray
fetches. It stops gracefully on `SIGINT` and `SIGTERM`.

## Version

```console
$ xray-exporter --version
xray-exporter 1.0.0 (revision 3f2c…, committed 2026-09-28T00:00:00Z, go1.26.2 linux/arm64)
```

The same data is exposed as the `xray_exporter_build_info` metric. See
[Metrics](metrics.md#exporter-self-metrics).
