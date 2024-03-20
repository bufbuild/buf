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

package bufworkspace

import (
	"fmt"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

type moduleTargeting struct {
	// Whether this module is really a target module.
	//
	// False if this was not specified as a target module by the caller.
	// Also false if there were bucketTargeting.TargetPaths() or bucketTargeting.protoFileTargetPath, but
	// these paths did not match anything in the module.
	isTargetModule bool
	// moduleDirPath is the directory path of the module
	moduleDirPath string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetPaths []string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleTargetExcludePaths []string
	// relative to the actual moduleDirPath and the roots parsed from the buf.yaml
	moduleProtoFileTargetPath string
	includePackageFiles       bool
}

func newModuleTargeting(
	moduleDirPath string,
	roots []string,
	bucketTargeting buftarget.BucketTargeting,
	config *workspaceBucketConfig,
	isTentativelyTargetModule bool,
) (*moduleTargeting, error) {
	if !isTentativelyTargetModule {
		// If this is not a target Module, we do not want to target anything, as targeting
		// paths for non-target Modules is an error.
		return &moduleTargeting{moduleDirPath: moduleDirPath}, nil
	}
	// If we have no target paths, then we always match the value of isTargetModule.
	// Otherwise, we need to see that at least one path matches the moduleDirPath for us
	// to consider this module a target.
	isTargetModule := len(bucketTargeting.TargetPaths()) == 0 && config.protoFileTargetPath == ""

	var moduleTargetPaths []string
	var moduleTargetExcludePaths []string
	var moduleProtoFileTargetPath string
	var includePackageFiles bool
	if config.protoFileTargetPath != "" {
		var err error
		// We are currently returning the mapped proto file path as a target path in bucketTargeting.
		// At the user level, we do not allow setting target paths or exclude paths for proto file
		// references, so we do this check here.
		if len(bucketTargeting.TargetPaths()) != 1 {
			return nil, syserror.New("ProtoFileTargetPath not properly returned with bucketTargeting")
		}
		protoFileTargetPath := bucketTargeting.TargetPaths()[0]
		if normalpath.ContainsPath(moduleDirPath, protoFileTargetPath, normalpath.Relative) {
			isTargetModule = true
			moduleProtoFileTargetPath, err = normalpath.Rel(moduleDirPath, protoFileTargetPath)
			if err != nil {
				return nil, err
			}
			moduleProtoFileTargetPath, err = applyRootsToTargetPath(roots, moduleProtoFileTargetPath, normalpath.Relative)
			if err != nil {
				return nil, err
			}
			includePackageFiles = config.includePackageFiles
		}
	} else {
		var err error
		// We use the bucketTargeting.TargetPaths() instead of the workspace config target paths
		// since those are stripped of the path to the module.
		for _, targetPath := range bucketTargeting.TargetPaths() {
			if targetPath == moduleDirPath {
				// We're just going to be realists in our error messages here.
				// TODO FUTURE: Do we error here currently? If so, this error remains. For extra credit in the future,
				// if we were really clever, we'd go back and just add this as a module path.
				return nil, fmt.Errorf("module %q was specified with --path, specify this module path directly as an input", targetPath)
			}
			if normalpath.ContainsPath(moduleDirPath, targetPath, normalpath.Relative) {
				isTargetModule = true
				moduleTargetPath, err := normalpath.Rel(moduleDirPath, targetPath)
				if err != nil {
					return nil, err
				}
				moduleTargetPaths = append(moduleTargetPaths, moduleTargetPath)
			}
		}
		// If this is not a target module, then exclude-paths should not apply to the module.
		//
		// Otherwise, you will run into scenarios like this:
		//
		// buf build --exclude-path googleapis/google --path books/acme
		// Failure: cannot set TargetPaths for a non-target Module when calling AddLocalModule, bucketID="googleapis", targetPaths=[], targetExcludePaths=[google]
		//
		// This syserror is valid: LocalModuleWithTargetPaths should only be called for target modules. If we pass --exclude-path for a non-target module (which in the above
		// example, googleapis is not targeted because --path books/acme only targeted books), then it should be as if --exclude-path pointed to a non-existent file or directory.
		if isTargetModule {
			// We use the bucketTargeting.TargetExcludePaths() instead of the workspace config target
			// exclude paths since those are stripped of the path to the module.
			for _, targetExcludePath := range bucketTargeting.TargetExcludePaths() {
				if targetExcludePath == moduleDirPath {
					// We're just going to be realists in our error messages here.
					// TODO FUTURE: Do we error here currently? If so, this error remains. For extra credit in the future,
					// if we were really clever, we'd go back and just remove this as a module path if it was specified.
					// This really should be allowed - how else do you exclude from a workspace?
					return nil, fmt.Errorf("module %q was specified with --exclude-path, this flag cannot be used to specify module directories", targetExcludePath)
				}
				if normalpath.ContainsPath(moduleDirPath, targetExcludePath, normalpath.Relative) {
					moduleTargetExcludePath, err := normalpath.Rel(moduleDirPath, targetExcludePath)
					if err != nil {
						return nil, err
					}
					moduleTargetExcludePaths = append(moduleTargetExcludePaths, moduleTargetExcludePath)
				}
			}
		}
		moduleTargetPaths, err = slicesext.MapError(
			moduleTargetPaths,
			func(moduleTargetPath string) (string, error) {
				return applyRootsToTargetPath(roots, moduleTargetPath, normalpath.Relative)
			},
		)
		if err != nil {
			return nil, err
		}
		moduleTargetExcludePaths, err = slicesext.MapError(
			moduleTargetExcludePaths,
			func(moduleTargetExcludePath string) (string, error) {
				return applyRootsToTargetPath(roots, moduleTargetExcludePath, normalpath.Relative)
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return &moduleTargeting{
		moduleDirPath:             moduleDirPath,
		isTargetModule:            isTargetModule,
		moduleTargetPaths:         moduleTargetPaths,
		moduleTargetExcludePaths:  moduleTargetExcludePaths,
		moduleProtoFileTargetPath: moduleProtoFileTargetPath,
		includePackageFiles:       includePackageFiles,
	}, nil
}

func applyRootsToTargetPath(roots []string, path string, pathType normalpath.PathType) (string, error) {
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
		return normalpath.Normalize(targetPath), nil
	default:
		// this should never happen
		return "", fmt.Errorf("%q is contained in multiple roots %s", path, stringutil.SliceToHumanStringQuoted(roots))
	}
}
