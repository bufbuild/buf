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

package modinit

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	documentationCommentsFlagName = "doc"
	outDirPathFlagName            = "output"
	outDirPathFlagShortName       = "o"
	uncommentFlagName             = "uncomment"
)

// NewCommand returns a new init Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " [buf.build/owner/foobar]",
		Short: fmt.Sprintf("Initializes and writes a new %s configuration file.", bufconfig.ExternalConfigV1FilePath),
		Args:  cobra.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	DocumentationComments bool
	OutDirPath            string

	// Hidden.
	// Just used for generating docs.buf.build.
	Uncomment bool
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(
		&f.DocumentationComments,
		documentationCommentsFlagName,
		false,
		"Write inline documentation in the form of comments in the resulting configuration file",
	)
	flagSet.StringVarP(
		&f.OutDirPath,
		outDirPathFlagName,
		outDirPathFlagShortName,
		".",
		`The directory to write the configuration file to`,
	)
	flagSet.BoolVar(
		&f.Uncomment,
		uncommentFlagName,
		false,
		"Uncomment examples in the resulting configuration file",
	)
	_ = flagSet.MarkHidden(uncommentFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if flags.OutDirPath == "" {
		return appcmd.NewInvalidArgumentErrorf("required flag %q not set", outDirPathFlagName)
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		flags.OutDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	existingConfigFilePath, err := bufconfig.ExistingConfigFilePath(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	if existingConfigFilePath != "" {
		return fmt.Errorf("%s already exists, not overwriting", existingConfigFilePath)
	}
	var writeConfigOptions []bufconfig.WriteConfigOption
	if container.NumArgs() > 0 {
		moduleIdentity, err := bufmoduleref.ModuleIdentityForString(container.Arg(0))
		if err != nil {
			return err
		}
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithModuleIdentity(moduleIdentity),
		)
	}
	if flags.DocumentationComments {
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithDocumentationComments(),
		)
	}
	if flags.Uncomment {
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithUncomment(),
		)
	}
	// Need to include the default version (v1), lint config, and breaking config.
	version := bufconfig.V1Version
	writeConfigOptions = append(
		writeConfigOptions,
		bufconfig.WriteConfigWithVersion(version),
	)
	writeConfigOptions = append(
		writeConfigOptions,
		bufconfig.WriteConfigWithBreakingConfig(
			&bufbreakingconfig.Config{
				Version: version,
				Use:     []string{"FILE"},
			},
		),
	)
	writeConfigOptions = append(
		writeConfigOptions,
		bufconfig.WriteConfigWithLintConfig(
			&buflintconfig.Config{
				Version: version,
				Use:     []string{"DEFAULT"},
			},
		),
	)
	return bufconfig.WriteConfig(
		ctx,
		readWriteBucket,
		writeConfigOptions...,
	)
}
