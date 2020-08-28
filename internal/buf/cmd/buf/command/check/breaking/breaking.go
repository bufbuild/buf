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
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	errorFormatFlagName        = "error-format"
	excludeImportsFlagName     = "exclude-imports"
	filesFlagName              = "file"
	limitToInputFilesFlagName  = "limit-to-input-files"
	inputFlagName              = "input"
	inputConfigFlagName        = "input-config"
	againstInputFlagName       = "against-input"
	againstInputConfigFlagName = "against-input-config"
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
		Short: "Check that the input location has no breaking changes compared to the against location.",
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
	ErrorFormat        string
	ExcludeImports     bool
	Files              []string
	LimitToInputFiles  bool
	Input              string
	InputConfig        string
	AgainstInput       string
	AgainstInputConfig string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
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
		&f.Input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to check for breaking changes. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The config file or data to use.`,
	)
	flagSet.StringVar(
		&f.AgainstInput,
		againstInputFlagName,
		"",
		fmt.Sprintf(
			`Required. The source or image to check against. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.AgainstInputConfig,
		againstInputConfigFlagName,
		"",
		`The config file or data to use for the against source or image.`,
	)
}

func run(
	ctx context.Context,
	container applog.Container,
	flags *flags,
	moduleReaderProvider bufcli.ModuleReaderProvider,
) error {
	if flags.AgainstInput == "" {
		return fmt.Errorf("--%s is required", againstInputFlagName)
	}
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, flags.Input)
	if err != nil {
		return fmt.Errorf("--%s: %v", inputFlagName, err)
	}
	moduleReader, err := moduleReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	env, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		inputConfigFlagName,
		moduleReader,
	).GetEnv(
		ctx,
		container,
		ref,
		flags.InputConfig,
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

	againstRef, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, flags.AgainstInput)
	if err != nil {
		return fmt.Errorf("--%s: %v", againstInputFlagName, err)
	}
	againstEnv, fileAnnotations, err := internal.NewBufwireEnvReader(
		container.Logger(),
		againstInputConfigFlagName,
		moduleReader,
	).GetEnv(
		ctx,
		container,
		againstRef,
		flags.AgainstInputConfig,
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
	fileAnnotations, err = internal.NewBufbreakingHandler(container.Logger()).Check(
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
