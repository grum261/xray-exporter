# Contributing

Bug reports, feature requests and pull requests are welcome. Please open an
issue before starting a large change, so we can agree on the approach first.

## Development

Requirements:

- Go (the version is set in [`go.mod`](go.mod))
- [golangci-lint](https://golangci-lint.run/) v2
- [GoReleaser](https://goreleaser.com/) v2, needed only for `make snapshot`
- Docker with buildx, needed only for `make docker` and `make snapshot`

| Command | What it does |
|---------|--------------|
| `make build` | Builds `bin/xray-exporter` for the host. Cross-compile with `make build GOOS=linux GOARCH=arm64`. |
| `make run` | Builds the binary and runs it with debug logging. |
| `make test` | Runs the unit tests with the race detector. Use `make test RACE=` to turn the race detector off. |
| `make test-cover` | Runs the tests and prints a coverage report. |
| `make lint` | Runs golangci-lint with [`.golangci.yml`](.golangci.yml). |
| `make docker` | Builds the container image `xray-exporter:dev` from source ([`Dockerfile`](Dockerfile)). |
| `make snapshot` | Builds all release archives into `dist/` and the release images for each platform, without publishing them. |

CI runs `lint`, `test` and `snapshot` on every push and pull request.

## Project layout

```
cmd/xray-exporter/      entry point: wiring, signal handling
config/                 flags and environment variables → Config
expvar/                 /debug/vars HTTP client, JSON types, and CachedClient
                        (merges concurrent fetches into one request per scrape)
collectors/             one prometheus.Collector per metric group
server/                 HTTP server: /metrics, /-/healthy, landing page
logger/                 logging interface, with a zap implementation in logger/zap
internal/singleflight/  generic singleflight used by CachedClient
internal/version/       build metadata (version, revision, commit date)
option/                 generic functional-option helper
deploy/                 systemd unit, Prometheus scrape job, Grafana dashboard
Dockerfile              container image built from source
goreleaser.Dockerfile   release image built from the GoReleaser binaries
docs/                   user documentation
```

To add a metric, change the collector for its group in `collectors/` and add
a case to `collectors/collectors_test.go`. Then document the metric in
[`docs/metrics.md`](docs/metrics.md).

## Commit messages

The project uses [Conventional Commits](https://www.conventionalcommits.org/):
`feat(collectors): …`, `fix: …`, `docs: …`, `ci: …`. Release notes are
generated from these messages. `feat` and `fix` commits get their own
sections, and `docs`, `test`, `ci` and `chore` commits are left out.

## Versioning and releases

The project uses [Semantic Versioning](https://semver.org/). Metric names and
labels are part of the public interface. Renaming or removing a metric or
label is a breaking change.

Version metadata comes from git. The Go toolchain stamps the tag, commit and
commit time into every binary, and
[`internal/version`](internal/version/version.go) reads them. They show up in
`xray-exporter --version` and in the `xray_exporter_build_info` metric. A
local `go build` on a tagged commit therefore reports that tag. Other commits
report a pseudo-version.

To cut a release, push a tag to `main`:

```bash
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

The [release workflow](.github/workflows/release.yml) runs the tests. Then
GoReleaser builds archives for every platform, with checksums, and publishes
a GitHub release with the changelog. It also pushes a multi-arch image to
`ghcr.io/grum261/xray-exporter`, tagged `X.Y.Z`, `X.Y`, `X` and `latest`. The
image is built from [`goreleaser.Dockerfile`](goreleaser.Dockerfile) and
reuses the release binaries.
