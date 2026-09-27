# Deployment

This guide runs xray-exporter as a systemd service on the host where Xray
runs. It then adds the exporter to Prometheus and imports the Grafana
dashboard. First, set up Xray as described in [Xray setup](xray-setup.md).

## Files

| File | Installed to |
|------|--------------|
| `xray-exporter` binary | `/usr/local/bin/xray-exporter` (mode `0755`) |
| [`deploy/xray-exporter.service`](../deploy/xray-exporter.service) | `/etc/systemd/system/xray-exporter.service` (mode `0644`) |
| [`deploy/prometheus-scrape.yml`](../deploy/prometheus-scrape.yml) | merged into your `prometheus.yml` |
| [`deploy/xray-grafana-dashboard.json`](../deploy/xray-grafana-dashboard.json) | imported in the Grafana UI |

Each release archive contains the binary and the whole `deploy/` directory.

## 1. Install the binary and the unit

Download the archive for your platform from the
[releases page](https://github.com/grum261/xray-exporter/releases/latest):

```bash
VERSION=1.0.0      # the release you're installing
ARCH=amd64         # amd64, arm64, armv7 or armv6
curl -fsSLO "https://github.com/grum261/xray-exporter/releases/download/v${VERSION}/xray-exporter-${VERSION}.linux-${ARCH}.tar.gz"
curl -fsSLO "https://github.com/grum261/xray-exporter/releases/download/v${VERSION}/sha256sums.txt"
sha256sum --check --ignore-missing sha256sums.txt

tar -xzf "xray-exporter-${VERSION}.linux-${ARCH}.tar.gz"
cd "xray-exporter-${VERSION}.linux-${ARCH}"
```

Install the binary and the unit, then start the service:

```bash
sudo install -m 0755 xray-exporter /usr/local/bin/xray-exporter
sudo install -m 0644 deploy/xray-exporter.service /etc/systemd/system/xray-exporter.service
sudo systemctl daemon-reload
sudo systemctl enable --now xray-exporter
```

Check that it's running:

```bash
systemctl status xray-exporter
curl -s http://127.0.0.1:9356/metrics | grep '^xray_scrape_success'   # → xray_scrape_success 1
```

About the unit:

- It runs the exporter as `nobody:nogroup` with a strict sandbox
  (`ProtectSystem=strict`, a system call filter, and no privileges). It also
  limits the process to `MemoryMax=64M` and `CPUQuota=20%`. On distributions
  without a `nogroup` group, such as Fedora or RHEL, change `Group=` to
  `nobody`.
- It listens on `127.0.0.1:9356` only. If Prometheus runs on another host,
  change `--web.listen-address` and restrict access with a firewall. The
  exporter has no TLS or authentication of its own.
- To change flags, override the unit with `sudo systemctl edit
  xray-exporter`. Don't edit the installed file, because an upgrade
  overwrites it.

### Upgrading

Replace the binary and restart the service:

```bash
sudo install -m 0755 xray-exporter /usr/local/bin/xray-exporter
sudo systemctl restart xray-exporter
```

## 2. Add the Prometheus scrape job

Add the job from [`deploy/prometheus-scrape.yml`](../deploy/prometheus-scrape.yml)
under `scrape_configs` in `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "xray"
    static_configs:
      - targets: ["127.0.0.1:9356"]
```

Then make Prometheus reload its configuration. Sending `SIGHUP` works with
any packaging and doesn't restart Prometheus:

```bash
sudo systemctl kill -s HUP prometheus
```

`systemctl reload prometheus` works only if the unit defines `ExecReload`.
`POST /-/reload` works only if Prometheus runs with `--web.enable-lifecycle`.
When Prometheus has reloaded, the `xray` target is `UP` on the
**Status → Targets** page.

## 3. Import the Grafana dashboard

1. In Grafana, go to **Dashboards → New → Import**.
2. Upload [`deploy/xray-grafana-dashboard.json`](../deploy/xray-grafana-dashboard.json).
3. Save, then select your Prometheus data source in the **Datasource**
   selector at the top of the dashboard.

The dashboard has these sections: scrape status, throughput and traffic
share per inbound and outbound, observatory latency and uptime, the
exporter's own health, and Xray's memory stats. Use the **Instance** and
**Outbound** selectors at the top to pick hosts and outbounds. For file-based
provisioning, put the JSON file into a directory that a Grafana dashboard
provider reads.

## Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| `xray_scrape_success` is always `0` | The exporter can't reach Xray. | Check `journalctl -u xray-exporter`, then run `curl http://127.0.0.1:11111/debug/vars` on the host. Compare the URL with `--xray.endpoint`. |
| No `xray_inbound_*` or `xray_outbound_*` series | `stats` or `policy` is missing, or there has been no traffic yet. | See [Stats and policy](xray-setup.md#stats-and-policy). Check `jq '.stats'` on `/debug/vars`. |
| No `xray_user_*` series | The clients have no `email`, or their `level` has no `statsUser*` policy. | See [Stats and policy](xray-setup.md#stats-and-policy). |
| No observatory metrics | No observatory is configured, or the first probe hasn't run yet. | Wait one probe interval. Check `jq '.observatory'` on `/debug/vars`. |
| Some observatory series are missing | The two observatory types emit different metrics. | See [Observatory](xray-setup.md#observatory-optional). |
| Counters suddenly drop to zero | Xray restarted, which resets its counters. | Nothing to fix. `rate()` and `increase()` handle resets. |
| The Prometheus target is `DOWN` | Wrong target address, or a firewall blocks the port. | Run `curl` against the target address from the Prometheus host. |
| The exporter uses more than a few tens of MB of RSS | This is unexpected. | Please [open an issue](https://github.com/grum261/xray-exporter/issues). As a workaround, raise `MemoryMax` in the unit. |

## Resource footprint

Measured on a 4-core arm64 host with one inbound, four outbounds and the
observatory enabled:

| | RSS | CPU |
|---|---|---|
| Idle (no scrapes) | ~7 MB | ~0% |
| Scraped every 15s | ~10 MB | < 0.3% of one core |

The exporter does no background work. It does everything while handling a
`/metrics` request.
