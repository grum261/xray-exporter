# Release image, built by GoReleaser (dockers_v2 in .goreleaser.yaml) from the
# binaries it has already compiled. To build from source, use Dockerfile.

FROM gcr.io/distroless/static-debian13:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/xray-exporter /usr/bin/xray-exporter
EXPOSE 9356
ENTRYPOINT ["/usr/bin/xray-exporter"]
