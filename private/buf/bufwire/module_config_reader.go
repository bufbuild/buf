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

package bufwire

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/buflock"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type moduleConfigReader struct {
	logger              *zap.Logger
	storageosProvider   storageos.Provider
	fetchReader         buffetch.Reader
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
	tracer              trace.Tracer
}

func newModuleConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) *moduleConfigReader {
	return &moduleConfigReader{
		logger:              logger,
		storageosProvider:   storageosProvider,
		fetchReader:         fetchReader,
		moduleBucketBuilder: moduleBucketBuilder,
		tracer:              otel.GetTracerProvider().Tracer("bufbuild/buf"),
	}
}

func (m *moduleConfigReader) GetModuleConfigSet(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfigSet, retErr error) {
	ctx, span := m.tracer.Start(ctx, "get_module_config")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()
	// We construct a new WorkspaceBuilder here so that the cache is only used for a single call.
	workspaceBuilder := bufwork.NewWorkspaceBuilder()
	switch t := sourceOrModuleRef.(type) {
	case buffetch.ProtoFileRef:
		return m.getProtoFileModuleSourceConfigSet(
			ctx,
			container,
			t,
			workspaceBuilder,
			configOverride,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	case buffetch.SourceRef:
		return m.getSourceModuleConfigSet(
			ctx,
			container,
			t,
			workspaceBuilder,
			configOverride,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	case buffetch.ModuleRef:
		moduleConfig, err := m.getModuleModuleConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
		if err != nil {
			return nil, err
		}
		return newModuleConfigSet(
			[]ModuleConfig{
				moduleConfig,
			},
			nil,
		), nil
	default:
		return nil, fmt.Errorf("invalid ref: %T", sourceOrModuleRef)
	}
}

func (m *moduleConfigReader) getSourceModuleConfigSet(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceRef buffetch.SourceRef,
	workspaceBuilder bufwork.WorkspaceBuilder,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfigSet, retErr error) {
	readBucketCloser, err := m.fetchReader.GetSourceBucket(ctx, container, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	existingConfigFilePath, err := bufwork.ExistingConfigFilePath(ctx, readBucketCloser)
	if err != nil {
		return nil, err
	}
	if existingConfigFilePath != "" {
		return m.getWorkspaceModuleConfigSet(
			ctx,
			sourceRef,
			workspaceBuilder,
			readBucketCloser,
			readBucketCloser.RelativeRootPath(),
			readBucketCloser.SubDirPath(),
			configOverride,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	}
	moduleConfig, err := m.getSourceModuleConfig(
		ctx,
		sourceRef,
		readBucketCloser,
		readBucketCloser.RelativeRootPath(),
		readBucketCloser.SubDirPath(),
		configOverride,
		workspaceBuilder,
		nil,
		nil,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, err
	}
	return newModuleConfigSet(
		[]ModuleConfig{
			moduleConfig,
		},
		nil,
	), nil
}

func (m *moduleConfigReader) getModuleModuleConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	moduleRef buffetch.ModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfig, retErr error) {
	module, err := m.fetchReader.GetModule(ctx, container, moduleRef)
	if err != nil {
		return nil, err
	}
	var targetPaths []string
	if len(externalDirOrFilePaths) > 0 {
		targetPaths = make([]string, len(externalDirOrFilePaths))
		for i, externalDirOrFilePath := range externalDirOrFilePaths {
			targetPath, err := moduleRef.PathForExternalPath(externalDirOrFilePath)
			if err != nil {
				return nil, err
			}
			targetPaths[i] = targetPath
		}
	}
	var excludePaths []string
	if len(externalExcludeDirOrFilePaths) > 0 {
		excludePaths = make([]string, len(externalExcludeDirOrFilePaths))
		for i, excludeDirOrFilePath := range externalExcludeDirOrFilePaths {
			excludePath, err := moduleRef.PathForExternalPath(excludeDirOrFilePath)
			if err != nil {
				return nil, err
			}
			excludePaths[i] = excludePath
		}
	}
	if externalDirOrFilePathsAllowNotExist {
		module, err = bufmodule.ModuleWithTargetPathsAllowNotExist(module, targetPaths, excludePaths)
		if err != nil {
			return nil, err
		}
	} else {
		module, err = bufmodule.ModuleWithTargetPaths(module, targetPaths, excludePaths)
		if err != nil {
			return nil, err
		}
	}
	// TODO: we should read the config from the module when configuration
	// is added to modules
	readWriteBucket, err := m.storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	config, err := bufconfig.ReadConfigOS(
		ctx,
		readWriteBucket,
		bufconfig.ReadConfigOSWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	return newModuleConfig(module, config), nil
}

func (m *moduleConfigReader) getProtoFileModuleSourceConfigSet(
	ctx context.Context,
	container app.EnvStdinContainer,
	protoFileRef buffetch.ProtoFileRef,
	workspaceBuilder bufwork.WorkspaceBuilder,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (_ ModuleConfigSet, retErr error) {
	readBucketCloser, err := m.fetchReader.GetSourceBucket(ctx, container, protoFileRef)
	if err != nil {
		return nil, err
	}
	workspaceConfigs := slicesext.ToStructMap(bufwork.AllConfigFilePaths)
	moduleConfigs := slicesext.ToStructMap(bufconfig.AllConfigFilePaths)
	terminateFileProvider := readBucketCloser.TerminateFileProvider()
	var workspaceConfigDirectory string
	var moduleConfigDirectory string
	for _, terminateFile := range terminateFileProvider.GetTerminateFiles() {
		if _, ok := workspaceConfigs[terminateFile.Name()]; ok {
			workspaceConfigDirectory = terminateFile.Path()
			continue
		}
		if _, ok := moduleConfigs[terminateFile.Name()]; ok {
			moduleConfigDirectory = terminateFile.Path()
		}
	}
	// If a workspace and module are both found, then we need to check of the module is within
	// the workspace. If it is, we use the workspace. Otherwise, we use the module.
	if workspaceConfigDirectory != "" {
		if moduleConfigDirectory != "" {
			relativePath, err := normalpath.Rel(workspaceConfigDirectory, moduleConfigDirectory)
			if err != nil {
				return nil, err
			}
			readBucketCloser.SetSubDirPath(normalpath.Normalize(relativePath))
		} else {
			// If there are no module configs in the path to the workspace, we need to check whether or not
			// proto file ref is contained within one of the workspace directories.
			// If yes, we can set the `SubDirPath` for the bucket to the directory, to ensure we build all the
			// dependencies for the directory. If not, then we will keep the `SubDirPath` as the working directory.
			workspaceConfig, err := bufwork.GetConfigForBucket(ctx, readBucketCloser, readBucketCloser.RelativeRootPath())
			if err != nil {
				return nil, err
			}
			for _, directory := range workspaceConfig.Directories {
				if normalpath.EqualsOrContainsPath(directory, readBucketCloser.SubDirPath(), normalpath.Relative) {
					readBucketCloser.SetSubDirPath(normalpath.Normalize(directory))
					break
				}
			}
		}
		return m.getWorkspaceModuleConfigSet(
			ctx,
			protoFileRef,
			workspaceBuilder,
			readBucketCloser,
			readBucketCloser.RelativeRootPath(),
			readBucketCloser.SubDirPath(),
			configOverride,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
	}
	moduleConfig, err := m.getSourceModuleConfig(
		ctx,
		protoFileRef,
		readBucketCloser,
		readBucketCloser.RelativeRootPath(),
		readBucketCloser.SubDirPath(),
		configOverride,
		workspaceBuilder,
		nil,
		nil,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, err
	}
	return newModuleConfigSet(
		[]ModuleConfig{
			moduleConfig,
		},
		nil,
	), nil
}

func (m *moduleConfigReader) getWorkspaceModuleConfigSet(
	ctx context.Context,
	sourceRef buffetch.SourceRef,
	workspaceBuilder bufwork.WorkspaceBuilder,
	readBucket storage.ReadBucket,
	relativeRootPath string,
	subDirPath string,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (ModuleConfigSet, error) {
	workspaceConfig, err := bufwork.GetConfigForBucket(ctx, readBucket, relativeRootPath)
	if err != nil {
		return nil, err
	}
	workspace, err := workspaceBuilder.BuildWorkspace(
		ctx,
		workspaceConfig,
		readBucket,
		relativeRootPath,
		subDirPath, // this is used to only apply the config override to this directory
		configOverride,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, err
	}
	if subDirPath != "." {
		moduleConfig, err := m.getSourceModuleConfig(
			ctx,
			sourceRef,
			readBucket,
			relativeRootPath,
			subDirPath,
			configOverride,
			workspaceBuilder,
			workspaceConfig,
			workspace,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
		if err != nil {
			return nil, err
		}
		return newModuleConfigSet(
			[]ModuleConfig{
				moduleConfig,
			},
			workspace,
		), nil
	}
	if configOverride != "" {
		return nil, errors.New("the --config flag is not compatible with workspaces")
	}
	// The target subDirPath points to the workspace configuration,
	// so we construct a separate workspace for each of the configured
	// directories.
	var moduleConfigs []ModuleConfig
	// We need to first get the map of externalDirOrFilePath to subDirRelPath and the map
	// of excludeDirOrFilePath to subDirRelExcludePath so we can check that all paths that
	// have been provided at the top level have been accounted for across the workspace.
	externalPathToRelPaths := make(map[string]string)
	externalExcludePathToRelPaths := make(map[string]string)
	for _, directory := range workspaceConfig.Directories {
		// We are unfortunately adding this logic in two difference places, once at the top level
		// here, and when we build each workspace for the build options. We need to do the work
		// at this level because we need to check across all workspaces once.
		// We need the same logic again for each workspace build because a module can span across
		// several workspaces.
		// That being said, the work will be done once, since the module build may be cached as
		// as a dependency via bufwork.BuildWorkspace, so the module will always be built once.
		externalPathToSubDirRelPaths, err := bufwork.ExternalPathsToSubDirRelPaths(
			relativeRootPath,
			directory,
			externalDirOrFilePaths,
		)
		if err != nil {
			return nil, err
		}
		externalExcludeToSubDirRelExcludePaths, err := bufwork.ExternalPathsToSubDirRelPaths(
			relativeRootPath,
			directory,
			externalExcludeDirOrFilePaths,
		)
		if err != nil {
			return nil, err
		}
		for externalFileOrDirPath, subDirRelPath := range externalPathToSubDirRelPaths {
			externalPathToRelPaths[externalFileOrDirPath] = subDirRelPath
		}
		for excludeFileOrDirPath, subDirRelExcludePath := range externalExcludeToSubDirRelExcludePaths {
			externalExcludePathToRelPaths[excludeFileOrDirPath] = subDirRelExcludePath
		}
		moduleConfig, err := m.getSourceModuleConfig(
			ctx,
			sourceRef,
			readBucket,
			relativeRootPath,
			directory,
			configOverride,
			workspaceBuilder,
			workspaceConfig,
			workspace,
			externalDirOrFilePaths,
			externalExcludeDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
		)
		if err != nil {
			return nil, err
		}
		moduleConfigs = append(moduleConfigs, moduleConfig)
	}
	// This is only a requirement if we do not allow paths to not exist.
	if !externalDirOrFilePathsAllowNotExist {
		for _, externalDirOrFilePath := range externalDirOrFilePaths {
			if _, ok := externalPathToRelPaths[externalDirOrFilePath]; !ok {
				return nil, fmt.Errorf("path does not exist: %s", externalDirOrFilePath)
			}
		}
		for _, excludeDirOrFilePath := range externalExcludeDirOrFilePaths {
			if _, ok := externalExcludePathToRelPaths[excludeDirOrFilePath]; !ok {
				return nil, fmt.Errorf("path does not exist: %s", excludeDirOrFilePath)
			}
		}
	}
	return newModuleConfigSet(moduleConfigs, workspace), nil
}

func (m *moduleConfigReader) getSourceModuleConfig(
	ctx context.Context,
	sourceRef buffetch.SourceRef,
	readBucket storage.ReadBucket,
	relativeRootPath string,
	subDirPath string,
	configOverride string,
	workspaceBuilder bufwork.WorkspaceBuilder,
	workspaceConfig *bufwork.Config,
	workspace bufmodule.Workspace,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (ModuleConfig, error) {
	mappedReadBucket := readBucket
	if subDirPath != "." {
		mappedReadBucket = storage.MapReadBucket(readBucket, storage.MapOnPrefix(subDirPath))
	}
	if err := buflock.CheckDeprecatedDigests(ctx, m.logger, mappedReadBucket); err != nil {
		return nil, err
	}
	if module, moduleConfig, ok := workspaceBuilder.GetModuleConfig(subDirPath); ok {
		// The module was already built while we were constructing the workspace.
		// However, we still need to perform some additional validation based on
		// the sourceRef.
		if len(externalDirOrFilePaths) > 0 {
			if workspaceDirectoryEqualsOrContainsSubDirPath(workspaceConfig, subDirPath) {
				// We have to do this ahead of time as we are not using PathForExternalPath
				// in this if branch. This is really bad.
				for _, externalDirOrFilePath := range externalDirOrFilePaths {
					if _, err := sourceRef.PathForExternalPath(externalDirOrFilePath); err != nil {
						return nil, err
					}
				}
			}
		}
		return newModuleConfig(module, moduleConfig), nil
	}
	moduleConfig, err := bufconfig.ReadConfigOS(
		ctx,
		mappedReadBucket,
		bufconfig.ReadConfigOSWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	var buildOptions []bufmodulebuild.BuildOption
	if len(externalDirOrFilePaths) > 0 {
		if workspaceDirectoryEqualsOrContainsSubDirPath(workspaceConfig, subDirPath) {
			// We have to do this ahead of time as we are not using PathForExternalPath
			// in this if branch. This is really bad.
			for _, externalDirOrFilePath := range externalDirOrFilePaths {
				if _, err := sourceRef.PathForExternalPath(externalDirOrFilePath); err != nil {
					return nil, err
				}
			}
			// The subDirPath is contained within one of the workspace directories, so
			// we first need to reformat the externalDirOrFilePaths so that they accommodate
			// for the relativeRootPath (the path to the directory containing the buf.work.yaml).
			//
			// For example,
			//
			//  $ buf build ../../proto --path ../../proto/buf
			//
			//  // buf.work.yaml
			//  version: v1
			//  directories:
			//    - proto
			//    - enterprise/proto
			//
			// Note that we CANNOT simply use the sourceRef because we would not be able to
			// determine which workspace directory the paths apply to afterwards. To be clear,
			// if we determined the bucketRelPath from the sourceRef, the bucketRelPath would be equal
			// to ./buf/... which is ambiguous to the workspace directories ('proto' and 'enterprise/proto'
			// in this case).
			externalPathToSubDirRelPaths, err := bufwork.ExternalPathsToSubDirRelPaths(
				relativeRootPath,
				subDirPath,
				externalDirOrFilePaths,
			)
			if err != nil {
				return nil, err
			}
			externalExcludeToSubDirRelExcludePaths, err := bufwork.ExternalPathsToSubDirRelPaths(
				relativeRootPath,
				subDirPath,
				externalExcludeDirOrFilePaths,
			)
			if err != nil {
				return nil, err
			}
			subDirRelPaths := make([]string, 0, len(externalPathToSubDirRelPaths))
			for _, subDirRelPath := range externalPathToSubDirRelPaths {
				subDirRelPaths = append(subDirRelPaths, subDirRelPath)
			}
			subDirRelExcludePaths := make([]string, 0, len(externalExcludeToSubDirRelExcludePaths))
			for _, subDirRelExcludePath := range externalExcludeToSubDirRelExcludePaths {
				subDirRelExcludePaths = append(subDirRelExcludePaths, subDirRelExcludePath)
			}
			buildOptions, err = bufwork.BuildOptionsForWorkspaceDirectory(
				ctx,
				workspaceConfig,
				moduleConfig,
				externalDirOrFilePaths,
				externalExcludeDirOrFilePaths,
				subDirRelPaths,
				subDirRelExcludePaths,
				externalDirOrFilePathsAllowNotExist,
			)
			if err != nil {
				return nil, err
			}
		} else {
			// The subDirPath isn't a workspace directory, so we can determine the bucketRelPaths
			// from the sourceRef on its own.
			buildOptions = []bufmodulebuild.BuildOption{
				// We can't determine the module's commit from the local file system.
				// This also may be nil.
				//
				// This is particularly useful for the GoPackage modifier used in
				// managed mode, which supports module-specific overrides.
				bufmodulebuild.WithModuleIdentity(moduleConfig.ModuleIdentity),
			}
			bucketRelPaths := make([]string, len(externalDirOrFilePaths))
			for i, externalDirOrFilePath := range externalDirOrFilePaths {
				bucketRelPath, err := sourceRef.PathForExternalPath(externalDirOrFilePath)
				if err != nil {
					return nil, err
				}
				bucketRelPaths[i] = bucketRelPath
			}
			if externalDirOrFilePathsAllowNotExist {
				buildOptions = append(buildOptions, bufmodulebuild.WithPathsAllowNotExist(bucketRelPaths))
			} else {
				buildOptions = append(buildOptions, bufmodulebuild.WithPaths(bucketRelPaths))
			}
		}
	}
	buildOptions = append(buildOptions, bufmodulebuild.WithWorkspaceDirectory(subDirPath))
	if len(externalExcludeDirOrFilePaths) > 0 {
		bucketRelPaths := make([]string, len(externalExcludeDirOrFilePaths))
		for i, excludeDirOrFilePath := range externalExcludeDirOrFilePaths {
			bucketRelPath, err := sourceRef.PathForExternalPath(excludeDirOrFilePath)
			if err != nil {
				return nil, err
			}
			bucketRelPaths[i] = bucketRelPath
		}
		buildOptions = append(buildOptions, bufmodulebuild.WithExcludePaths(bucketRelPaths))
	}
	module, err := bufmodulebuild.NewModuleBucketBuilder().BuildForBucket(
		ctx,
		mappedReadBucket,
		moduleConfig.Build,
		buildOptions...,
	)
	if err != nil {
		return nil, err
	}
	if missingReferences := detectMissingDependencies(
		moduleConfig.Build.DependencyModuleReferences,
		module.DependencyModulePins(),
		workspace,
	); len(missingReferences) > 0 {
		var builder strings.Builder
		_, _ = builder.WriteString(`Specified deps are not covered in your buf.lock, run "buf mod update":`)
		for _, moduleReference := range missingReferences {
			_, _ = builder.WriteString("\n\t- " + moduleReference.IdentityString())
		}
		m.logger.Warn(builder.String())
	}
	return newModuleConfig(module, moduleConfig), nil
}

func workspaceDirectoryEqualsOrContainsSubDirPath(workspaceConfig *bufwork.Config, subDirPath string) bool {
	if workspaceConfig == nil {
		return false
	}
	for _, directory := range workspaceConfig.Directories {
		if normalpath.EqualsOrContainsPath(directory, subDirPath, normalpath.Relative) {
			return true
		}
	}
	return false
}

func detectMissingDependencies(
	references []bufmoduleref.ModuleReference,
	pins []bufmoduleref.ModulePin,
	workspace bufmodule.Workspace,
) []bufmoduleref.ModuleReference {
	pinSet := make(map[string]struct{})
	for _, pin := range pins {
		pinSet[pin.IdentityString()] = struct{}{}
	}

	var missingReferences []bufmoduleref.ModuleReference
	for _, reference := range references {
		if _, ok := pinSet[reference.IdentityString()]; !ok {
			if workspace != nil {
				if _, ok := workspace.GetModule(reference); !ok {
					missingReferences = append(missingReferences, reference)
				}
			} else {
				missingReferences = append(missingReferences, reference)
			}
		}
	}
	return missingReferences
}
