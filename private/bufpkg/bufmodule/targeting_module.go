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

package bufmodule

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

type targetingModule struct {
	Module
	targetPaths                    []string
	targetPathsAllowNotExistOnWalk bool
	excludePaths                   []string
	allowAllTargetPaths            bool
}

func newTargetingModule(
	delegate Module,
	targetPaths []string,
	targetPathsAllowNotExistOnWalk bool,
	excludePaths []string,
	allowAllTargetPaths bool,
) (*targetingModule, error) {
	if err := normalpath.ValidatePathsNormalizedValidatedUnique(targetPaths); err != nil {
		return nil, err
	}
	return &targetingModule{
		Module:                         delegate,
		targetPaths:                    targetPaths,
		targetPathsAllowNotExistOnWalk: targetPathsAllowNotExistOnWalk,
		excludePaths:                   excludePaths,
		allowAllTargetPaths:            allowAllTargetPaths,
	}, nil
}

func (m *targetingModule) TargetFileInfos(ctx context.Context) (fileInfos []bufmoduleref.FileInfo, retErr error) {
	defer func() {
		if retErr == nil {
			bufmoduleref.SortFileInfos(fileInfos)
		}
	}()
	sourceReadBucket := m.getSourceReadBucket()
	// potentialDirPaths are paths that we need to check if they are directories.
	// These are any files that do not end in .proto, as well as files that end in .proto, but
	// do not have a corresponding file in the source ReadBucket.
	// If there is not an file the path ending in .proto could be a directory
	// that itself contains files, i.e. a/b.proto/c.proto is valid.
	var potentialDirPaths []string
	// fileInfoPaths are the paths that are files, so we return them as a separate set.
	fileInfoPaths := make(map[string]struct{})
	for _, targetPath := range m.targetPaths {
		if normalpath.Ext(targetPath) != ".proto" {
			// not a .proto file, therefore must be a directory
			potentialDirPaths = append(potentialDirPaths, targetPath)
		} else {
			// Since all of these are specific files to include, we will only need to check that an
			// equivalent exclude has not been set for this path.
			for _, excludePath := range m.excludePaths {
				if excludePath == targetPath {
					return nil, fmt.Errorf("cannot set the same path for both --path and --exclude flags: %s", excludePath)
				}
			}
			objectInfo, err := sourceReadBucket.Stat(ctx, targetPath)
			if err != nil {
				if !storage.IsNotExist(err) {
					return nil, err
				}
				// we do not have a file, so even though this path ends
				// in .proto,  this could be a directory - we need to check it
				potentialDirPaths = append(potentialDirPaths, targetPath)
			} else {
				// we have a file, therefore the targetPath was a file path
				// add to the nonImportImageFiles if does not already exist
				if _, ok := fileInfoPaths[targetPath]; !ok {
					fileInfoPaths[targetPath] = struct{}{}
					fileInfo, err := bufmoduleref.NewFileInfo(
						objectInfo.Path(),
						objectInfo.ExternalPath(),
						false,
						m.Module.getModuleIdentity(),
						m.Module.getCommit(),
					)
					if err != nil {
						return nil, err
					}
					fileInfos = append(fileInfos, fileInfo)
				}
			}
		}
	}
	if len(potentialDirPaths) == 0 && !m.allowAllTargetPaths {
		// We had no potential directory paths as we were able to get
		// an file for all targetPaths, so we can return the FileInfos now
		// this means we do not have to do the expensive O(sourceReadBucketSize) operation
		// to check to see if each file is within a potential directory path.
		// This only works if we are expecting target paths, if we are allowing all paths other
		// than excludes, we need to still take into account the excludes for the walk.
		return fileInfos, nil
	}
	var potentialDirPathMap map[string]struct{}
	if !m.allowAllTargetPaths {
		// We have potential directory paths, do the expensive operation to
		// make a map of the directory paths.
		potentialDirPathMap = stringutil.SliceToMap(potentialDirPaths)
	}
	excludePathMap := stringutil.SliceToMap(m.excludePaths)
	// The map of paths within potentialDirPath that matches a file.
	// This needs to contain all paths in potentialDirPathMap at the end for us to
	// have had matches for every targetPath input.
	matchingPotentialDirPathMap := make(map[string]struct{})
	if walkErr := sourceReadBucket.Walk(
		ctx,
		"",
		func(objectInfo storage.ObjectInfo) error {
			path := objectInfo.Path()
			excludeMatchingPathMap := normalpath.MapAllEqualOrContainingPathMap(
				excludePathMap,
				path,
				normalpath.Relative,
			)
			var fileMatchingPathMap map[string]struct{}
			if !m.allowAllTargetPaths {
				// get the paths in potentialDirPathMap that match this path
				fileMatchingPathMap = normalpath.MapAllEqualOrContainingPathMap(
					potentialDirPathMap,
					path,
					normalpath.Relative,
				)
			}
			// 1. If we find a match for both target and exclude paths, we need to resolve the union
			// of the paths.
			// 2. If we find it only for exclude paths, then we exclude.
			// 3. If we do not find it for our target path and allowAllTargetPaths=false, we exclude.
			// 4. If allowAllTargetPaths=false but we find it in our target paths, we need to add
			// the file to `matchingPotentialDirPathMap` to check against the `targetPathsAllowNotExistOnWalk`
			// condition.
			// 5. In the remaining case, we add the file if `len(fileMatchingPathMap) > 0 || allowAllTargetPaths == true`.
			if len(excludeMatchingPathMap) > 0 && len(fileMatchingPathMap) > 0 {
				// We need to merge the results for exclude and target paths based on the most specific
				// configuration.
				// Case 1 - target path is more specific than exclude, we include:
				//   --path a/b.proto --exclude a
				// Case 2 - exclude path is more specific than target, we exclude:
				//   --path a --exclude a/b.proto
				// Case 3 - they are equal, we error:
				//   --path a --exclude a
				for fileMatchingPath := range fileMatchingPathMap {
					for excludeMatchingPath := range excludeMatchingPathMap {
						if fileMatchingPath == excludeMatchingPath {
							return fmt.Errorf("cannot set the same path for both --path and --exclude flags: %s", excludeMatchingPath)
						}
						// Since we already checked for the equal condition, we know that this must be contain.
						// In this case
						if normalpath.EqualsOrContainsPath(fileMatchingPath, excludeMatchingPath, normalpath.Relative) {
							return nil
						}
					}
				}
			}
			if (len(excludeMatchingPathMap) > 0) || (len(fileMatchingPathMap) == 0 && !m.allowAllTargetPaths) {
				return nil
			}
			if !m.allowAllTargetPaths {
				// We had a match, this means that some path in potentialDirPaths matched
				// the path, add all the paths in potentialDirPathMap that
				// matched to matchingPotentialDirPathMap.
				for key := range fileMatchingPathMap {
					matchingPotentialDirPathMap[key] = struct{}{}
				}
			}
			// then, add the file if it is not added
			if _, ok := fileInfoPaths[path]; !ok {
				fileInfoPaths[path] = struct{}{}
				fileInfo, err := bufmoduleref.NewFileInfo(
					objectInfo.Path(),
					objectInfo.ExternalPath(),
					false,
					m.Module.getModuleIdentity(),
					m.Module.getCommit(),
				)
				if err != nil {
					return err
				}
				fileInfos = append(fileInfos, fileInfo)
			}
			return nil
		},
	); walkErr != nil {
		return nil, walkErr
	}
	// if !allowNotExist, i.e. if all targetPaths must have a matching file,
	// we check the matchingPotentialDirPathMap against the potentialDirPathMap
	// to make sure that potentialDirPathMap is covered
	if !m.targetPathsAllowNotExistOnWalk && !m.allowAllTargetPaths {
		for potentialDirPath := range potentialDirPathMap {
			if _, ok := matchingPotentialDirPathMap[potentialDirPath]; !ok {
				// no match, this is an error given that allowNotExist is false
				return nil, fmt.Errorf("path %q has no matching file in the module", potentialDirPath)
			}
		}
	}
	return fileInfos, nil
}
