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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

// workspaceTargeting figures out if we are working with a v1 or v2 workspace based on
// buftarget.BucketTargeting information and provides the workspace targeting information
// to WorkspaceProvider and WorkspaceDepManagerProvider.
//
// For v1 workspaces, this provides the bucket IDs to module configs, mapped module buckets and
// module targeting information, and all configured dependency module refs.
//
// # For v2 workspaces, this provides the bucket IDs to module configs, module dir paths,
// and mapped module buckets and module targeting.
//
// In the case where no controlling workspace is found, we default to a v1 workspace with
// a single module directory path, which is equivalent to the input dir.
//
// We only ever return one of v1 or v2.
type workspaceTargeting struct {
	v1 *v1Targeting
	v2 *v2Targeting
}

type v1Targeting struct {
	bucketIDToModuleConfig     map[string]bufconfig.ModuleConfig
	moduleBucketsAndTargeting  []*moduleBucketAndModuleTargeting
	allConfiguredDepModuleRefs []bufmodule.ModuleRef
}

type v2Targeting struct {
	bufYAMLFile               bufconfig.BufYAMLFile
	bucketIDToModuleConfig    map[string]bufconfig.ModuleConfig
	moduleBucketsAndTargeting []*moduleBucketAndModuleTargeting
}

type moduleBucketAndModuleTargeting struct {
	bucket storage.ReadBucket
	// A bucketID, which uniquely identifies a moduleBucketAndModuleTargeting, is needed
	// because in a v2 workspace multiple local modules may have the same DirPath.
	bucketID        string
	moduleTargeting *moduleTargeting
}

func newWorkspaceTargeting(
	ctx context.Context,
	logger *slog.Logger,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
	ignoreAndDisallowV1BufWorkYAMLs bool,
) (*workspaceTargeting, error) {
	if err := validateBucketTargeting(bucketTargeting, config.protoFileTargetPath); err != nil {
		return nil, err
	}
	if overrideBufYAMLFile != nil {
		logger.DebugContext(
			ctx,
			"targeting workspace with config override",
			slog.String("subDirPath", bucketTargeting.SubDirPath()),
		)
		switch fileVersion := overrideBufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			return v1WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				[]string{bucketTargeting.SubDirPath()},
				overrideBufYAMLFile,
			)
		case bufconfig.FileVersionV2:
			return v2WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				overrideBufYAMLFile,
				// If the user specifies a `--config path/to/config/buf.yaml` and the input workspace
				// modules do not have their own doc/license files, we do not want to use "path/to/config/README(or LICENSE)"
				// or "moduleDir/README(or LICENSE)" as the module's doc/license.
				false,
			)
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}
	if controllingWorkspace := bucketTargeting.ControllingWorkspace(); controllingWorkspace != nil {
		// This is a v2 workspace.
		if controllingWorkspace.BufYAMLFile() != nil {
			logger.DebugContext(
				ctx,
				"targeting workspace based on v2 buf.yaml",
				slog.String("subDirPath", bucketTargeting.SubDirPath()),
			)
			return v2WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				controllingWorkspace.BufYAMLFile(),
				// For a v2 controlling workspace, if a module inside does not have its own doc/license,
				// we want to the doc/license at the root of the workspace if they exist.
				true,
			)
		}
		// This is a v1 workspace.
		if bufWorkYAMLFile := controllingWorkspace.BufWorkYAMLFile(); bufWorkYAMLFile != nil {
			if ignoreAndDisallowV1BufWorkYAMLs {
				// This means that we attempted to target a v1 workspace at the buf.work.yaml, not
				// an individual module within the v1 workspace defined in buf.work.yaml.
				// This is disallowed.
				if bucketTargeting.SubDirPath() == "." {
					return nil, errors.New(`Workspaces defined with buf.work.yaml cannot be updated or pushed, only
the individual modules within a workspace can be updated or pushed. Workspaces
defined with a v2 buf.yaml can be updated, see the migration documentation for more details.`)
				}
				// We targeted a specific module within the workspace. Based on the option we provided, we're going to ignore
				// the workspace entirely, and just act as if the buf.work.yaml did not exist.
				logger.DebugContext(
					ctx,
					"targeting workspace, ignoring v1 buf.work.yaml, just building on module at target",
					slog.String("subDirPath", bucketTargeting.SubDirPath()),
				)
				return v1WorkspaceTargeting(
					ctx,
					config,
					bucket,
					bucketTargeting,
					[]string{bucketTargeting.SubDirPath()}, // Assume we are targeting only the module at the input dir
					nil,
				)
			}
			return v1WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				bufWorkYAMLFile.DirPaths(),
				nil,
			)
		}
	}
	logger.DebugContext(
		ctx,
		"targeting workspace with no found buf.work.yaml or v2 buf.yaml",
		slog.String("subDirPath", bucketTargeting.SubDirPath()),
	)
	// We did not find any buf.work.yaml or v2 buf.yaml, we invoke fallback logic.
	return fallbackWorkspaceTargeting(
		ctx,
		logger,
		config,
		bucket,
		bucketTargeting,
	)
}

