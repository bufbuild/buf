// Package clicobra contains helper functionality for applications using Cobra.
package clicobra

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/cli/internal"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// Flags are base flags.
type Flags struct {
	logLevel          string
	logFormat         string
	profile           bool
	profilePath       string
	profileLoops      int
	profileType       string
	profileAllowError bool

	devel bool
}

// NewFlags returns a new Flags.
//
// Devel should not be set for release binaries.
func NewFlags(devel bool) *Flags {
	return &Flags{
		devel: devel,
	}
}

// BindRootCommandFlags binds the root-command flags.
func (f *Flags) BindRootCommandFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.logLevel, "log-level", "info", "The log level [debug,info,warn,error].")
	flagSet.StringVar(&f.logFormat, "log-format", "color", "The log format [text,color,json].")
	if f.devel {
		flagSet.BoolVar(&f.profile, "profile", false, "Run profiling.")
		flagSet.StringVar(&f.profilePath, "profile-path", "", "The profile base directory path.")
		flagSet.IntVar(&f.profileLoops, "profile-loops", 1, "The number of loops to run.")
		flagSet.StringVar(&f.profileType, "profile-type", "cpu", "The profile type [cpu,mem,block,mutex].")
		flagSet.BoolVar(&f.profileAllowError, "profile-allow-error", false, "Allow errors for profiled commands.")
	}
}

// NewRunFunc creates a new run function.
func (f *Flags) NewRunFunc(
	fn func(
		*cli.ExecEnv,
		*zap.Logger,
		*bytepool.SegList,
	) error,
) func(*cli.ExecEnv) error {
	return func(execEnv *cli.ExecEnv) error {
		return doRun(execEnv, f, fn)
	}
}

// Devel returns true if devel was set.
func (f *Flags) Devel() bool {
	return f.devel
}

// TimeoutFlags are base flags with a root timeout.
type TimeoutFlags struct {
	flags   *Flags
	timeout time.Duration
}

// NewTimeoutFlags returns a new TimeoutFlags.
//
// Devel should not be set for release binaries.
func NewTimeoutFlags(devel bool) *TimeoutFlags {
	return &TimeoutFlags{
		flags: NewFlags(devel),
	}
}

// BindRootCommandFlags binds the root-command flags with timeout.
func (t *TimeoutFlags) BindRootCommandFlags(flagSet *pflag.FlagSet, defaultTimeout time.Duration) {
	t.flags.BindRootCommandFlags(flagSet)
	flagSet.DurationVar(&t.timeout, "timeout", defaultTimeout, `The duration until timing out.`)
}

// NewRunFunc creates a new run function.
func (t *TimeoutFlags) NewRunFunc(
	fn func(
		context.Context,
		*cli.ExecEnv,
		*zap.Logger,
		*bytepool.SegList,
	) error,
) func(*cli.ExecEnv) error {
	return func(execEnv *cli.ExecEnv) error {
		ctx := context.Background()
		var cancel context.CancelFunc
		if !t.flags.profile && t.timeout != 0 {
			ctx, cancel = context.WithTimeout(context.Background(), t.timeout)
			defer cancel()
		}
		return doRun(
			execEnv,
			t.flags,
			func(
				execEnv *cli.ExecEnv,
				logger *zap.Logger,
				segList *bytepool.SegList,
			) error {
				return fn(ctx, execEnv, logger, segList)
			},
		)
	}
}

// Devel returns true if devel was set.
func (t *TimeoutFlags) Devel() bool {
	return t.flags.devel
}

// Main runs the application using the OS runtime and calling os.Exit on the return value of Run.
func Main(rootCommand *Command, version string) {
	os.Exit(Run(rootCommand, version, internal.NewOSRunEnv()))
}

// Run runs the application, returning the exit code.
//
// RunEnv will be modified to have dummy values if fields are not set.
func Run(rootCommand *Command, version string, runEnv *cli.RunEnv) int {
	start := time.Now()
	internal.SetRunEnvDefaults(runEnv)
	var exitCode int
	if err := runRootCommand(rootCommand, version, start, runEnv, &exitCode); err != nil {
		printError(runEnv.Stderr, err)
		return 1
	}
	return exitCode
}

// Command is a command.
type Command struct {
	// Use is the one-line usage message. Required.
	Use string
	// Short is the short message shown in the 'help' output. Required if Long is set.
	Short string
	// Long is the long message shown in the 'help <this-command>' output. Optional.
	// The Short field will be prepended to the Long field with a newline.
	Long string
	// Args are the expected arguments. Optional.
	Args cobra.PositionalArgs
	// Run is the command to run. Optional.
	Run func(*cli.ExecEnv) error
	// BindFlags allows binding of flags on build. Optional.
	BindFlags func(*pflag.FlagSet)
	// SubCommands are the sub-commands. Optional.
	SubCommands []*Command
}

