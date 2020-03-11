// Package clipflag contains functionality to work with pflag.
package clipflag

import (
	"context"
	"time"

	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// Flags are base flags.
type Flags interface {
	BindRootCommandFlags(flagSet *pflag.FlagSet)
	NewRunFunc(func(context.Context, clienv.Env, *zap.Logger) error) func(clienv.Env) error
}

// NewFlags returns a new Flags.
func NewFlags() Flags {
	return newFlags(false, 0)
}

// NewTimeoutFlags returns a new Flags with a timeout flag.
func NewTimeoutFlags(defaultTimeout time.Duration) Flags {
	return newFlags(true, defaultTimeout)
}
