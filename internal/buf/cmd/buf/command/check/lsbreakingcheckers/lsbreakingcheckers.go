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

package lsbreakingcheckers

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	checkinternal "github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/internal"
	"github.com/bufbuild/buf/internal/buf/cmd/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	allFlagName        = "all"
	categoriesFlagName = "category"
	configFlagName     = "config"
	formatFlagName     = "format"
)

// NewCommand returns a new Command
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List breaking checkers.",
		Args:  cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container applog.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	All        bool
	Categories []string
	Config     string
	Format     string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	checkinternal.BindLSCheckersAll(flagSet, &f.All, allFlagName)
	checkinternal.BindLSCheckersCategories(flagSet, &f.Categories, categoriesFlagName)
	checkinternal.BindLSCheckersConfig(flagSet, &f.Config, configFlagName)
	checkinternal.BindLSCheckersFormat(flagSet, &f.Format, formatFlagName)
}

func run(
	ctx context.Context,
	container applog.Container,
	flags *flags,
) error {
	var checkers []bufcheck.Checker
	var err error
	if flags.All {
		checkers, err = bufbreaking.GetAllCheckers(flags.Categories...)
		if err != nil {
			return err
		}
	} else {
		config, err := internal.NewBufwireConfigReader(
			container.Logger(),
			configFlagName,
		).GetConfig(
			ctx,
			flags.Config,
		)
		if err != nil {
			return err
		}
		checkers, err = config.Breaking.GetCheckers(flags.Categories...)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintCheckers(
		container.Stdout(),
		checkers,
		flags.Format,
	)
}
