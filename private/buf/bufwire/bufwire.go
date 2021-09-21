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

// Package bufwire wires everything together.
//
// TODO: This package should be split up into individual functionality.
package bufwire

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufconfig"
	"github.com/bufbuild/buf/private/buf/buffetch"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

// ImageConfig is an image and configuration.
type ImageConfig interface {
	Image() bufimage.Image
	Config() *bufconfig.Config
}

// ImageConfigReader is an ImageConfig reader.
type ImageConfigReader interface {
	// GetImageConfig gets the ImageConfig for the fetch value.
	//
	// If externalDirOrFilePaths is empty, this builds all files under Buf control.
	GetImageConfigs(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
		externalDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) ([]ImageConfig, []bufanalysis.FileAnnotation, error)
}

// NewImageConfigReader returns a new ImageConfigReader.
func NewImageConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder,
	imageBuilder bufimagebuild.Builder,
) ImageConfigReader {
	return newImageConfigReader(
		logger,
		storageosProvider,
		fetchReader,
		moduleBucketBuilder,
		moduleFileSetBuilder,
		imageBuilder,
	)
}

// ModuleConfig is an module and configuration.
type ModuleConfig interface {
	Module() bufmodule.Module
	Config() *bufconfig.Config
	Workspace() bufmodule.Workspace
	WorkspaceConfig() *bufwork.Config
}

// BuildModuleFileSetOptionForTargetPaths returns the bufmodulebuild.BuildModuleFileSetOption
// required for the given ModuleConfigs. This is required to support the --path filter.
func BuildModuleFileSetOptionForTargetPaths(
	moduleConfigs []ModuleConfig,
	sourceOrModuleRef buffetch.SourceOrModuleRef,
	externalDirOrFilePaths []string,
	externalDirOrFilePathsAllowNotExist bool,
) (bufmodulebuild.BuildModuleFileSetOption, error) {
	if len(moduleConfigs) == 0 {
		// This should never happen, but it's included for additional safety.
		return nil, errors.New("expected at least one module, but found none")
	}
	// All of the ModuleConfigs are expected to have the same *bufwork.Config,
	// so we can arbitrarily select the first one.
	workspaceConfig := moduleConfigs[0].WorkspaceConfig()
	rootToExcludesForWorkspaceDirectory := make(map[string]map[string][]string, len(moduleConfigs))
	if len(moduleConfigs) > 1 {
		if len(workspaceConfig.Directories) != len(moduleConfigs) {
			// This should be unreachable.
			return nil, fmt.Errorf(
				"received %d modules, but %d directories were listed in the workspace",
				len(moduleConfigs),
				len(workspaceConfig.Directories),
			)
		}
		// We only need to collect the roots for each workspace directory if the
		// user targeted a directory containing a buf.work.yaml.
		for i, moduleConfig := range moduleConfigs {
			// ModuleConfigs are constructed and returned in the same order they're
			// listed as directories in the user's buf.work.yaml.
			workspaceDirectory := workspaceConfig.Directories[i]
			rootToExcludesForWorkspaceDirectory[workspaceDirectory] = moduleConfig.Config().Build.RootToExcludes
		}
	}
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
			return nil, err
		default:
			// We need to determine if the given path is relative to the
			// workspace directory and/or build.roots.
			buildRootTargetPath := targetPath
			var currentWorkspaceDirectory string
			if workspaceConfig != nil {
				for _, directory := range workspaceConfig.Directories {
					if !normalpath.ContainsPath(directory, buildRootTargetPath, normalpath.Relative) {
						continue
					}
					buildRootTargetPath, err = normalpath.Rel(directory, buildRootTargetPath)
					if err != nil {
						// Unreachable according to the check above.
						return nil, fmt.Errorf(
							`a relative path could not be resolved between "%s" and workspace directory "%s"`,
							normalpath.Unnormalize(externalDirOrFilePaths[i]),
							directory,
						)
					}
					currentWorkspaceDirectory = directory
					break
				}
			}
			var rootToExcludes map[string][]string
			if len(moduleConfigs) == 1 {
				// There's only one set of roots we need to check.
				rootToExcludes = moduleConfigs[0].Config().Build.RootToExcludes
			} else if currentWorkspaceDirectory != "" {
				// Use the roots configured for the ModuleConfig that matches
				// the current workspace directory.
				rootToExcludes = rootToExcludesForWorkspaceDirectory[currentWorkspaceDirectory]
			}
			for root := range rootToExcludes {
				// We don't actually care about the excludes in this case; we
				// just need the root (if it exists).
				if !normalpath.ContainsPath(root, buildRootTargetPath, normalpath.Relative) {
					continue
				}
				buildRootTargetPath, err = normalpath.Rel(root, buildRootTargetPath)
				if err != nil {
					// Unreachable according to the check above.
					return nil, fmt.Errorf(
						`a relative path could not be resolved between "%s" and root "%s"`,
						normalpath.Unnormalize(externalDirOrFilePaths[i]),
						root,
					)
				}
				break
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
	if externalDirOrFilePathsAllowNotExist {
		return bufmodulebuild.WithTargetPathsAllowNotExist(targetPaths), nil
	}
	return bufmodulebuild.WithTargetPaths(targetPaths), nil
}

// ModuleConfigReader is a ModuleConfig reader.
type ModuleConfigReader interface {
	// GetModuleConfig gets the ModuleConfig for the fetch value.
	//
	// If externalDirOrFilePaths is empty, this builds all files under Buf control.
	//
	// Note that as opposed to ModuleReader, this will return a Module for either
	// a source or module reference, not just a module reference.
	GetModuleConfigs(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceOrModuleRef buffetch.SourceOrModuleRef,
		configOverride string,
		externalDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
	) ([]ModuleConfig, error)
}

// NewModuleConfigReader returns a new ModuleConfigReader
func NewModuleConfigReader(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
) ModuleConfigReader {
	return newModuleConfigReader(
		logger,
		storageosProvider,
		fetchReader,
		moduleBucketBuilder,
	)
}

// FileLister lists files.
type FileLister interface {
	// ListFiles lists the files.
	ListFiles(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
	) ([]bufmoduleref.FileInfo, error)
}

// NewFileLister returns a new FileLister.
func NewFileLister(
	logger *zap.Logger,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	imageBuilder bufimagebuild.Builder,
) FileLister {
	return newFileLister(
		logger,
		fetchReader,
		moduleBucketBuilder,
		imageBuilder,
	)
}

// ImageReader is an image reader.
type ImageReader interface {
	// GetImage reads the image from the value.
	GetImage(
		ctx context.Context,
		container app.EnvStdinContainer,
		imageRef buffetch.ImageRef,
		externalDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (bufimage.Image, error)
}

// NewImageReader returns a new ImageReader.
func NewImageReader(
	logger *zap.Logger,
	fetchReader buffetch.ImageReader,
) ImageReader {
	return newImageReader(
		logger,
		fetchReader,
	)
}

// ImageWriter is an image writer.
type ImageWriter interface {
	// PutImage writes the image to the value.
	//
	// The file must be an image format.
	// This is a no-np if value is the equivalent of /dev/null.
	PutImage(
		ctx context.Context,
		container app.EnvStdoutContainer,
		imageRef buffetch.ImageRef,
		image bufimage.Image,
		asFileDescriptorSet bool,
		excludeImports bool,
	) error
}

// NewImageWriter returns a new ImageWriter.
func NewImageWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) ImageWriter {
	return newImageWriter(
		logger,
		fetchWriter,
	)
}
