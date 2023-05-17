// Copyright 2020-2023 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type imageConfigReader struct {
	logger               *zap.Logger
	storageosProvider    storageos.Provider
	fetchReader          buffetch.Reader
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
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *imageConfigReader {
	return &imageConfigReader{
		logger:               logger.Named("bufwire"),
		storageosProvider:    storageosProvider,
		fetchReader:          fetchReader,
		moduleBucketBuilder:  moduleBucketBuilder,
		moduleFileSetBuilder: moduleFileSetBuilder,
		imageBuilder:         imageBuilder,
		moduleConfigReader: newModuleConfigReader(
			logger,
			storageosProvider,
			fetchReader,
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
	externalExcludeDirOrFilePaths []string,
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
			externalExcludeDirOrFilePaths,
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
			externalExcludeDirOrFilePaths,
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
			externalExcludeDirOrFilePaths,
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
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) ([]ImageConfig, []bufanalysis.FileAnnotation, error) {
	moduleConfigs, err := i.moduleConfigReader.GetModuleConfigs(
		ctx,
		container,
		sourceOrModuleRef,
		configOverride,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
		externalDirOrFilePathsAllowNotExist,
	)
	if err != nil {
		return nil, nil, err
	}
	imageConfigs := make([]ImageConfig, 0, len(moduleConfigs))
	var allFileAnnotations []bufanalysis.FileAnnotation
	for _, moduleConfig := range moduleConfigs {
		moduleFileSet, err := i.moduleFileSetBuilder.Build(
			ctx,
			moduleConfig.Module(),
			bufmodulebuild.WithWorkspace(moduleConfig.Workspace()),
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
			if importsErr := i.validateSourceImports(
				moduleConfig.Module().DirectDependencies(),
				imageConfig.Image(),
			); importsErr != nil {
				return nil, nil, fmt.Errorf("source imports: %w", importsErr)
			}
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
	if protoFileRef, ok := sourceOrModuleRef.(buffetch.ProtoFileRef); ok {
		imageConfigs, err = filterImageConfigs(imageConfigs, protoFileRef)
		if err != nil {
			return nil, nil, err
		}
	}
	return imageConfigs, nil, nil
}

// validateSourceImports takes the direct dependencies from a module, and a built image for that
// module, and validates that all the source image files have valid imports clauses that come from a
// direct dependency and not a transitive (or unknown) one. Error message is safe to pass to the
// user.
func (i *imageConfigReader) validateSourceImports(
	directModuleDeps []bufmoduleref.ModuleReference,
	builtImage bufimage.Image,
) error {
	// prepare direct deps as a map
	directDepsIdentities := make(map[string]struct{}, len(directModuleDeps))
	for _, directDep := range directModuleDeps {
		directDepsIdentities[directDep.IdentityString()] = struct{}{}
	}
	i.logger.Debug(
		"module",
		zap.Any("module_direct_dependencies", directDepsIdentities),
	)

	// categorize image files into direct vs transitive dependencies
	allImgFiles := make(map[string]map[string][]string)    // for logging purposes only, modIdentity:filepath:imports
	directDepsFilesToModule := make(map[string]string)     // filepath:modIdentity
	transitiveDepsFilesToModule := make(map[string]string) // filepath:modIdentity
	for _, file := range builtImage.Files() {
		modIdentity := "source"
		if file.ModuleIdentity() != nil {
			modIdentity = file.ModuleIdentity().IdentityString()
		}
		if _, ok := allImgFiles[modIdentity]; !ok {
			allImgFiles[modIdentity] = make(map[string][]string)
		}
		allImgFiles[modIdentity][file.Path()] = file.FileDescriptor().GetDependency()
		if file.IsImport() {
			if _, ok := directDepsIdentities[modIdentity]; ok {
				directDepsFilesToModule[file.Path()] = modIdentity
			} else {
				transitiveDepsFilesToModule[file.Path()] = modIdentity
			}
		}
	}
	i.logger.Debug(
		"image_files",
		zap.Any("all_with_imports", allImgFiles),
		zap.Any("from_direct_dependencies", directDepsFilesToModule),
		zap.Any("from_transitive_dependencies", transitiveDepsFilesToModule),
	)

	// validate import statements of source files against dependencies categorization above
	var importsErr error
	for _, file := range builtImage.Files() {
		if !file.IsImport() {
			for _, importFilePath := range file.FileDescriptor().GetDependency() {
				if _, ok := directDepsFilesToModule[importFilePath]; ok {
					continue // import comes from direct dep
				}
				errorMsg := fmt.Sprintf(
					"source proto file %q imports %q, not found in your direct dependencies",
					file.Path(), importFilePath,
				)
				if transitiveDepModule, ok := transitiveDepsFilesToModule[importFilePath]; ok {
					errorMsg += fmt.Sprintf(
						", but found in transitive dependency %q, please declare that one as explicit dependency in your buf.yaml file",
						transitiveDepModule,
					)
				} else {
					errorMsg += ", or any of your transitive dependencies, please check that your imports are declared as explicit dependencies in your buf.yaml file"
				}
				importsErr = multierr.Append(importsErr, errors.New(errorMsg))
			}
		}
	}
	return importsErr
}

func (i *imageConfigReader) getImageImageConfig(
	ctx context.Context,
	container app.EnvStdinContainer,
	imageRef buffetch.ImageRef,
	configOverride string,
	externalDirOrFilePaths []string,
	externalExcludeDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
	excludeSourceCodeInfo bool,
) (_ ImageConfig, retErr error) {
	image, err := i.imageReader.GetImage(
		ctx,
		container,
		imageRef,
		externalDirOrFilePaths,
		externalExcludeDirOrFilePaths,
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
	config, err := bufconfig.ReadConfigOS(
		ctx,
		readWriteBucket,
		bufconfig.ReadConfigOSWithOverride(configOverride),
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
	ctx, span := otel.GetTracerProvider().Tracer("bufbuild/buf").Start(ctx, "build_module")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		return nil, fileAnnotations, nil
	}
	return newImageConfig(image, config), nil, nil
}

// filterImageConfigs takes in image configs and filters them based on the proto file ref.
// First, we get the package, path, and config for the file ref. And then we merge the images
// across the ImageConfigs, then filter them based on the paths for the package.
//
// The image merge is needed because if the `include_package_files=true` option is set, we
// need to gather all the files for the package, including files spread out across workspace
// directories, which would result in multiple image configs.
func filterImageConfigs(imageConfigs []ImageConfig, protoFileRef buffetch.ProtoFileRef) ([]ImageConfig, error) {
	var pkg string
	var path string
	var config *bufconfig.Config
	var images []bufimage.Image
	for _, imageConfig := range imageConfigs {
		for _, imageFile := range imageConfig.Image().Files() {
			// TODO: Ideally, we have the path returned from PathForExternalPath, however for a protoFileRef,
			// PathForExternalPath returns only ".", <nil> when matched on the exact path of the proto file
			// provided as the ref. This is expected since `PathForExternalPath` is meant to return the relative
			// path based on the reference, which in this case will always be a specific file.
			if _, err := protoFileRef.PathForExternalPath(imageFile.ExternalPath()); err == nil {
				pkg = imageFile.Proto().GetPackage()
				path = imageFile.Path()
				config = imageConfig.Config()
				break
			}
		}
		images = append(images, imageConfig.Image())
	}
	if path == "" {
		return nil, errors.New("did not find a matching image file for the ProtoFileRef")
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return nil, err
	}
	var paths []string
	if protoFileRef.IncludePackageFiles() {
		for _, imageFile := range image.Files() {
			if imageFile.Proto().GetPackage() == pkg {
				paths = append(paths, imageFile.Path())
			}
		}
	} else {
		paths = []string{path}
	}
	prunedImage, err := bufimage.ImageWithOnlyPaths(image, paths, nil)
	if err != nil {
		return nil, err
	}
	return []ImageConfig{newImageConfig(prunedImage, config)}, nil
}
