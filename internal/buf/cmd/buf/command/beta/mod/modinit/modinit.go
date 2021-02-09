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

package modinit

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	documentationCommentsFlagName = "doc"
	outDirPathFlagName            = "output"
	outDirPathFlagShortName       = "o"
	nameFlagName                  = "name"
	depFlagName                   = "dep"
	uncommentFlagName             = "uncomment"
)

// NewCommand returns a new init Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	deprecated string,
	hidden bool,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:        name,
		Short:      fmt.Sprintf("Initializes and writes a new %s configuration file.", bufconfig.ExternalConfigV1Beta1FilePath),
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
	DocumentationComments bool
	OutDirPath            string
	Name                  string
	Deps                  []string

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
		"Do not write inline documentation in the form of comments in the resulting configuration file.",
	)
	flagSet.StringVarP(
		&f.OutDirPath,
		outDirPathFlagName,
		outDirPathFlagShortName,
		".",
		`The directory to write the configuration file to.`,
	)
	flagSet.StringVar(
		&f.Name,
		nameFlagName,
		"",
		"The module name.",
	)
	flagSet.StringSliceVar(
		&f.Deps,
		depFlagName,
		nil,
		"The module dependencies.",
	)
	flagSet.BoolVar(
		&f.Uncomment,
		uncommentFlagName,
		false,
		"Uncomment examples in the resutling configuration file.",
	)
	_ = flagSet.MarkHidden(uncommentFlagName)
}

func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	if flags.OutDirPath == "" {
		return appcmd.NewInvalidArgumentErrorf("Flag --%s is required.", outDirPathFlagName)
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		flags.OutDirPath,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return err
	}
	exists, err := bufconfig.ConfigExists(ctx, readWriteBucket)
	if err != nil {
		return err
	}
	if exists {
		return appcmd.NewInvalidArgumentErrorf("%s already exists, not overwriting", bufconfig.ExternalConfigV1Beta1FilePath)
	}
	var writeConfigOptions []bufconfig.WriteConfigOption
	if flags.DocumentationComments {
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithDocumentationComments(),
		)
	}
	if flags.Name != "" {
		moduleIdentity, err := bufmodule.ModuleIdentityForString(flags.Name)
		if err != nil {
			return err
		}
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithModuleIdentity(moduleIdentity),
		)
	}
	if len(flags.Deps) > 0 {
		dependencyModuleReferences := make([]bufmodule.ModuleReference, len(flags.Deps))
		for i, dep := range flags.Deps {
			dependencyModuleReference, err := bufmodule.ModuleReferenceForString(dep)
			if err != nil {
				return err
			}
			dependencyModuleReferences[i] = dependencyModuleReference
		}
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithDependencyModuleReferences(dependencyModuleReferences...),
		)
	}
	if flags.Uncomment {
		writeConfigOptions = append(
			writeConfigOptions,
			bufconfig.WriteConfigWithUncomment(),
		)
	}
	return bufconfig.WriteConfig(
		ctx,
		readWriteBucket,
		writeConfigOptions...,
	)
}
