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
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syncext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
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
// If the addedModule is a remote Module, the ModuleDataProvider and CommitProvider are queried to get the Module.
func (a *addedModule) ToModule(
	ctx context.Context,
	moduleDataProvider ModuleDataProvider,
	commitProvider CommitProvider,
) (Module, error) {
	// If the addedModule is a local Module, just return it.
	if a.localModule != nil {
		return a.localModule, nil
	}
	// Else, get the remote Module.
	getModuleData := syncext.OnceValues(
		func() (ModuleData, error) {
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
			return moduleData, nil
		},
	)
	getBucket := syncext.OnceValues(
		func() (storage.ReadBucket, error) {
			moduleData, err := getModuleData()
			if err != nil {
				return nil, err
			}
			// ModuleData.Bucket has sync.OnceValues and getStorageMatchers applied since it can
			// only be constructed via NewModuleData.
			//
			// TODO FUTURE: This is a bit shady.
			return moduleData.Bucket()
		},
	)
	getV1BufYAMLObjectData := func() (ObjectData, error) {
		moduleData, err := getModuleData()
		if err != nil {
			return nil, err
		}
		return moduleData.V1Beta1OrV1BufYAMLObjectData()
	}
	getV1BufLockObjectData := func() (ObjectData, error) {
		moduleData, err := getModuleData()
		if err != nil {
			return nil, err
		}
		return moduleData.V1Beta1OrV1BufLockObjectData()
	}
	// Imagine the following scenario:
	//
	//   module-a (local)
	//     README.md
	//     a.proto
	//     b.proto
	//   module-b:c1
	//     README.md
	//     c.proto
	//     d.proto
	//   module-b:c2
	//     README.md
	//     c.proto
	//     d.proto
	//   module-c:c1
	//     e.proto
	//     f.proto
	//   module-c:c2
	//     e.proto
	//     f.proto
	//     g.proto
	//
	// Then, you have this dependency graph:
	//
	// module-a -> module-b:c1, module-c:c2
	// module-b:c1 -> module-c:c1
	//
	// Note that module-b depends on an earlier commit of module-c than module-a does.
	//
	// If we were to just use the dependencies in the ModuleSet to compute the digest, the following
	// would happen as a calculation:
	//
	//   DIGEST(module-a) = digest(
	//     // module-a contents
	//     README.md,
	//     a.proto,
	//     b.proto,
	//     // module-b:c1 digest
	//     DIGEST(
	//       README.md,
	//       c.proto,
	//       d.proto,
	//       // module-c:c2 digest
	//       DIGEST(
	//         README.md,
	//         e.proto,
	//         f.proto,
	//         g.proto,
	//       ),
	//     ),
	//     // module-c:c2 digest
	//     DIGEST(
	//         README.md,
	//         e.proto,
	//         f.proto,
	//         g.proto,
	//     ),
	//   )
	//
	// Note that to compute the digest of module-b:c1, we would actually use the digest of
	// module-c:c2, as opposed to module-c:c1, since within the ModuleSet, we would resolve
	// to use module-c:c2 instead of module-c:c1.
	//
	// We should be using module-c:c1 to compute the digest of module-b:c1:
	//
	//   DIGEST(module-a) = digest(
	//     // module-a contents
	//     README.md,
	//     a.proto,
	//     b.proto,
	//     // module-b:c1 digest
	//     DIGEST(
	//       README.md,
	//       c.proto,
	//       d.proto,
	//       // module-c:c1 digest
	//       DIGEST(
	//         README.md,
	//         e.proto,
	//         f.proto,
	//       ),
	//     ),
	//     // module-c:c2 digest
	//     DIGEST(
	//         README.md,
	//         e.proto,
	//         f.proto,
	//         g.proto,
	//     ),
	//   )
	//
	// To accomplish this, we need to take the dependencies of the declared ModuleKeys (ie what
	// the Module actually says is in its buf.lock). This function enables us to do that for
	// digest calculations. Within the Module, we say that if we get a remote Module, use the
	// declared ModuleKeys instead of whatever Module we have resolved to for a given ModuleFullName.
	getDeclaredDepModuleKeysB5 := func() ([]ModuleKey, error) {
		moduleData, err := getModuleData()
		if err != nil {
			return nil, err
		}
		declaredDepModuleKeys, err := moduleData.DeclaredDepModuleKeys()
		if err != nil {
			return nil, err
		}
		if len(declaredDepModuleKeys) == 0 {
			return nil, nil
		}
		var digestType DigestType
		for i, moduleKey := range declaredDepModuleKeys {
			digest, err := moduleKey.Digest()
			if err != nil {
				return nil, err
			}
			if i == 0 {
				digestType = digest.Type()
			} else if digestType != digest.Type() {
				return nil, syserror.Newf("multiple digest types found in DeclaredDepModuleKeys: %v, %v", digestType, digest.Type())
			}
		}
		switch digestType {
		case DigestTypeB4:
			// The declared ModuleKey dependencies for a commit may be stored in v1 buf.lock file,
			// in which case they will use B4 digests. B4 digests aren't allowed to be used as
			// input to the B5 digest calculation, so we perform a call to convert all ModuleKeys
			// from B4 to B5 by using the commit provider.
			commitKeysToFetch := make([]CommitKey, len(declaredDepModuleKeys))
			for i, declaredDepModuleKey := range declaredDepModuleKeys {
				commitKey, err := NewCommitKey(declaredDepModuleKey.ModuleFullName().Registry(), declaredDepModuleKey.CommitID(), DigestTypeB5)
				if err != nil {
					return nil, err
				}
				commitKeysToFetch[i] = commitKey
			}
			commits, err := commitProvider.GetCommitsForCommitKeys(ctx, commitKeysToFetch)
			if err != nil {
				return nil, err
			}
			if len(commits) != len(commitKeysToFetch) {
				return nil, syserror.Newf("expected %d commit(s), got %d", commitKeysToFetch, len(commits))
			}
			return slicesext.Map(commits, func(commit Commit) ModuleKey {
				return commit.ModuleKey()
			}), nil
		case DigestTypeB5:
			// No need to fetch b5 digests - we've already got them stored in the module's declared dependencies.
			return declaredDepModuleKeys, nil
		default:
			return nil, syserror.Newf("unsupported digest type: %v", digestType)
		}
	}
	return newModule(
		ctx,
		getBucket,
		"",
		a.remoteModuleKey.ModuleFullName(),
		a.remoteModuleKey.CommitID(),
		a.isTarget,
		false,
		getV1BufYAMLObjectData,
		getV1BufLockObjectData,
		getDeclaredDepModuleKeysB5,
		a.remoteTargetPaths,
		a.remoteTargetExcludePaths,
		"",
		false,
	)
}

