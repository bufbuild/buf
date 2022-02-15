// Copyright 2020-2022 Buf Technologies, Inc.
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

package modlsbreakingrules

import (
	"context"
	"fmt"

	modinternal "github.com/bufbuild/buf/private/buf/cmd/buf/command/mod/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	allFlagName     = "all"
	configFlagName  = "config"
	formatFlagName  = "format"
	versionFlagName = "version"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "List breaking rules.",
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
	All     bool
	Config  string
	Format  string
	Version string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	modinternal.BindLSRulesAll(flagSet, &f.All, allFlagName)
	modinternal.BindLSRulesConfig(flagSet, &f.Config, configFlagName, allFlagName, versionFlagName)
	modinternal.BindLSRulesFormat(flagSet, &f.Format, formatFlagName)
	modinternal.BindLSRulesVersion(flagSet, &f.Version, versionFlagName, allFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if flags.All {
		// We explicitly document that if all is set, config is ignored.
		// If a user wants to override the version while using all, they should use version.
		flags.Config = ""
	}
	if flags.Version != "" {
		// If version is set, all is implied, and we use the config override to specify the
		// version that bufconfig.ReadConfig will return.
		flags.All = true
		// This also results in config being ignored per the documentation.
		flags.Config = fmt.Sprintf(`{"version":"%s"}`, flags.Version)
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	config, err := bufconfig.ReadConfigOS(
		ctx,
		readWriteBucket,
		bufconfig.ReadConfigOSWithOverride(flags.Config),
	)
	if err != nil {
		return err
	}
	var rules []bufcheck.Rule
	if flags.All {
		switch config.Version {
		case bufconfig.V1Beta1Version:
			rules, err = bufbreaking.GetAllRulesV1Beta1()
			if err != nil {
				return err
			}
		case bufconfig.V1Version:
			rules, err = bufbreaking.GetAllRulesV1()
			if err != nil {
				return err
			}
		}
	} else {
		rules, err = bufbreaking.RulesForConfig(config.Breaking)
		if err != nil {
			return err
		}
	}
	return bufcheck.PrintRules(
		container.Stdout(),
		rules,
		flags.Format,
	)
}
