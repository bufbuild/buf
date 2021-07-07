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

package modupdate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/buflock"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/rpc"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	dirFlagName   = "dir"
	defaultRemote = "buf.build"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appflag.Builder,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name,
		Short: "Update the modules dependencies. Updates the " + buflock.ExternalConfigFilePath + " file.",
		Long: "Gets the latest digests for the specified branches in the config file, " +
			"and writes them and their transitive dependencies to the " +
			buflock.ExternalConfigFilePath +
			" file.",
		Args: cobra.NoArgs,
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appflag.Container) error {
				return run(ctx, container, flags, moduleResolverReaderProvider)
			},
			bufcli.NewErrorInterceptor(),
		),
		BindFlags: flags.Bind,
	}
}

type flags struct {
	// for testing only
	Dir string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringVar(
		&f.Dir,
		dirFlagName,
		".",
		"The directory to operate in. For testing only.",
	)
	_ = flagSet.MarkHidden(dirFlagName)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	moduleResolverReaderProvider bufcli.ModuleResolverReaderProvider,
) error {
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		flags.Dir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	exists, err := bufconfig.ConfigExists(ctx, readWriteBucket)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	if !exists {
		return bufcli.ErrNoConfigFile
	}
	moduleConfig, err := bufconfig.NewProvider(container.Logger()).GetConfig(ctx, readWriteBucket)
	if err != nil {
		return err
	}

	remote := defaultRemote
	if moduleConfig.ModuleIdentity != nil && moduleConfig.ModuleIdentity.Remote() != "" {
		remote = moduleConfig.ModuleIdentity.Remote()
	}

	var dependencyModulePins []bufmodule.ModulePin
	if len(moduleConfig.Build.DependencyModuleReferences) != 0 {
		apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
		if err != nil {
			return err
		}
		service, err := apiProvider.NewResolveService(ctx, remote)
		if err != nil {
			return err
		}
		protoDependencyModuleReferences := bufmodule.NewProtoModuleReferencesForModuleReferences(
			moduleConfig.Build.DependencyModuleReferences...,
		)
		protoDependencyModulePins, err := service.GetModulePins(ctx, protoDependencyModuleReferences)
		if err != nil {
			if rpc.GetErrorCode(err) == rpc.ErrorCodeUnimplemented && remote != defaultRemote {
				return fmt.Errorf("%w. Are you sure %q (derived from module name %q) is a Buf Schema Registry?", err, remote, moduleConfig.ModuleIdentity.IdentityString())
			}
			return err
		}
		dependencyModulePins, err = bufmodule.NewModulePinsForProtos(protoDependencyModulePins...)
		if err != nil {
			return bufcli.NewInternalError(err)
		}
	}
	module, err := bufmodule.NewModuleForBucketWithDependencyModulePins(
		ctx,
		readWriteBucket,
		dependencyModulePins,
	)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	if err := bufmodule.PutModuleDependencyModulePinsToBucket(ctx, readWriteBucket, module); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}
