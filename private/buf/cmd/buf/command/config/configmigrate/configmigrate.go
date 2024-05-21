// Copyright 2020-2024 Buf Technologies, Inc.
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

package configmigrate

import (
	"context"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufmigrate"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/pflag"
)

const (
	workspaceDirectoriesFlagName = "workspace"
	moduleDirectoriesFlagName    = "module"
	bufGenYAMLFilePathFlagName   = "buf-gen-yaml"
	diffFlagName                 = "diff"
	diffFlagShortName            = "d"
)

// ignoreDirPaths are paths we ignore when calling MigrateAll or DiffAll.
//
// This is just a hardcoded list for the 99% case. If users actually want to migrate files in one
// of these locations, they can use the flags, but typically we do not want to pick these up.
var ignoreDirPaths = []string{
	".git",
	".github",
}

// NewCommand returns a new Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: `Migrate all buf.yaml, buf.work.yaml, buf.gen.yaml, and buf.lock files at the specified directories or paths to v2`,
		Long: `If no flags are specified, the current directory is searched for buf.yamls, buf.work.yamls, and buf.gen.yamls.

The effects of this command may change over time `,
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
	WorkspaceDirPaths   []string
	ModuleDirPaths      []string
	BufGenYAMLFilePaths []string
	Diff                bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.WorkspaceDirPaths,
		workspaceDirectoriesFlagName,
		nil,
		"The workspace directories to migrate. buf.work.yaml, buf.yamls and buf.locks will be migrated",
	)
	flagSet.StringSliceVar(
		&f.ModuleDirPaths,
		moduleDirectoriesFlagName,
		nil,
		"The module directories to migrate. buf.yaml and buf.lock will be migrated",
	)
	flagSet.StringSliceVar(
		&f.BufGenYAMLFilePaths,
		bufGenYAMLFilePathFlagName,
		nil,
		"The paths to the buf.gen.yaml generation templates to migrate",
	)
	flagSet.BoolVarP(
		&f.Diff,
		diffFlagName,
		diffFlagShortName,
		false,
		"Write a diff to stdout instead of migrating files on disk. Useful for performing a dry run.",
	)
}

func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	runner := command.NewRunner()
	moduleKeyProvider, err := bufcli.NewModuleKeyProvider(container)
	if err != nil {
		return err
	}
	commitProvider, err := bufcli.NewCommitProvider(container)
	if err != nil {
		return err
	}
	bucket, err := storageos.NewProvider(storageos.ProviderWithSymlinks()).NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	migrator := bufmigrate.NewMigrator(
		container.Logger(),
		runner,
		moduleKeyProvider,
		commitProvider,
	)
	all := len(flags.WorkspaceDirPaths) == 0 && len(flags.ModuleDirPaths) == 0 && len(flags.BufGenYAMLFilePaths) == 0
	if flags.Diff {
		if all {
			return bufmigrate.DiffAll(
				ctx,
				migrator,
				bucket,
				container.Stdout(),
				ignoreDirPaths,
			)
		}
		return migrator.Diff(
			ctx,
			bucket,
			container.Stdout(),
			flags.WorkspaceDirPaths,
			flags.ModuleDirPaths,
			flags.BufGenYAMLFilePaths,
		)
	}
	if all {
		return bufmigrate.MigrateAll(
			ctx,
			migrator,
			bucket,
			ignoreDirPaths,
		)
	}
	return migrator.Migrate(
		ctx,
		bucket,
		flags.WorkspaceDirPaths,
		flags.ModuleDirPaths,
		flags.BufGenYAMLFilePaths,
	)
}
