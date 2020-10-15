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
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	checkinternal "github.com/bufbuild/buf/internal/buf/cmd/buf/command/check/internal"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	allFlagName        = "all"
	categoriesFlagName = "category"
	configFlagName     = "config"
	formatFlagName     = "format"
)

// NewCommand returns a new Command.
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
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	All bool
	// TODO: remove for v1.0
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
	checkinternal.BindLSCheckersConfig(flagSet, &f.Config, configFlagName, allFlagName)
	checkinternal.BindLSCheckersFormat(flagSet, &f.Format, formatFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if err := checkinternal.CheckLSCheckersCategories(flags.Categories, categoriesFlagName); err != nil {
		return err
	}
	var checkers []bufcheck.Checker
	var err error
	if flags.All {
		checkers, err = bufbreaking.GetAllCheckersV1Beta1()
		if err != nil {
			return err
		}
	} else {
		readWriteBucket, err := storageos.NewReadWriteBucket(".")
		if err != nil {
			return err
		}
		config, err := bufconfig.ReadConfig(
			ctx,
			bufconfig.NewProvider(container.Logger()),
			readWriteBucket,
			flags.Config,
		)
		if err != nil {
			return err
		}
		checkers = config.Breaking.GetCheckers()
	}
	return bufcheck.PrintCheckers(
		container.Stdout(),
		checkers,
		flags.Format,
	)
}
