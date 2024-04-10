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

package bufimage

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufprotocompile"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/editionstesting"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func init() {
	// This enables editions support in protocompile. Buf doesn't yet support editions, but rejecting
	// such files happens elsewhere (in github.com/bufbuilder/buf/private/pkg/protodescriptor). We
	// need the compiler's support enabled so that we can start implementing and testing editions
	// support in the CLI, before we enable that support for users.
	editionstesting.AllowEditions()
}

func buildImage(
	ctx context.Context,
	tracer tracing.Tracer,
	moduleReadBucket bufmodule.ModuleReadBucket,
	excludeSourceCodeInfo bool,
	noParallelism bool,
) (_ Image, retErr error) {
	ctx, span := tracer.Start(ctx, tracing.WithErr(&retErr))
	defer span.End()

	if !moduleReadBucket.ShouldBeSelfContained() {
		return nil, syserror.New("passed a ModuleReadBucket to BuildImage that was not expected to be self-contained")
	}
	moduleReadBucket = bufmodule.ModuleReadBucketWithOnlyProtoFiles(moduleReadBucket)
	parserAccessorHandler := newParserAccessorHandler(ctx, moduleReadBucket)
	targetFileInfos, err := bufmodule.GetTargetFileInfos(ctx, moduleReadBucket)
	if err != nil {
		return nil, err
	}
	if len(targetFileInfos) == 0 {
		// If we had no no target files within the module after path filtering, this is an error.
		// We could have a better user error than this. This gets back to the lack of allowNotExist.
		return nil, bufmodule.ErrNoTargetProtoFiles
	}
	paths := bufmodule.FileInfoPaths(targetFileInfos)

	buildResult := getBuildResult(
		ctx,
		parserAccessorHandler,
		paths,
		excludeSourceCodeInfo,
		noParallelism,
	)
	if buildResult.Err != nil {
		return nil, buildResult.Err
	}
	fileDescriptors, err := checkAndSortFileDescriptors(buildResult.FileDescriptors, paths)
	if err != nil {
		return nil, err
	}
	image, err := getImage(
		ctx,
		excludeSourceCodeInfo,
		fileDescriptors,
		parserAccessorHandler,
		buildResult.SyntaxUnspecifiedFilenames,
		buildResult.FilenameToUnusedDependencyFilenames,
	)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func getBuildResult(
	ctx context.Context,
	parserAccessorHandler *parserAccessorHandler,
	paths []string,
	excludeSourceCodeInfo bool,
	noParallelism bool,
) *buildResult {
	var errorsWithPos []reporter.ErrorWithPos
	var warningErrorsWithPos []reporter.ErrorWithPos
	// With "extra option locations", buf can include more comments
	// for an option value than protoc can. In particular, this allows
	// it to preserve comments inside of message literals.
	sourceInfoMode := protocompile.SourceInfoExtraOptionLocations
	if excludeSourceCodeInfo {
		sourceInfoMode = protocompile.SourceInfoNone
	}
	parallelism := thread.Parallelism()
	if noParallelism {
		parallelism = 1
	}
	compiler := protocompile.Compiler{
		MaxParallelism: parallelism,
		SourceInfoMode: sourceInfoMode,
		Resolver:       &protocompile.SourceResolver{Accessor: parserAccessorHandler.Open},
		Reporter: reporter.NewReporter(
			func(errorWithPos reporter.ErrorWithPos) error {
				errorsWithPos = append(errorsWithPos, errorWithPos)
				return nil
			},
			func(errorWithPos reporter.ErrorWithPos) {
				warningErrorsWithPos = append(warningErrorsWithPos, errorWithPos)
			},
		),
	}
	// fileDescriptors are in the same order as paths per the documentation
	compiledFiles, err := compiler.Compile(ctx, paths...)
	if err != nil {
		if err == reporter.ErrInvalidSource {
			if len(errorsWithPos) == 0 {
				return newBuildResult(
					nil,
					nil,
					nil,
					errors.New("got invalid source error from parse but no errors reported"),
				)
			}
			fileAnnotationSet, err := bufprotocompile.FileAnnotationSetForErrorsWithPos(
				errorsWithPos,
				bufprotocompile.WithExternalPathResolver(parserAccessorHandler.ExternalPath),
			)
			if err != nil {
				return newBuildResult(nil, nil, nil, err)
			}
			return newBuildResult(nil, nil, nil, fileAnnotationSet)
		}
		if errorWithPos, ok := err.(reporter.ErrorWithPos); ok {
			fileAnnotation, err := bufprotocompile.FileAnnotationForErrorWithPos(
				errorWithPos,
				bufprotocompile.WithExternalPathResolver(parserAccessorHandler.ExternalPath),
			)
			if err != nil {
				return newBuildResult(nil, nil, nil, err)
			}
			return newBuildResult(nil, nil, nil, bufanalysis.NewFileAnnotationSet(fileAnnotation))
		}
		return newBuildResult(nil, nil, nil, err)
	} else if len(errorsWithPos) > 0 {
		// https://github.com/jhump/protoreflect/pull/331
		return newBuildResult(
			nil,
			nil,
			nil,
			errors.New("got no error from parse but errors reported"),
		)
	}
	if len(compiledFiles) != len(paths) {
		return newBuildResult(
			nil,
			nil,
			nil,
			fmt.Errorf("expected FileDescriptors to be of length %d but was %d", len(paths), len(compiledFiles)),
		)
	}
	for i, fileDescriptor := range compiledFiles {
		path := paths[i]
		filename := fileDescriptor.Path()
		// doing another rough verification
		// NO LONGER NEED TO DO SUFFIX SINCE WE KNOW THE ROOT FILE NAME
		if path != filename {
			return newBuildResult(
				nil,
				nil,
				nil,
				fmt.Errorf("expected fileDescriptor name %s to be a equal to %s", filename, path),
			)
		}
	}
	syntaxUnspecifiedFilenames := make(map[string]struct{})
	filenameToUnusedDependencyFilenames := make(map[string]map[string]struct{})
	for _, warningErrorWithPos := range warningErrorsWithPos {
		maybeAddSyntaxUnspecified(syntaxUnspecifiedFilenames, warningErrorWithPos)
		maybeAddUnusedImport(filenameToUnusedDependencyFilenames, warningErrorWithPos)
	}
	return newBuildResult(
		compiledFiles,
		syntaxUnspecifiedFilenames,
		filenameToUnusedDependencyFilenames,
		nil,
	)
}

// We need to sort the FileDescriptors as they may/probably are out of order
// relative to input order after concurrent builds. This mimics the output
// order of protoc.
func checkAndSortFileDescriptors(
	fileDescriptors linker.Files,
	rootRelFilePaths []string,
) (linker.Files, error) {
	if len(fileDescriptors) != len(rootRelFilePaths) {
		return nil, fmt.Errorf("rootRelFilePath length was %d but FileDescriptor length was %d", len(rootRelFilePaths), len(fileDescriptors))
	}
	nameToFileDescriptor := make(map[string]linker.File, len(fileDescriptors))
	for _, fileDescriptor := range fileDescriptors {
		name := fileDescriptor.Path()
		if name == "" {
			return nil, errors.New("no name on FileDescriptor")
		}
		if _, ok := nameToFileDescriptor[name]; ok {
			return nil, fmt.Errorf("duplicate FileDescriptor: %s", name)
		}
		nameToFileDescriptor[name] = fileDescriptor
	}
	// We now know that all FileDescriptors had unique names and the number of FileDescriptors
	// is equal to the number of rootRelFilePaths. We also verified earlier that rootRelFilePaths
	// has only unique values. Now we can put them in order.
	sortedFileDescriptors := make(linker.Files, 0, len(fileDescriptors))
	for _, rootRelFilePath := range rootRelFilePaths {
		fileDescriptor, ok := nameToFileDescriptor[rootRelFilePath]
		if !ok {
			return nil, fmt.Errorf("no FileDescriptor for rootRelFilePath: %q", rootRelFilePath)
		}
		sortedFileDescriptors = append(sortedFileDescriptors, fileDescriptor)
	}
	return sortedFileDescriptors, nil
}

// getImage gets the Image for the protoreflect.FileDescriptors.
//
// This mimics protoc's output order.
// This assumes checkAndSortFileDescriptors was called.
func getImage(
	ctx context.Context,
	excludeSourceCodeInfo bool,
	sortedFileDescriptors linker.Files,
	parserAccessorHandler *parserAccessorHandler,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
) (Image, error) {
	// if we aren't including imports, then we need a set of file names that
	// are included so we can create a topologically sorted list w/out
	// including imports that should not be present.
	//
	// if we are including imports, then we need to know what filenames
	// are imports are what filenames are not
	// all input protoreflect.FileDescriptors are not imports, we derive the imports
	// from GetDependencies.
	nonImportFilenames := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		nonImportFilenames[fileDescriptor.Path()] = struct{}{}
	}

	var imageFiles []ImageFile
	var err error
	alreadySeen := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		imageFiles, err = getImageFilesRec(
			ctx,
			excludeSourceCodeInfo,
			fileDescriptor,
			parserAccessorHandler,
			syntaxUnspecifiedFilenames,
			filenameToUnusedDependencyFilenames,
			alreadySeen,
			nonImportFilenames,
			imageFiles,
		)
		if err != nil {
			return nil, err
		}
	}
	return newImage(imageFiles, false, &resolverFromBuildResult{sortedFileDescriptors.AsResolver()})
}