// getUniqueSortedModulesByOpaqueID deduplicates and sorts the addedModule list.
//
// Modules that are targets are preferred, followed by Modules that are local.
// Otherwise, remote Modules with later create times are preferred.
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
func getUniqueSortedAddedModulesByOpaqueID(
	ctx context.Context,
	commitProvider CommitProvider,
	addedModules []*addedModule,
) ([]*addedModule, error) {
	opaqueIDToAddedModules := slicesext.ToValuesMap(addedModules, (*addedModule).OpaqueID)
	resultAddedModules := make([]*addedModule, 0, len(opaqueIDToAddedModules))
	for _, addedModulesForOpaqueID := range opaqueIDToAddedModules {
		resultAddedModule, err := selectAddedModuleForOpaqueID(ctx, commitProvider, addedModulesForOpaqueID)
		if err != nil {
			return nil, err
		}
		resultAddedModules = append(resultAddedModules, resultAddedModule)
	}
	sort.Slice(
		resultAddedModules,
		func(i int, j int) bool {
			return resultAddedModules[i].OpaqueID() < resultAddedModules[j].OpaqueID()
		},
	)
	return resultAddedModules, nil
}

// selectAddedModuleForOpaqueID selects the single addedModule that should be used for a list
// of addedModules that all have ths same OpaqueID.
//
// Note from earlier, not deleting:
//
// Digest *cannot* be used here - it's a chicken or egg problem. Computing the digest requires the cache,
// the cache requires the unique Modules, the unique Modules require this function. This is OK though -
// we want to add all Modules that we *think* are unique to the cache. If there is a duplicate, it
// will be detected via cache usage.
func selectAddedModuleForOpaqueID(
	ctx context.Context,
	commitProvider CommitProvider,
	addedModules []*addedModule,
) (*addedModule, error) {
	// First, we see if there are any target Modules. If so, we prefer those.
	targetAddedModules := slicesext.Filter(addedModules, (*addedModule).IsTarget)
	switch len(targetAddedModules) {
	case 0:
		// We have no target Modules. We will select a non-target Module via
		// selectAddedModuleForOpaqueIDIgnoreTargeting
		return selectAddedModuleForOpaqueIDIgnoreTargeting(ctx, commitProvider, addedModules)
	case 1:
		// We have one target Module. Use this Module.
		return targetAddedModules[0], nil
	default:
		// We have multiple target Modules. We will select one of them, but go to the next step
		// within selectAddedModuleForOpaqueIDIgnoreTargeting.
		return selectAddedModuleForOpaqueIDIgnoreTargeting(ctx, commitProvider, targetAddedModules)
	}
}

