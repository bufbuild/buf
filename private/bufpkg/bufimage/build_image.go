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
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
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
	"github.com/google/uuid"
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
		// If we had no target files within the module after path filtering, this is an error.
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
	if err != nil {
		return nil, err
	}

	var fileAnnotations []bufanalysis.FileAnnotation
	for _, diagnostic := range diagnostics.Diagnostics {
		primary := diagnostic.Primary()
		if primary.IsZero() || diagnostic.Level() > report.Error {
			// We only surface [report.Error] level or more sever diagnostics as build errors.
			// Warnings will still be rendered in the diagnostics report if errors are found,
			// but if there are no errors, then the warnings are not surfaced to the user.
			//
			// In the future, we should handle warnings and other checks in a unified way.
			continue
		}
		start := primary.Location(primary.Start, length.Bytes)
		end := primary.Location(primary.End, length.Bytes)

		// We resolve the path and external path using moduleFileResolver, since the span
		// uses the path set by moduleFileResolver, which is the moduleFile.LocalPath().
		path := moduleFileResolver.PathForLocalPath(primary.Path())
		if path == "" {
			// If there is no path, fallback to using the path from the diagnostic span directly.
			path = primary.Path()
		}
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
		return nil, bufanalysis.NewFileAnnotationSet(fileAnnotations...)
	}

	// Validate that there is a single result for all files
	if len(results) != 1 {
		return nil, fmt.Errorf("expected a single result from query, instead got: %d", len(results))
	}

	if results[0].Fatal != nil {
		return nil, results[0].Fatal
	}
	irFiles := results[0].Value

	// Validate that all files have completed lowering. If not, then symbols have not been
	// properly resolved and cannot generate a valid file descriptor set.
	for _, irFile := range irFiles {
		if !irFile.Lowered() {
			return nil, fmt.Errorf(`The symbols for file %q have not been fully resolved due to an invalid version of descriptor.proto, located at %q. This is likely due to a vendored descriptor.proto.`,
				moduleFileResolver.ExternalPath(irFile.Path()),
				moduleFileResolver.ExternalPath("google/protobuf/descriptor.proto"),
			)
		}
	}

	fds, resolver, err := irFilesToFileDescriptorSet(irFiles)
	if err != nil {
		return nil, err
	}
	return fileDescriptorSetToImage(resolver, moduleFileResolver, paths, fds, excludeSourceCodeInfo)
}

// imageFileMetadataResolver provides path metadata when constructing [ImageFile]s.
// This abstraction allows image building to work both with a full [moduleFileResolver]
// (which tracks module names, commit IDs, and external/local paths) and with a simpler
// identity resolver (for cases like the LSP where path == externalPath).
type imageFileMetadataResolver interface {
	ExternalPath(path string) string
	LocalPath(path string) string
	FullName(path string) bufparse.FullName
	CommitID(path string) uuid.UUID
}

// identityImageFileMetadataResolver is an [imageFileMetadataResolver] that returns
// identity values: external path equals path, and no module metadata.
// Used by [BuildImageFromOpener] where the caller owns the file contents directly.
type identityImageFileMetadataResolver struct{}

func (identityImageFileMetadataResolver) ExternalPath(path string) string  { return path }
func (identityImageFileMetadataResolver) LocalPath(_ string) string        { return "" }
func (identityImageFileMetadataResolver) FullName(_ string) bufparse.FullName { return nil }
func (identityImageFileMetadataResolver) CommitID(_ string) uuid.UUID      { return uuid.Nil }

