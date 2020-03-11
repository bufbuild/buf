package clipflag

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"github.com/bufbuild/buf/internal/pkg/cli/clizap"
	"github.com/pkg/profile"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

type flags struct {
	logLevel          string
	logFormat         string
	profile           bool
	profilePath       string
	profileLoops      int
	profileType       string
	profileAllowError bool
	timeout           time.Duration

	bindTimeout    bool
	defaultTimeout time.Duration
}

func newFlags(bindTimeout bool, defaultTimeout time.Duration) *flags {
	return &flags{
		bindTimeout:    bindTimeout,
		defaultTimeout: defaultTimeout,
	}
}

func (f *flags) BindRootCommandFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.logLevel, "log-level", "info", "The log level [debug,info,warn,error].")
	flagSet.StringVar(&f.logFormat, "log-format", "color", "The log format [text,color,json].")
	if f.bindTimeout {
		flagSet.DurationVar(&f.timeout, "timeout", f.defaultTimeout, `The duration until timing out.`)
	}

	flagSet.BoolVar(&f.profile, "profile", false, "Run profiling.")
	_ = flagSet.MarkHidden("profile")
	flagSet.StringVar(&f.profilePath, "profile-path", "", "The profile base directory path.")
	_ = flagSet.MarkHidden("profile-path")
	flagSet.IntVar(&f.profileLoops, "profile-loops", 1, "The number of loops to run.")
	_ = flagSet.MarkHidden("profile-loops")
	flagSet.StringVar(&f.profileType, "profile-type", "cpu", "The profile type [cpu,mem,block,mutex].")
	_ = flagSet.MarkHidden("profile-type")
	flagSet.BoolVar(&f.profileAllowError, "profile-allow-error", false, "Allow errors for profiled commands.")
	_ = flagSet.MarkHidden("profile-allow-error")
}

func (f *flags) NewRunFunc(
	fn func(
		context.Context,
		clienv.Env,
		*zap.Logger,
	) error,
) func(clienv.Env) error {
	return func(env clienv.Env) error {
		return doRun(env, f, fn)
	}
}

func doRun(
	env clienv.Env,
	flags *flags,
	fn func(
		context.Context,
		clienv.Env,
		*zap.Logger,
	) error,
) error {
	logger, err := clizap.NewLogger(env.Stderr(), flags.logLevel, flags.logFormat)
	if err != nil {
		return err
	}
	start := time.Now()
	logger.Debug("command_start")
	defer func() {
		logger.Debug("command_end", zap.Duration("duration", time.Since(start)))
	}()

	ctx := context.Background()
	var cancel context.CancelFunc
	if !flags.profile && flags.timeout != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), flags.timeout)
		defer cancel()
	}

	if !flags.profile {
		return fn(ctx, env, logger)
	}
	return doProfile(
		logger,
		flags.profilePath,
		flags.profileType,
		flags.profileLoops,
		flags.profileAllowError,
		func() error {
			return fn(ctx, env, logger)
		},
	)
}

// doProfile profiles the function.
func doProfile(
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
