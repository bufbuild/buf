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

package modupdate

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/connectclient"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/stringutil"
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
		Short: "Update a module's dependencies by updating the " + buflock.ExternalConfigFilePath + " file",
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
		"The name of the dependency to update. When set, only this dependency is updated (along with any of its sub-dependencies). May be passed multiple times",
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
	clientConfig, err := bufcli.NewConnectClientConfig(container)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	pinnedRepositories, err := getDependencies(
		ctx,
		clientConfig,
		container,
		flags,
		moduleConfig,
		readWriteBucket,
		existingConfigFilePath,
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
	// Before updating buf.lock file, verify that existing dependency digests didn't change for the same commit.
	if err := bufmoduleref.ValidateModulePinsConsistentDigests(ctx, readWriteBucket, dependencyModulePins); err != nil {
		if bufmoduleref.IsDigestChanged(err) {
			return err
		}
		return bufcli.NewInternalError(err)
	}
	// Before updating buf.lock file, verify that no file path exists in more than one module.
	pathToModuleIdentityStrings := make(map[string][]string)
	currentModule, err := bufmodule.NewModuleForBucket(ctx, readWriteBucket)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	currentModuleIdentityString := "the current module"
	if currentModuleIdentity := currentModule.ModuleIdentity(); currentModuleIdentity != nil {
		currentModuleIdentityString = currentModuleIdentity.IdentityString()
	}
	currentModuleSourceFileInfos, err := currentModule.SourceFileInfos(ctx)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	for _, sourceFileInfo := range currentModuleSourceFileInfos {
		path := sourceFileInfo.Path()
		pathToModuleIdentityStrings[path] = append(pathToModuleIdentityStrings[path], currentModuleIdentityString)
	}
	moduleReader, err := bufcli.NewModuleReaderAndCreateCacheDirs(container, clientConfig)
	if err != nil {
		return bufcli.NewInternalError(err)
	}
	for _, modulePin := range dependencyModulePins {
		module, err := moduleReader.GetModule(ctx, modulePin)
		if err != nil {
			return bufcli.NewInternalError(err)
		}
		sourceFileInfos, err := module.SourceFileInfos(ctx)
		if err != nil {
			return bufcli.NewInternalError(err)
		}
		for _, sourceFileInfo := range sourceFileInfos {
			path := sourceFileInfo.Path()
			pathToModuleIdentityStrings[path] = append(pathToModuleIdentityStrings[path], modulePin.IdentityString())
		}
	}
	for path, moduleIdentityStrings := range pathToModuleIdentityStrings {
		if len(moduleIdentityStrings) > 1 {
			explanation := "Multiple files with the same path are not allowed because Protobuf files import each other by their paths, and each import path must uniquely identify a file."
			return fmt.Errorf("%s is found in multiple modules: %s\n%s", path, stringutil.SliceToHumanString(moduleIdentityStrings), explanation)
		}
	}
	if err := bufmoduleref.PutDependencyModulePinsToBucket(ctx, readWriteBucket, dependencyModulePins); err != nil {
		return bufcli.NewInternalError(err)
	}
	return nil
}

func getDependencies(
	ctx context.Context,
	clientConfig *connectclient.Config,
	container appflag.Container,
	flags *flags,
	moduleConfig *bufconfig.Config,
	readWriteBucket storage.ReadWriteBucket,
	existingConfigFilePath string,
) ([]*pinnedRepository, error) {
	if len(moduleConfig.Build.DependencyModuleReferences) == 0 {
		return nil, nil
	}
	var (
		remote               string
		moduleIdentityString string
	)
	if moduleConfig.ModuleIdentity != nil && moduleConfig.ModuleIdentity.Remote() != "" {
		remote = moduleConfig.ModuleIdentity.Remote()
		moduleIdentityString = moduleConfig.ModuleIdentity.IdentityString()
	} else {
		// At this point we know there's at least one dependency. If it's an unnamed module, select
		// the right remote from the list of dependencies.
		selectedRef := bufcli.SelectReferenceForRemote(moduleConfig.Build.DependencyModuleReferences)
		if selectedRef == nil {
			return nil, fmt.Errorf(`File %q has invalid "deps" references`, existingConfigFilePath)
		}
		remote = selectedRef.Remote()
		moduleIdentityString = selectedRef.IdentityString()
		container.Logger().Debug(fmt.Sprintf(
			`File %q does not specify the "name" field. Based on the dependency %q, it appears that you are using a BSR instance at %q. Did you mean to specify "name: %s/..." within %q?`,
			existingConfigFilePath,
			moduleIdentityString,
			remote,
			remote,
			existingConfigFilePath,
		))
	}
	service := connectclient.Make(clientConfig, remote, registryv1alpha1connect.NewResolveServiceClient)
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
	resp, err := service.GetModulePins(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetModulePinsRequest{
			ModuleReferences:  protoDependencyModuleReferences,
			CurrentModulePins: currentProtoModulePins,
		}),
	)
	if err != nil {
		if !connect.IsWireError(err) && remote != bufconnect.DefaultRemote {
			return nil, bufcli.NewInvalidRemoteError(err, remote, moduleIdentityString)
		}
		return nil, err
	}
	dependencyModulePins, err := bufmoduleref.NewModulePinsForProtos(resp.Msg.ModulePins...)
	if err != nil {
		return nil, bufcli.NewInternalError(err)
	}
	// We want to create one repository service per relevant remote.
	remoteToRepositoryService := make(map[string]registryv1alpha1connect.RepositoryServiceClient)
	remoteToDependencyModulePins := make(map[string][]bufmoduleref.ModulePin)
	for _, pin := range dependencyModulePins {
		if _, ok := remoteToRepositoryService[pin.Remote()]; !ok {
			remoteToRepositoryService[pin.Remote()] = connectclient.Make(clientConfig, pin.Remote(), registryv1alpha1connect.NewRepositoryServiceClient)
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