func (c *Command) toCobra(start time.Time, runEnv *cli.RunEnv, exitCodeAddr *int) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = c.Use
	cmd.Short = strings.TrimSpace(c.Short)
	if c.Long != "" {
		cmd.Long = fmt.Sprintf("%s\n%s", cmd.Short, strings.TrimSpace(c.Long))
	}
	cmd.Args = c.Args
	if c.Run != nil {
		cmd.Run = func(_ *cobra.Command, args []string) {
			execEnv, err := internal.NewExecEnv(
				args,
				runEnv.Stdin,
				runEnv.Stdout,
				runEnv.Stderr,
				runEnv.Environ,
				start,
			)
			if err != nil {
				printError(runEnv.Stderr, err)
				*exitCodeAddr = 1
				return
			}
			if err := c.Run(execEnv); err != nil {
				printError(runEnv.Stderr, err)
				*exitCodeAddr = 1
			}
		}
	}
	if c.BindFlags != nil {
		c.BindFlags(cmd.PersistentFlags())
	}
	for _, subCommand := range c.SubCommands {
		cmd.AddCommand(subCommand.toCobra(start, runEnv, exitCodeAddr))
	}
	return cmd
}

func runRootCommand(
	rootCommand *Command,
	version string,
	start time.Time,
	runEnv *cli.RunEnv,
	exitCodeAddr *int,
) error {
	rootCmd := rootCommand.toCobra(start, runEnv, exitCodeAddr)

	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Version = version

	rootCmd.AddCommand(&cobra.Command{
		Use:    "bash-completion",
		Args:   cobra.NoArgs,
		Hidden: true,
		Run: func(*cobra.Command, []string) {
			if err := rootCmd.GenBashCompletion(runEnv.Stdout); err != nil {
				printError(runEnv.Stderr, err)
				*exitCodeAddr = 1
			}
		},
	})
	rootCmd.AddCommand(&cobra.Command{
		Use:    "zsh-completion",
		Args:   cobra.NoArgs,
		Hidden: true,
		Run: func(*cobra.Command, []string) {
			if err := rootCmd.GenZshCompletion(runEnv.Stdout); err != nil {
				printError(runEnv.Stderr, err)
				*exitCodeAddr = 1
			}
		},
	})
	//rootCmd.AddCommand(&cobra.Command{
	//Use:    "manpages dirPath",
	//Args:   cobra.ExactArgs(1),
	//Hidden: true,
	//Run: func(_ *cobra.Command, args []string) {
	//if err := doc.GenManTree(rootCmd, &doc.GenManHeader{
	//Source: rootCmd.Use,
	//}, args[0]); err != nil {
	//printError(runEnv.Stderr, err)
	//*exitCodeAddr = 1
	//}
	//},
	//})

	rootCmd.SetArgs(runEnv.Args)
	rootCmd.SetOutput(runEnv.Stderr)

	return rootCmd.Execute()
}

func printError(writer io.Writer, err error) {
	if errString := err.Error(); errString != "" {
		_, _ = fmt.Fprintln(writer, errString)
	}
}

func doRun(
	execEnv *cli.ExecEnv,
	flags *Flags,
	f func(*cli.ExecEnv, *zap.Logger, *bytepool.SegList) error,
) error {
	logger, err := logutil.NewLogger(execEnv.Stderr, flags.logLevel, flags.logFormat)
	if err != nil {
		return err
	}
	defer logutil.Defer(logger.Named("command"), "run")()

	segList := bytepool.NewNoPoolSegList()
	if strings.TrimSpace(strings.ToLower(flags.logLevel)) == "debug" {
		defer func() {
			var unrecycled uint64
			for _, listStats := range segList.ListStats() {
				logger.Debug("memory", zap.Any("list_stats", listStats))
				unrecycled += listStats.TotalUnrecycled
			}
			if unrecycled != 0 {
				logger.Debug("memory_leak", zap.Uint64("unrecycled", unrecycled))
			}
		}()
	}

	if !flags.profile {
		return f(execEnv, logger, segList)
	}

	return doProfile(
		logger,
		flags.profilePath,
		flags.profileType,
		flags.profileLoops,
		flags.profileAllowError,
		func() error {
			return f(execEnv, logger, segList)
		},
	)
}

// doProfile profiles the function.
//
// The function should only return error on system error.
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
		return errs.NewInvalidArgumentf("unknown profile type: %q", profileType)
	}
	profileStart := time.Now()
	logger.Debug("profile_start", zap.String("profile_path", profilePath))
	stop := profile.Start(
		profile.Quiet,
		profile.ProfilePath(profilePath),
		profileFunc,
	)
	for i := 0; i < profileLoops; i++ {
		if err := f(); err != nil {
			if !profileAllowError {
				logger.Error("profile_end_with_error")
				return err
			}
		}
	}
	stop.Stop()
	logger.Debug("profile_end", zap.Duration("duration", time.Since(profileStart)))
	return nil
}
