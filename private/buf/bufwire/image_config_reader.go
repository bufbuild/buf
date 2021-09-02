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

package bufwire

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type imageConfigReader struct {
	logger               *zap.Logger
	storageosProvider    storageos.Provider
	fetchReader          buffetch.Reader
	configProvider       bufconfig.Provider
	moduleBucketBuilder  bufmodulebuild.ModuleBucketBuilder
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	imageBuilder         bufimagebuild.Builder
	moduleConfigReader   *moduleConfigReader
	imageReader          *imageReader
}

func newImageConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	configProvider bufconfig.Provider,
	workspaceConfigProvider bufwork.Provider,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *imageConfigReader {
	return &imageConfigReader{
		logger:               logger.Named("bufwire"),
		storageosProvider:    storageosProvider,
		fetchReader:          fetchReader,
		configProvider:       configProvider,
		moduleBucketBuilder:  moduleBucketBuilder,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
		moduleConfigReader: newModuleConfigReader(
			logger,
			storageosProvider,
			fetchReader,
			configProvider,
			workspaceConfigProvider,
			moduleBucketBuilder,
		),
		imageReader: newImageReader(
			logger,
			fetchReader,
		),
	}
}

func (i *imageConfigReader) GetImageConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) ([]ImageConfig, []bufanalysis.FileAnnotation, error) {
	switch t := ref.(type) {
	case buffetch.ImageRef:
		env, err := i.getImageImageConfig(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
		return []ImageConfig{env}, nil, err
	case buffetch.SourceRef:
		return i.getSourceOrModuleImageConfigs(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	case buffetch.ModuleRef:
		return i.getSourceOrModuleImageConfigs(
			ctx,
			container,
			t,
			configOverride,
			externalDirOrFilePaths,
			externalDirOrFilePathsAllowNotExist,
			excludeSourceCodeInfo,
		)
	default:
		return nil, nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

func (i *imageConfigReader) getSourceOrModuleImageConfigs(
	ctx context.Context,
	container app.EnvStdinContainer,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) ([]ImageConfig, []bufanalysis.FileAnnotation, error) {
	moduleConfigs, err := i.moduleConfigReader.GetModuleConfigs(
		ctx,
		container,
		sourceOrModuleRef,
		configOverride,
		externalDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, nil, err
	}
	// We need to collect all the target paths before we can construct the ModuleFileSet.
	// Paths will vary depending on the module's build.roots configuration, so we perform
	// this step upfront.
	//
	// Target paths belong to one of the following categories:
	//
	//  1. An import path, not actually on the local filesystem (e.g. an import like `buf build petapis --path acme/payment/v2/payment.proto`)
	//  2. A path relative to the sourceOrModuleRef (e.g. `buf build petapis --path petapis/acme/pet/v1/pet.proto`)
	//  3. A path contained in a workspace directory (e.g. `buf build --path petapis/acme/pet/v1/pet.proto` - this file should be interpreted as `acme/pet/v1/pet.proto` in the ModuleFileSet).
	//  4. A path contained in a build root (e.g. `buf build --path root/foo.proto` - this file should be interpreted as `foo.proto` in the ModuleFileSet if the buf.yaml has build.roots set to ["root"]).
	//  5. (2), (3), and (4) combined (i.e. a path contained in a workspace directory that defines multiple build.roots).
	//
	// In short, the user's intent is ambiguous, so we must provide multiple options to the ModuleFileSet. For each path,
	// we include the possible cases in a single set, and the ModuleFileSet will consider the externalDirOrFilePath
	// satisfied if at least one of its associated paths is matched.
	//
	// Note that only two files will ever be possible for any given externalDirOrFilePath:
	// the file provided as-is (1), or any combination of (2), (3), and (4).
	var workspaceConfig *bufwork.Config
	if len(moduleConfigs) > 0 {
		// All of the ModuleConfigs returned by the ModuleConfigReader will have
		// the same *bufwork.Config, so we can arbitrarily select the first one.
		workspaceConfig = moduleConfigs[0].WorkspaceConfig()
	}
	allRootsToExcludes := make([]map[string][]string, 0, len(moduleConfigs))
	for _, moduleConfig := range moduleConfigs {
		// TODO(alex): This isn't actually right as-is; see comment below.
		allRootsToExcludes = append(allRootsToExcludes, moduleConfig.Config().Build.RootToExcludes)
	}
	targetPaths := make([][]string, len(externalDirOrFilePaths))
	for i, externalDirOrFilePath := range externalDirOrFilePaths {
		targetPath, err := sourceOrModuleRef.PathForExternalPath(externalDirOrFilePath)
		switch {
		case normalpath.IsOutsideContextDirError(err):
			// If the path is outside the context directory, then we provide it as
			// it was specified by the user. This is the case for import paths, like
			// the first case shown above.
			targetPaths[i] = []string{externalDirOrFilePath}
		case err != nil:
			return nil, nil, err
		default:
			// We need to determine if the given path is relative to the
			// workspace directory and/or build.roots.
			buildRootTargetPath := targetPath
			if workspaceConfig != nil {
				for _, directory := range workspaceConfig.Directories {
					if !normalpath.ContainsPath(directory, buildRootTargetPath, normalpath.Relative) {
						continue
					}
					buildRootTargetPath, err = normalpath.Rel(directory, buildRootTargetPath)
					if err != nil {
						// Unreachable according to the check above.
						return nil, nil, fmt.Errorf(
							`a relative path could not be resolved between "%s" and workspace directory "%s"`,
							normalpath.Unnormalize(externalDirOrFilePaths[i]),
							directory,
						)
					}
				}
			}
			// TODO(alex): This won't work as-is, but it's close to what we need. We actually need to know
			// whether or not the sourceOrModuleRef represents the given root configuration in question.
			//
			// Counterexample:
			//
			//  $ buf build petapis --path petapis/foo/bar.proto
			//
			//  # buf.work.yaml
			//  version: v1
			//  directories:
			//    - petapis
			//    - paymentapis
			//
			// If the paymentapis module has a 'foo' root with a single 'foo/bar.proto' file,
			// then we would reformat the path as 'bar.proto' so that the ModuleFileSet
			// could match it.
			//
			// However, the user actually meant to match the foo/bar.proto file in the petapis
			// module.
			//
			// We need to add more metadata to the ModuleConfig type so that we could
			// determine the ModuleConfig reprsented by the given sourceOrModuleRef. That
			// way, we only apply the roots for the correct ModuleConfig.
			//
			// We could add a 'WorkspaceDirectory() string' method and adopt the same
			// logic used in bufwork.BuildOptionsForWorkspaceDirectory.
			for _, rootToExcludes := range allRootsToExcludes {
				for root := range rootToExcludes {
					// We don't actually care about the excludes in this case; we
					// just need the root (if it exists).
					if !normalpath.ContainsPath(root, buildRootTargetPath, normalpath.Relative) {
						continue
					}
					buildRootTargetPath, err = normalpath.Rel(root, buildRootTargetPath)
					if err != nil {
						// Unreachable according to the check above.
						return nil, nil, fmt.Errorf(
							`a relative path could not be resolved between "%s" and root "%s"`,
							normalpath.Unnormalize(externalDirOrFilePaths[i]),
							root,
						)
					}
				}
			}
			if buildRootTargetPath != targetPath {
				// If the target path was in a workspace directory and/or
				// a single build.roots, then we want to include the mapped
				// path.
				targetPaths[i] = []string{externalDirOrFilePath, buildRootTargetPath}
				continue
			}
			targetPaths[i] = []string{externalDirOrFilePath, targetPath}
		}
	}
	imageConfigs := make([]ImageConfig, 0, len(moduleConfigs))
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, moduleConfig := range moduleConfigs {
		var buildModuleFileSetOptions []bufmodulebuild.BuildModuleFileSetOption
		if len(externalDirOrFilePaths) > 0 {
			if externalDirOrFilePathsAllowNotExist {
				buildModuleFileSetOptions = append(buildModuleFileSetOptions, bufmodulebuild.WithTargetPathsAllowNotExist(targetPaths))
			} else {
				buildModuleFileSetOptions = append(buildModuleFileSetOptions, bufmodulebuild.WithTargetPaths(targetPaths))
			}
		}
		moduleFileSet, err := i.moduleFileSetBuilder.Build(
			ctx,
			moduleConfig.Module(),
			append(buildModuleFileSetOptions, bufmodulebuild.WithWorkspace(moduleConfig.Workspace()))...,
		)
		if err != nil {
			return nil, nil, err
		}
		targetFileInfos, err := moduleFileSet.TargetFileInfos(ctx)
		if err != nil {
			return nil, nil, err
		}
		if len(targetFileInfos) == 0 {
			// This ModuleFileSet doesn't have any targets, so we shouldn't build
			// an image for it.
			continue
		}
		imageConfig, fileAnnotations, err := i.buildModule(
			ctx,
			moduleConfig.Config(),
			moduleFileSet,
			excludeSourceCodeInfo,
		)
		if err != nil {
			return nil, nil, err
		}
		if imageConfig != nil {
			imageConfigs = append(imageConfigs, imageConfig)
		}
		allFileAnnotations = append(allFileAnnotations, fileAnnotations...)
	}
	if len(allFileAnnotations) > 0 {
		// Deduplicate and sort the file annotations again now that we've
		// consolidated them across multiple images.
		return nil, bufanalysis.DeduplicateAndSortFileAnnotations(allFileAnnotations), nil
	}
	if len(imageConfigs) == 0 {
		return nil, nil, errors.New("no .proto target files found")
	}
	return imageConfigs, nil, nil
}

func (i *imageConfigReader) getImageImageConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ ImageConfig, retErr error) {
	image, err := i.imageReader.GetImage(
		ctx,
		container,
		imageRef,
		externalDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
		excludeSourceCodeInfo,
	)
	if err != nil {
		return nil, err
	}
	readWriteBucket, err := i.storageosProvider.NewReadWriteBucket(
		".",
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	if err != nil {
		return nil, err
	}
	config, err := bufconfig.ReadConfig(
		ctx,
		i.configProvider,
		readWriteBucket,
		bufconfig.ReadConfigWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	return newImageConfig(image, config), nil
}

func (i *imageConfigReader) buildModule(
	ctx context.Context,
	config *bufconfig.Config,
	moduleFileSet bufmodule.ModuleFileSet,
	excludeSourceCodeInfo bool,
) (ImageConfig, []bufanalysis.FileAnnotation, error) {
	ctx, span := trace.StartSpan(ctx, "build_module")
	defer span.End()
	var options []bufimagebuild.BuildOption
	if excludeSourceCodeInfo {
		options = append(options, bufimagebuild.WithExcludeSourceCodeInfo())
	}
	image, fileAnnotations, err := i.imageBuilder.Build(
		ctx,
		moduleFileSet,
		options...,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		return nil, fileAnnotations, nil
	}
	return newImageConfig(image, config), nil, nil
}
