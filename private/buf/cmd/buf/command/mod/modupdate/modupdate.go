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

package modupdate

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/spf13/pflag"
)

const (
	onlyFlagName = "only"
)

// NewCommand returns a new update Command.
func NewCommand(
	name string,
	builder appext.SubCommandBuilder,
) *appcmd.Command {
	flags := newFlags()
	return &appcmd.Command{
		Use:   name + " <directory>",
		Short: "Update a module's locked dependencies in buf.lock",
		Long: `Fetch the latest digests for the specified references in buf.yaml,
and write them and their transitive dependencies to buf.lock.

The first argument is the directory of the local module to update.
Defaults to "." if no argument is specified.

Note that updating is only allowed for v2 buf.yaml files. Run "buf migrate" to migrate to v2.`,
		Args: appcmd.MaximumNArgs(1),
		Run: builder.NewRunFunc(
			func(ctx context.Context, container appext.Container) error {
				return run(ctx, container, flags)
			},
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
		"The name of the dependency to update. When set, only this dependency and its transitive dependencies are updated. May be passed multiple times",
	)
}

// run update the buf.lock file for a specific module.
func run(
	ctx context.Context,
	container appext.Container,
	flags *flags,
) error {
	dirPath := "."
	if container.NumArgs() > 0 {
		dirPath = container.Arg(0)
	}
	controller, err := bufcli.NewController(container)
	if err != nil {
		return err
	}
	moduleKeyProvider, err := bufcli.NewModuleKeyProvider(container)
	if err != nil {
		return err
	}
	updateableWorkspace, err := controller.GetUpdateableWorkspace(ctx, dirPath)
	if err != nil {
		return err
	}
	onlyNameMap, err := getOnlyNameMapForModuleSet(updateableWorkspace, flags.Only)
	if err != nil {
		return err
	}
	remoteDeps, err := bufmodule.RemoteDepsForModuleSet(updateableWorkspace)
	if err != nil {
		return err
	}
	// All the ModuleKeys we get from the current dependency list in the workspace.
	// This includes transitive dependencies.
	remoteDepNameToModuleKey, err := getModuleFullNameToModuleKey(remoteDeps)
	if err != nil {
		return err
	}
	// All the ModuleKeys we get back from buf.yaml.
	//
	// ModuleKeyProvider provides updated references for all the ModuleKeys.
	bufYAMLUpdatedModuleKeys, err := moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		updateableWorkspace.ConfiguredDepModuleRefs(),
	)
	if err != nil {
		return err
	}
	bufYAMLNameToUpdatedModuleKey, err := slicesext.ToUniqueValuesMapError(
		bufYAMLUpdatedModuleKeys,
		func(moduleKey bufmodule.ModuleKey) (string, error) {
			return moduleKey.ModuleFullName().String(), nil
		},
	)
	if err != nil {
		return err
	}
	for bufYAMLName, updatedModuleKey := range bufYAMLNameToUpdatedModuleKey {
		if _, ok := remoteDepNameToModuleKey[bufYAMLName]; !ok {
			// In updated list from buf.yaml, but not in dependency list.
			//
			// This is an unused module.
			//
			// Delete from our update map, we won't write this to buf.lock
			delete(bufYAMLNameToUpdatedModuleKey, bufYAMLName)
			// We determine if its unused because its local, or if because it is not
			// a dependency at all.
			module := updateableWorkspace.GetModuleForModuleFullName(updatedModuleKey.ModuleFullName())
			if module == nil || !module.IsLocal() {
				container.Logger().Sugar().Warnf("%s is specified in buf.yaml but is not used", bufYAMLName)
			} else { // module.IsLocal()
				container.Logger().Sugar().Warnf("%s is specified in buf.yaml but is within the workspace, so does not need to be specified in your deps", bufYAMLName)
			}
		}
	}

	isTransitveRemoteDep, err := getIsTransitiveRemoteDepFunc(remoteDeps)
	if err != nil {
		return err
	}
	// Our result buf.lock needs to have everything in deps. We will only use the new values from bufYAMLNameToModuleKey
	// if either (1) onlyNameMap is empty (2) they are within onlyNameMap, AND they are a remote dependency.
	//
	// Note we deleted unused dependencies from bufYAMLNameToModuleKey above.
	for remoteDepName := range remoteDepNameToModuleKey {
		updatedModuleKey, ok := bufYAMLNameToUpdatedModuleKey[remoteDepName]
		if ok {
			if len(onlyNameMap) > 0 {
				if _, ok := onlyNameMap[remoteDepName]; ok {
					// This was a dependency (or transitive dependency) in --only. Update.
					remoteDepNameToModuleKey[remoteDepName] = updatedModuleKey
				}
			} else {
				// We didn't specify --only. Update indiscriminately.
				remoteDepNameToModuleKey[remoteDepName] = updatedModuleKey
			}
		} else {
			// This was in our deps list but was not specified in buf.yaml. Check if it was only transitive dependency.
			// If so, we're fine. If not, we should error, as this means it was unspecified in buf.yaml as of now (but
			// was at some point in the past), but we require it.
			//
			// Note if something wasn't PREVIOUSLY specified in our buf.lock, we would have failed on the building
			// of the workspace, as we just wouldn't have a dep.
			if isTransitveRemoteDep(remoteDepName) {
				return fmt.Errorf("previously present dependency %q is not longer specified in buf.yaml but is still depended on", remoteDepName)
			}
		}
	}
	// NewBufLockFile will sort the deps.
	bufLockFile, err := bufconfig.NewBufLockFile(bufconfig.FileVersionV2, slicesext.MapValuesToSlice(remoteDepNameToModuleKey))
	if err != nil {
		return err
	}
	return updateableWorkspace.PutBufLockFile(ctx, bufLockFile)
}

