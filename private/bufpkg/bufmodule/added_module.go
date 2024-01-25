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

package bufmodule

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
)

// addedModule represents a Module that was added in moduleSetBuilder.
//
// It either represents a local Module, or a remote Module.
//
// This is needed because when we add a remote Module, we make
// a call out to the API to get the ModuleData by ModuleKey. However, if we are in
// a situation where we have a v1 workspace with named modules, but those modules
// do not actually exist in the BSR, and only in the workspace, AND we have a buf.lock
// that represents those modules, we don't want to actually do the work to retrieve
// the Module from the BSR, as in the end, the local Module in the workspace will win out
// when get deduplicated.
//
// Even if this weren't the case, we don't want to make unnecessary BSR calls. So, instead of
// making the call, we store the information that we will need to deduplicate, and once we've
// filtered out the modules we don't need, we actually create the remote Module. At this point,
// any modules that were both local (in the workspace) and remote (via a buf.lock) will have the
// buf.lock-added Modules filtered out, and no BSR call will be made.
type addedModule struct {
	localModule              Module
	remoteModuleKey          ModuleKey
	remoteTargetPaths        []string
	remoteTargetExcludePaths []string
	isTarget                 bool
}

func newLocalAddedModule(
	localModule Module,
	isTarget bool,
) *addedModule {
	return &addedModule{
		localModule: localModule,
		isTarget:    isTarget,
	}
}

func newRemoteAddedModule(
	remoteModuleKey ModuleKey,
	remoteTargetPaths []string,
	remoteTargetExcludePaths []string,
	isTarget bool,
) *addedModule {
	return &addedModule{
		remoteModuleKey:          remoteModuleKey,
		remoteTargetPaths:        remoteTargetPaths,
		remoteTargetExcludePaths: remoteTargetExcludePaths,
		isTarget:                 isTarget,
	}
}

// IsLocal returns true if the addedModule is a local Module.
func (a *addedModule) IsLocal() bool {
	return a.localModule != nil
}

// IsTarget returns true if the addedModule was targeted.
func (a *addedModule) IsTarget() bool {
	return a.isTarget
}

// OpaqueID returns the OpaqueID of the addedModule.
func (a *addedModule) OpaqueID() string {
	if a.remoteModuleKey != nil {
		return a.remoteModuleKey.ModuleFullName().String()
	}
	return a.localModule.OpaqueID()
}

// ToModule converts the addedModule to a Module.
//
// If the addedModule is a local Module, this is just returned.
// If the addedModule is a remote Module, the ModuleDataProvider is queried to get the Module.
func (a *addedModule) ToModule(
	ctx context.Context,
	logger *zap.Logger,
	moduleDataProvider ModuleDataProvider,
) (Module, error) {
	// If the addedModule is a local Module, just return it.
	if a.localModule != nil {
		return a.localModule, nil
	}
	// Else, get ther remote Module.
	moduleDatas, err := moduleDataProvider.GetModuleDatasForModuleKeys(
		ctx,
		[]ModuleKey{a.remoteModuleKey},
	)
	if err != nil {
		return nil, err
	}
	if len(moduleDatas) != 1 {
		return nil, syserror.Newf("expected 1 ModuleData, got %d", len(moduleDatas))
	}
	moduleData := moduleDatas[0]
	if moduleData.ModuleKey().ModuleFullName() == nil {
		return nil, syserror.New("got nil ModuleFullName for a ModuleKey returned from a ModuleDataProvider")
	}
	if a.remoteModuleKey.ModuleFullName().String() != moduleData.ModuleKey().ModuleFullName().String() {
		return nil, syserror.Newf(
			"mismatched ModuleFullName from ModuleDataProvider: input %q, output %q",
			a.remoteModuleKey.ModuleFullName().String(),
			moduleData.ModuleKey().ModuleFullName().String(),
		)
	}
	v1BufYAMLObjectData, err := moduleData.V1Beta1OrV1BufYAMLObjectData()
	if err != nil {
		return nil, err
	}
	v1BufLockObjectData, err := moduleData.V1Beta1OrV1BufLockObjectData()
	if err != nil {
		return nil, err
	}
	// TODO: normalize and validate all paths
	return newModule(
		ctx,
		logger,
		// ModuleData.Bucket has sync.OnceValues and getStorageMatchers applied since it can
		// only be constructed via NewModuleData.
		//
		// TODO: This is a bit shady.
		moduleData.Bucket,
		"",
		moduleData.ModuleKey().ModuleFullName(),
		moduleData.ModuleKey().CommitID(),
		a.isTarget,
		false,
		v1BufYAMLObjectData,
		v1BufLockObjectData,
		a.remoteTargetPaths,
		a.remoteTargetExcludePaths,
		"",
		false,
	)
}

