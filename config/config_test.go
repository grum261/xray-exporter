package config

import (
	"testing"
	"time"
)

// Tests here use t.Setenv, so none of them can run in parallel.

func TestParse_Defaults(t *testing.T) {
	// Also guards envVars: an entry naming a missing flag fails Parse.
	cfg, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.ListenAddr != "" || cfg.XrayEndpoint != "" || cfg.ScrapeTimeout != 0 {
		t.Errorf("unset options must stay zero so downstream defaults apply, got %+v", cfg)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, defaultLogLevel)
	}
	if !cfg.EnableScrape || !cfg.EnableMemory || !cfg.EnableTraffic || !cfg.EnableObservatory {
		t.Errorf("all collectors must be enabled by default, got %+v", cfg)
	}
}

func TestParse_Env(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "127.0.0.1:9000")
	t.Setenv("XRAY_TIMEOUT", "3s")
	t.Setenv("COLLECTOR_MEMORY", "false")
	t.Setenv("LOG_LEVEL", "") // empty means unset

	cfg, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9000" {
		t.Errorf("ListenAddr = %q, want %q", cfg.ListenAddr, "127.0.0.1:9000")
	}
	if cfg.ScrapeTimeout != 3*time.Second {
		t.Errorf("ScrapeTimeout = %v, want 3s", cfg.ScrapeTimeout)
	}
	if cfg.EnableMemory {
		t.Error("EnableMemory = true, want false")
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, defaultLogLevel)
	}
}

func TestParse_FlagOverridesEnv(t *testing.T) {
	t.Setenv("XRAY_TIMEOUT", "3s")
	t.Setenv("COLLECTOR_TRAFFIC", "false")

	cfg, err := Parse([]string{"--xray.timeout=7s", "--collector.traffic=true"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.ScrapeTimeout != 7*time.Second {
		t.Errorf("ScrapeTimeout = %v, want 7s", cfg.ScrapeTimeout)
	}
	if !cfg.EnableTraffic {
		t.Error("EnableTraffic = false, want true")
	}
}

func TestParse_InvalidEnv(t *testing.T) {
	tests := []struct{ key, value string }{
		{"XRAY_TIMEOUT", "5"},
		{"COLLECTOR_SCRAPE", "maybe"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			if _, err := Parse(nil); err == nil {
				t.Errorf("%s=%q: want error, got nil", tt.key, tt.value)
			}
		})
	}
}