func getImageFilesRec(
	ctx context.Context,
	excludeSourceCodeInfo bool,
	fileDescriptor protoreflect.FileDescriptor,
	parserAccessorHandler *parserAccessorHandler,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
	alreadySeen map[string]struct{},
	nonImportFilenames map[string]struct{},
	imageFiles []ImageFile,
) ([]ImageFile, error) {
	if fileDescriptor == nil {
		return nil, errors.New("nil FileDescriptor")
	}
	path := fileDescriptor.Path()
	if _, ok := alreadySeen[path]; ok {
		return imageFiles, nil
	}
	alreadySeen[path] = struct{}{}

	unusedDependencyFilenames, ok := filenameToUnusedDependencyFilenames[path]
	var unusedDependencyIndexes []int32
	if ok {
		unusedDependencyIndexes = make([]int32, 0, len(unusedDependencyFilenames))
	}
	var err error
	for i := 0; i < fileDescriptor.Imports().Len(); i++ {
		dependency := fileDescriptor.Imports().Get(i).FileDescriptor
		if unusedDependencyFilenames != nil {
			if _, ok := unusedDependencyFilenames[dependency.Path()]; ok {
				unusedDependencyIndexes = append(
					unusedDependencyIndexes,
					int32(i),
				)
			}
		}
		imageFiles, err = getImageFilesRec(
			ctx,
			excludeSourceCodeInfo,
			dependency,
			parserAccessorHandler,
			syntaxUnspecifiedFilenames,
			filenameToUnusedDependencyFilenames,
			alreadySeen,
			nonImportFilenames,
			imageFiles,
		)
		if err != nil {
			return nil, err
		}
	}

	fileDescriptorProto := protoutil.ProtoFromFileDescriptor(fileDescriptor)
	if fileDescriptorProto == nil {
		return nil, errors.New("nil FileDescriptorProto")
	}
	if excludeSourceCodeInfo {
		// need to do this anyways as Parser does not respect this for FileDescriptorProtos
		fileDescriptorProto.SourceCodeInfo = nil
	}
	_, isNotImport := nonImportFilenames[path]
	_, syntaxUnspecified := syntaxUnspecifiedFilenames[path]
	imageFile, err := NewImageFile(
		fileDescriptorProto,
		parserAccessorHandler.ModuleFullName(path),
		parserAccessorHandler.CommitID(path),
		// if empty, defaults to path
		parserAccessorHandler.ExternalPath(path),
		parserAccessorHandler.LocalPath(path),
		!isNotImport,
		syntaxUnspecified,
		unusedDependencyIndexes,
	)
	if err != nil {
		return nil, err
	}
	return append(imageFiles, imageFile), nil
}

