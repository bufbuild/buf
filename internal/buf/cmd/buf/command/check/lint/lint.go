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

package lint

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
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
	errorFormatFlagName = "error-format"
	filesFlagName       = "file"
	inputFlagName       = "input"
	inputConfigFlagName = "input-config"
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
		Short: "Check that the input location passes lint checks.",
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
	ErrorFormat string
	Files       []string
	Input       string
	InputConfig string
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
			stringutil.SliceToString(buflint.AllFormatStrings),
		),
	)
	flagSet.StringSliceVar(
		&f.Files,
		filesFlagName,
		nil,
		`Limit to specific files. This is an advanced feature and is not recommended.`,
	)
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to lint. Must be one of format %s.`,
			buffetch.AllFormatsString,
		),
	)
	flagSet.StringVar(
		&f.InputConfig,
		inputConfigFlagName,
		"",
		`The config file or data to use.`,
	)
}

func run(
	ctx context.Context,
	container applog.Container,
	flags *flags,
	moduleReaderProvider bufcli.ModuleReaderProvider,
) (retErr error) {
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
		false,       // input files must exist
		false,       // we must include source info for linting
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
	fileAnnotations, err = internal.NewBuflintHandler(container.Logger()).Check(
		ctx,
		env.Config().Lint,
		bufimage.ImageWithoutImports(env.Image()),
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