// BuildImageFromOpener is like [BuildImage] but accepts a [source.Opener] directly
// instead of a [bufmodule.ModuleReadBucket]. It is intended for use cases where the
// caller controls the file contents directly, such as the LSP where files may have
// unsaved modifications.
//
// The returned [*report.Report] always contains the full diagnostic output from
// compilation, including both errors and warnings, regardless of whether the [Image]
// is nil. The [Image] is nil only when a fatal error prevents descriptor generation.
func BuildImageFromOpener(
	ctx context.Context,
	logger *slog.Logger,
	opener source.Opener,
	paths []string,
	options ...BuildImageOption,
) (Image, *report.Report, error) {
	opts := newBuildImageOptions()
	for _, option := range options {
		option(opts)
	}

	session := new(ir.Session)
	parallelism := thread.Parallelism()
	if opts.noParallelism {
		parallelism = 1
	}
	exec := incremental.New(incremental.WithParallelism(int64(parallelism)))

	// Use context.WithoutCancel to prevent external context cancellation from
	// triggering an early return in incremental.Resolve while background goroutines
	// are still running. A bug in protocompile causes those goroutines to access
	// the (now-nil) named return slice after early return, causing a nil-index panic.
	// The queries are pure in-memory work and complete quickly; shielding them from
	// cancellation does not cause meaningful delays. We check ctx.Err() after Run
	// completes to propagate the original cancellation to the caller.
	results, diagnostics, err := incremental.Run(
		context.WithoutCancel(ctx),
		exec,
		queries.Link{
			Opener:    opener,
			Session:   session,
			Workspace: source.NewWorkspace(paths...),
		},
	)
	// incremental.Run can return a nil report on a fatal internal error. Normalize
	// to non-nil so callers can always range over Diagnostics safely.
	if diagnostics == nil {
		diagnostics = new(report.Report)
	}
	if err != nil {
		return nil, diagnostics, err
	}
	// Check if the caller's context was cancelled while we were compiling; if so,
	// treat it as a cancellation error even though the build completed.
	if ctx.Err() != nil {
		return nil, diagnostics, context.Cause(ctx)
	}

	if len(results) != 1 {
		return nil, diagnostics, fmt.Errorf("expected a single result from query, instead got: %d", len(results))
	}
	if results[0].Fatal != nil {
		return nil, diagnostics, results[0].Fatal
	}
	irFiles := results[0].Value

	for _, irFile := range irFiles {
		if !irFile.Lowered() {
			logger.WarnContext(ctx, "file not fully lowered", slog.String("path", irFile.Path()))
		}
	}

	fds, resolver, err := irFilesToFileDescriptorSet(irFiles)
	if err != nil {
		return nil, diagnostics, err
	}
	image, err := fileDescriptorSetToImage(resolver, identityImageFileMetadataResolver{}, paths, fds, opts.excludeSourceCodeInfo)
	if err != nil {
		return nil, diagnostics, err
	}
	return image, diagnostics, nil
}

// irFilesToFileDescriptorSet serializes linked [ir.File]s to a
// [descriptorpb.FileDescriptorSet] and builds a [protoencoding.Resolver] with
// Buf's custom descriptor extensions recognised, ready for [fileDescriptorSetToImage].
func irFilesToFileDescriptorSet(irFiles []*ir.File) (*descriptorpb.FileDescriptorSet, protoencoding.Resolver, error) {
	descriptorSetBytes, err := fdp.DescriptorSetBytes(
		irFiles,
		// Always include source code info to capture [descriptorv1.E_BufSourceCodeInfoExtension].
		// Source code info may still be stripped from the final Image later.
		fdp.IncludeSourceCodeInfo(true),
		// Needed for lint and breaking change detection annotations.
		fdp.GenerateExtraOptionLocations(true),
	)
	if err != nil {
		return nil, nil, err
	}
	fds := new(descriptorpb.FileDescriptorSet)
	if err := protoencoding.NewWireUnmarshaler(nil).Unmarshal(descriptorSetBytes, fds); err != nil {
		return nil, nil, err
	}
	// Include Buf's descriptor proto alongside the compiled files so that
	// ReparseExtensions can recognise [descriptorv1.E_BufSourceCodeInfoExtension]
	// and convert unknown fields to typed extensions.
	resolverFiles := []*descriptorpb.FileDescriptorProto{
		protodesc.ToFileDescriptorProto(descriptorv1.File_buf_descriptor_v1_descriptor_proto),
	}
	resolverFiles = append(resolverFiles, fds.File...)
	resolver := protoencoding.NewLazyResolver(resolverFiles...)
	for _, fileDescriptor := range fds.File {
		if err := protoencoding.ReparseExtensions(resolver, fileDescriptor.ProtoReflect()); err != nil {
			return nil, nil, err
		}
	}
	return fds, resolver, nil
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
	metadataResolver imageFileMetadataResolver,
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
	nonImportPaths := xslices.ToStructMap(paths)

	for _, path := range paths {
		imageFiles, err = getImageFilesForPath(
			path,
			pathToDescriptor,
			metadataResolver,
			excludeSourceCodeInfo,
			seen,
			nonImportPaths,
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
	metadataResolver imageFileMetadataResolver,
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
			metadataResolver,
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
		metadataResolver,
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
	metadataResolver imageFileMetadataResolver,
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
		metadataResolver.FullName(fileDescriptor.GetName()),
		metadataResolver.CommitID(fileDescriptor.GetName()),
		metadataResolver.ExternalPath(fileDescriptor.GetName()),
		metadataResolver.LocalPath(fileDescriptor.GetName()),
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
