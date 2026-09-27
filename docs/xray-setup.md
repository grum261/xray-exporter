# Xray setup

The exporter reads Xray's expvar endpoint, `/debug/vars`. Xray serves it only
when the `metrics` object is configured. Traffic counters appear there only
when `stats` and `policy` are also set.

All keys on this page are **top-level** keys of the Xray config. They sit
next to `inbounds`, `outbounds` and `routing`, not inside them.

## Metrics endpoint

```json
{
  "metrics": { "tag": "metrics", "listen": "127.0.0.1:11111" }
}
```

With `listen` set, Xray starts its own HTTP server that serves `/debug/vars`.
No extra inbound or routing rule is needed. Keep it on `127.0.0.1`. The
endpoint has no authentication, and it reveals your tags and user emails.

Upstream reference:

- [Metrics config reference](https://xtls.github.io/en/config/metrics.html): the `tag` and `listen` fields.
- [`infra/conf/metrics.go`](https://github.com/XTLS/Xray-core/blob/main/infra/conf/metrics.go): the config schema. Either `tag` or `listen` is required.
- [`app/metrics/metrics.go`](https://github.com/XTLS/Xray-core/blob/main/app/metrics/metrics.go): `listen` starts a dedicated HTTP server.

<details>
<summary>Older Xray without <code>metrics.listen</code></summary>

Older builds can expose `/debug/vars` only through a dokodemo-door inbound
that is routed to the metrics tag. Use this instead of the `listen` block
above:

```json
{
  "metrics": { "tag": "metrics_out" },
  "inbounds": [
    {
      "tag": "metrics_in",
      "listen": "127.0.0.1",
      "port": 11111,
      "protocol": "dokodemo-door",
      "settings": { "address": "127.0.0.1" }
    }
  ],
  "routing": {
    "rules": [
      { "type": "field", "inboundTag": ["metrics_in"], "outboundTag": "metrics_out" }
    ]
  }
}
```

Merge the inbound and the rule into your existing `inbounds` and
`routing.rules`. The rule must come **first**.

</details>

## Stats and policy

Without these settings, `/debug/vars` has an empty `stats` object, and no
`xray_inbound_*`, `xray_outbound_*` or `xray_user_*` series are exported.

```json
{
  "stats": {},
  "policy": {
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true,
      "statsOutboundUplink": true,
      "statsOutboundDownlink": true
    },
    "levels": {
      "0": { "statsUserUplink": true, "statsUserDownlink": true }
    }
  }
}
```

- The `policy.system.stats*` switches enable the counters per inbound and
  outbound tag.
- `policy.levels.<N>.statsUser*` enables the counters per user. They count
  only clients that have an `email` and whose `level` equals `N`.

## Observatory (optional)

Xray has two observatory types. Each has its own top-level key and reports
different data. Choose the one whose metrics you need; you normally configure
only one. `subjectSelector` matches outbound tags **by prefix**.

### `observatory`: latest probe, with timestamps

```json
{
  "observatory": {
    "subjectSelector": ["proxy"],
    "probeURL": "https://www.gstatic.com/generate_204",
    "probeInterval": "30s"
  }
}
```

Metrics: `xray_observatory_alive`, `xray_observatory_probe_delay_milliseconds`,
`xray_observatory_probe_last_seen_timestamp_seconds`,
`xray_observatory_probe_last_try_timestamp_seconds`.

### `burstObservatory`: statistics over a sliding window

```json
{
  "burstObservatory": {
    "subjectSelector": ["proxy"],
    "pingConfig": {
      "destination": "https://www.gstatic.com/generate_204",
      "interval": "30s",
      "timeout": "10s",
      "sampling": 3
    }
  }
}
```

Metrics: `xray_observatory_alive`, `xray_observatory_probe_delay_milliseconds`,
`xray_burst_observatory_samples`, `xray_burst_observatory_failures` and
`xray_burst_observatory_rtt_{average,min,max,deviation}_seconds`.

See [Metrics](metrics.md#observatory) for what each series means.

## Verify

Check and apply the config:

```bash
xray run -test -config /usr/local/etc/xray/config.json   # "Configuration OK."
sudo systemctl restart xray
```

Check that the endpoint responds:

```console
$ curl -s http://127.0.0.1:11111/debug/vars | jq 'keys'
["cmdline", "memstats", "observatory", "stats"]
```

Check that the counters are filled in. Send some traffic through the proxy
first:

```console
$ curl -s http://127.0.0.1:11111/debug/vars | jq '.stats'
{ "inbound": { "<tag>": { "uplink": …, "downlink": … } }, "outbound": { … }, "user": { … } }
```

Check the observatory. The first probe runs about one `probeInterval` after
Xray starts.

```console
$ curl -s http://127.0.0.1:11111/debug/vars | jq '.observatory'
# observatory:
{ "proxy-a": { "alive": true, "delay": 211, "last_seen_time": 1747200000, "last_try_time": 1747200030 } }
# burstObservatory:
{ "proxy-a": { "alive": true, "delay": 199, "health_ping": { "all": 3, "fail": 0, "average": 199590283, … } } }
```
