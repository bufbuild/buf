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

package build

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	imageinternal "github.com/bufbuild/buf/internal/buf/cmd/buf/command/image/internal"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	asFileDescriptorSetFlagName = "as-file-descriptor-set"
	errorFormatFlagName         = "error-format"
	excludeImportsFlagName      = "exclude-imports"
	excludeSourceInfoFlagName   = "exclude-source-info"
	filesFlagName               = "file"
	outputFlagName              = "output"
	outputFlagShortName         = "o"
	sourceFlagName              = "source"
	sourceConfigFlagName        = "source-config"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleReaderProvider bufcli.ModuleReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Build all files from the input location and output an Image or FileDescriptorSet.",
		Args:  cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container applog.Container) error {
				return run(ctx, container, flags, moduleReaderProvider)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	AsFileDescriptorSet bool
	ErrorFormat         string
	ExcludeImports      bool
	ExcludeSourceInfo   bool
	Files               []string
	Output              string
	Source              string
	SourceConfig        string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	imageinternal.BindAsFileDescriptorSet(flagSet, &f.AsFileDescriptorSet, asFileDescriptorSetFlagName)
	flagSet.StringVar(
		&f.ErrorFormat,
		errorFormatFlagName,
		"text",
		fmt.Sprintf(
			"The format for build errors, printed to stderr. Must be one of %s.",
			stringutil.SliceToString(bufanalysis.AllFormatStrings),
		),
	)
	imageinternal.BindExcludeImports(flagSet, &f.ExcludeImports, excludeImportsFlagName)
	imageinternal.BindExcludeSourceInfo(flagSet, &f.ExcludeSourceInfo, excludeSourceInfoFlagName)
	imageinternal.BindFiles(flagSet, &f.Files, filesFlagName)
	flagSet.StringVarP(
		&f.Output,
		outputFlagName,
		outputFlagShortName,
		"",
		fmt.Sprintf(
			`Required. The location to write the image to. Must be one of format %s.`,
			buffetch.ImageFormatsString,
		),
	)
	flagSet.StringVar(
		&f.Source,
		sourceFlagName,
		".",
		fmt.Sprintf(
			`The source or module to build. Must be one of format %s.`,
			buffetch.SourceOrModuleFormatsString,
		),
	)
	flagSet.StringVar(
		&f.SourceConfig,
		sourceConfigFlagName,
		"",
		`The config file or data to use.`,
	)
}

func run(
	ctx context.Context,
	container applog.Container,
	flags *flags,
	moduleReaderProvider bufcli.ModuleReaderProvider,
) error {
	if flags.Output == "" {
		return fmt.Errorf("--%s is required", outputFlagName)
	}
	sourceOrModuleRef, err := buffetch.NewSourceOrModuleRefParser(
		container.Logger(),
	).GetSourceOrModuleRef(ctx, flags.Source)
	if err != nil {
		return fmt.Errorf("--%s: %v", sourceFlagName, err)
	}
	moduleReader, err := moduleReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		sourceConfigFlagName,
		moduleReader,
		// must be source or module only
	).GetSourceOrModuleEnv(
		ctx,
		container,
		sourceOrModuleRef,
		flags.SourceConfig,
		flags.Files,
		false,
		flags.ExcludeSourceInfo,
	)
	if err != nil {
		return err
	}
	if len(fileAnnotations) > 0 {
		// stderr since we do output to stdout potentially
		if err := bufanalysis.PrintFileAnnotations(
			container.Stderr(),
			fileAnnotations,
			flags.ErrorFormat,
		); err != nil {
			return err
		}
		// app works on the concept that an error results in a non-zero exit code
		// we already printed the messages with PrintFileAnnotations so we do
		// not want to print any additional error message
		// we could put the FileAnnotations in this error, but in general with
		// linting/breaking change detection we actually print them to stdout
		// so doing this here is consistent with lint/breaking change detection
		return errors.New("")
	}
	imageRef, err := buffetch.NewImageRefParser(container.Logger()).GetImageRef(ctx, flags.Output)
	if err != nil {
		return fmt.Errorf("--%s: %v", outputFlagName, err)
	}
	return internal.NewBufwireImageWriter(
		container.Logger(),
	).PutImage(
		ctx,
		container,
		imageRef,
		env.Image(),
		flags.AsFileDescriptorSet,
		flags.ExcludeImports,
	)
}
