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

package lint

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName = "error-format"
	configFlagName      = "config"
	pathsFlagName       = "path"

	// deprecated
	inputFlagName = "input"
	// deprecated
	inputConfigFlagName = "input-config"
	// deprecated
	filesFlagName = "file"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
	deprecated string,
	hidden bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name + " <input>",
		Short:      "Check that the input location passes lint checks.",
		Long:       bufcli.GetInputLong(`the source, module, or image to lint`),
		Args:       cobra.MaximumNArgs(1),
		Deprecated: deprecated,
		Hidden:     hidden,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	ErrorFormat string
	Config      string
	Paths       []string

	// deprecated
	Input string
	// deprecated
	InputConfig string
	// deprecated
	Files []string
	// special
	InputHashtag string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	bufcli.BindInputHashtag(flagSet, &f.InputHashtag)
	bufcli.BindPathsAndDeprecatedFiles(flagSet, &f.Paths, pathsFlagName, &f.Files, filesFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors or check violations, printed to stdout. Must be one of %s.",
			stringutil.SliceToString(buflint.AllFormatStrings),
		),
	)
	flagSet.StringVar(
		&f.Config,
		configFlagName,
		"",
		`The config file or data to use.`,
	)

	// deprecated
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		"",
		fmt.Sprintf(
			`The source or image to lint. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	_ = flagSet.MarkDeprecated(
		inputFlagName,
		`input as the first argument instead.`+bufcli.FlagDeprecationMessageSuffix,
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
		fmt.Sprintf("use --%s instead.%s", configFlagName, bufcli.FlagDeprecationMessageSuffix),
	)
	_ = flagSet.MarkHidden(inputConfigFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) (retErr error) {
	input, err := bufcli.GetInputValue(container, flags.InputHashtag, flags.Input, inputFlagName, ".")
	if err != nil {
		return err
	}
	inputConfig, err := bufcli.GetStringFlagOrDeprecatedFlag(
		flags.Config,
		configFlagName,
		flags.InputConfig,
		inputConfigFlagName,
	)
	if err != nil {
		return err
	}
	paths, err := bufcli.GetStringSliceFlagOrDeprecatedFlag(
		flags.Paths,
		pathsFlagName,
		flags.Files,
		filesFlagName,
	)
	if err != nil {
		return err
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
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	imageConfig, fileAnnotations, err := bufcli.NewWireImageConfigReader(
		container.Logger(),
		storageosProvider,
		configProvider,
		moduleResolver,
		moduleReader,
	).GetImageConfig(
		ctx,
		container,
		ref,
		inputConfig,
		paths, // we filter checks for files
		false, // input files must exist
		false, // we must include source info for linting
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		formatString := flags.ErrorFormat
		if formatString == "config-ignore-yaml" {
			formatString = "text"
		}
		if err := bufanalysis.PrintFileAnnotations(container.Stdout(), fileAnnotations, formatString); err != nil {
			return err
		}
		return errors.New("")
	}
	fileAnnotations, err = buflint.NewHandler(container.Logger()).Check(
		ctx,
		imageConfig.Config().Lint,
		bufimage.ImageWithoutImports(imageConfig.Image()),
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		if err := buflint.PrintFileAnnotations(
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
