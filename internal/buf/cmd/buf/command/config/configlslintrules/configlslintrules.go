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

package configlslintrules

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcheck"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	configinternal "github.com/bufbuild/buf/internal/buf/cmd/buf/command/config/internal"
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
	deprecated string,
	hidden bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name,
		Short:      "List lint rules.",
		Args:       cobra.NoArgs,
		Deprecated: deprecated,
		Hidden:     hidden,
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
	configinternal.BindLSRulesAll(flagSet, &f.All, allFlagName)
	configinternal.BindLSRulesCategories(flagSet, &f.Categories, categoriesFlagName)
	configinternal.BindLSRulesConfig(flagSet, &f.Config, configFlagName, allFlagName)
	configinternal.BindLSRulesFormat(flagSet, &f.Format, formatFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if err := configinternal.CheckLSRulesCategories(flags.Categories, categoriesFlagName); err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	var rules []bufcheck.Rule
	var err error
	if flags.All {
		rules, err = buflint.GetAllRulesV1Beta1()
		if err != nil {
			return err
		}
	} else {
		readWriteBucket, err := storageosProvider.NewReadWriteBucket(
			".",
			storageos.ReadWriteBucketWithSymlinksIfSupported(),
		)
		if err != nil {
			return err
		}
		config, err := bufconfig.ReadConfig(
			ctx,
			bufconfig.NewProvider(container.Logger()),
			readWriteBucket,
			bufconfig.ReadConfigWithOverride(flags.Config),
		)
		if err != nil {
			return err
		}
		rules = config.Lint.GetRules()
	}
	return bufcheck.PrintRules(
		container.Stdout(),
		rules,
		flags.Format,
	)
}
