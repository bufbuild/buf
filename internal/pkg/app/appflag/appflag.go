// Package appflag contains functionality to work with flags.
package appflag

import (
	"context"
	"time"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/pflag"
)

// Builder builds run functions.
type Builder interface {
	BindRoot(flagSet *pflag.FlagSet)
	NewRunFunc(func(context.Context, applog.Container) error) func(context.Context, app.Container) error
}

// NewBuilder returns a new Builder.
func NewBuilder(options ...BuilderOption) Builder {
	return newBuilder(options...)
}

// BuilderOption is an option for a new Builder
type BuilderOption func(*builder)

// BuilderWithTimeout returns a new BuilderOption that adds a timeout flag and the default timeout.
func BuilderWithTimeout(defaultTimeout time.Duration) BuilderOption {
	return func(builder *builder) {
		builder.defaultTimeout = defaultTimeout
	}
}
