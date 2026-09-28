# syntax=docker/dockerfile:1

# Builds xray-exporter from source:
#
#   docker build -t xray-exporter .
#
# Release images on ghcr.io are built by GoReleaser from goreleaser.Dockerfile
# instead, reusing the binaries it has already compiled.

FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
# .git is part of the build context on purpose: the Go toolchain reads the
# version, revision and commit time from it (see internal/version).
COPY . .
ARG TARGETOS TARGETARCH TARGETVARIANT
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -trimpath -ldflags="-s -w" -o /out/xray-exporter ./cmd/xray-exporter

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/xray-exporter /usr/bin/xray-exporter
EXPOSE 9356
ENTRYPOINT ["/usr/bin/xray-exporter"]
