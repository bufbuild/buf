package buf

import (
	"context"
	"fmt"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufos"
	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/cli"
	"github.com/bufbuild/buf/internal/pkg/cli/clicobra"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	imageBuildInputFlagName  = "source"
	imageBuildConfigFlagName = "source-config"
	imageBuildOutputFlagName = "output"

	checkLintInputFlagName  = "input"
	checkLintConfigFlagName = "input-config"

	checkBreakingInputFlagName         = "input"
	checkBreakingConfigFlagName        = "input-config"
	checkBreakingAgainstInputFlagName  = "against-input"
	checkBreakingAgainstConfigFlagName = "against-input-config"

	lsFilesInputFlagName  = "input"
	lsFilesConfigFlagName = "input-config"

	checkLsCheckersConfigFlagName = "config"

	errorFormatFlagName           = "error-format"
	checkLsCheckersFormatFlagName = "format"
)

// Flags are flags for the buf CLI.
type Flags struct {
	*clicobra.Flags

	// root command flags
	// LogLevel and LogFormat are also bound
	Timeout time.Duration
	// root devel command flags
	Profile           bool
	ProfilePath       string
	ProfileLoops      int
	ProfileType       string
	ProfileAllowError bool

	Config        string
	AgainstConfig string

	Input        string
	AgainstInput string

	Output              string
	AsFileDescriptorSet bool

	ExcludeImports    bool
	ExcludeSourceInfo bool

	Files             []string
	LimitToInputFiles bool

	CheckerAll        bool
	CheckerCategories []string

	ErrorFormat string
	Format      string
}

// newFlags returns a new Flags.
//
// Devel should not be set for release binaries.
func newFlags(devel bool) *Flags {
	return &Flags{Flags: clicobra.NewFlags(devel)}
}

// newRunFunc creates a new run function.
func (f *Flags) newRunFunc(
	fn func(
		context.Context,
		*cli.ExecEnv,
		*Flags,
		*zap.Logger,
		*bytepool.SegList,
	) error,
) func(*cli.ExecEnv) error {
	return func(execEnv *cli.ExecEnv) error {
		return conditionalProfile(execEnv, f, fn)
	}
}

func (f *Flags) bindAllRootCommandFlags(flagSet *pflag.FlagSet) {
	f.BindLogLevel(flagSet)
	f.BindLogFormat(flagSet)
	flagSet.DurationVar(&f.Timeout, "timeout", 10*time.Second, `The duration until timing out.`)
	if f.Devel() {
		flagSet.BoolVar(&f.Profile, "profile", false, "Run profiling.")
		flagSet.StringVar(&f.ProfilePath, "profile-path", "", "The profile base directory path.")
		flagSet.IntVar(&f.ProfileLoops, "profile-loops", 10, "The number of loops to run.")
		flagSet.StringVar(&f.ProfileType, "profile-type", "cpu", "The profile type [cpu,mem,block,mutex].")
		flagSet.BoolVar(&f.ProfileAllowError, "profile-allow-error", false, "Allow errors for profiled commands.")
	}
}

func (f *Flags) bindImageBuildInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, imageBuildInputFlagName, ".", fmt.Sprintf(`The source to build. Must be one of format %s.`, bufos.SourceFormatsToString()))
}

func (f *Flags) bindImageBuildConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, imageBuildConfigFlagName, "", `The config file or data to use.`)
}

func (f *Flags) bindImageBuildOutput(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&f.Output, imageBuildOutputFlagName, "o", "", fmt.Sprintf(`Required. The location to write the image. Must be one of format %s.`, bufos.ImageFormatsToString()))
}

func (f *Flags) bindImageBuildAsFileDescriptorSet(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.AsFileDescriptorSet, "as-file-descriptor-set", false, `Output as a google.protobuf.FileDescriptorSet instead of an image.

Note that images are wire-compatible with FileDescriptorSets, however this flag will strip
the additional metadata added for Buf usage.`)
}

func (f *Flags) bindImageBuildExcludeImports(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeImports, "exclude-imports", false, "Exclude imports.")
}

