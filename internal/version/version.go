// Package version reports build metadata of the xray-exporter binary.
//
// Plain `go build` and `go install ...@vX.Y.Z` need no extra flags: the Go
// toolchain stamps the module version, VCS revision and commit time into the
// binary, and Get reads them back via runtime/debug.ReadBuildInfo.
//
// Release builds may still pin the values explicitly with -ldflags -X, which
// take precedence over the toolchain-provided ones:
//
//	go build -ldflags "\
//	  -X github.com/grum261/xray-exporter/internal/version.version=1.2.3 \
//	  -X github.com/grum261/xray-exporter/internal/version.revision=<sha> \
//	  -X github.com/grum261/xray-exporter/internal/version.commitDate=<rfc3339>" \
//	  ./cmd/xray-exporter
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Overridden at link time via -ldflags -X; see the package doc.
var (
	version    string
	revision   string
	commitDate string
)

const unknown = "unknown"

// Info is the resolved build metadata.
type Info struct {
	Version    string // semantic version without the "v" prefix, or "dev"
	Revision   string // full VCS commit hash, with "-dirty" for modified trees
	CommitDate string // commit time in RFC 3339
	GoVersion  string
	GOOS       string
	GOARCH     string
}

// Get returns the build metadata of the running binary.
func Get() Info {
	bi, _ := debug.ReadBuildInfo()
	return resolve(version, revision, commitDate, bi)
}

// String formats Info for `xray-exporter --version`.
func (i Info) String() string {
	return fmt.Sprintf("xray-exporter %s (revision %s, committed %s, %s %s/%s)",
		i.Version, i.Revision, i.CommitDate, i.GoVersion, i.GOOS, i.GOARCH)
}

// resolve merges link-time values with the toolchain's build info. Explicit
// link-time values win; bi may be nil when build info is unavailable.
func resolve(ver, rev, date string, bi *debug.BuildInfo) Info {
	info := Info{
		Version:    ver,
		Revision:   rev,
		CommitDate: date,
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	if bi != nil {
		if info.Version == "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}
		info.fillFromVCS(bi.Settings)
	}

	info.Version = strings.TrimPrefix(info.Version, "v")
	if info.Version == "" {
		info.Version = "dev"
	}
	if info.Revision == "" {
		info.Revision = unknown
	}
	if info.CommitDate == "" {
		info.CommitDate = unknown
	}
	return info
}

func (i *Info) fillFromVCS(settings []debug.BuildSetting) {
	var vcsRevision, vcsTime string
	var modified bool
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			vcsRevision = s.Value
		case "vcs.time":
			vcsTime = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}

	if i.Revision == "" && vcsRevision != "" {
		i.Revision = vcsRevision
		if modified {
			i.Revision += "-dirty"
		}
	}
	if i.CommitDate == "" {
		i.CommitDate = vcsTime
	}
}
