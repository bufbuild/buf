// Copyright 2020-2021 Buf Technologies, Inc.
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

package appflag

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/observability"
	"github.com/bufbuild/buf/internal/pkg/observability/observabilityzap"
	"github.com/pkg/profile"
	"github.com/spf13/pflag"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type builder struct {
	appName string

	verbose   bool
	logLevel  string
	logFormat string

	profile           bool
	profilePath       string
	profileLoops      int
	profileType       string
	profileAllowError bool

	timeout time.Duration

	defaultTimeout time.Duration

	tracing bool
}

func newBuilder(appName string, options ...BuilderOption) *builder {
	builder := &builder{
		appName: appName,
	}
	for _, option := range options {
		option(builder)
	}
	return builder
}

func (b *builder) BindRoot(flagSet *pflag.FlagSet) {
	// True means debug, false means info.
	flagSet.BoolVarP(&b.verbose, "verbose", "v", false, "Enable verbose logging.")
	flagSet.StringVar(&b.logLevel, "log-level", "", "The log level [debug,info,warn,error].")
	// Effectively deprecated - we only ever cared about debug vs info, which is now controlled
	// by verbose, but we keep this around in case it is being used and handle it being set.
	// Note that both verbose and log-level cannot be set together.
	_ = flagSet.MarkHidden("log-level")
	flagSet.StringVar(&b.logFormat, "log-format", "color", "The log format [text,color,json].")
	if b.defaultTimeout > 0 {
		flagSet.DurationVar(&b.timeout, "timeout", b.defaultTimeout, `The duration until timing out.`)
	}

	flagSet.BoolVar(&b.profile, "profile", false, "Run profiling.")
	_ = flagSet.MarkHidden("profile")
	flagSet.StringVar(&b.profilePath, "profile-path", "", "The profile base directory path.")
	_ = flagSet.MarkHidden("profile-path")
	flagSet.IntVar(&b.profileLoops, "profile-loops", 1, "The number of loops to run.")
	_ = flagSet.MarkHidden("profile-loops")
	flagSet.StringVar(&b.profileType, "profile-type", "cpu", "The profile type [cpu,mem,block,mutex].")
	_ = flagSet.MarkHidden("profile-type")
	flagSet.BoolVar(&b.profileAllowError, "profile-allow-error", false, "Allow errors for profiled commands.")
	_ = flagSet.MarkHidden("profile-allow-error")
}

func (b *builder) NewRunFunc(
	f func(context.Context, Container) error,
) func(context.Context, app.Container) error {
	return func(ctx context.Context, appContainer app.Container) error {
		return b.run(ctx, appContainer, f)
	}
}

func (b *builder) run(
	ctx context.Context,
	appContainer app.Container,
	f func(context.Context, Container) error,
) (retErr error) {
	logLevel, err := getLogLevel(b.verbose, b.logLevel)
	if err != nil {
		return err
	}
	logger, err := applog.NewLogger(appContainer.Stderr(), logLevel, b.logFormat)
	if err != nil {
		return err
	}
	container, err := newContainer(appContainer, b.appName, logger)
	if err != nil {
		return err
	}

	var cancel context.CancelFunc
	if !b.profile && b.timeout != 0 {
		ctx, cancel = context.WithTimeout(ctx, b.timeout)
		defer cancel()
	}

	if b.tracing {
		closer := observability.Start(
			observability.StartWithTraceExportCloser(
				observabilityzap.NewTraceExportCloser(logger),
			),
		)
		defer func() {
			retErr = multierr.Append(retErr, closer.Close())
		}()
		var span *trace.Span
		ctx, span = trace.StartSpan(ctx, "command")
		defer span.End()
	}
	if !b.profile {
		return f(ctx, container)
	}
	return runProfile(
		logger,
		b.profilePath,
		b.profileType,
		b.profileLoops,
		b.profileAllowError,
		func() error {
			return f(ctx, container)
		},
	)
}

// runProfile profiles the function.
func runProfile(
	logger *zap.Logger,
	profilePath string,
	profileType string,
	profileLoops int,
	profileAllowError bool,
	f func() error,
) error {
	var err error
	if profilePath == "" {
		profilePath, err = ioutil.TempDir("", "")
		if err != nil {
			return err
		}
	}
	logger.Debug("profile", zap.String("path", profilePath))
	if profileType == "" {
		profileType = "cpu"
	}
	if profileLoops == 0 {
		profileLoops = 10
	}
	var profileFunc func(*profile.Profile)
	switch profileType {
	case "cpu":
		profileFunc = profile.CPUProfile
	case "mem":
		profileFunc = profile.MemProfile
	case "block":
		profileFunc = profile.BlockProfile
	case "mutex":
		profileFunc = profile.MutexProfile
	default:
		return fmt.Errorf("unknown profile type: %q", profileType)
	}
	stop := profile.Start(
		profile.Quiet,
		profile.ProfilePath(profilePath),
		profileFunc,
	)
	for i := 0; i < profileLoops; i++ {
		if err := f(); err != nil {
			if !profileAllowError {
				return err
			}
		}
	}
	stop.Stop()
	return nil
}

func getLogLevel(verboseFlag bool, logLevelFlag string) (string, error) {
	if verboseFlag && logLevelFlag != "" {
		return "", errors.New("cannot set both --verbose and --log-level")
	}
	if logLevelFlag != "" {
		return logLevelFlag, nil
	}
	if verboseFlag {
		return "debug", nil
	}
	return "info", nil
}
