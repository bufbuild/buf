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
	targetPaths              []string
	pathsAllowNotExistOnWalk bool
	excludePaths             []string
}

func newTargetingModule(
	delegate Module,
	targetPaths []string,
	excludePaths []string,
	pathsAllowNotExistOnWalk bool,
) (*targetingModule, error) {
	if err := normalpath.ValidatePathsNormalizedValidatedUnique(targetPaths); err != nil {
		return nil, err
	}
	return &targetingModule{
		Module:                   delegate,
		targetPaths:              targetPaths,
		pathsAllowNotExistOnWalk: pathsAllowNotExistOnWalk,
		excludePaths:             excludePaths,
	}, nil
}

func (m *targetingModule) TargetFileInfos(ctx context.Context) (fileInfos []bufmoduleref.FileInfo, retErr error) {
	defer func() {
		if retErr == nil {
			bufmoduleref.SortFileInfos(fileInfos)
		}
	}()
	// We start by ensuring that no paths have been duplicated between target and exclude pathes.
	for _, targetPath := range m.targetPaths {
		for _, excludePath := range m.excludePaths {
			if excludePath == targetPath {
				return nil, fmt.Errorf(
					"cannot set the same path for both --path and --exclude flags: %s",
					normalpath.Unnormalize(excludePath),
				)
			}
		}
	}
	sourceReadBucket := m.getSourceReadBucket()
	// potentialDirPaths are paths that we need to check if they are directories.
	// These are any files that do not end in .proto, as well as files that end in .proto, but
	// do not have a corresponding file in the source ReadBucket.
	// If there is not an file the path ending in .proto could be a directory
	// that itself contains files, i.e. a/b.proto/c.proto is valid.
	var potentialDirPaths []string
	// fileInfoPaths are the paths that are files, so we return them as a separate set.
	fileInfoPaths := make(map[string]struct{})
	// If m.targetPaths == nil then we are accepting all paths and we only need to filter on
	// the excluded paths.
	//
	// In the event that we do have target paths, we need first gather up all the target paths
	// that are proto files. If all target paths proto files, we can return them first.
	if m.targetPaths != nil {
		for _, targetPath := range m.targetPaths {
			if normalpath.Ext(targetPath) != ".proto" {
				// not a .proto file, therefore must be a directory
				potentialDirPaths = append(potentialDirPaths, targetPath)
			} else {
				objectInfo, err := sourceReadBucket.Stat(ctx, targetPath)
				if err != nil {
					if !storage.IsNotExist(err) {
						return nil, err
					}
					// we do not have a file, so even though this path ends
					// in .proto,  this could be a directory - we need to check it
					potentialDirPaths = append(potentialDirPaths, targetPath)
				} else {
					// Since all of these are specific files to include, and we've already checked
					// for duplicated excludes, we know that this file is not excluded.
					// We have a file, therefore the targetPath was a file path
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
		if len(potentialDirPaths) == 0 {
			// We had no potential directory paths as we were able to get
			// an file for all targetPaths, so we can return the FileInfos now
			// this means we do not have to do the expensive O(sourceReadBucketSize) operation
			// to check to see if each file is within a potential directory path.
			// TODO: we need to check for the exclude paths existing here...
			return fileInfos, nil
		}
	}
	// We have potential directory paths, do the expensive operation to
	// make a map of the directory paths.
	potentialDirPathMap := stringutil.SliceToMap(potentialDirPaths)
	excludePathMap := stringutil.SliceToMap(m.excludePaths)
	// The map of paths within potentialDirPath that matches a file.
	// This needs to contain all paths in potentialDirPathMap at the end for us to
	// have had matches for every targetPath input.
	matchingPotentialDirPathMap := make(map[string]struct{})
	// The map of exclude paths that have a match on the walk. This is used to check against
	// pathsAllowNotExistOnWalk.
	matchingExcludePaths := make(map[string]struct{})
	if walkErr := sourceReadBucket.Walk(
		ctx,
		"",
		func(objectInfo storage.ObjectInfo) error {
			path := objectInfo.Path()
			fileMatchingExcludePathMap := normalpath.MapAllEqualOrContainingPathMap(
				excludePathMap,
				path,
				normalpath.Relative,
			)
			for excludeMatchingPath := range fileMatchingExcludePathMap {
				if _, ok := matchingExcludePaths[excludeMatchingPath]; !ok {
					matchingExcludePaths[excludeMatchingPath] = struct{}{}
				}
			}
			// get the paths in potentialDirPathMap that match this path
			fileMatchingPathMap := normalpath.MapAllEqualOrContainingPathMap(
				potentialDirPathMap,
				path,
				normalpath.Relative,
			)
			// if m.targetPaths == nil, then len(potentialDirPathMap) == 0 and fileMatchingPathMap == nil,
			// so we'll exclude as long as len(fileMatchingExcludePathMap) > 0.
			if excludeFile(fileMatchingPathMap, fileMatchingExcludePathMap) {
				return nil
			}
			if m.targetPaths != nil {
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
	if !m.pathsAllowNotExistOnWalk {
		for potentialDirPath := range potentialDirPathMap {
			if _, ok := matchingPotentialDirPathMap[potentialDirPath]; !ok {
				// no match, this is an error given that allowNotExist is false
				return nil, fmt.Errorf("path %q has no matching file in the module", potentialDirPath)
			}
		}
		for excludePath := range excludePathMap {
			if _, ok := matchingExcludePaths[excludePath]; !ok {
				// no match, this is an error given that allowNotExist is false
				return nil, fmt.Errorf("path %q has no matching file in the module", excludePath)
			}
		}
	}
	return fileInfos, nil
}

func excludeFile(
	fileMatchingPathMap map[string]struct{},
	fileMatchingExcludePathMap map[string]struct{},
) bool {
	if fileMatchingPathMap == nil {
		return !(len(fileMatchingExcludePathMap) == 0)
	}
	if len(fileMatchingPathMap) == 0 {
		return true
	}
	if len(fileMatchingExcludePathMap) == 0 {
		return false
	}
	seenExcludePaths := make(map[string]struct{})
	for fileMatchingPath := range fileMatchingPathMap {
		for fileMatchingExcludePath := range fileMatchingExcludePathMap {
			if normalpath.EqualsOrContainsPath(fileMatchingPath, fileMatchingExcludePath, normalpath.Relative) {
				delete(fileMatchingPathMap, fileMatchingPath)
				seenExcludePaths[fileMatchingExcludePath] = struct{}{}
				continue
			}
		}
	}
	// For paths that have never been seen for excludes, we should trim them, since they are
	// outside of the target paths found.
	for fileMatchingExcludePath := range fileMatchingExcludePathMap {
		if _, ok := seenExcludePaths[fileMatchingExcludePath]; ok {
			continue
		}
		delete(fileMatchingExcludePathMap, fileMatchingExcludePath)
	}
	return excludeFile(fileMatchingPathMap, fileMatchingExcludePathMap)
}
