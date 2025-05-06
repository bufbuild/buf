// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package appext

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/spf13/pflag"
)

type builder struct {
	appName string

	debug     bool
	noWarn    bool
	logFormat string

	parallelism int

	timeout time.Duration

	defaultTimeout time.Duration

	interceptors   []Interceptor
	loggerProvider LoggerProvider
}

func newBuilder(appName string, options ...BuilderOption) *builder {
	builder := &builder{
		appName:        appName,
		loggerProvider: defaultLoggerProvider,
	}
	for _, option := range options {
		option(builder)
	}
	return builder
}

func (b *builder) BindRoot(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&b.debug, "debug", false, "Turn on debug logging")
	flagSet.StringVar(&b.logFormat, "log-format", "color", "The log format [text,color,json]")
	if b.defaultTimeout > 0 {
		flagSet.DurationVar(&b.timeout, "timeout", b.defaultTimeout, `The duration until timing out, setting it to zero means no timeout`)
	}

	// We do not officially support this flag, this is for testing, where we need warnings turned off.
	flagSet.BoolVar(&b.noWarn, "no-warn", false, "Turn off warn logging")
	_ = flagSet.MarkHidden("no-warn")
	flagSet.IntVar(&b.parallelism, "parallelism", 0, "Manually control the parallelism")
	_ = flagSet.MarkHidden("parallelism")

	// We used to have this as a global flag, so we still need to not error when it is called.
	var verbose bool
	flagSet.BoolVarP(&verbose, "verbose", "v", false, "")
	_ = flagSet.MarkHidden("verbose")
}

func (b *builder) NewRunFunc(
	f func(context.Context, Container) error,
) func(context.Context, app.Container) error {
	interceptor := chainInterceptors(b.interceptors...)
	return func(ctx context.Context, appContainer app.Container) error {
		if interceptor != nil {
			return b.run(ctx, appContainer, interceptor(f))
		}
		return b.run(ctx, appContainer, f)
	}
}

func (b *builder) run(
	ctx context.Context,
	appContainer app.Container,
	f func(context.Context, Container) error,
) (retErr error) {
	logLevel, err := getLogLevel(b.debug, b.noWarn)
	if err != nil {
		return err
	}
	logFormat, err := ParseLogFormat(b.logFormat)
	if err != nil {
		return err
	}
	nameContainer, err := newNameContainer(appContainer, b.appName)
	if err != nil {
		return err
	}
	logger, err := b.loggerProvider(nameContainer, logLevel, logFormat)
	if err != nil {
		return err
	}
	container := newContainer(nameContainer, logger)

	if b.parallelism > 0 {
		thread.SetParallelism(b.parallelism)
	}

	var cancel context.CancelFunc
	if b.timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}

	return f(ctx, container)
}

func getLogLevel(debugFlag bool, noWarnFlag bool) (LogLevel, error) {
	if debugFlag && noWarnFlag {
		return 0, fmt.Errorf("cannot set both --debug and --no-warn")
	}
	if noWarnFlag {
		return LogLevelError, nil
	}
	if debugFlag {
		return LogLevelDebug, nil
	}
	return LogLevelInfo, nil
}

func defaultLoggerProvider(container NameContainer, logLevel LogLevel, logFormat LogFormat) (*slog.Logger, error) {
	switch logFormat {
	case LogFormatText, LogFormatColor:
		return slog.New(slog.NewTextHandler(container.Stderr(), &slog.HandlerOptions{Level: logLevel.SlogLevel()})), nil
	case LogFormatJSON:
		return slog.New(slog.NewJSONHandler(container.Stderr(), &slog.HandlerOptions{Level: logLevel.SlogLevel()})), nil
	default:
		return nil, fmt.Errorf("unknown appext.LogFormat: %v", logFormat)
	}
}

// chainInterceptors consolidates the given interceptors into one.
// The interceptors are applied in the order they are declared.
func chainInterceptors(interceptors ...Interceptor) Interceptor {
	if len(interceptors) == 0 {
		return nil
	}
	filtered := make([]Interceptor, 0, len(interceptors))
	for _, interceptor := range interceptors {
		if interceptor != nil {
			filtered = append(filtered, interceptor)
		}
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		first := filtered[0]
		return func(next func(context.Context, Container) error) func(context.Context, Container) error {
			for i := len(filtered) - 1; i > 0; i-- {
				next = filtered[i](next)
			}
			return first(next)
		}
	}
}