// Returns a function that returns true if the named module is a transitive remote
// dependency of the Workspace.
func getIsTransitiveRemoteDepFunc(remoteDeps []bufmodule.RemoteDep) (func(name string) bool, error) {
	transitiveRemoteDeps := slicesext.Filter(
		remoteDeps,
		func(remoteDep bufmodule.RemoteDep) bool {
			return !remoteDep.IsDirect()
		},
	)
	transitiveRemoteDepNameToModuleKey, err := getModuleFullNameToModuleKey(transitiveRemoteDeps)
	if err != nil {
		return nil, err
	}
	return func(name string) bool {
		_, ok := transitiveRemoteDepNameToModuleKey[name]
		return ok
	}, nil
}

// Returns a map from name to ModuleKey.
//
// Can be used for Modules, ModuleDeps, RemoteDeps.
//
// Expects al Modules to have a ModuleFullName and CommitID. This cannot be used
// for local Modules.
//
// All the ModuleKeys we get from the current dependency list in the workspace.
// This includes transitive dependencies.
func getModuleFullNameToModuleKey[T bufmodule.Module, S ~[]T](values S) (map[string]bufmodule.ModuleKey, error) {
	moduleKeys, err := slicesext.MapError(
		values,
		func(module T) (bufmodule.ModuleKey, error) {
			if module.IsLocal() {
				return nil, syserror.New("cannot pass local Modules to getModuleFullNameToModuleKey")
			}
			return bufmodule.ModuleToModuleKey(module)
		},
	)
	if err != nil {
		return nil, err
	}
	return slicesext.ToUniqueValuesMapError(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) (string, error) {
			return moduleKey.ModuleFullName().String(), nil
		},
	)
}

// Returns the dependencies and transitive dependencies to be updated.
//
// Returns nil if onlyModuleFullNames was empty.
func getOnlyNameMapForModuleSet(
	moduleSet bufmodule.ModuleSet,
	onlyNames []string,
) (map[string]struct{}, error) {
	if len(onlyNames) == 0 {
		return nil, nil
	}
	onlyModuleFullNames := make([]bufmodule.ModuleFullName, len(onlyNames))
	for i, onlyName := range onlyNames {
		onlyModuleFullName, err := bufmodule.ParseModuleFullName(onlyName)
		if err != nil {
			return nil, appcmd.NewInvalidArgumentErrorf("--%s value %q is not a valid module name", onlyFlagName, onlyName)
		}
		onlyModuleFullNames[i] = onlyModuleFullName
	}
	onlyNameMap := make(map[string]struct{}, len(onlyModuleFullNames))
	for _, onlyModuleFullName := range onlyModuleFullNames {
		module := moduleSet.GetModuleForModuleFullName(onlyModuleFullName)
		if module == nil {
			return nil, appcmd.NewInvalidArgumentErrorf("--%s value %q does not represent a dependency of this workspace", onlyFlagName, onlyModuleFullName.String())
		}
		onlyNameMap[onlyModuleFullName.String()] = struct{}{}
		moduleDeps, err := module.ModuleDeps()
		if err != nil {
			return nil, err
		}
		// ModuleDeps are transitive.
		for _, moduleDep := range moduleDeps {
			if depModuleFullName := moduleDep.ModuleFullName(); depModuleFullName != nil {
				onlyNameMap[depModuleFullName.String()] = struct{}{}
			} else if !moduleDep.IsLocal() {
				// This is a system error, this should not happen. This is just a sanity check.
				return nil, syserror.Newf("module %s was remote but did not have a name", moduleDep.OpaqueID())
			}
		}
	}
	return onlyNameMap, nil
}
