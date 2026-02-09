// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufimage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	descriptorv1 "buf.build/gen/go/bufbuild/protodescriptor/protocolbuffers/go/buf/descriptor/v1"
	"buf.build/go/standard/xlog/xslog"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile/experimental/fdp"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/experimental/source/length"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func buildImage(
	ctx context.Context,
	logger *slog.Logger,
	moduleReadBucket bufmodule.ModuleReadBucket,
	excludeSourceCodeInfo bool,
	noParallelism bool,
) (Image, error) {
	defer xslog.DebugProfile(logger)()

	if !moduleReadBucket.ShouldBeSelfContained() {
		return nil, syserror.New(
			"passed a ModuleReadBucket to BuildImage that was not expected to be self-contained",
		)
	}

	targetFileInfos, err := bufmodule.GetTargetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return nil, err
	}

	if len(targetFileInfos) == 0 {
		// If we had no no target files within the module after path filtering, this is an error.
		// We could have a better user error than this. This gets back to the lack of allowNotExist.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}

	image, err := compileImage(
		ctx,
		bufmodule.ModuleReadBucketWithOnlyProtoFiles(moduleReadBucket),
		bufmodule.FileInfoPaths(targetFileInfos),
		excludeSourceCodeInfo,
		noParallelism,
	)
	if err != nil {
		return nil, err
	}

	return image, nil
}

// compileImage compiles the [Image] for the given [bufmodule.ModuleReadBucket].
func compileImage(
	ctx context.Context,
	moduleReadBucket bufmodule.ModuleReadBucket,
	paths []string,
	excludeSourceCodeInfo bool,
	noParallelism bool,
) (Image, error) {
	session := new(ir.Session)
	moduleFileResolver := newModuleFileResolver(ctx, moduleReadBucket)

	parallelism := thread.Parallelism()
	if noParallelism {
		parallelism = 1
	}

	exec := incremental.New(
		incremental.WithParallelism(int64(parallelism)),
	)
	results, diagnostics, err := incremental.Run(
		ctx,
		exec,
		queries.Link{
			Opener:    moduleFileResolver,
			Session:   session,
			Workspace: source.NewWorkspace(paths...),
		},
	)

	var fileAnnotations []bufanalysis.FileAnnotation
	for _, diagnostic := range diagnostics.Diagnostics {
		primary := diagnostic.Primary()
		if primary.IsZero() || diagnostic.Level() > report.Error {
			continue
		}
		start := primary.Location(primary.Start, length.Bytes)
		end := primary.Location(primary.End, length.Bytes)
		// We resolve the path and external path using moduleFileResolver, since the span
		// uses the path set by moduleFileResolver, which is the moduleFile.LocalPath().
		path := moduleFileResolver.PathForLocalPath(primary.Path())
		fileAnnotations = append(
			fileAnnotations,
			bufanalysis.NewFileAnnotation(
				&fileInfo{
					path:         path,
					externalPath: moduleFileResolver.ExternalPath(path),
				},
				start.Line,
				start.Column,
				end.Line,
				end.Column,
				"COMPILE",
				diagnostic.Message(),
				"", // pluginName
				"", // policyName
			),
		)
	}
	if len(fileAnnotations) > 0 {
		return nil, bufanalysis.NewFileAnnotationSet(diagnostics, fileAnnotations...)
	}

	// Validate that there is a single result for all files
	if len(results) != 1 {
		return nil, fmt.Errorf("expected a single result from query, instead got: %d", len(results))
	}

	if results[0].Fatal != nil {
		return nil, results[0].Fatal
	}
	irFiles := results[0].Value

	bytes, err := fdp.DescriptorSetBytes(
		irFiles,
		// When compiling the [descriptorpb.FileDescriptorSet], we always include the source
		// code info to get [descriptorv1.E_BufSourceCodeInfoExtension] information. The source
		// code info may still be excluded from the final [Image] based on the options passed in.
		fdp.IncludeSourceCodeInfo(true),
		// This is needed for lint and breaking change detection annotations.
		fdp.GenerateExtraOptionLocations(true),
	)
	if err != nil {
		return nil, err
	}

	// First unmarshal to get the descriptors
	fds := new(descriptorpb.FileDescriptorSet)
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(bytes, fds); err != nil {
		return nil, err
	}

	// Create a resolver from the descriptors so extensions can be recognized.
	// Specifically, we need to ensure that we are able to resolve the Buf-specific descriptor
	// extensions for propagating compiler errors. In the future, we should have better
	// integration with [report.Report] to handle warnings.
	resolverFiles := []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(descriptorv1.File_buf_descriptor_v1_descriptor_proto),
	}
	resolverFiles = append(resolverFiles, fds.File...)
	resolver := protoencoding.NewLazyResolver(resolverFiles...)

	// Reparse extensions with the resolver in all FileDescriptorProtos to convert unknown
	// fields into recognized extensions
	for _, fileDescriptor := range fds.File {
		if err := protoencoding.ReparseExtensions(resolver, fileDescriptor.ProtoReflect()); err != nil {
			return nil, err
		}
	}

	return fileDescriptorSetToImage(resolver, moduleFileResolver, paths, fds, excludeSourceCodeInfo)
}

