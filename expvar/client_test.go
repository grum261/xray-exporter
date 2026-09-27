package expvar

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		body              string
		wantErr           bool
		wantInboundTags   int
		wantOutboundTags  int
		wantObsTags       int
		wantProxyDelay    int64
		wantProxyAlive    bool
		wantHeapInuse     uint64
		wantSpecificCheck func(t *testing.T, s Snapshot)
	}{
		{
			name: "full snapshot",
			body: `{
				"cmdline": ["/usr/local/bin/xray"],
				"memstats": {
					"Alloc": 12345678,
					"Sys": 50000000,
					"HeapAlloc": 12000000,
					"HeapInuse": 15000000,
					"HeapIdle": 5000000,
					"HeapReleased": 2000000,
					"NumGC": 42,
					"PauseTotalNs": 5000000,
					"GCCPUFraction": 0.001,
					"NextGC": 16000000,
					"Mallocs": 1000000,
					"Frees": 900000
				},
				"observatory": {
					"proxy": {
						"alive": true, "delay": 234,
						"outbound_tag": "proxy",
						"last_seen_time": 1747200000,
						"last_try_time": 1747200030
					}
				},
				"stats": {
					"inbound": {
						"transparent": {"uplink": 5000000, "downlink": 50000000},
						"dns-in": {"uplink": 1000, "downlink": 5000}
					},
					"outbound": {
						"proxy": {"uplink": 4500000, "downlink": 45000000},
						"direct": {"uplink": 500000, "downlink": 5000000},
						"block": {"uplink": 0, "downlink": 0}
					}
				}
			}`,
			wantInboundTags:  2,
			wantOutboundTags: 3,
			wantObsTags:      1,
			wantProxyDelay:   234,
			wantProxyAlive:   true,
			wantHeapInuse:    15000000,
			wantSpecificCheck: func(t *testing.T, s Snapshot) {
				t.Helper()
				if got := s.Stats.Outbound["proxy"].Downlink; got != 45000000 {
					t.Errorf("proxy downlink = %d, want 45000000", got)
				}
				if got := s.Stats.Inbound["transparent"].Uplink; got != 5000000 {
					t.Errorf("transparent uplink = %d, want 5000000", got)
				}
			},
		},
		{
			name: "missing observatory is fine",
			body: `{
				"memstats": {"HeapInuse": 1000},
				"stats": {"inbound": {}, "outbound": {}}
			}`,
			wantInboundTags:  0,
			wantOutboundTags: 0,
			wantObsTags:      0,
			wantHeapInuse:    1000,
		},
		{
			name:    "malformed json",
			body:    `{not json at all`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			snap, err := Parse([]byte(tt.body))

			if tt.wantErr {
				if err == nil {
					t.Fatal("Parse() expected error, got nil")
				}
				if !errors.Is(err, ErrScrapeFailed) {
					t.Errorf("Parse() error = %v, want wrapped ErrScrapeFailed", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if got := len(snap.Stats.Inbound); got != tt.wantInboundTags {
				t.Errorf("inbound tags = %d, want %d", got, tt.wantInboundTags)
			}
			if got := len(snap.Stats.Outbound); got != tt.wantOutboundTags {
				t.Errorf("outbound tags = %d, want %d", got, tt.wantOutboundTags)
			}
			if got := len(snap.Observatory); got != tt.wantObsTags {
				t.Errorf("observatory tags = %d, want %d", got, tt.wantObsTags)
			}
			if tt.wantObsTags > 0 {
				if got := snap.Observatory["proxy"].Delay; got != tt.wantProxyDelay {
					t.Errorf("proxy delay = %d, want %d", got, tt.wantProxyDelay)
				}
				if got := snap.Observatory["proxy"].Alive; got != tt.wantProxyAlive {
					t.Errorf("proxy alive = %v, want %v", got, tt.wantProxyAlive)
				}
			}
			if got := snap.MemStats.HeapInuse; got != tt.wantHeapInuse {
				t.Errorf("HeapInuse = %d, want %d", got, tt.wantHeapInuse)
			}

			if tt.wantSpecificCheck != nil {
				tt.wantSpecificCheck(t, snap)
			}
		})
	}
}
