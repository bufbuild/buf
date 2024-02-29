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

	"github.com/bufbuild/buf/private/buf/buftarget"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"go.uber.org/zap"
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
// a single module directory path, which is equivalent to the input path.
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
	bucket          storage.ReadBucket
	moduleTargeting *moduleTargeting
}

func newWorkspaceTargeting(
	ctx context.Context,
	logger *zap.Logger,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	overrideBufYAMLFile bufconfig.BufYAMLFile,
	ignoreAndDisallowV1BufWorkYAMLs bool,
) (*workspaceTargeting, error) {
	if overrideBufYAMLFile != nil {
		logger.Debug(
			"targeting workspace with config override",
			zap.String("input path", bucketTargeting.InputPath()),
		)
		switch fileVersion := overrideBufYAMLFile.FileVersion(); fileVersion {
		case bufconfig.FileVersionV1Beta1, bufconfig.FileVersionV1:
			return v1WorkspaceTargeting(
				ctx,
				config,
				bucket,
				bucketTargeting,
				[]string{bucketTargeting.InputPath()},
				overrideBufYAMLFile,
			)
		case bufconfig.FileVersionV2:
			return v2WorkspaceTargeting(ctx, config, bucket, bucketTargeting, overrideBufYAMLFile)
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}
	if controllingWorkspace := bucketTargeting.ControllingWorkspace(); controllingWorkspace != nil {
		// This is a v2 workspace.
		if controllingWorkspace.BufYAMLFile() != nil {
			logger.Debug(
				"targeting workspace based on v2 buf.yaml",
				zap.String("input path", bucketTargeting.InputPath()),
			)
			return v2WorkspaceTargeting(ctx, config, bucket, bucketTargeting, controllingWorkspace.BufYAMLFile())
		}
		// This is a v1 workspace.
		if bufWorkYAMLFile := controllingWorkspace.BufWorkYAMLFile(); bufWorkYAMLFile != nil {
			if ignoreAndDisallowV1BufWorkYAMLs {
				// This means that we attempted to target a v1 workspace at the buf.work.yaml, not
				// an individual module within the v1 workspace defined in buf.work.yaml.
				// This is disallowed.
				if bucketTargeting.InputPath() == "." {
					return nil, errors.New(`Workspaces defined with buf.work.yaml cannot be updated or pushed, only
the individual modules within a workspace can be updated or pushed. Workspaces
defined with a v2 buf.yaml can be updated, see the migration documentation for more details.`)
				}
				// We targeted a specific module within the workspace. Based on the option we provided, we're going to ignore
				// the workspace entirely, and just act as if the buf.work.yaml did not exist.
				logger.Debug(
					"targeting workspace, ignoring v1 buf.work.yaml, just building on module at target",
					zap.String("input path", bucketTargeting.InputPath()),
				)
				return v1WorkspaceTargeting(
					ctx,
					config,
					bucket,
					bucketTargeting,
					[]string{bucketTargeting.InputPath()}, // Assume we are targeting only the module at the input path
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
	logger.Debug(
		"targeting workspace with no found buf.work.yaml or buf.yaml",
		zap.String("input path", bucketTargeting.InputPath()),
	)
	// We did not find any buf.work.yaml or buf.yaml, we invoke fallback logic.
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
) (*workspaceTargeting, error) {
	// We keep track of if any module was tentatively targeted, and then actually targeted via
	// the paths flags. We use this pre-building of the ModuleSet to see if the --path and
	// --exclude-path flags resulted in no targeted modules. This condition is represented
	// by hadIsTentativelyTargetModule == true && hadIsTargetModule = false
	//
	// If hadIsTentativelyTargetModule is false, this means that our input bucketTargeting.InputPath() was not
	// actually representative of any module that we detected in buf.work.yaml or v2 buf.yaml
	// directories, and this is a system error - this should be verified before we reach this function.
	var hadIsTentativelyTargetModule bool
	var hadIsTargetModule bool
	moduleDirPaths := make([]string, 0, len(bufYAMLFile.ModuleConfigs()))
	bucketIDToModuleConfig := make(map[string]bufconfig.ModuleConfig)
	moduleBucketsAndTargeting := make([]*moduleBucketAndModuleTargeting, 0, len(bufYAMLFile.ModuleConfigs()))
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		moduleDirPath := moduleConfig.DirPath()
		moduleDirPaths = append(moduleDirPaths, moduleDirPath)
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		// bucketTargeting.InputPath() is the input targetSubDirPath. We only want to target modules that are inside
		// this targetSubDirPath. Example: bufWorkYAMLDirPath is "foo", targetSubDirPath is "foo/bar",
		// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
		// and "foo/bar/bat".
		//
		// This is new behavior - before, we required that you input an exact match for the module directory path,
		// but now, we will take all the modules underneath this workspace.
		isTentativelyTargetModule := normalpath.EqualsOrContainsPath(bucketTargeting.InputPath(), moduleDirPath, normalpath.Relative)
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
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleBucketsAndTargeting = append(moduleBucketsAndTargeting, &moduleBucketAndModuleTargeting{
			bucket:          mappedModuleBucket,
			moduleTargeting: moduleTargeting,
		})
	}
	if !hadIsTentativelyTargetModule {
		// Check if the input is overlapping within a module dir path. If so, return a nicer
		// error. In the future, we want to remove special treatment for input path, and it
		// should be treated just like any target path.
		return nil, checkForOverlap(bucketTargeting.InputPath(), moduleDirPaths)
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
	// If hadIsTentativelyTargetModule is false, this means that our input bucketTargeting.InputPath() was not
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
		bucketIDToModuleConfig[moduleDirPath] = moduleConfig
		// We only want to target modules that are inside the bucketTargeting.InputPath().
		// Example: bufWorkYAMLDirPath is "foo", bucketTargeting.InputPath() is "foo/bar",
		// listed directories are "bar/baz", "bar/bat", "other". We want to include "foo/bar/baz"
		// and "foo/bar/bat".
		//
		// This is new behavior - before, we required that you input an exact match for the module directory path,
		// but now, we will take all the modules underneath this workspace.
		isTentativelyTargetModule := normalpath.EqualsOrContainsPath(bucketTargeting.InputPath(), moduleDirPath, normalpath.Relative)
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
		)
		if err != nil {
			return nil, err
		}
		if moduleTargeting.isTargetModule {
			hadIsTargetModule = true
		}
		moduleBucketsAndTargeting = append(moduleBucketsAndTargeting, &moduleBucketAndModuleTargeting{
			bucket:          mappedModuleBucket,
			moduleTargeting: moduleTargeting,
		})
	}
	if !hadIsTentativelyTargetModule {
		// Check if the input is overlapping within a module dir path. If so, return a nicer
		// error. In the future, we want to remove special treatment for input path, and it
		// should be treated just like any target path.
		return nil, checkForOverlap(bucketTargeting.InputPath(), moduleDirPaths)
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
// 1. We check if there are target paths. If there are no target paths, then we return
// a v1 workspace with the only module directory path as the input, since we have no other
// information to go on.
//
// 2. If there are target paths, then we check from each target path, walking up, if there
// is a workspace or v1 module.
//
//	a. If we find nothing, then same fallback as 1.
//	b. If we find a v1 or v2 workspace, we ensure that all them resolve to the same workspace.
//	c. If we find no workspace, then we can check for v1 module configs.
func fallbackWorkspaceTargeting(
	ctx context.Context,
	logger *zap.Logger,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
) (*workspaceTargeting, error) {
	// No target paths, return a v1 workspace with a single module dir path at the input
	if len(bucketTargeting.TargetPaths()) == 0 {
		return v1WorkspaceTargeting(
			ctx,
			config,
			bucket,
			bucketTargeting,
			[]string{bucketTargeting.InputPath()},
			nil,
		)
	}
	var v1BufWorkYAML bufconfig.BufWorkYAMLFile
	var v2BufYAMLFile bufconfig.BufYAMLFile
	var controllingWorkspacePath string
	var v1ModulePaths []string
	for _, targetPath := range bucketTargeting.TargetPaths() {
		controllingWorkspaceOrModule, err := checkForControllingWorkspaceOrV1Module(
			ctx,
			logger,
			bucket,
			targetPath,
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
	// We should not have found a controlling workspace and also have a list of v1 modules we
	// need to build.
	// In the future, we may be able to support multi-building, but for now, we error. We have
	// the option to prioritise the workspace here and drop the v1ModulePaths, but that might
	// be confusing to users.
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
		)
	}
	if v1BufWorkYAML != nil {
		v1ModulePaths = v1BufWorkYAML.DirPaths()
	}
	// If we still have no v1 module paths, then we go to the final fallback and set a v1
	// module at the input path.
	if len(v1ModulePaths) == 0 {
		v1ModulePaths = append(v1ModulePaths, bucketTargeting.InputPath())
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

func getMappedModuleBucketAndModuleTargeting(
	ctx context.Context,
	config *workspaceBucketConfig,
	bucket storage.ReadBucket,
	bucketTargeting buftarget.BucketTargeting,
	moduleDirPath string,
	moduleConfig bufconfig.ModuleConfig,
	isTargetModule bool,
) (storage.ReadBucket, *moduleTargeting, error) {
	moduleBucket := storage.MapReadBucket(
		bucket,
		storage.MapOnPrefix(moduleDirPath),
	)
	rootToExcludes := moduleConfig.RootToExcludes()
	var rootBuckets []storage.ReadBucket
	for root, excludes := range rootToExcludes {
		// Roots only applies to .proto files.
		mappers := []storage.Mapper{
			// need to do match extension here
			// https://github.com/bufbuild/buf/issues/113
			storage.MatchPathExt(".proto"),
			storage.MapOnPrefix(root),
		}
		if len(excludes) != 0 {
			var notOrMatchers []storage.Matcher
			for _, exclude := range excludes {
				notOrMatchers = append(
					notOrMatchers,
					storage.MatchPathContained(exclude),
				)
			}
			mappers = append(
				mappers,
				storage.MatchNot(
					storage.MatchOr(
						notOrMatchers...,
					),
				),
			)
		}
		rootBuckets = append(
			rootBuckets,
			storage.MapReadBucket(
				moduleBucket,
				mappers...,
			),
		)
	}
	rootBuckets = append(
		rootBuckets,
		bufmodule.GetDocStorageReadBucket(ctx, moduleBucket),
		bufmodule.GetLicenseStorageReadBucket(moduleBucket),
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
	// Just a sanity check. This should have already been validated, but let's make sure.
	if bufYAMLFile.FileVersion() != bufconfig.FileVersionV1Beta1 && bufYAMLFile.FileVersion() != bufconfig.FileVersionV1 {
		return nil, nil, syserror.Newf("buf.yaml at %s did not have version v1beta1 or v1", moduleDirPath)
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
func checkForControllingWorkspaceOrV1Module(
	ctx context.Context,
	logger *zap.Logger,
	bucket storage.ReadBucket,
	path string,
) (buftarget.ControllingWorkspace, error) {
	path, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return nil, err
	}
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
		// First, check for a v1 or v2 controlling workspace
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
			logger.Debug(
				"buffetch termination found",
				zap.String("curDirPath", curDirPath),
				zap.String("path", path),
			)
			return controllingWorkspace, nil
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
	logger.Debug(
		"buffetch no termination found",
		zap.String("path", path),
	)
	return fallbackV1Module, nil
}

func checkForOverlap(inputPath string, moduleDirPaths []string) error {
	for _, moduleDirPath := range moduleDirPaths {
		if normalpath.ContainsPath(moduleDirPath, inputPath, normalpath.Relative) {
			return fmt.Errorf("failed to build input %q because it is contained by directory %q", inputPath, moduleDirPath)
		}
	}
	return fmt.Errorf("input path %q did not contain modules found in workspace %v", inputPath, moduleDirPaths)
}
