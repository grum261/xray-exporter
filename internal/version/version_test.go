package version

import (
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	vcs := &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc123"},
			{Key: "vcs.time", Value: "2026-09-27T12:00:00Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}

	tests := []struct {
		name             string
		ver, rev, date   string
		bi               *debug.BuildInfo
		wantVer, wantRev string
		wantDate         string
	}{
		{
			name:    "no build info",
			wantVer: "dev", wantRev: unknown, wantDate: unknown,
		},
		{
			name:    "local build outside VCS",
			bi:      &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantVer: "dev", wantRev: unknown, wantDate: unknown,
		},
		{
			name:    "toolchain stamping",
			bi:      vcs,
			wantVer: "1.2.3", wantRev: "abc123", wantDate: "2026-09-27T12:00:00Z",
		},
		{
			name: "dirty tree",
			bi: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.4-0.20260927120000-abc123+dirty"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVer: "1.2.4-0.20260927120000-abc123+dirty", wantRev: "abc123-dirty", wantDate: unknown,
		},
		{
			name: "ldflags win over toolchain",
			ver:  "v2.0.0", rev: "def456", date: "2026-10-01T00:00:00Z",
			bi:      vcs,
			wantVer: "2.0.0", wantRev: "def456", wantDate: "2026-10-01T00:00:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolve(tt.ver, tt.rev, tt.date, tt.bi)
			if got.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVer)
			}
			if got.Revision != tt.wantRev {
				t.Errorf("Revision = %q, want %q", got.Revision, tt.wantRev)
			}
			if got.CommitDate != tt.wantDate {
				t.Errorf("CommitDate = %q, want %q", got.CommitDate, tt.wantDate)
			}
		})
	}
}
