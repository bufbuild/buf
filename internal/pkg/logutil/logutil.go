// Package logutil implements log utilities.
package logutil

import (
	"io"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/pkg/errs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger returns a new Logger.
func NewLogger(stderr io.Writer, level string, format string) (*zap.Logger, error) {
	level = strings.TrimSpace(strings.ToLower(level))
	format = strings.TrimSpace(strings.ToLower(format))

	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "":
		zapLevel = zapcore.InfoLevel
	default:
		return nil, errs.NewUserErrorf("unknown log level [debug,info,warn,error]: %q", level)
	}

	var encoder zapcore.Encoder
	switch format {
	case "text":
		encoder = zapcore.NewConsoleEncoder(textEncoderConfig)
	case "color":
		encoder = zapcore.NewConsoleEncoder(colortextEncoderConfig)
	case "json":
		encoder = zapcore.NewJSONEncoder(jsonEncoderConfig)
	case "":
		encoder = zapcore.NewConsoleEncoder(colortextEncoderConfig)
	default:
		return nil, errs.NewUserErrorf("unknown log format [text,color,json]: %q", format)
	}

	return zap.New(
		zapcore.NewCore(
			encoder,
			zapcore.Lock(zapcore.AddSync(stderr)),
			zap.NewAtomicLevelAt(zapLevel),
		),
	), nil
}

// Defer returns a function to defer that logs at the debug level.
//
// defer logutil.Defer(logger, "foo")()
func Defer(logger *zap.Logger, name string, fields ...zap.Field) func() {
	start := time.Now()
	return func() {
		fields = append(fields, zap.Duration("duration", time.Since(start)))
		logger.Debug(name, fields...)
	}
}

// DeferWithError returns a function to defer that logs at the debug level.
//
// defer logutil.DeferWithError(logger, "foo", &retErr)()
func DeferWithError(logger *zap.Logger, name string, retErrPtr *error, fields ...zap.Field) func() {
	start := time.Now()
	return func() {
		fields = append(fields, zap.Duration("duration", time.Since(start)))
		if retErrPtr != nil && *retErrPtr != nil {
			fields = append(fields, zap.Error(*retErrPtr))
		}
		logger.Debug(name, fields...)
	}
}