func maybeAddSyntaxUnspecified(
	syntaxUnspecifiedFilenames map[string]struct{},
	errorWithPos reporter.ErrorWithPos,
) {
	if errorWithPos.Unwrap() != parser.ErrNoSyntax {
		return
	}
	syntaxUnspecifiedFilenames[errorWithPos.GetPosition().Filename] = struct{}{}
}

func maybeAddUnusedImport(
	filenameToUnusedImportFilenames map[string]map[string]struct{},
	errorWithPos reporter.ErrorWithPos,
) {
	errorUnusedImport, ok := errorWithPos.Unwrap().(linker.ErrorUnusedImport)
	if !ok {
		return
	}
	pos := errorWithPos.GetPosition()
	unusedImportFilenames, ok := filenameToUnusedImportFilenames[pos.Filename]
	if !ok {
		unusedImportFilenames = make(map[string]struct{})
		filenameToUnusedImportFilenames[pos.Filename] = unusedImportFilenames
	}
	unusedImportFilenames[errorUnusedImport.UnusedImport()] = struct{}{}
}

type buildResult struct {
	FileDescriptors                     linker.Files
	SyntaxUnspecifiedFilenames          map[string]struct{}
	FilenameToUnusedDependencyFilenames map[string]map[string]struct{}
	Err                                 error
}

func newBuildResult(
	fileDescriptors linker.Files,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
	err error,
) *buildResult {
	return &buildResult{
		FileDescriptors:                     fileDescriptors,
		SyntaxUnspecifiedFilenames:          syntaxUnspecifiedFilenames,
		FilenameToUnusedDependencyFilenames: filenameToUnusedDependencyFilenames,
		Err:                                 err,
	}
}

type buildImageOptions struct {
	excludeSourceCodeInfo bool
	noParallelism         bool
}

func newBuildImageOptions() *buildImageOptions {
	return &buildImageOptions{}
}

type resolverFromBuildResult struct {
	linker.Resolver
}

func (r *resolverFromBuildResult) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	descriptor, err := r.Resolver.FindDescriptorByName(enum)
	if err != nil {
		return nil, err
	}
	enumDescriptor, ok := descriptor.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not an enum", enum)
	}
	return dynamicpb.NewEnumType(enumDescriptor), nil
}
