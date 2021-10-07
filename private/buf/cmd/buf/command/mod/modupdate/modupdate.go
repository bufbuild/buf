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

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/bufpkg/bufrpc"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/rpc"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	onlyFlagName = "only"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update the modules dependencies. Updates the " + buflock.ExternalConfigFilePath + " file.",
		Long: "Gets the latest digests for the specified references in the config file, " +
			"and writes them and their transitive dependencies to the " +
			buflock.ExternalConfigFilePath +
			` file. The first argument is the directory of the local module to update. If no argument is specified, defaults to "."`,
		Args: cobra.MaximumNArgs(1),
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
	Only []string
}

func newFlags() *flags {
	return &flags{}
}

func (f *flags) Bind(flagSet *pflag.FlagSet) {
	flagSet.StringSliceVar(
		&f.Only,
		onlyFlagName,
		nil,
		"The name of a dependency to update. When used, only this dependency (and possibly its dependencies) will be updated. May be passed multiple times.",
	)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	directoryInput, err := bufcli.GetInputValue(container, "", "", "", ".")
	if err != nil {
		return err
	}
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	readWriteBucket, err := storageosProvider.NewReadWriteBucket(
		directoryInput,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	existingConfigFilePath, err := bufconfig.ExistingConfigFilePath(ctx, readWriteBucket)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	if existingConfigFilePath == "" {
		return bufcli.ErrNoConfigFile
	}
	moduleConfig, err := bufconfig.GetConfigForBucket(ctx, readWriteBucket)
	if err != nil {
		return err
	}

	remote := bufrpc.DefaultRemote
	if moduleConfig.ModuleIdentity != nil && moduleConfig.ModuleIdentity.Remote() != "" {
		remote = moduleConfig.ModuleIdentity.Remote()
	}

	pinnedRepositories, err := getDependencies(
		ctx,
		container,
		flags,
		remote,
		moduleConfig,
		readWriteBucket,
	)
	if err != nil {
		return err
	}

	dependencyModulePins := make([]bufmoduleref.ModulePin, len(pinnedRepositories))
	for i := range pinnedRepositories {
		dependencyModulePins[i] = pinnedRepositories[i].modulePin
		modulePin := pinnedRepositories[i].modulePin
		repository := pinnedRepositories[i].repository
		if !repository.Deprecated {
			continue
		}
		warnMsg := fmt.Sprintf(
			`Repository "%s/%s/%s" is deprecated`,
			modulePin.Remote(),
			modulePin.Owner(),
			modulePin.Repository(),
		)
		if repository.DeprecationMessage != "" {
			warnMsg = fmt.Sprintf("%s: %s", warnMsg, repository.DeprecationMessage)
		}
		container.Logger().Warn(warnMsg)
	}

	if err := bufmoduleref.PutDependencyModulePinsToBucket(ctx, readWriteBucket, dependencyModulePins); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}

func getDependencies(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
	remote string,
	moduleConfig *bufconfig.Config,
	readWriteBucket storage.ReadWriteBucket,
) ([]*pinnedRepository, error) {
	if len(moduleConfig.Build.DependencyModuleReferences) == 0 {
		return nil, nil
	}
	apiProvider, err := bufcli.NewRegistryProvider(ctx, container)
	if err != nil {
		return nil, err
	}
	service, err := apiProvider.NewResolveService(ctx, remote)
	if err != nil {
		return nil, err
	}
	var protoDependencyModuleReferences []*modulev1alpha1.ModuleReference
	var currentProtoModulePins []*modulev1alpha1.ModulePin
	if len(flags.Only) > 0 {
		referencesByIdentity := map[string]bufmoduleref.ModuleReference{}
		for _, reference := range moduleConfig.Build.DependencyModuleReferences {
			referencesByIdentity[reference.IdentityString()] = reference
		}
		for _, only := range flags.Only {
			moduleReference, ok := referencesByIdentity[only]
			if !ok {
				return nil, fmt.Errorf("%q is not a valid --only input: no such dependency in current module deps", only)
			}
			protoDependencyModuleReferences = append(protoDependencyModuleReferences, bufmoduleref.NewProtoModuleReferenceForModuleReference(moduleReference))
		}
		currentModulePins, err := bufmoduleref.DependencyModulePinsForBucket(ctx, readWriteBucket)
		if err != nil {
			return nil, fmt.Errorf("couldn't read current dependencies: %w", err)
		}
		currentProtoModulePins = bufmoduleref.NewProtoModulePinsForModulePins(currentModulePins...)
	} else {
		protoDependencyModuleReferences = bufmoduleref.NewProtoModuleReferencesForModuleReferences(
			moduleConfig.Build.DependencyModuleReferences...,
		)
	}
	protoDependencyModulePins, err := service.GetModulePins(
		ctx,
		protoDependencyModuleReferences,
		currentProtoModulePins,
	)
	if err != nil {
		if rpc.GetErrorCode(err) == rpc.ErrorCodeUnimplemented && remote != bufrpc.DefaultRemote {
			return nil, bufcli.NewUnimplementedRemoteError(err, remote, moduleConfig.ModuleIdentity.IdentityString())
		}
		return nil, err
	}
	dependencyModulePins, err := bufmoduleref.NewModulePinsForProtos(protoDependencyModulePins...)
	if err != nil {
		return nil, bufcli.NewInternalError(err)
	}
	repositoryService, err := apiProvider.NewRepositoryService(ctx, remote)
	if err != nil {
		return nil, err
	}
	dependencyFullNames := make([]string, len(dependencyModulePins))
	for i, pin := range dependencyModulePins {
		dependencyFullNames[i] = fmt.Sprintf("%s/%s", pin.Owner(), pin.Repository())
	}
	dependencyRepos, err := repositoryService.GetRepositoriesByFullName(ctx, dependencyFullNames)
	if err != nil {
		return nil, err
	}
	pinnedRepositories := make([]*pinnedRepository, len(dependencyFullNames))
	for i := range dependencyModulePins {
		pinnedRepositories[i] = &pinnedRepository{
			modulePin:  dependencyModulePins[i],
			repository: dependencyRepos[i],
		}
	}
	return pinnedRepositories, nil
}

type pinnedRepository struct {
	modulePin  bufmoduleref.ModulePin
	repository *registryv1alpha1.Repository
}