func v2WorkspaceTargeting(
	ctx context.Context,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	bufYAMLFile bufconfig.BufYAMLFile,
	// If true and if a workspace module does not have a license/doc at its moduleDirPath,
	// use the license/doc respectively at the workspace root for this module.
	useWorkspaceLicenseDocIfNotFoundAtMoudle bool,
) (*workspaceTargeting, error) {
	// We keep track of if any module was tentatively targeted, and then actually targeted via
	// the paths flags. We use this pre-building of the ModuleSet to see if the --path and
	// --exclude-path flags resulted in no targeted modules. This condition is represented
	// by hadIsTentativelyTargetModule == true && hadIsTargetModule = false
	//
	// If hadIsTentativelyTargetModule is false, this means that our input bucketTargeting.SubDirPath() was not
	// actually representative of any module that we detected in buf.work.yaml or v2 buf.yaml
	// directories, and this is a system error - this should be verified before we reach this function.
	var hadIsTentativelyTargetModule bool
	var hadIsTargetModule bool
	moduleDirPaths := make([]string, 0, len(bufYAMLFile.ModuleConfigs()))
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	moduleBucketsAndTargeting := make([]*moduleBucketAndModuleTargeting, 0, len(bufYAMLFile.ModuleConfigs()))
	moduleConfigs := bufYAMLFile.ModuleConfigs()
	bucketIDsForModuleConfigs := bucketIDsForModuleConfigsV2(moduleConfigs)
	if len(bucketIDsForModuleConfigs) != len(moduleConfigs) {
		// This is impossible, as the length is guaranteed by bucketIDsForModuleConfigsV2.
		return nil, syserror.Newf("expected %d bucketIDs computed but got %d", len(moduleConfigs), len(bucketIDsForModuleConfigs))
	}
	for i, moduleConfig := range moduleConfigs {
		moduleDirPath := moduleConfig.DirPath()
		moduleDirPaths = append(moduleDirPaths, moduleDirPath)
		// bucketIDs have the same order as moduleConfigs
		bucketID := bucketIDsForModuleConfigs[i]
		bucketIDToModuleConfig[bucketID] = moduleConfig
		// bucketTargeting.SubDirPath() is the input targetSubDirPath. We only want to target modules that are inside
		// this targetSubDirPath. Example: bufWorkYAMLDirPath is "foo", targetSubDirPath is "foo/bar",
		// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
		// and "foo/bar/bat".
		//
		// This is new behavior - before, we required that you input an exact match for the module directory path,
		// but now, we will take all the modules underneath this workspace.
		isTentativelyTargetModule := normalpath.EqualsOrContainsPath(bucketTargeting.SubDirPath(), moduleDirPath, normalpath.Relative)
		// We ignore this check for proto file refs, since the input is considered the directory
		// of the proto file reference, which is unlikely to contain a module in its entirety.
		// In the future, it would be nice to handle this more elegently.
		if config.protoFileTargetPath != "" {
			isTentativelyTargetModule = true
		}
		if isTentativelyTargetModule {
			hadIsTentativelyTargetModule = true
		}
		mappedModuleBucket, moduleTargeting, err := getMappedModuleBucketAndModuleTargeting(
			ctx,
			config,
			bucket,
			bucketTargeting,
			moduleDirPath,
			moduleConfig,
			isTentativelyTargetModule,
			useWorkspaceLicenseDocIfNotFoundAtMoudle,
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleBucketsAndTargeting = append(moduleBucketsAndTargeting, &moduleBucketAndModuleTargeting{
			bucket:          mappedModuleBucket,
			bucketID:        bucketID,
			moduleTargeting: moduleTargeting,
		})
	}
	if !hadIsTentativelyTargetModule {
		// Check if the input is overlapping within a module dir path. If so, return a nicer
		// error. In the future, we want to remove special treatment for input dir, and it
		// should be treated just like any target path.
		return nil, checkForOverlap(ctx, bucket, bucketTargeting.SubDirPath(), moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	return &workspaceTargeting{
		v2: &v2Targeting{
			bufYAMLFile:               bufYAMLFile,
			bucketIDToModuleConfig:    bucketIDToModuleConfig,
			moduleBucketsAndTargeting: moduleBucketsAndTargeting,
		},
	}, nil
}

func v1WorkspaceTargeting(
	ctx context.Context,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	moduleDirPaths []string,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (*workspaceTargeting, error) {
	// We keep track of if any module was tentatively targeted, and then actually targeted via
	// the paths flags. We use this pre-building of the ModuleSet to see if the --path and
	// --exclude-path flags resulted in no targeted modules. This condition is represented
	// by hadIsTentativelyTargetModule == true && hadIsTargetModule = false
	//
	// If hadIsTentativelyTargetModule is false, this means that our input bucketTargeting.SubDirPath() was not
	// actually representative of any module that we detected in buf.work.yaml or v2 buf.yaml
	// directories, and this is a system error - this should be verified before we reach this function.
	var hadIsTentativelyTargetModule bool
	var hadIsTargetModule bool
	var allConfiguredDepModuleRefs []bufmodule.ModuleRef
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	// We use this to detect different refs across different files.
	moduleFullNameStringToConfiguredDepModuleRefString := make(map[string]string)
	moduleBucketsAndTargeting := make([]*moduleBucketAndModuleTargeting, 0, len(moduleDirPaths))
	for _, moduleDirPath := range moduleDirPaths {
		moduleConfig, configuredDepModuleRefs, err := getModuleConfigAndConfiguredDepModuleRefsV1Beta1OrV1(
			ctx,
			bucket,
			moduleDirPath,
			overrideBufYAMLFile,
		)
		if err != nil {
			return nil, err
		}
		for _, configuredDepModuleRef := range configuredDepModuleRefs {
			moduleFullNameString := configuredDepModuleRef.ModuleFullName().String()
			configuredDepModuleRefString := configuredDepModuleRef.String()
			existingConfiguredDepModuleRefString, ok := moduleFullNameStringToConfiguredDepModuleRefString[moduleFullNameString]
			if !ok {
				// We haven't encountered a ModuleRef with this ModuleFullName yet, add it.
				allConfiguredDepModuleRefs = append(allConfiguredDepModuleRefs, configuredDepModuleRef)
				moduleFullNameStringToConfiguredDepModuleRefString[moduleFullNameString] = configuredDepModuleRefString
			} else if configuredDepModuleRefString != existingConfiguredDepModuleRefString {
				// We encountered the same ModuleRef by ModuleFullName, but with a different Ref.
				return nil, fmt.Errorf("found different refs for the same module within buf.yaml deps in the workspace: %s %s", configuredDepModuleRefString, existingConfiguredDepModuleRefString)
			}
		}
		// DirPaths are unique within a v1 workspace, and so it's safe to use them as bucketIDs.
		bucketID := moduleDirPath
		bucketIDToModuleConfig[bucketID] = moduleConfig
		// We only want to target modules that are inside the bucketTargeting.SubDirPath().
		// Example: bufWorkYAMLDirPath is "foo", bucketTargeting.SubDirPath() is "foo/bar",
		// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
		// and "foo/bar/bat".
		//
		// This is new behavior - before, we required that you input an exact match for the module directory path,
		// but now, we will take all the modules underneath this workspace.
		isTentativelyTargetModule := normalpath.EqualsOrContainsPath(bucketTargeting.SubDirPath(), moduleDirPath, normalpath.Relative)
		// We ignore this check for proto file refs, since the input is considered the directory
		// of the proto file reference, which is unlikely to contain a module in its entirety.
		// In the future, it would be nice to handle this more elegently.
		if config.protoFileTargetPath != "" {
			isTentativelyTargetModule = true
		}
		if isTentativelyTargetModule {
			hadIsTentativelyTargetModule = true
		}
		mappedModuleBucket, moduleTargeting, err := getMappedModuleBucketAndModuleTargeting(
			ctx,
			config,
			bucket,
			bucketTargeting,
			moduleDirPath,
			moduleConfig,
			isTentativelyTargetModule,
			// In a v1 workspace, if a module does not have its own doc/license next to its v1 buf.yaml,
			// we do NOT want to fall back to the doc/license next to it's buf.work.yaml.
			false,
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleBucketsAndTargeting = append(moduleBucketsAndTargeting, &moduleBucketAndModuleTargeting{
			bucket:          mappedModuleBucket,
			bucketID:        bucketID,
			moduleTargeting: moduleTargeting,
		})
	}
	if !hadIsTentativelyTargetModule {
		// Check if the input is overlapping within a module dir path. If so, return a nicer
		// error. In the future, we want to remove special treatment for input dir, and it
		// should be treated just like any target path.
		return nil, checkForOverlap(ctx, bucket, bucketTargeting.SubDirPath(), moduleDirPaths)
	}
	if !hadIsTargetModule {
		// It would be nice to have a better error message than this in the long term.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	return &workspaceTargeting{
		v1: &v1Targeting{
			bucketIDToModuleConfig:     bucketIDToModuleConfig,
			moduleBucketsAndTargeting:  moduleBucketsAndTargeting,
			allConfiguredDepModuleRefs: allConfiguredDepModuleRefs,
		},
	}, nil
}

// fallbackWorkspaceTargeting is the fallback logic when there is no config override or
// controlling workspace discovered through bucket targeting.
//
// 1. We check if the input is in a v1 module. If yes, then we simply use that.
// 2. If no v1 module was found for the input, we check the target paths, if there are any.
// For each target path, we check if it is part of a workspace or v1 module.
//
//	a. If we find a v1 or v2 workspace, we ensure that all them resolve to the same workspace.
//	b. If we find no workspace, we keep track of any v1 modules we find along the way. We
//	then build those v1 modules as a v1 workspace.
//
// 3. In the case where we find nothing, we set the input as a v1 module in a v1 workspace.
func fallbackWorkspaceTargeting(
	ctx context.Context,
	logger *slog.Logger,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
) (*workspaceTargeting, error) {
	var v1ModulePaths []string
	inputDirV1Module, err := checkForControllingWorkspaceOrV1Module(
		ctx,
		logger,
		bucket,
		bucketTargeting.SubDirPath(),
		true,
	)
	if err != nil {
		return nil, err
	}
	if inputDirV1Module != nil {
		v1ModulePaths = append(v1ModulePaths, inputDirV1Module.Path())
	} else if config.protoFileTargetPath == "" {
		// No v1 module found for the input dir, if the input was not a protoFileRef, then
		// check the target paths to see if a workspace or v1 module exists.
		var v1BufWorkYAML bufconfig.BufWorkYAMLFile
		var v2BufYAMLFile bufconfig.BufYAMLFile
		var controllingWorkspacePath string
		for _, targetPath := range bucketTargeting.TargetPaths() {
			controllingWorkspaceOrModule, err := checkForControllingWorkspaceOrV1Module(
				ctx,
				logger,
				bucket,
				targetPath,
				false,
			)
			if err != nil {
				return nil, err
			}
			if controllingWorkspaceOrModule != nil {
				// v1 workspace found
				if bufWorkYAMLFile := controllingWorkspaceOrModule.BufWorkYAMLFile(); bufWorkYAMLFile != nil {
					if controllingWorkspacePath != "" && controllingWorkspaceOrModule.Path() != controllingWorkspacePath {
						return nil, fmt.Errorf("different controlling workspaces found: %q, %q", controllingWorkspacePath, controllingWorkspaceOrModule.Path())
					}
					controllingWorkspacePath = controllingWorkspaceOrModule.Path()
					v1BufWorkYAML = bufWorkYAMLFile
					continue
				}
				// v2 workspace or v1 module found
				if bufYAMLFile := controllingWorkspaceOrModule.BufYAMLFile(); bufYAMLFile != nil {
					if bufYAMLFile.FileVersion() == bufconfig.FileVersionV2 {
						if controllingWorkspacePath != "" && controllingWorkspaceOrModule.Path() != controllingWorkspacePath {
							return nil, fmt.Errorf("different controlling workspaces found: %q, %q", controllingWorkspacePath, controllingWorkspaceOrModule.Path())
						}
						controllingWorkspacePath = controllingWorkspaceOrModule.Path()
						v2BufYAMLFile = bufYAMLFile
						continue
					}
					if bufYAMLFile.FileVersion() == bufconfig.FileVersionV1 {
						v1ModulePaths = append(v1ModulePaths, controllingWorkspaceOrModule.Path())
					}
				}
			}
		}
		// If multiple workspaces were found, we return an error since we don't support building
		// multiple workspaces.
		if v1BufWorkYAML != nil && v2BufYAMLFile != nil {
			return nil, fmt.Errorf("multiple workspaces found")
		}
		// If we found a workspace and v1 modules that were not contained in the workspace, we
		// do not support building multiple workspaces.
		if controllingWorkspacePath != "" && len(v1ModulePaths) > 0 {
			return nil, fmt.Errorf("found a workspace %q that does not contain all found modules: %v", controllingWorkspacePath, v1ModulePaths)
		}
		if v2BufYAMLFile != nil {
			return v2WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				v2BufYAMLFile,
				// Since the v2 buf.yaml is found at the workspace root we allow falling back
				// to the doc/license at the workspace root, even though this function (fallbackWorkspaceTargeting)
				// is the logic for handling the situation where the a controlling workspace not found.
				true,
			)
		}
		if v1BufWorkYAML != nil {
			v1ModulePaths = v1BufWorkYAML.DirPaths()
		}
	}
	// If we still have no v1 module paths, then we go to the final fallback and set a v1
	// module at the input dir.
	if len(v1ModulePaths) == 0 {
		v1ModulePaths = append(v1ModulePaths, bucketTargeting.SubDirPath())
	}
	return v1WorkspaceTargeting(
		ctx,
		config,
		bucket,
		bucketTargeting,
		v1ModulePaths,
		nil,
	)
}

func validateBucketTargeting(
	bucketTargeting buftarget.BucketTargeting,
	protoFilePath string,
) error {
	if protoFilePath != "" {
		// We set the proto file path as a target path, which we handle in module targeting,
		// so we expect len(bucketTargeting.TargetPaths()) to be exactly 1.
		if len(bucketTargeting.TargetPaths()) != 1 || len(bucketTargeting.TargetExcludePaths()) > 0 {
			// This is just a system error. We messed up and called both exclusive options.
			return syserror.New("cannot set targetPaths/targetExcludePaths with protoFileTargetPaths")
		}
	}
	// These are actual user errors. This is us verifying --path and --exclude-path.
	// An argument could be made this should be at a higher level for user errors, and then
	// if it gets to this point, this should be a system error.
	//
	// We don't use --path, --exclude-path here because these paths have had ExternalPathToPath
	// applied to them. Which is another argument to do this at a higher level.
	for _, targetPath := range bucketTargeting.TargetPaths() {
		if targetPath == bucketTargeting.SubDirPath() {
			// The targetPath/SubDirPath may not equal something on the command line as we have done
			// targeting via workspaces by now, so do not print them.
			return errors.New("given input is equal to a value of --path, this has no effect and is disallowed")
		}
		// We want this to be deterministic.  We don't have that many paths in almost all cases.
		// This being n^2 shouldn't be a huge issue unless someone has a diabolical wrapping shell script.
		// If this becomes a problem, there's optimizations we can do by turning targetExcludePaths into
		// a map but keeping the index in targetExcludePaths around to prioritize what error
		// message to print.
		for _, targetExcludePath := range bucketTargeting.TargetExcludePaths() {
			if targetPath == targetExcludePath {
				unnormalizedTargetPath := filepath.Clean(normalpath.Unnormalize(targetPath))
				return fmt.Errorf(`cannot set the same path for both --path and --exclude-path: "%s"`, unnormalizedTargetPath)
			}
			// This is new post-refactor. Before, we gave precedence to --path. While a change,
			// doing --path foo/bar --exclude-path foo seems like a bug rather than expected behavior to maintain.
			if normalpath.EqualsOrContainsPath(targetExcludePath, targetPath, normalpath.Relative) {
				// We clean and unnormalize the target paths to show in the error message
				unnormalizedTargetExcludePath := filepath.Clean(normalpath.Unnormalize(targetExcludePath))
				unnormalizedTargetPath := filepath.Clean(normalpath.Unnormalize(targetPath))
				return fmt.Errorf(`excluded path "%s" contains targeted path "%s", which means all paths in "%s" will be excluded`, unnormalizedTargetExcludePath, unnormalizedTargetPath, unnormalizedTargetPath)
			}
		}
	}
	for _, targetExcludePath := range bucketTargeting.TargetExcludePaths() {
		if targetExcludePath == bucketTargeting.SubDirPath() {
			unnormalizedTargetSubDirPath := filepath.Clean(normalpath.Unnormalize(bucketTargeting.SubDirPath()))
			unnormalizedTargetExcludePath := filepath.Clean(normalpath.Unnormalize(targetExcludePath))
			return fmt.Errorf(`given input "%s" is equal to a value of --exclude-path "%s", this would exclude everything`, unnormalizedTargetSubDirPath, unnormalizedTargetExcludePath)
		}
	}
	return nil
}

func getMappedModuleBucketAndModuleTargeting(
	ctx context.Context,
	config *workspaceBucketConfig,
	workspaceBucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	moduleDirPath string,
	moduleConfig bufconfig.ModuleConfig,
	isTargetModule bool,
	// If true and if a workspace module does not have a license/doc at its moduleDirPath,
	// use the license/doc respectively at the workspace root for this module.
	useWorkspaceLicenseDocIfNotFoundAtMoudle bool,
) (storage.ReadBucket, *moduleTargeting, error) {
	moduleBucket := storage.MapReadBucket(
		workspaceBucket,
		storage.MapOnPrefix(moduleDirPath),
	)
	rootToExcludes := moduleConfig.RootToExcludes()
	rootToIncludes := moduleConfig.RootToIncludes()
	var rootBuckets []storage.ReadBucket
	for root, excludes := range rootToExcludes {
		includes, ok := rootToIncludes[root]
		if !ok {
			// This should never happen because ModuleConfig guarantees that they have the same keys.
			return nil, nil, syserror.Newf("expected root %q to be also in rootToIncludes but not found", root)
		}
		// Roots only applies to .proto files.
		mappers := []storage.Mapper{
			storage.MapOnPrefix(root),
		}
		matchers := []storage.Matcher{
			// need to do match extension here
			// https://github.com/bufbuild/buf/issues/113
			storage.MatchPathExt(".proto"),
		}
		if len(excludes) != 0 {
			var notOrMatchers []storage.Matcher
			for _, exclude := range excludes {
				notOrMatchers = append(
					notOrMatchers,
					storage.MatchPathContained(exclude),
				)
			}
			matchers = append(
				matchers,
				storage.MatchNot(
					storage.MatchOr(
						notOrMatchers...,
					),
				),
			)
		}
		// An includes with length 0 adds no filter to the proto files.
		if len(includes) > 0 {
			var orMatchers []storage.Matcher
			for _, include := range includes {
				orMatchers = append(
					orMatchers,
					storage.MatchPathContained(include),
				)
			}
			matchers = append(
				matchers,
				storage.MatchOr(
					orMatchers...,
				),
			)
		}
		rootBuckets = append(
			rootBuckets,
			storage.FilterReadBucket(
				storage.MapReadBucket(
					moduleBucket,
					mappers...,
				),
				matchers...,
			),
		)
	}
	docStorageReadBucket, err := bufmodule.GetDocStorageReadBucket(ctx, moduleBucket)
	if err != nil {
		return nil, nil, err
	}
	licenseStorageReadBucket, err := bufmodule.GetLicenseStorageReadBucket(ctx, moduleBucket)
	if err != nil {
		return nil, nil, err
	}
	if useWorkspaceLicenseDocIfNotFoundAtMoudle {
		isModuleDocBucketEmpty, err := storage.IsEmpty(ctx, docStorageReadBucket, "")
		if err != nil {
			return nil, nil, err
		}
		// If at moduleDirPath there isn't a doc file, we fall back to use the doc file
		// at the workspace root if it exists.
		if isModuleDocBucketEmpty {
			// We do not need to check if a doc file exists at the workspace root by
			// checking whether the doc bucket for the workspace is empty, because
			// this bucket will just be empty there isn't one, which is what we want.
			docStorageReadBucket, err = bufmodule.GetDocStorageReadBucket(ctx, workspaceBucket)
			if err != nil {
				return nil, nil, err
			}
		}
		isModuleLicenseBucketEmpty, err := storage.IsEmpty(ctx, licenseStorageReadBucket, "")
		if err != nil {
			return nil, nil, err
		}
		// If at moduleDirPath there isn't a license, we fall back to use the license
		// at the workspace root if it exists.
		if isModuleLicenseBucketEmpty {
			// We do not need to check if this bucket is empty for the same reason, see comment for doc bucket.
			licenseStorageReadBucket, err = bufmodule.GetLicenseStorageReadBucket(ctx, workspaceBucket)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	rootBuckets = append(
		rootBuckets,
		docStorageReadBucket,
		licenseStorageReadBucket,
	)
	mappedModuleBucket := storage.MultiReadBucket(rootBuckets...)
	moduleTargeting, err := newModuleTargeting(
		moduleDirPath,
		slicesext.MapKeysToSlice(rootToExcludes),
		bucketTargeting,
		config,
		isTargetModule,
	)
	if err != nil {
		return nil, nil, err
	}
	return mappedModuleBucket, moduleTargeting, nil
}

func getModuleConfigAndConfiguredDepModuleRefsV1Beta1OrV1(
	ctx context.Context,
	bucket storage.ReadBucket,
	moduleDirPath string,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
) (bufconfig.ModuleConfig, []bufmodule.ModuleRef, error) {
	var bufYAMLFile bufconfig.BufYAMLFile
	var err error
	if overrideBufYAMLFile != nil {
		bufYAMLFile = overrideBufYAMLFile
	} else {
		bufYAMLFile, err = bufconfig.GetBufYAMLFileForPrefix(ctx, bucket, moduleDirPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// If we do not have a buf.yaml, we use the default config.
				// This is a v1 config.
				return bufconfig.DefaultModuleConfigV1, nil, nil
			}
			return nil, nil, err
		}
	}
	// Ensure buf.yaml matches the expected version for a v1 module.
	if bufYAMLFile.FileVersion() != bufconfig.FileVersionV1Beta1 && bufYAMLFile.FileVersion() != bufconfig.FileVersionV1 {
		return nil, nil, fmt.Errorf("buf.yaml at %s did not have version v1beta1 or v1", moduleDirPath)
	}
	moduleConfigs := bufYAMLFile.ModuleConfigs()
	if len(moduleConfigs) != 1 {
		// This is a system error. This should never happen.
		return nil, nil, syserror.Newf("received %d ModuleConfigs from a v1beta1 or v1 BufYAMLFile", len(moduleConfigs))
	}
	return moduleConfigs[0], bufYAMLFile.ConfiguredDepModuleRefs(), nil
}

// checkForControllingWorkspaceOrV1Module take a bucket and path, and walks up the bucket
// from the base of the path, checking for a controlling workspace or v1 module.
//
// If ignoreWorkspaceCheck is set to true, then we only look for v1 modules. This is done
// for the input, since we already did the workspace check in the initial bucketTargeting.
// Note that this is something that we could build into the initial bucketTargeting,
// however, moving forward, we want to encourage users to move to v2 workspaces, so it is
// nice to be able to isolate this as fallback logic here.
func checkForControllingWorkspaceOrV1Module(
	ctx context.Context,
	logger *slog.Logger,
	bucket storage.ReadBucket,
	path string,
	ignoreWorkspaceCheck bool,
) (buftarget.ControllingWorkspace, error) {
	path = normalpath.Normalize(path)
	// Keep track of any v1 module found along the way. If we find a v1 or v2 workspace, we
	// return that over the v1 module, but we return this as the fallback.
	var fallbackV1Module buftarget.ControllingWorkspace
	// Similar to the mapping loop in buftarget for buftarget.BucketTargeting, we can't do
	// this in a traditional loop like this:
	//
	// for curDirPath := path; curDirPath != "."; curDirPath = normalpath.Dir(curDirPath) {
	//
	// If we do that, then we don't run terminateFunc for ".", which we want to so that we get
	// the correct value for the terminate bool.
	//
	// Instead, we effectively do a do-while loop.
	curDirPath := path
	for {
		if !ignoreWorkspaceCheck {
			controllingWorkspace, err := buftarget.TerminateAtControllingWorkspace(
				ctx,
				bucket,
				curDirPath,
				path,
			)
			if err != nil {
				return nil, err
			}
			if controllingWorkspace != nil {
				logger.DebugContext(
					ctx,
					"buffetch termination found",
					slog.String("curDirPath", curDirPath),
					slog.String("path", path),
				)
				return controllingWorkspace, nil
			}
		}
		// Then check for a v1 module
		v1Module, err := buftarget.TerminateAtV1Module(ctx, bucket, curDirPath, path)
		if err != nil {
			return nil, err
		}
		if v1Module != nil {
			if fallbackV1Module != nil {
				return nil, fmt.Errorf("nested modules %q and %q are not allowed", fallbackV1Module.Path(), v1Module.Path())
			}
			fallbackV1Module = v1Module
		}
		if curDirPath == "." {
			break
		}
		curDirPath = normalpath.Dir(curDirPath)
	}
	logger.DebugContext(
		ctx,
		"buffetch no termination found",
		slog.String("path", path),
	)
	return fallbackV1Module, nil
}

func checkForOverlap(
	ctx context.Context,
	bucket storage.ReadBucket,
	inputPath string,
	moduleDirPaths []string,
) error {
	for _, moduleDirPath := range moduleDirPaths {
		if normalpath.ContainsPath(moduleDirPath, inputPath, normalpath.Relative) {
			// In the case where the inputPath would appear to be relative to moduleDirPath,
			// but does not exist, for example, moduleDirPath == "." and inputPath == "fake-path",
			// or moduleDirPath == "real-path" and inputPath == "real-path/fake-path", the error
			// returned below is not very clear (in particular the first case, "." and "fake-path").
			// We do a check here and return ErrNoTargetProtoFiles if the inputPath is empty
			// and/or it does not exist.
			empty, err := storage.IsEmpty(ctx, bucket, inputPath)
			if err != nil {
				return err
			}
			if empty {
				return bufmodule.ErrNoTargetProtoFiles
			}
			return fmt.Errorf("failed to build input %q because it is contained by module at path %q specified in your configuration, you must provide the workspace or module as the input, and filter to this path using --path", inputPath, moduleDirPath)
		}
	}
	return fmt.Errorf("input %q did not contain modules found in workspace %v", inputPath, moduleDirPaths)
}

// bucketIDsForModuleConfigsV2 returns the bucket IDs for the given moduleDirPaths in the same order.
// moduleDirPaths must already be normalized.
//
// v1 does not need to call a function like this because bucketIDs are just their module roots relative to the workspace root, which are unique.
func bucketIDsForModuleConfigsV2(moduleConfigs []bufconfig.ModuleConfig) []string {
	moduleDirPaths := slicesext.Map(moduleConfigs, bufconfig.ModuleConfig.DirPath)
	// In a v2 bufYAMLFile, multiple module configs may have the same DirPath, but we still want to
	// make sure each local module has a unique BucketID, which means we cannot use their DirPaths as
	// BucketIDs directly. Instead, we append an index (1-indexed) to each DirPath to deduplicate, and
	// each module's bucketID becomes "<path>[index]", except for the first one does not need the index
	// and its bucketID is stil "<path>".
	// As an example, bucketIDs are shown for modules in the buf.yaml below:
	// ...
	// modules:
	//   - path: foo # bucketID: foo
	//   - path: bar # bucketID: bar
	//   - path: foo # bucketID: foo-2
	//   - path: bar # bucketID: bar-2
	//   - path: bar # bucketID: bar-3
	//   - path: new # bucketID: new
	//   - path: foo # bucketID: foo-3
	// ...
	// Note: The BufYAMLFile interface guarantees that the relative order among module configs with
	// the same path is the same order among these modules in the external buf.yaml v2, e.g. the 2nd
	// "foo" in the external buf.yaml above is also the 2nd "foo" in module configs, even though sorted:
	// [bar, bar, bar, foo, foo, foo, new].
	//                       ^
	bucketIDs := bucketIDsForDirPaths(moduleDirPaths, false)
	if len(slicesext.Duplicates(bucketIDs)) == 0 {
		// This approach does not produce duplicate bucketIDs in the result most of the time, we return
		// the bucketIDs if we don't detect any duplicates and it gives us the nice property that for
		// 99% of the v2 workspaces, each module's bucketID is its DirPath.
		return bucketIDs
	}
	// However, the approach above may create duplicates in the result bucketIDs, for example:
	// ...
	// modules:
	//   - path: foo-2 # bucketID: foo-2
	//   - path: foo   # bucketID: foo
	//   - path: foo   # bucketID: foo-2
	// Notice that both the module at "foo-2" and the second module at "foo" have bucketID "foo-2".
	// In this case, we append an index to every DirPath, including the first occurrence of each DirPath.
	return bucketIDsForDirPaths(moduleDirPaths, true)
}

func bucketIDsForDirPaths(moduleDirPaths []string, firstIDHasSuffix bool) []string {
	bucketIDs := make([]string, 0, len(moduleDirPaths))
	// Use dirPathToRunningCount to keep track of how many modules of this path has been seen (before the
	// current module) in this BufYAMLFile.
	dirPathToRunningCount := make(map[string]int)
	for _, moduleDirPath := range moduleDirPaths {
		// If n modules before this one has the same DirPath, then this module has index n+1 (1-indexed).
		currentModuleDirPathIndex := dirPathToRunningCount[moduleDirPath] + 1
		bucketID := fmt.Sprintf("%s-%d", moduleDirPath, currentModuleDirPathIndex)
		if currentModuleDirPathIndex == 1 && !firstIDHasSuffix {
			bucketID = moduleDirPath
		}
		bucketIDs = append(bucketIDs, bucketID)
		dirPathToRunningCount[moduleDirPath]++
	}
	return bucketIDs
}