// getUniqueSortedModulesByOpaqueID deduplicates and sorts the addedModule list.
//
// Modules that are targets are preferred, followed by Modules that are local.
// Otherwise, Modules earlier in the slice are preferred. Note that this means that if two
// remote non-target Modules are added for different Commit IDs, the one that was added
// first will be preferred (ie we are not doing any dependency resolution here).
//
// Duplication determined based opaqueID, that is if a Module has an equal
// opaqueID, it is considered a duplicate.
//
// We want to account for Modules with the same name but different digests, that is a dep in a workspace
// that has the same name as something in a buf.lock file, we prefer the local dep in the workspace.
//
// When returned, all modules have unique opaqueIDs and Digests.
//
// Note: Modules with the same ModuleFullName will automatically have the same commit and Digest after this,
// as there will be exactly one Module with a given ModuleFullName, given that an OpaqueID will be equal
// for Modules with equal ModuleFullNames.
func getUniqueSortedAddedModulesByOpaqueID(ctx context.Context, addedModules []*addedModule) ([]*addedModule, error) {
	// sort.SliceStable keeps equal elements in their original order, so this does
	// not affect the "earlier preferred" property.
	//
	// However, after this, we can really apply "earlier" preferred to denote "prefer targets over
	// non-targets, then prefer local over remote."
	sort.SliceStable(
		addedModules,
		func(i int, j int) bool {
			m1 := addedModules[i]
			m2 := addedModules[j]
			// If this ever comes up in the future: by preferring remote targets over local non-targets,
			// we are in a situation where we might have a local module, but we use the remote module
			// anyways, which leads to a BSR call we didn't want to make. See addedModule documentation.
			// We're making the bet that if we did add a remote target module, we had a good reason
			// to do so (i.e. we want that version of the module for some reason) so we're going
			// to prefer it.
			if m1.IsTarget() && !m2.IsTarget() {
				return true
			}
			if !m1.IsTarget() && m2.IsTarget() {
				return false
			}
			if m1.IsLocal() && !m2.IsLocal() {
				return true
			}
			// includes if !m1.IsLocal() && m2.IsLocal()
			return false
		},
	)
	// Digest *cannot* be used here - it's a chicken or egg problem. Computing the digest requires the cache,
	// the cache requires the unique Modules, the unique Modules require this function. This is OK though -
	// we want to add all Modules that we *think* are unique to the cache. If there is a duplicate, it
	// will be detected via cache usage.
	alreadySeenOpaqueIDs := make(map[string]struct{})
	uniqueAddedModules := make([]*addedModule, 0, len(addedModules))
	for _, addedModule := range addedModules {
		opaqueID := addedModule.OpaqueID()
		if opaqueID == "" {
			return nil, syserror.New("OpaqueID was empty which should never happen")
		}
		if _, ok := alreadySeenOpaqueIDs[opaqueID]; !ok {
			alreadySeenOpaqueIDs[opaqueID] = struct{}{}
			uniqueAddedModules = append(uniqueAddedModules, addedModule)
		}
	}
	sort.Slice(
		uniqueAddedModules,
		func(i int, j int) bool {
			return uniqueAddedModules[i].OpaqueID() < uniqueAddedModules[j].OpaqueID()
		},
	)
	return uniqueAddedModules, nil
}

// resolveModuleKeys gets the ModuleKey with the latest create time.
//
// All ModuleKeys expected to have the same ModuleFullName.
func resolveModuleKeys(
	ctx context.Context,
	commitProvider CommitProvider,
	moduleKeys []ModuleKey,
) (ModuleKey, error) {
	if len(moduleKeys) == 0 {
		return nil, syserror.New("expected at least one ModuleKey")
	}
	if len(moduleKeys) == 1 {
		return moduleKeys[0], nil
	}
	// Validate we're all within one registry for now.
	if moduleFullNameStrings := slicesext.ToUniqueSorted(
		slicesext.Map(
			moduleKeys,
			func(moduleKey ModuleKey) string { return moduleKey.ModuleFullName().String() },
		),
	); len(moduleFullNameStrings) > 1 {
		return nil, fmt.Errorf("multiple ModuleFullNames detected: %s", strings.Join(moduleFullNameStrings, ", "))
	}
	// Returned commits are in same order as input ModuleKeys
	commits, err := commitProvider.GetCommitsForModuleKeys(ctx, moduleKeys)
	if err != nil {
		return nil, err
	}
	createTime, err := commits[0].CreateTime()
	if err != nil {
		return nil, err
	}
	moduleKey := moduleKeys[0]
	// i+1 is index inside moduleKeys.
	//
	// Find the commit with the latest CreateTime, this is the ModuleKey you want to return.
	for i, commit := range commits[1:] {
		iCreateTime, err := commit.CreateTime()
		if err != nil {
			return nil, err
		}
		if iCreateTime.After(createTime) {
			moduleKey = moduleKeys[i+1]
			createTime = iCreateTime
		}
	}
	return moduleKey, nil
}
