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

// Package bufwire wires everything together.
//
// TODO: This package should be split up into individual functionality.
package bufwire

import (
	"context"

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
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// ImageConfig is an image and configuration.
type ImageConfig interface {
	Image() bufimage.Image
	Config() *bufconfig.Config
}

// ImageConfigReader is an ImageConfig reader.
type ImageConfigReader interface {
	// GetImageConfigs gets the ImageConfig for the fetch value.
	//
	// If externalDirOrFilePaths is empty, this builds all files under Buf control.
	GetImageConfigs(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
		externalDirOrFilePaths []string,
		externalExcludeDirOrFilePaths []string,
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
	imageBuilder bufimagebuild.Builder,
) ImageConfigReader {
	return newImageConfigReader(
		logger,
		storageosProvider,
		fetchReader,
		moduleBucketBuilder,
		imageBuilder,
	)
}

// ModuleConfig is a Module and configuration.
type ModuleConfig interface {
	Module() bufmodule.Module
	Config() *bufconfig.Config
}

// ModuleConfigSet is a set of ModuleConfigs with a potentially associated Workspace.
type ModuleConfigSet interface {
	ModuleConfigs() []ModuleConfig
	// Optional. May be nil.
	Workspace() bufmodule.Workspace
}

// ModuleConfigReader is a ModuleConfig reader.
type ModuleConfigReader interface {
	// GetModuleConfigSet gets the ModuleConfigSet for the fetch value.
	//
	// If externalDirOrFilePaths is empty, this builds all files under Buf control.
	//
	// Note that as opposed to ModuleReader, this will return Modules for either
	// a source or module reference, not just a module reference.
	GetModuleConfigSet(
		ctx context.Context,
		container app.EnvStdinContainer,
		sourceOrModuleRef buffetch.SourceOrModuleRef,
		configOverride string,
		externalDirOrFilePaths []string,
		externalExcludeDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
	) (ModuleConfigSet, error)
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
	//
	// If includeImports is set, the ref is built, which can result in FileAnnotations.
	// There is no defined returned sorting order. If you need the FileInfos to
	// be sorted, do so with bufmoduleref.SortFileInfos or bufmoduleref.SortFileInfosByExternalPath.
	ListFiles(
		ctx context.Context,
		container app.EnvStdinContainer,
		ref buffetch.Ref,
		configOverride string,
		includeImports bool,
	) ([]bufmoduleref.FileInfo, []bufanalysis.FileAnnotation, error)
}

// NewFileLister returns a new FileLister.
func NewFileLister(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	fetchReader buffetch.Reader,
	moduleBucketBuilder bufmodulebuild.ModuleBucketBuilder,
	imageBuilder bufimagebuild.Builder,
) FileLister {
	return newFileLister(
		logger,
		storageosProvider,
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
		messageRef buffetch.MessageRef,
		externalDirOrFilePaths []string,
		externalExcludeDirOrFilePaths []string,
		externalDirOrFilePathsAllowNotExist bool,
		excludeSourceCodeInfo bool,
	) (bufimage.Image, error)
}

// NewImageReader returns a new ImageReader.
func NewImageReader(
	logger *zap.Logger,
	fetchReader buffetch.MessageReader,
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
		messageRef buffetch.MessageRef,
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

// ProtoEncodingReader is a reader that reads a protobuf message in different encoding.
type ProtoEncodingReader interface {
	// GetMessage reads the message by the messageRef.
	GetMessage(
		ctx context.Context,
		container app.EnvStdinContainer,
		image bufimage.Image,
		typeName string,
		messageRef buffetch.MessageRef,
	) (proto.Message, error)
}

// NewProtoEncodingReader returns a new ProtoEncodingReader.
func NewProtoEncodingReader(
	logger *zap.Logger,
	fetchReader buffetch.MessageReader,
) ProtoEncodingReader {
	return newProtoEncodingReader(
		logger,
		fetchReader,
	)
}

// ProtoEncodingWriter is a writer that writes a protobuf message in different encoding.
type ProtoEncodingWriter interface {
	// PutMessage writes the message to the path, which can be
	// a path in file system, or stdout represented by "-".
	//
	// Currently, this support bin and JSON format.
	PutMessage(
		ctx context.Context,
		container app.EnvStdoutContainer,
		image bufimage.Image,
		message proto.Message,
		messageRef buffetch.MessageRef,
	) error
}

// NewProtoEncodingWriter returns a new ProtoEncodingWriter.
func NewProtoEncodingWriter(
	logger *zap.Logger,
	fetchWriter buffetch.Writer,
) ProtoEncodingWriter {
	return newProtoEncodingWriter(
		logger,
		fetchWriter,
	)
}