// selectAddedModuleForOpaqueIDIgnoreTargeting is a child function of selectAddedModuleForOpaqueID
// that assumes targeting has already been taken into account.
//
// This function will just take into account local vs remote, and then resolution between
// remote Modules.
func selectAddedModuleForOpaqueIDIgnoreTargeting(
	ctx context.Context,
	commitProvider CommitProvider,
	addedModules []*addedModule,
) (*addedModule, error) {
	// Now, we see if there are any local Modules. If so, we prefer those
	localAddedModules := slicesext.Filter(addedModules, (*addedModule).IsLocal)
	switch len(localAddedModules) {
	case 0:
		// We have no local Modules. We will select a remote Module.
		return selectRemoteAddedModuleForOpaqueIDIgnoreTargeting(ctx, commitProvider, addedModules)
	default:
		// We have one or more added Modules. We just return the first one - we have
		// no way to differentiate between local Modules. Note that this will result
		// in the first Module added with AddLocalModule to be used, given that we
		// have not messed with ordering.
		return localAddedModules[0], nil
	}
}

// selectRemoteAddedModuleForOpaqueIDIgnoreTargeting is a child function of
// selectAddedModuleForOpaqueIDIgnoreTargeting that assumes targeting and local vs remote
// has already been taken into account.
//
// All addedModules are assumed to have the same OpaqueID, and therefore the same
// ModuleFullName, since they are remote Modules. We validate this.
//
// Note that there may be straight duplicates, ie two modules with the same ModuleFullName and CommitID! This
// function deduplicates these.
//
// The ModuleKey with the latest create time is used.
func selectRemoteAddedModuleForOpaqueIDIgnoreTargeting(
	ctx context.Context,
	commitProvider CommitProvider,
	addedModules []*addedModule,
) (*addedModule, error) {
	if len(addedModules) == 0 {
		return nil, syserror.New("expected at least one remote addedModule in selectRemoteAddedModuleForOpaqueIDIgnoreTargeting")
	}
	for _, addedModule := range addedModules {
		// Just a sanity check.
		if addedModule.remoteModuleKey == nil {
			return nil, syserror.Newf("got nil remoteModuleKey in selectRemoteAddedModuleForOpaqueIDIgnoreTargeting for addedModule %q", addedModule.OpaqueID())
		}
	}
	if len(addedModules) == 1 {
		return addedModules[0], nil
	}
	if moduleFullNameStrings := slicesext.ToUniqueSorted(
		slicesext.Map(
			addedModules,
			func(addedModule *addedModule) string { return addedModule.remoteModuleKey.ModuleFullName().String() },
		),
	); len(moduleFullNameStrings) > 1 {
		return nil, syserror.Newf("multiple ModuleFullNames detected in selectRemoteAddedModuleForOpaqueIDIgnoreTargeting: %s", strings.Join(moduleFullNameStrings, ", "))
	}

	// We now know that we have >1 addedModules, and all of them have a remoteModuleKey, and all the remoteModuleKeys have the same ModuleFullName.

	// Now, we deduplicate by commit ID. If we end up with a single Module, we return that, otherwise we select exactly one Module
	// based on the create time of the corresponding commit ID.
	commitIDToAddedModules := slicesext.ToValuesMap(
		addedModules,
		func(addedModule *addedModule) uuid.UUID { return addedModule.remoteModuleKey.CommitID() },
	)
	uniqueAddedModules := make([]*addedModule, 0, len(commitIDToAddedModules))
	for _, addedModules := range commitIDToAddedModules {
		uniqueAddedModules = append(uniqueAddedModules, addedModules[0])
	}
	if len(uniqueAddedModules) == 1 {
		return uniqueAddedModules[0], nil
	}

	// We now know that we have non-unique remote added Modules, and have selected exactly one addedModule per commit ID.

	uniqueModuleKeys := slicesext.Map(
		uniqueAddedModules,
		func(addedModule *addedModule) ModuleKey {
			return addedModule.remoteModuleKey
		},
	)
	// Returned commits are in same order as input ModuleKeys
	commits, err := commitProvider.GetCommitsForModuleKeys(ctx, uniqueModuleKeys)
	if err != nil {
		return nil, fmt.Errorf("could not resolve modules from buf.lock: %w", err)
	}
	createTime, err := commits[0].CreateTime()
	if err != nil {
		return nil, err
	}
	uniqueAddedModule := uniqueAddedModules[0]
	// i+1 is index inside moduleKeys and addedModules.
	//
	// Find the commit with the latest CreateTime, this is the addedModule you want to return.
	for i, commit := range commits[1:] {
		iCreateTime, err := commit.CreateTime()
		if err != nil {
			return nil, err
		}
		if iCreateTime.After(createTime) {
			uniqueAddedModule = uniqueAddedModules[i+1]
			createTime = iCreateTime
		}
	}
	return uniqueAddedModule, nil
}
