// Package logger defines the structured logging abstraction used across
// the exporter, decoupled from any concrete backend (zap).
package logger

import "context"

// Level is the severity of a log record. It is deliberately independent of
// any concrete logging backend (zap, ...) so the Logger abstraction does
// not leak a specific implementation. Adapters map it to their own level
// type.
type Level int

// Severity levels, ordered from least to most severe.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger is the structured logging abstraction used across the exporter.
// Message methods take a message plus alternating key/value pairs
// (slog-style): Info("scrape failed", "error", err, "tag", tag).
type Logger interface {
	Debug(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	Enabled(ctx context.Context, level Level) bool
	Error(msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
	Info(msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	Warn(msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
}
