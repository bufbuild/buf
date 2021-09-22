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
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/buf/bufwork"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type fileLister struct {
	logger              *zap.Logger
	fetchReader         buffetch.Reader
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder
	imageBuilder        bufimagebuild.Builder
	// imageReaders require ImageRefs, we only use this in the withoutImports flow
	// the imageConfigReader is used when we need to build an image
	imageReader       *imageReader
	imageConfigReader *imageConfigReader
}

func newFileLister(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) *fileLister {
	return &fileLister{
		logger:              logger.Named("bufwire"),
		fetchReader:         fetchReader,
		moduleBucketBuilder: moduleBucketBuilder,
		imageBuilder:        imageBuilder,
		imageReader: newImageReader(
			logger,
			fetchReader,
		),
		imageConfigReader: newImageConfigReader(
			logger,
			storageosProvider,
			fetchReader,
			moduleBucketBuilder,
			moduleFileSetBuilder,
			imageBuilder,
		),
	}
}

func (e *fileLister) ListFiles(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
	includeImports bool,
) ([]bufmoduleref.FileInfo, []bufanalysis.FileAnnotation, error) {
	if includeImports {
		// To get imports, we need to build an image so we keep this flow separate and
		// re-use the logic in imageConfigReader.
		return e.listFilesWithImports(
			ctx,
			container,
			ref,
			configOverride,
		)
	}
	// We don't need to build in the withoutImports flow, so we just completely separate the logic
	// to make sure the common case is as quick as it can be.
	fileInfos, err := e.listFilesWithoutImports(
		ctx,
		container,
		ref,
		configOverride,
	)
	return fileInfos, nil, err
}

func (e *fileLister) listFilesWithImports(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
) ([]bufmoduleref.FileInfo, []bufanalysis.FileAnnotation, error) {
	imageConfigs, fileAnnotations, err := e.imageConfigReader.GetImageConfigs(
		ctx,
		container,
		ref,
		configOverride,
		nil,
		false,
		true,
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		return nil, fileAnnotations, nil
	}
	images := make([]bufimage.Image, len(imageConfigs))
	for i, imageConfig := range imageConfigs {
		images[i] = imageConfig.Image()
	}
	image, err := bufimage.MergeImages(images...)
	if err != nil {
		return nil, nil, err
	}
	imageFiles := image.Files()
	fileInfos := make([]bufmoduleref.FileInfo, len(imageFiles))
	for i, imageFile := range imageFiles {
		fileInfos[i] = imageFile
	}
	bufmoduleref.SortFileInfos(fileInfos)
	return fileInfos, nil, nil
}

func (e *fileLister) listFilesWithoutImports(
	ctx context.Context,
	container app.EnvStdinContainer,
	ref buffetch.Ref,
	configOverride string,
) (_ []bufmoduleref.FileInfo, retErr error) {
	switch t := ref.(type) {
	case buffetch.ImageRef:
		// if we have an image, list the files in the image
		image, err := e.imageReader.GetImage(
			ctx,
			container,
			t,
			nil,
			false,
			true,
		)
		if err != nil {
			return nil, err
		}
		files := image.Files()
		fileInfos := make([]bufmoduleref.FileInfo, len(files))
		for i, file := range files {
			fileInfos[i] = file
		}
		return fileInfos, nil
	case buffetch.SourceRef:
		readBucketCloser, err := e.fetchReader.GetSourceBucket(ctx, container, t)
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
		if subDirPath := readBucketCloser.SubDirPath(); existingConfigFilePath == "" || subDirPath != "." {
			return e.sourceFileInfosForDirectory(ctx, readBucketCloser, subDirPath, configOverride)
		}
		workspaceConfig, err := bufwork.GetConfigForBucket(ctx, readBucketCloser, readBucketCloser.RelativeRootPath())
		if err != nil {
			return nil, err
		}
		var allSourceFileInfos []bufmoduleref.FileInfo
		for _, directory := range workspaceConfig.Directories {
			sourceFileInfos, err := e.sourceFileInfosForDirectory(ctx, readBucketCloser, directory, configOverride)
			if err != nil {
				return nil, err
			}
			allSourceFileInfos = append(allSourceFileInfos, sourceFileInfos...)
		}
		return allSourceFileInfos, nil
	case buffetch.ModuleRef:
		module, err := e.fetchReader.GetModule(ctx, container, t)
		if err != nil {
			return nil, err
		}
		return module.SourceFileInfos(ctx)
	default:
		return nil, fmt.Errorf("invalid ref: %T", ref)
	}
}

// sourceFileInfosForDirectory returns the source file infos
// for the module defined in the given diretory of the read bucket.
func (e *fileLister) sourceFileInfosForDirectory(
	ctx context.Context,
	readBucket storage.ReadBucket,
	directory string,
	configOverride string,
) ([]bufmoduleref.FileInfo, error) {
	mappedReadBucket := storage.MapReadBucket(readBucket, storage.MapOnPrefix(directory))
	config, err := bufconfig.ReadConfigOS(
		ctx,
		mappedReadBucket,
		bufconfig.ReadConfigOSWithOverride(configOverride),
	)
	if err != nil {
		return nil, err
	}
	module, err := e.moduleBucketBuilder.BuildForBucket(
		ctx,
		mappedReadBucket,
		config.Build,
	)
	if err != nil {
		return nil, err
	}
	return module.SourceFileInfos(ctx)
}
