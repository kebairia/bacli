package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
}

// zapLogger wraps a *zap.SugaredLogger and implements Logger.
type zapLogger struct {
	sugar *zap.SugaredLogger
}

// Ensure zapLogger satisfies Logger.
var _ Logger = (*zapLogger)(nil)

// Debug logs at DebugLevel. keysAndValues are alternating key/value pairs.
func (l *zapLogger) Debug(msg string, keysAndValues ...any) {
	l.sugar.Debugw(msg, keysAndValues...)
}

// Info logs at InfoLevel.
func (l *zapLogger) Info(msg string, keysAndValues ...any) {
	l.sugar.Infow(msg, keysAndValues...)
}

// Warn logs at WarnLevel.
func (l *zapLogger) Warn(msg string, keysAndValues ...any) {
	l.sugar.Warnw(msg, keysAndValues...)
}

// Error logs at ErrorLevel.
func (l *zapLogger) Error(msg string, keysAndValues ...any) {
	l.sugar.Errorw(msg, keysAndValues...)
}

// ----------------------------------------------------------------------------
// globalSugar holds the SugaredLogger for easy global use (optional).
var globalSugar *zap.SugaredLogger

// Init creates a Zap logger, wraps it, and returns your Logger interface.
// Call this once at startup.
func Init() (Logger, error) {
	// 1) Start from a development-friendly config:
	cfg := zap.NewDevelopmentConfig()
	// cfg := zap.NewProductionConfig()

	// 2) Tweak encoder: ISO8601 timestamps + capital, colored levels
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	// 3) Build the zap.Logger
	zapLog, err := cfg.Build(
		zap.AddCaller(),      // include file:line
		zap.AddCallerSkip(1), // skip this Init frame
	)
	if err != nil {
		return nil, err
	}

	// 4) Create a SugaredLogger for easier key/value logging
	sugar := zapLog.Sugar()
	globalSugar = sugar

	// 5) Return your interface implementation
	return &zapLogger{sugar: sugar}, nil
}

// Cleanup flushes any buffered log entries. Call at program exit.
func Cleanup() {
	if globalSugar != nil {
		_ = globalSugar.Sync()
	}
}

// Global returns the Logger created by Init(), for use in libraries.
// You may choose to remove this and inject Logger everywhere instead.
func Global() Logger {
	return &zapLogger{sugar: globalSugar}
}
