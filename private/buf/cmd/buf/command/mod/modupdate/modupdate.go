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

package modupdate

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/connect-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	onlyFlagName   = "only"
	bufTeamsRemote = "buf.team"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appflag.Builder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update a module's dependencies by updating the " + buflock.ExternalConfigFilePath + " file.",
		Long: "Fetch the latest digests for the specified references in the config file, " +
			"and write them and their transitive dependencies to the " +
			buflock.ExternalConfigFilePath +
			` file. The first argument is the directory of the local module to update. Defaults to "." if no argument is specified.`,
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
		"The name of the dependency to update. When set, only this dependency is updated (along with any of its sub-dependencies). May be passed multiple times.",
	)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appflag.Container,
	flags *flags,
) error {
	directoryInput, err := bufcli.GetInputValue(container, "", ".")
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

	remote := bufconnect.DefaultRemote
	if moduleConfig.ModuleIdentity != nil && moduleConfig.ModuleIdentity.Remote() != "" {
		remote = moduleConfig.ModuleIdentity.Remote()
	} else {
		for _, moduleReference := range moduleConfig.Build.DependencyModuleReferences {
			if strings.HasSuffix(moduleReference.Remote(), bufTeamsRemote) && !strings.HasSuffix(bufconnect.DefaultRemote, bufTeamsRemote) {
				warnMsg := fmt.Sprintf(
					`%q does not specify a "name", so Buf is defaulting to using remote %q for dependency resolution. This remote may be unable to resolve %q if it's an enterprise BSR module. Did you mean to specify a "name: %s/..." on this module?`,
					existingConfigFilePath,
					bufconnect.DefaultRemote,
					moduleReference.IdentityString(),
					moduleReference.Remote(),
				)
				container.Logger().Warn(warnMsg)
				break
			}
		}
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
		if connect.CodeOf(err) == connect.CodeUnimplemented && remote != bufconnect.DefaultRemote {
			return nil, bufcli.NewUnimplementedRemoteError(err, remote, moduleConfig.ModuleIdentity.IdentityString())
		}
		return nil, err
	}
	dependencyModulePins, err := bufmoduleref.NewModulePinsForProtos(protoDependencyModulePins...)
	if err != nil {
		return nil, bufcli.NewInternalError(err)
	}
	// We want to create one repository service per relevant remote.
	remoteToRepositoryService := make(map[string]registryv1alpha1connect.RepositoryServiceClient)
	remoteToDependencyModulePins := make(map[string][]bufmoduleref.ModulePin)
	for _, pin := range dependencyModulePins {
		if _, ok := remoteToRepositoryService[pin.Remote()]; !ok {
			remoteToRepositoryService[pin.Remote()] = connectclient.Make(apiProvider.ToClientConfig(), pin.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
		}
		remoteToDependencyModulePins[pin.Remote()] = append(remoteToDependencyModulePins[pin.Remote()], pin)
	}
	var allPinnedRepositories []*pinnedRepository
	for dependencyRemote, dependencyModulePins := range remoteToDependencyModulePins {
		repositoryService, ok := remoteToRepositoryService[dependencyRemote]
		if !ok {
			return nil, fmt.Errorf("a repository service is not available for %s", dependencyRemote)
		}
		dependencyFullNames := make([]string, len(dependencyModulePins))
		for i, pin := range dependencyModulePins {
			dependencyFullNames[i] = fmt.Sprintf("%s/%s", pin.Owner(), pin.Repository())
		}
		resp, err := repositoryService.GetRepositoriesByFullName(ctx,
			connect.NewRequest(&registryv1alpha1.GetRepositoriesByFullNameRequest{
				FullNames: dependencyFullNames,
			}))
		if err != nil {
			return nil, err
		}
		pinnedRepositories := make([]*pinnedRepository, len(dependencyModulePins))
		for i, modulePin := range dependencyModulePins {
			pinnedRepositories[i] = &pinnedRepository{
				modulePin:  modulePin,
				repository: resp.Msg.Repositories[i],
			}
		}
		allPinnedRepositories = append(allPinnedRepositories, pinnedRepositories...)
	}
	return allPinnedRepositories, nil
}

type pinnedRepository struct {
	modulePin  bufmoduleref.ModulePin
	repository *registryv1alpha1.Repository
}
