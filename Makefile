.DEFAULT_GOAL := build

BINARY := xray-exporter
PKG    := ./cmd/xray-exporter

# Version metadata (xray_exporter_build_info, --version) is stamped by the Go
# toolchain from git: the tag or pseudo-version, the commit and its time. See
# internal/version. Pass VERSION to override it, e.g. `make build VERSION=1.2.3`.
VERSION_VAR := github.com/grum261/xray-exporter/internal/version.version

# Trim paths for reproducibility, strip debug info to slim the binary.
LDFLAGS := -s -w
ifdef VERSION
LDFLAGS += -X $(VERSION_VAR)=$(VERSION)
endif
BUILDFLAGS := -trimpath -ldflags="$(LDFLAGS)"

# Builds for the host platform. Cross-compile with GOOS/GOARCH, e.g.
# `make build GOOS=linux GOARCH=arm64`; the binary is static (no CGO).
.PHONY: build
build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -o bin/$(BINARY) $(PKG)

# Race detector is on by default. Disable it with `make test RACE=` on hosts
# where ThreadSanitizer is unsupported.
RACE ?= -race

.PHONY: test
test:
	go test $(RACE) -count=1 ./...

.PHONY: test-cover
test-cover:
	go test $(RACE) -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

.PHONY: lint
lint:
	golangci-lint run ./...

# Builds every release artifact locally into dist/ without publishing.
.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --clean

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: run
run: build
	./bin/$(BINARY) --log.level=debug

.PHONY: clean
clean:
	rm -rf bin/ dist/ coverage.out
