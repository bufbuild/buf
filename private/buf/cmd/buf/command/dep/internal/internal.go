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

package internal

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/buf/bufctl"
	"github.com/bufbuild/buf/private/buf/bufworkspace"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/app/appext"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// ModuleKeysAndTransitiveDepModuleKeysForModuleKeys gets the ModuleKeys for the
// ModuleRefs, and all the transitive dependencies.
func ModuleKeysAndTransitiveDepModuleKeysForModuleRefs(
	ctx context.Context,
	container appext.Container,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]bufmodule.ModuleKey, error) {
	moduleKeyProvider, err := bufcli.NewModuleKeyProvider(container)
	if err != nil {
		return nil, err
	}
	moduleKeys, err := moduleKeyProvider.GetModuleKeysForModuleRefs(
		ctx,
		moduleRefs,
		digestType,
	)
	if err != nil {
		return nil, err
	}
	return moduleKeysAndTransitiveDepModuleKeysForModuleKeys(ctx, container, moduleKeys)
}

// Prune prunes the buf.lock.
//
// Used by dep/mod prune.
func Prune(
	ctx context.Context,
	logger *zap.Logger,
	controller bufctl.Controller,
	// Contains all the Modules and their transitive dependencies based on the  buf.yaml.
	//
	// All dependencies must be within this group from RemoteDepsForModuleSet. If a dependency
	// is not within this group, this means it existed in the buf.lock from a previous buf dep update
	// call, but no longer is a declared remote dependency based on the current buf.yaml. In this
	// case, we error.
	//
	// This list is computed based on the result of ModuleKeysAndTransitiveDepModuleKeysForModuleRefs.
	bufYAMLBasedDepModuleKeys []bufmodule.ModuleKey,
	workspaceDepManager bufworkspace.WorkspaceDepManager,
	dirPath string,
) error {
	workspace, err := controller.GetWorkspace(ctx, dirPath, bufctl.WithIgnoreAndDisallowV1BufWorkYAMLs())
	if err != nil {
		return err
	}
	// Make sure the workspace builds.
	if _, err := controller.GetImageForWorkspace(
		ctx,
		workspace,
		bufctl.WithImageExcludeSourceInfo(true),
	); err != nil {
		return err
	}
	// Compute those dependencies that are in buf.yaml that are not used at all, and warn
	// about them.
	if err := LogUnusedConfiguredDepsForWorkspace(workspace, logger); err != nil {
		return err
	}
	// Step that actually computes remote dependencies based on imports. These are all
	// that is needed for buf.lock.
	depModules, err := bufmodule.RemoteDepsForModuleSet(workspace)
	if err != nil {
		return err
	}
	depModuleKeys, err := slicesext.MapError(
		depModules,
		func(remoteDep bufmodule.RemoteDep) (bufmodule.ModuleKey, error) {
			return bufmodule.ModuleToModuleKey(remoteDep, workspaceDepManager.BufLockFileDigestType())
		},
	)
	if err != nil {
		return err
	}
	if err := validateModuleKeysContains(bufYAMLBasedDepModuleKeys, depModuleKeys); err != nil {
		return err
	}
	return workspaceDepManager.UpdateBufLockFile(ctx, depModuleKeys)
}

// LogUnusedConfiugredDepsForWorkspace takes a workspace and logs the unused configured
// dependencies as warnings to the user.
func LogUnusedConfiguredDepsForWorkspace(
	workspace bufworkspace.Workspace,
	logger *zap.Logger,
) error {
	malformedDeps, err := bufworkspace.MalformedDepsForWorkspace(workspace)
	if err != nil {
		return err
	}
	for _, malformedDep := range malformedDeps {
		switch t := malformedDep.Type(); t {
		case bufworkspace.MalformedDepTypeUnused:
			logger.Sugar().Warnf(
				`Module %[1]s is declared in your buf.yaml deps but is unused. This command only modifies buf.lock files, not buf.yaml files. Please remove %[1]s from your buf.yaml deps if it is not needed.`,
				malformedDep.ModuleRef().ModuleFullName(),
			)
		default:
			return fmt.Errorf("unknown MalformedDepType: %v", t)
		}
	}
	return nil
}

// moduleKeysAndTransitiveDepModuleKeysForModuleKeys returns the ModuleKeys
// and all the transitive dependencies.
func moduleKeysAndTransitiveDepModuleKeysForModuleKeys(
	ctx context.Context,
	container appext.Container,
	moduleKeys []bufmodule.ModuleKey,
) ([]bufmodule.ModuleKey, error) {
	graphProvider, err := bufcli.NewGraphProvider(container)
	if err != nil {
		return nil, err
	}
	// Walk the graph to get all ModuleKeys including transitive dependencies.
	graph, err := graphProvider.GetGraphForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	var newModuleKeys []bufmodule.ModuleKey
	if err := graph.WalkNodes(
		func(moduleKey bufmodule.ModuleKey, _ []bufmodule.ModuleKey, _ []bufmodule.ModuleKey) error {
			newModuleKeys = append(newModuleKeys, moduleKey)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return newModuleKeys, nil
}

// validateModuleKeysContains validates that containingModuleKeys is a superset of moduleKeys.
//
// This is used by Prune to validate that bufYAMLBasedDepModuleKeys are a superset of RemoteDepsForModuleSet.
//
// See comment on Prune.
func validateModuleKeysContains(containingModuleKeys []bufmodule.ModuleKey, moduleKeys []bufmodule.ModuleKey) error {
	containingModuleFullNameStringToModuleKey, err := getModuleFullNameStringToModuleKey(containingModuleKeys)
	if err != nil {
		return syserror.Newf("validateModuleKeysContains: containingModuleKeys: %w", err)
	}
	moduleFullNameStringToModuleKey, err := getModuleFullNameStringToModuleKey(moduleKeys)
	if err != nil {
		return syserror.Newf("validateModuleKeysContains: moduleKeys: %w", err)
	}
	for moduleFullNameString := range moduleFullNameStringToModuleKey {
		if _, ok := containingModuleFullNameStringToModuleKey[moduleFullNameString]; !ok {
			return fmt.Errorf(
				`Module %s is detected to be a still-used dependency from your existing buf.lock, but is not a declared dependency in your buf.yaml deps, and is not a transitive dependency of any declared dependency. Add %s to your buf.yaml deps.`,
				moduleFullNameString,
				moduleFullNameString,
			)
		}
	}
	return nil
}

// All ModuleKeys are expected to be unique by ModuleFullName.
func getModuleFullNameStringToModuleKey(moduleKeys []bufmodule.ModuleKey) (map[string]bufmodule.ModuleKey, error) {
	return slicesext.ToUniqueValuesMap(
		moduleKeys,
		func(moduleKey bufmodule.ModuleKey) string {
			return moduleKey.ModuleFullName().String()
		},
	)
}
