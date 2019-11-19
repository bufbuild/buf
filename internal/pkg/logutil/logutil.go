// Package logutil implements log utilities.
package logutil

import (
	"time"

	"go.uber.org/zap"
)

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
