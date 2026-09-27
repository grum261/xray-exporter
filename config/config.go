// Package config parses the xray-exporter binary's CLI flags and
// environment variables into a Config struct. The package is the
// boundary between OS-level configuration (argv, env) and the typed
// options consumed by the rest of the codebase.
//
// Empty / zero values in Config indicate "not set" — downstream
// packages (server, expvar, collectors) keep their own defaults and
// only override them when the option is explicitly provided.
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grum261/xray-exporter/logger"
	loggerzap "github.com/grum261/xray-exporter/logger/zap"
)

const defaultLogLevel = "info"

// Config holds the parsed flag/env configuration.
type Config struct {
	ListenAddr    string
	MetricsPath   string
	XrayEndpoint  string
	ScrapeTimeout time.Duration
	LogLevel      string
	ShowVersion   bool

	EnableScrape      bool
	EnableMemory      bool
	EnableTraffic     bool
	EnableObservatory bool
}

// Parse builds a Config from argv-style args. Pass os.Args[1:] from main.
// Returns flag.ErrHelp on -h / -help so callers can exit 0 cleanly.
// Uses flag.NewFlagSet (not the global flag.CommandLine) so it can be
// invoked from tests with arbitrary argument vectors.
func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("xray-exporter", flag.ContinueOnError)

	fs.StringVar(&cfg.ListenAddr, "web.listen-address", "", "Address on which to expose metrics (default :9356).")
	fs.StringVar(&cfg.MetricsPath, "web.telemetry-path", "", "Path under which to expose metrics (default /metrics).")
	fs.StringVar(&cfg.XrayEndpoint, "xray.endpoint", "", "xray /debug/vars URL (default http://127.0.0.1:11111/debug/vars).")
	fs.DurationVar(&cfg.ScrapeTimeout, "xray.timeout", 0, "Timeout for a single xray scrape (default 5s).")
	fs.StringVar(&cfg.LogLevel, "log.level", defaultLogLevel, "Log level: debug, info, warn, error.")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Print version information and exit.")

	fs.BoolVar(&cfg.EnableScrape, "collector.scrape", true, "Enable scrape health collector.")
	fs.BoolVar(&cfg.EnableMemory, "collector.memory", true, "Enable Go runtime memstats collector.")
	fs.BoolVar(&cfg.EnableTraffic, "collector.traffic", true, "Enable inbound/outbound/user traffic collector.")
	fs.BoolVar(&cfg.EnableObservatory, "collector.observatory", true, "Enable observatory probe results collector.")

	if err := applyEnv(fs); err != nil {
		return nil, err
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return cfg, nil
}

// envVars maps flag names to the environment variables that can set them.
var envVars = map[string]string{
	"web.listen-address":    "LISTEN_ADDR",
	"web.telemetry-path":    "TELEMETRY_PATH",
	"xray.endpoint":         "XRAY_ENDPOINT",
	"xray.timeout":          "XRAY_TIMEOUT",
	"log.level":             "LOG_LEVEL",
	"collector.scrape":      "COLLECTOR_SCRAPE",
	"collector.memory":      "COLLECTOR_MEMORY",
	"collector.traffic":     "COLLECTOR_TRAFFIC",
	"collector.observatory": "COLLECTOR_OBSERVATORY",
}

// applyEnv sets flags from their environment variables. It must run before
// fs.Parse so that command-line flags take precedence. Values go through the
// flag's own parser, so a malformed variable is an error, same as a malformed
// flag. It also appends the variable name to each flag's usage text.
func applyEnv(fs *flag.FlagSet) error {
	for name, key := range envVars {
		f := fs.Lookup(name)
		if f == nil {
			return fmt.Errorf("env %s: no flag -%s", key, name)
		}
		f.Usage += " [env " + key + "]"

		v, ok := os.LookupEnv(key)
		if !ok || v == "" {
			continue
		}
		if err := fs.Set(name, v); err != nil {
			return fmt.Errorf("env %s=%q: %w", key, v, err)
		}
	}
	return nil
}

// NewLogger builds a JSON zap logger at the level configured on c,
// wrapped in the zap adapter so it satisfies logger.Logger. Unknown
// levels fall back to info.
func (c *Config) NewLogger() logger.Logger {
	var lvl logger.Level
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		lvl = logger.LevelDebug
	case "warn", "warning":
		lvl = logger.LevelWarn
	case "error":
		lvl = logger.LevelError
	default:
		lvl = logger.LevelInfo
	}
	return loggerzap.NewAtLevel(lvl)
}
