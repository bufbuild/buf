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

package lsfiles

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
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
		Short: "List all Protobuf files for the input location.",
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
	Input       string
	InputConfig string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Input,
		inputFlagName,
		".",
		fmt.Sprintf(
			`The source or image to list the files from. Must be one of format %s.`,
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
) error {
	ref, err := buffetch.NewRefParser(container.Logger()).GetRef(ctx, flags.Input)
	if err != nil {
		return fmt.Errorf("--%s: %v", inputFlagName, err)
	}
	moduleReader, err := moduleReaderProvider.GetModuleReader(ctx, container)
	if err != nil {
		return err
	}
	fileRefs, err := internal.NewBufwireEnvReader(
		container.Logger(),
		inputConfigFlagName,
		moduleReader,
	).ListFiles(
		ctx,
		container,
		ref,
		flags.InputConfig,
	)
	if err != nil {
		return err
	}
	for _, fileRef := range fileRefs {
		if _, err := fmt.Fprintln(container.Stdout(), fileRef.ExternalPath()); err != nil {
			return err
		}
	}
	return nil
}