func (f *Flags) bindImageBuildExcludeSourceInfo(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeSourceInfo, "exclude-source-info", false, "Exclude source info.")
}

func (f *Flags) bindImageBuildErrorFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.ErrorFormat, errorFormatFlagName, "text", "The format for build errors, printed to stderr. Must be one of [text,json].")
}

func (f *Flags) bindCheckLintInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, checkLintInputFlagName, ".", fmt.Sprintf(`The source or image to lint. Must be one of format %s.`, bufos.AllFormatsToString()))
}

func (f *Flags) bindCheckLintConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkLintConfigFlagName, "", `The config file or data to use.`)
}

func (f *Flags) bindCheckBreakingInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, checkBreakingInputFlagName, ".", fmt.Sprintf(`The source or image to check for breaking changes. Must be one of format %s.`, bufos.AllFormatsToString()))
}

func (f *Flags) bindCheckBreakingConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkBreakingConfigFlagName, "", `The config file or data to use.`)
}

func (f *Flags) bindCheckBreakingAgainstInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.AgainstInput, checkBreakingAgainstInputFlagName, "", fmt.Sprintf(`Required. The source or image to check against. Must be one of format %s.`, bufos.AllFormatsToString()))
}

func (f *Flags) bindCheckBreakingAgainstConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.AgainstConfig, checkBreakingAgainstConfigFlagName, "", `The config file or data to use for the against source or image.`)
}

func (f *Flags) bindCheckBreakingLimitToInputFiles(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.LimitToInputFiles, "limit-to-input-files", false, `Only run breaking checks against the files in the input.
This has the effect of filtering the against input to only contain the files in the input.
Overrides --file.`)
}

func (f *Flags) bindCheckBreakingExcludeImports(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeImports, "exclude-imports", false, "Exclude imports from breaking change detection.")
}

func (f *Flags) bindCheckFiles(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.Files, "file", nil, `Limit to specific files. This is an advanced feature and is not recommended.`)
}

func (f *Flags) bindCheckErrorFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.ErrorFormat, errorFormatFlagName, "text", "The format for build errors or check violations, printed to stdout. Must be one of [text,json].")
}

func (f *Flags) bindLsFilesInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, lsFilesInputFlagName, ".", fmt.Sprintf(`The source or image to list the files from. Must be one of format %s.`, bufos.AllFormatsToString()))
}

func (f *Flags) bindLsFilesConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, lsFilesConfigFlagName, "", `The config file or data to use.`)
}

func (f *Flags) bindCheckLsCheckersConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkLsCheckersConfigFlagName, "", `The config file or data to use. If --all is specified, this is ignored.`)
}

func (f *Flags) bindCheckLsCheckersAll(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.CheckerAll, "all", false, "List all checkers and not just those currently configured.")
}

func (f *Flags) bindCheckLsCheckersCategories(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.CheckerCategories, "category", nil, "Only list the checkers in these categories.")
}

func (f *Flags) bindCheckLsCheckersFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Format, checkLsCheckersFormatFlagName, "text", "The format to print checkers as. Must be one of [text,json].")
}

func conditionalProfile(
	execEnv *cli.ExecEnv,
	flags *Flags,
	f func(context.Context, *cli.ExecEnv, *Flags, *zap.Logger, *bytepool.SegList) error,
) error {
	logger, err := flags.NewLogger(execEnv.Stderr)
	if err != nil {
		return err
	}
	defer logutil.Defer(logger.Named("clicommand"), "run")()

	ctx := context.Background()
	var cancel context.CancelFunc
	if !flags.Profile && flags.Timeout != 0 {
		ctx, cancel = context.WithTimeout(context.Background(), flags.Timeout)
		defer cancel()
	}

	segList := bytepool.NewNoPoolSegList()

	if !flags.Profile {
		return f(ctx, execEnv, flags, logger, segList)
	}

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
	return clicobra.Profile(
		logger,
		flags.ProfilePath,
		flags.ProfileType,
		flags.ProfileLoops,
		flags.ProfileAllowError,
		func() error {
			return f(ctx, execEnv, flags, logger, segList)
		},
	)
}
