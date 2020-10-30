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

package breaking

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/cmd/buf/command/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName       = "error-format"
	excludeImportsFlagName    = "exclude-imports"
	filesFlagName             = "file"
	limitToInputFilesFlagName = "limit-to-input-files"
	configFlagName            = "config"
	againstFlagName           = "against"
	againstConfigFlagName     = "against-config"

	// deprecated
	inputFlagName = "input"
	// deprecated
	inputConfigFlagName = "input-config"
	// deprecated
	againstInputFlagName = "against-input"
	// deprecated
	againstInputConfigFlagName = "against-input-config"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " --against against-input <input>",
		Short: "Check that the input location has no breaking changes compared to the against location.",
		Long:  internal.GetInputLong(`the source, module, or image to check for breaking changes`),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	ErrorFormat       string
	ExcludeImports    bool
	Files             []string
	LimitToInputFiles bool
	Config            string
	Against           string
	AgainstConfig     string

	// deprecated
	Input string
	// deprecated
	InputConfig string
	// deprecated
	AgainstInput string
	// deprecated
	AgainstInputConfig string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	internal.BindInputHashtag(flagSet, &f.InputHashtag)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations, printed to stdout. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	flagSet.BoolVar(
		&f.ExcludeImports,
		excludeImportsFlagName,
		false,
		"Exclude imports from breaking change detection.",
	)
	flagSet.StringSliceVar(
		&f.Files,
		filesFlagName,
		nil,
		`Limit to specific files. This is an advanced feature and is not recommended.`,
	)
	flagSet.BoolVar(
		&f.LimitToInputFiles,
		limitToInputFilesFlagName,
		false,
		`Only run breaking checks against the files in the input.
This has the effect of filtering the against input to only contain the files in the input.
Overrides --file.`,
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The config file or data to use.`,
	)
	flagSet.StringVar(
		&f.Against,
		againstFlagName,
		"",
		fmt.Sprintf(
			`Required. The source, module, or image to check against. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.AgainstConfig,
		againstConfigFlagName,
		"",
		`The config file or data to use for the against source, module, or image.`,
	)

	// deprecated
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		"",
		fmt.Sprintf(
			`The source or image to check for breaking changes. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	_ = flagSet.MarkDeprecated(
		inputFlagName,
		`input as the first argument instead.`+internal.FlagDeprecationMessageSuffix,
	)
	_ = flagSet.MarkHidden(inputFlagName)
	// deprecated
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The config file or data to use.`,
	)
	_ = flagSet.MarkDeprecated(
		inputConfigFlagName,
		fmt.Sprintf("use --%s instead.%s", configFlagName, internal.FlagDeprecationMessageSuffix),
	)
	_ = flagSet.MarkHidden(inputConfigFlagName)
	// deprecated
	flagSet.StringVar(
		&f.AgainstInput,
		againstInputFlagName,
		"",
		fmt.Sprintf(
			`Required. The source or image to check against. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	_ = flagSet.MarkDeprecated(
		againstInputFlagName,
		fmt.Sprintf("use --%s instead.%s", againstFlagName, internal.FlagDeprecationMessageSuffix),
	)
	_ = flagSet.MarkHidden(againstInputFlagName)
	// deprecated
	flagSet.StringVar(
		&f.AgainstInputConfig,
		againstInputConfigFlagName,
		"",
		`The config file or data to use for the against source or image.`,
	)
	_ = flagSet.MarkDeprecated(
		againstInputConfigFlagName,
		fmt.Sprintf("use --%s instead.%s", againstConfigFlagName, internal.FlagDeprecationMessageSuffix),
	)
	_ = flagSet.MarkHidden(againstInputConfigFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) error {
	input, err := internal.GetInputValue(container, flags.InputHashtag, flags.Input, inputFlagName, ".")
	if err != nil {
		return err
	}
	inputConfig, err := internal.GetFlagOrDeprecatedFlag(
		flags.Config,
		configFlagName,
		flags.InputConfig,
		inputConfigFlagName,
	)
	if err != nil {
		return err
	}
	againstInput, err := internal.GetFlagOrDeprecatedFlag(
		flags.Against,
		againstFlagName,
		flags.AgainstInput,
		againstInputFlagName,
	)
	if err != nil {
		return err
	}
	againstInputConfig, err := internal.GetFlagOrDeprecatedFlag(
		flags.AgainstConfig,
		againstConfigFlagName,
		flags.AgainstInputConfig,
		againstInputConfigFlagName,
	)
	if err != nil {
		return err
	}
	if againstInput == "" {
		return appcmd.NewInvalidArgumentErrorf("Flag --%s is required.", againstFlagName)
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, input)
	if err != nil {
		return err
	}
	configProvider := bufconfig.NewProvider(container.Logger())
	moduleResolver, err := moduleResolverReaderProvider.GetModuleResolver(ctx, container)
	if err != nil {
		return err
	}
	moduleReader, err := moduleResolverReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := bufcli.NewWireEnvReader(
		container.Logger(),
		configProvider,
		moduleResolver,
		moduleReader,
	).GetEnv(
		ctx,
		container,
		ref,
		inputConfig,
		flags.Files, // we filter checks for files
		false,       // files specified must exist on the main input
		false,       // we must include source info for this side of the check
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	image := env.Image()
	if flags.ExcludeImports {
		image = bufimage.ImageWithoutImports(image)
	}

	// TODO: this doesn't actually work because we're using the same file paths for both sides
	// if the roots change, then we're torched
	externalPaths := flags.Files
	if flags.LimitToInputFiles {
		files := image.Files()
		// we know that the file descriptors have unique names from validation
		externalPaths = make([]string, len(files))
		for i, file := range files {
			externalPaths[i] = file.ExternalPath()
		}
	}

	againstRef, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, againstInput)
	if err != nil {
		return err
	}
	againstEnv, fileAnnotations, err := bufcli.NewWireEnvReader(
		container.Logger(),
		configProvider,
		moduleResolver,
		moduleReader,
	).GetEnv(
		ctx,
		container,
		againstRef,
		againstInputConfig,
		externalPaths, // we filter checks for files
		true,          // files are allowed to not exist on the against input
		true,          // no need to include source info for against
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	againstImage := againstEnv.Image()
	if flags.ExcludeImports {
		againstImage = bufimage.ImageWithoutImports(againstImage)
	}
	fileAnnotations, err = bufbreaking.NewHandler(container.Logger()).Check(
		ctx,
		env.Config().Breaking,
		againstImage,
		image,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := bufanalysis.PrintFileAnnotations(
			container.Stdout(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		return errors.New("")
	}
	return nil
}
