package zap

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/grum261/xray-exporter/logger"
)

// newObserved builds an adapter writing into an in-memory observer at the
// given enabled level.
func newObserved(level zapcore.Level) (logger.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	return New(zap.New(core)), logs
}

func TestZap_KeyValuePairs(t *testing.T) {
	l, logs := newObserved(zapcore.DebugLevel)

	l.Info("scrape ok", "tag", "transparent", "count", 42)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Message != "scrape ok" {
		t.Errorf("message = %q, want %q", e.Message, "scrape ok")
	}
	if e.Level != zapcore.InfoLevel {
		t.Errorf("level = %v, want Info", e.Level)
	}
	ctx := e.ContextMap()
	if got := ctx["tag"]; got != "transparent" {
		t.Errorf(`field tag = %v, want "transparent"`, got)
	}
	if got := ctx["count"]; got != int64(42) {
		t.Errorf("field count = %v (%T), want int64(42)", got, got)
	}
}

func TestZap_MalformedArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []any
		wantKey string
	}{
		{"trailing key without value", []any{"orphan"}, badKey},
		{"non-string key", []any{7, "value"}, badKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, logs := newObserved(zapcore.DebugLevel)
			l.Info("msg", tt.args...)

			ctx := logs.All()[0].ContextMap()
			if _, ok := ctx[tt.wantKey]; !ok {
				t.Errorf("expected a %q field, got context %v", tt.wantKey, ctx)
			}
		})
	}
}

func TestZap_NoArgs(t *testing.T) {
	l, logs := newObserved(zapcore.DebugLevel)
	l.Info("no fields")

	if n := logs.All()[0].Context; len(n) != 0 {
		t.Errorf("want no fields, got %v", n)
	}
}

func TestZap_LevelRouting(t *testing.T) {
	l, logs := newObserved(zapcore.DebugLevel)

	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")

	want := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}
	entries := logs.All()
	if len(entries) != len(want) {
		t.Fatalf("want %d entries, got %d", len(want), len(entries))
	}
	for i, lvl := range want {
		if entries[i].Level != lvl {
			t.Errorf("entry %d level = %v, want %v", i, entries[i].Level, lvl)
		}
	}
}

func TestZap_ContextVariantsDelegate(t *testing.T) {
	l, logs := newObserved(zapcore.DebugLevel)

	l.WarnContext(context.Background(), "warned", "key", "val")

	e := logs.All()[0]
	if e.Level != zapcore.WarnLevel || e.Message != "warned" {
		t.Errorf("got level=%v msg=%q", e.Level, e.Message)
	}
	if got := e.ContextMap()["key"]; got != "val" {
		t.Errorf(`field key = %v, want "val"`, got)
	}
}

func TestZap_Enabled(t *testing.T) {
	l, _ := newObserved(zapcore.WarnLevel)

	cases := []struct {
		level logger.Level
		want  bool
	}{
		{logger.LevelDebug, false},
		{logger.LevelInfo, false},
		{logger.LevelWarn, true},
		{logger.LevelError, true},
	}
	for _, c := range cases {
		if got := l.Enabled(context.Background(), c.level); got != c.want {
			t.Errorf("Enabled(%v) = %v, want %v", c.level, got, c.want)
		}
	}
}

func TestZap_NilLoggerNoPanic(t *testing.T) {
	l := New(nil)
	l.Info("should not panic", "k", "v") // no-op via zap.NewNop
}
