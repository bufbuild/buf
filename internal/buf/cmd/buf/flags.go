// Copyright 2020 Buf Technologies, Inc.
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

package buf

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/pflag"
)

const (
	imageBuildInputFlagName            = "source"
	imageBuildConfigFlagName           = "source-config"
	imageBuildOutputFlagName           = "output"
	imageConvertInputFlagName          = "image"
	imageConvertOutputFlagName         = "output"
	checkLintInputFlagName             = "input"
	checkLintConfigFlagName            = "input-config"
	checkBreakingInputFlagName         = "input"
	checkBreakingConfigFlagName        = "input-config"
	checkBreakingAgainstInputFlagName  = "against-input"
	checkBreakingAgainstConfigFlagName = "against-input-config"
	checkLsCheckersConfigFlagName      = "config"
	checkLsCheckersFormatFlagName      = "format"
	lsFilesInputFlagName               = "input"
	lsFilesConfigFlagName              = "input-config"
	errorFormatFlagName                = "error-format"
	experimentalGitCloneFlagName       = "experimental-git-clone"
)

// flags are the flags.
type flags struct {
	Config               string
	AgainstConfig        string
	Input                string
	AgainstInput         string
	ConvertInput         string
	Output               string
	AsFileDescriptorSet  bool
	ExcludeImports       bool
	ExcludeSourceInfo    bool
	Files                []string
	LimitToInputFiles    bool
	CheckerAll           bool
	CheckerCategories    []string
	ErrorFormat          string
	Format               string
	ExperimentalGitClone bool
}

func newFlags() *flags {
	return &flags{}
}

func newRunFunc(
	builder appflag.Builder,
	flags *flags,
	f func(context.Context, applog.Container, *flags) error,
) func(context.Context, app.Container) error {
	return builder.NewRunFunc(
		func(ctx context.Context, container applog.Container) error {
			return f(ctx, container, flags)
		},
	)
}

func (f *flags) bindImageBuildInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, imageBuildInputFlagName, ".", fmt.Sprintf(`The source to build. Must be one of format %s.`, buffetch.SourceFormatsString))
}

func (f *flags) bindImageBuildConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, imageBuildConfigFlagName, "", `The config file or data to use.`)
}

func (f *flags) bindImageBuildFiles(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.Files, "file", nil, `Limit to specific files. This is an advanced feature and is not recommended.`)
}

func (f *flags) bindImageBuildOutput(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&f.Output, imageBuildOutputFlagName, "o", "", fmt.Sprintf(`Required. The location to write the image. Must be one of format %s.`, buffetch.ImageFormatsString))
}

func (f *flags) bindImageBuildAsFileDescriptorSet(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.AsFileDescriptorSet, "as-file-descriptor-set", false, `Output as a google.protobuf.FileDescriptorSet instead of an image.

Note that images are wire-compatible with FileDescriptorSets, however this flag will strip
the additional metadata added for Buf usage.`)
}

func (f *flags) bindImageBuildExcludeImports(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeImports, "exclude-imports", false, "Exclude imports.")
}

func (f *flags) bindImageBuildExcludeSourceInfo(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeSourceInfo, "exclude-source-info", false, "Exclude source info.")
}

func (f *flags) bindImageBuildErrorFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
}

func (f *flags) bindImageConvertInput(flagSet *pflag.FlagSet) {
	// TODO: cobra cannot have the same variable with different inputs, we need
	// to refactor the variables to have different binds per function
	flagSet.StringVarP(&f.ConvertInput, imageConvertInputFlagName, "i", "", fmt.Sprintf(`The image to convert. Must be one of format %s.`, buffetch.ImageFormatsString))
}

func (f *flags) bindImageConvertFiles(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.Files, "file", nil, `Limit to specific files. This is an advanced feature and is not recommended.`)
}

func (f *flags) bindImageConvertOutput(flagSet *pflag.FlagSet) {
	flagSet.StringVarP(&f.Output, imageConvertOutputFlagName, "o", "", fmt.Sprintf(`Required. The location to write the image to. Must be one of format %s.`, buffetch.ImageFormatsString))
}

func (f *flags) bindImageConvertAsFileDescriptorSet(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.AsFileDescriptorSet, "as-file-descriptor-set", false, `Output as a google.protobuf.FileDescriptorSet instead of an image.

Note that images are wire-compatible with FileDescriptorSets, however this flag will strip
the additional metadata added for Buf usage.`)
}

func (f *flags) bindImageConvertExcludeImports(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeImports, "exclude-imports", false, "Exclude imports.")
}

func (f *flags) bindImageConvertExcludeSourceInfo(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeSourceInfo, "exclude-source-info", false, "Exclude source info.")
}

func (f *flags) bindCheckLintInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, checkLintInputFlagName, ".", fmt.Sprintf(`The source or image to lint. Must be one of format %s.`, buffetch.AllFormatsString))
}

func (f *flags) bindCheckLintConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkLintConfigFlagName, "", `The config file or data to use.`)
}

func (f *flags) bindCheckBreakingInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Input, checkBreakingInputFlagName, ".", fmt.Sprintf(`The source or image to check for breaking changes. Must be one of format %s.`, buffetch.AllFormatsString))
}

func (f *flags) bindCheckBreakingConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkBreakingConfigFlagName, "", `The config file or data to use.`)
}

func (f *flags) bindCheckBreakingAgainstInput(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.AgainstInput, checkBreakingAgainstInputFlagName, "", fmt.Sprintf(`Required. The source or image to check against. Must be one of format %s.`, buffetch.AllFormatsString))
}

func (f *flags) bindCheckBreakingAgainstConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.AgainstConfig, checkBreakingAgainstConfigFlagName, "", `The config file or data to use for the against source or image.`)
}

func (f *flags) bindCheckBreakingLimitToInputFiles(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.LimitToInputFiles, "limit-to-input-files", false, `Only run breaking checks against the files in the input.
This has the effect of filtering the against input to only contain the files in the input.
Overrides --file.`)
}

func (f *flags) bindCheckBreakingExcludeImports(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.ExcludeImports, "exclude-imports", false, "Exclude imports from breaking change detection.")
}

func (f *flags) bindCheckFiles(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.Files, "file", nil, `Limit to specific files. This is an advanced feature and is not recommended.`)
}

func (f *flags) bindCheckBreakingErrorFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations, printed to stdout. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
}

func (f *flags) bindCheckLintErrorFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations, printed to stdout. Must be one of %s.",
			stringutil.SliceToString(buflint.AllFormatStrings),
		),
	)
}

func (f *flags) bindCheckLsCheckersConfig(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.Config, checkLsCheckersConfigFlagName, "", `The config file or data to use. If --all is specified, this is ignored.`)
}

func (f *flags) bindCheckLsCheckersAll(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.CheckerAll, "all", false, "List all checkers and not just those currently configured.")
}

func (f *flags) bindCheckLsCheckersCategories(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(&f.CheckerCategories, "category", nil, "Only list the checkers in these categories.")
}

func (f *flags) bindCheckLsCheckersFormat(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Format,
		checkLsCheckersFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format to print checkers as. Must be one of %s.",
			stringutil.SliceToString(bufcheck.AllCheckerFormatStrings),
		),
	)
}

func (f *flags) bindExperimentalGitClone(flagSet *pflag.FlagSet) {
	internal.BindExperimentalGitClone(flagSet, &f.ExperimentalGitClone)
}
