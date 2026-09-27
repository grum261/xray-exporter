// Package zap adapts a *zap.Logger to the logger.Logger interface.
// It is the default and only logging backend for the exporter.
//
// Message methods accept alternating key/value pairs and convert them
// into typed zap.Fields via zap.Any.
package zap

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grum261/xray-exporter/logger"
)

// badKey labels values that arrive without a valid string key, mirroring
// slog's handling of malformed argument lists.
const badKey = "!BADKEY"

type zapLogger struct {
	l *zap.Logger
}

// New wraps l so it satisfies logger.Logger. If l is nil, a no-op logger
// is used so callers never have to nil-check.
func New(l *zap.Logger) logger.Logger {
	if l == nil {
		l = zap.NewNop()
	}
	return &zapLogger{l: l}
}

// Default returns an adapter over a production zap logger at info level.
// Used by packages that need a sensible fallback logger when none is
// supplied via options.
func Default() logger.Logger {
	return NewAtLevel(logger.LevelInfo)
}

// NewAtLevel builds a production (JSON, stderr) zap logger enabled at the
// given level, wrapped in the adapter. If the logger cannot be built, a
// no-op logger is returned so callers never have to nil-check.
func NewAtLevel(level logger.Level) logger.Logger {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(toZapLevel(level))
	l, err := cfg.Build()
	if err != nil {
		l = zap.NewNop()
	}
	return New(l)
}

func (z *zapLogger) Debug(msg string, args ...any) { z.l.Debug(msg, fields(args)...) }

func (z *zapLogger) DebugContext(_ context.Context, msg string, args ...any) {
	z.l.Debug(msg, fields(args)...)
}

func (z *zapLogger) Error(msg string, args ...any) { z.l.Error(msg, fields(args)...) }

func (z *zapLogger) ErrorContext(_ context.Context, msg string, args ...any) {
	z.l.Error(msg, fields(args)...)
}

func (z *zapLogger) Info(msg string, args ...any) { z.l.Info(msg, fields(args)...) }

func (z *zapLogger) InfoContext(_ context.Context, msg string, args ...any) {
	z.l.Info(msg, fields(args)...)
}

func (z *zapLogger) Warn(msg string, args ...any) { z.l.Warn(msg, fields(args)...) }

func (z *zapLogger) WarnContext(_ context.Context, msg string, args ...any) {
	z.l.Warn(msg, fields(args)...)
}

func (z *zapLogger) Enabled(_ context.Context, level logger.Level) bool {
	return z.l.Core().Enabled(toZapLevel(level))
}

// fields converts slog-style alternating key/value pairs into zap.Fields.
// A non-string key, or a trailing value without its key, is recorded under
// badKey — the same fallback slog uses.
func fields(args []any) []zap.Field {
	if len(args) == 0 {
		return nil
	}
	fs := make([]zap.Field, 0, len(args))
	for i := 0; i < len(args); {
		key, ok := args[i].(string)
		switch {
		case !ok:
			fs = append(fs, zap.Any(badKey, args[i]))
			i++
		case i+1 < len(args):
			fs = append(fs, zap.Any(key, args[i+1]))
			i += 2
		default:
			fs = append(fs, zap.String(badKey, key))
			i++
		}
	}
	return fs
}

func toZapLevel(level logger.Level) zapcore.Level {
	switch level {
	case logger.LevelDebug:
		return zapcore.DebugLevel
	case logger.LevelWarn:
		return zapcore.WarnLevel
	case logger.LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
