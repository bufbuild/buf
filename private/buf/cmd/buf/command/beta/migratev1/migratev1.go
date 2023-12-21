// Copyright 2020-2023 Buf Technologies, Inc.
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

package migratev1

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufmigrate"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/pflag"
)

const (
	// TODO: buf-work-yaml or bufwork-yaml?
	bufWorkYAMLPathFlagName = "buf-work-yaml"
	bufYAMLPathsFlagName    = "buf-yaml"
	dryRunFlagName          = "dry-run"
)

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		// TODO: update if we are taking a directory as input
		Use:   name,
		Short: `Migrate configuration to the latest version`,
		Long: `Migrate any v1 and v1beta1 configuration files in the directory to the latest version.
Defaults to the current directory if not specified.`,
		// TODO: update if we are taking a directory as input
		Args: appcmd.MaximumNArgs(0),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	BufWorkYAMLPath string
	BufYAMLPaths    []string
	DryRun          bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.BufWorkYAMLPath,
		bufWorkYAMLPathFlagName,
		"",
		"The buf.work.yaml to migrate",
	)
	flagSet.StringSliceVar(
		&f.BufYAMLPaths,
		bufYAMLPathsFlagName,
		nil,
		"The buf.yamls to migrate. Must not be included by the workspace specified",
	)
	flagSet.BoolVar(
		&f.DryRun,
		dryRunFlagName,
		false,
		"Print the changes to be made, without writing to the disk",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	var migrateOptions []bufmigrate.MigrateOption
	if flags.BufWorkYAMLPath != "" {
		option, err := bufmigrate.MigrateBufWorkYAMLFile(flags.BufWorkYAMLPath)
		if err != nil {
			return err
		}
		migrateOptions = append(migrateOptions, option)
	}
	if len(flags.BufYAMLPaths) > 0 {
		option, err := bufmigrate.MigrateBufYAMLFile(flags.BufYAMLPaths)
		if err != nil {
			return err
		}
		migrateOptions = append(migrateOptions, option)
	}
	if flags.DryRun {
		migrateOptions = append(migrateOptions, bufmigrate.MigrateAsDryRun(container.Stdout()))
	}
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return err
	}
	return bufmigrate.Migrate(
		ctx,
		bufapi.NewClientProvider(clientConfig),
		// TODO: Do we want to add a flag --disable-symlinks?
		storageos.NewProvider(storageos.ProviderWithSymlinks()),
		migrateOptions...,
	)
}