// fileDescriptorSetToImage is a helper function that converts a [descriptorpb.FileDescriptorSet]
// to an [Image], preserving the order of the paths based on the module paths.
//
// Note that this iterates through the given paths and constructs the the [ImageFile]s
// based on that rather than using the file descriptor set compiled through the compiler.
// This is because there is a difference in the topological sorting algo used in the
// compiler vs. expected protoc ordering, and so for conformance reasons, we reconstruct
// the ordering of the [ImageFile]s.
func fileDescriptorSetToImage(
	resolver protoencoding.Resolver,
	moduleFileResolver *moduleFileResolver,
	paths []string,
	fds *descriptorpb.FileDescriptorSet,
	excludeSourceCodeInfo bool,
) (Image, error) {
	pathToDescriptor := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, fileDescriptor := range fds.File {
		pathToDescriptor[fileDescriptor.GetName()] = fileDescriptor
	}

	var imageFiles []ImageFile
	var err error
	seen := make(map[string]struct{})

	for _, path := range paths {
		imageFiles, err = getImageFilesForPath(
			path,
			pathToDescriptor,
			moduleFileResolver,
			excludeSourceCodeInfo,
			seen,
			xslices.ToStructMap(paths),
			imageFiles,
		)

		if err != nil {
			return nil, err
		}
	}

	return newImage(imageFiles, false, resolver)
}

func getImageFilesForPath(
	path string,
	pathToDescriptor map[string]*descriptorpb.FileDescriptorProto,
	moduleFileResolver *moduleFileResolver,
	excludeSourceCodeInfo bool,
	seen map[string]struct{},
	nonImportFilenames map[string]struct{},
	imageFiles []ImageFile,
) ([]ImageFile, error) {
	fileDescriptor := pathToDescriptor[path]
	if fileDescriptor == nil {
		return nil, errors.New("nil FileDescriptor")
	}

	if _, ok := seen[path]; ok {
		return imageFiles, nil
	}
	seen[path] = struct{}{}

	var err error
	for _, dependency := range fileDescriptor.Dependency {
		imageFiles, err = getImageFilesForPath(
			dependency,
			pathToDescriptor,
			moduleFileResolver,
			excludeSourceCodeInfo,
			seen,
			nonImportFilenames,
			imageFiles,
		)
		if err != nil {
			return nil, err
		}
	}

	_, isNotImport := nonImportFilenames[path]

	imageFile, err := fileDescriptorProtoToImageFile(
		moduleFileResolver,
		fileDescriptor,
		excludeSourceCodeInfo,
		!isNotImport,
	)
	if err != nil {
		return nil, err
	}
	return append(imageFiles, imageFile), nil
}

// fileDescriptorProtoToImageFile is a helper function that converts a [descriptorpb.FileDescriptorProto]
// to an [ImageFile].
func fileDescriptorProtoToImageFile(
	moduleFileResolver *moduleFileResolver,
	fileDescriptor *descriptorpb.FileDescriptorProto,
	excludeSourceCodeInfo bool,
	isImport bool,
) (ImageFile, error) {
	var (
		isSyntaxUnspecified     bool
		unusedDependencyIndexes []int32
	)

	sourceCodeInfo := fileDescriptor.GetSourceCodeInfo()
	extensionDescriptor := descriptorv1.E_BufSourceCodeInfoExtension.TypeDescriptor()
	if sourceCodeInfo.ProtoReflect().Has(extensionDescriptor) {
		sourceCodeInfoExt := new(descriptorv1.SourceCodeInfoExtension)
		switch sourceCodeInfoExtMessage := sourceCodeInfo.ProtoReflect().Get(extensionDescriptor).Message().Interface().(type) {
		case *dynamicpb.Message:
			bytes, err := protoencoding.NewWireMarshaler().Marshal(sourceCodeInfoExtMessage)
			if err != nil {
				return nil, err
			}
			if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(bytes, sourceCodeInfoExt); err != nil {
				return nil, err
			}
		case *descriptorv1.SourceCodeInfoExtension:
			sourceCodeInfoExt = sourceCodeInfoExtMessage
		}
		isSyntaxUnspecified = sourceCodeInfoExt.GetIsSyntaxUnspecified()
		unusedDependencyIndexes = sourceCodeInfoExt.GetUnusedDependency()
	}

	if excludeSourceCodeInfo {
		fileDescriptor.SourceCodeInfo = nil
	}

	return NewImageFile(
		fileDescriptor,
		moduleFileResolver.FullName(fileDescriptor.GetName()),
		moduleFileResolver.CommitID(fileDescriptor.GetName()),
		moduleFileResolver.ExternalPath(fileDescriptor.GetName()),
		moduleFileResolver.LocalPath(fileDescriptor.GetName()),
		isImport,
		isSyntaxUnspecified,
		unusedDependencyIndexes,
	)
}

type buildImageOptions struct {
	excludeSourceCodeInfo bool
	noParallelism         bool
}

func newBuildImageOptions() *buildImageOptions {
	return &buildImageOptions{}
}

type fileInfo struct {
	path         string
	externalPath string
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	if f.externalPath != "" {
		return f.externalPath
	}
	return f.path
}
