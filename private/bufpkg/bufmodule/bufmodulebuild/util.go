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

package bufmodulebuild

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

func applyModulePaths(
	module bufmodule.Module,
	roots []string,
	fileOrDirPaths *[]string,
	excludeFileOrDirPaths []string,
	fileOrDirPathsAllowNotExist bool,
	pathType normalpath.PathType,
) (bufmodule.Module, error) {
	if fileOrDirPaths == nil && excludeFileOrDirPaths == nil {
		return module, nil
	}
	var excludePaths []string
	if len(excludeFileOrDirPaths) != 0 {
		var err error
		excludePaths, err = pathsToTargetPaths(roots, excludeFileOrDirPaths, pathType)
		if err != nil {
			return nil, err
		}
	}
	if fileOrDirPaths == nil {
		if fileOrDirPathsAllowNotExist {
			return bufmodule.ModuleWithExcludePathsAllowNotExist(module, excludePaths)
		}
		return bufmodule.ModuleWithExcludePaths(module, excludePaths)
	}
	targetPaths, err := pathsToTargetPaths(roots, *fileOrDirPaths, pathType)
	if err != nil {
		return nil, err
	}
	if fileOrDirPathsAllowNotExist {
		return bufmodule.ModuleWithTargetPathsAllowNotExist(module, targetPaths, excludePaths)
	}
	return bufmodule.ModuleWithTargetPaths(module, targetPaths, excludePaths)
}

func pathsToTargetPaths(roots []string, paths []string, pathType normalpath.PathType) ([]string, error) {
	if len(roots) == 0 {
		// this should never happen
		return nil, errors.New("no roots on config")
	}

	targetPaths := make([]string, len(paths))
	for i, path := range paths {
		targetPath, err := pathToTargetPath(roots, path, pathType)
		if err != nil {
			return nil, err
		}
		targetPaths[i] = targetPath
	}
	return targetPaths, nil
}

func pathToTargetPath(roots []string, path string, pathType normalpath.PathType) (string, error) {
	var matchingRoots []string
	for _, root := range roots {
		if normalpath.ContainsPath(root, path, pathType) {
			matchingRoots = append(matchingRoots, root)
		}
	}
	switch len(matchingRoots) {
	case 0:
		// this is a user error and will likely happen often
		return "", fmt.Errorf(
			"path %q is not contained within any of roots %s - note that specified paths "+
				"cannot be roots, but must be contained within roots",
			path,
			stringutil.SliceToHumanStringQuoted(roots),
		)
	case 1:
		targetPath, err := normalpath.Rel(matchingRoots[0], path)
		if err != nil {
			return "", err
		}
		// just in case
		return normalpath.NormalizeAndValidate(targetPath)
	default:
		// this should never happen
		return "", fmt.Errorf("%q is contained in multiple roots %s", path, stringutil.SliceToHumanStringQuoted(roots))
	}
}

type buildOptions struct {
	moduleIdentity bufmoduleref.ModuleIdentity
	// If nil, all files are considered targets.
	// If empty (but non-nil), the module will have no target paths.
	paths              *[]string
	pathsAllowNotExist bool
	// Paths that will be excluded from the module build process. This is handled in conjunction
	// with `paths`.
	excludePaths       []string
	workspaceDirectory string
}

type buildModuleFileSetOptions struct {
	workspace bufmodule.Workspace
}
