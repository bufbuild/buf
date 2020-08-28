// Copyright 2020 Buf Technologies, Inc.
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

package bufimagebuild

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcore"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type builder struct {
	logger *zap.Logger
}

func newBuilder(logger *zap.Logger) *builder {
	return &builder{
		logger: logger,
	}

}

func (b *builder) Build(
	ctx context.Context,
	moduleFileSet bufmodule.ModuleFileSet,
	options ...BuildOption,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		moduleFileSet,
		buildOptions.excludeSourceCodeInfo,
	)
}

func (b *builder) build(
	ctx context.Context,
	moduleFileSet bufmodule.ModuleFileSet,
	excludeSourceCodeInfo bool,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	ctx, span := trace.StartSpan(ctx, "build")
	defer span.End()

	parserAccessorHandler := newParserAccessorHandler(ctx, moduleFileSet)
	targetFileInfos, err := moduleFileSet.TargetFileInfos(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(targetFileInfos) == 0 {
		return nil, nil, errors.New("no input files specified")
	}
	paths := make([]string, len(targetFileInfos))
	for i, targetFileInfo := range targetFileInfos {
		paths[i] = targetFileInfo.Path()
	}

	buildResults := b.getBuildResults(
		ctx,
		parserAccessorHandler,
		paths,
		excludeSourceCodeInfo,
	)
	var buildResultErr error
	for _, buildResult := range buildResults {
		buildResultErr = multierr.Append(buildResultErr, buildResult.Err)
	}
	if buildResultErr != nil {
		return nil, nil, buildResultErr
	}
	var fileAnnotations []bufanalysis.FileAnnotation
	for _, buildResult := range buildResults {
		fileAnnotations = append(fileAnnotations, buildResult.FileAnnotations...)
	}
	if len(fileAnnotations) > 0 {
		bufanalysis.SortFileAnnotations(fileAnnotations)
		return nil, fileAnnotations, nil
	}

	descFileDescriptors, err := getDescFileDescriptorsFromBuildResults(buildResults, paths)
	if err != nil {
		return nil, nil, err
	}
	image, err := b.getImage(
		ctx,
		excludeSourceCodeInfo,
		descFileDescriptors,
		parserAccessorHandler,
	)
	if err != nil {
		return nil, nil, err
	}
	return image, nil, nil
}

func (b *builder) getBuildResults(
	ctx context.Context,
	parserAccessorHandler *parserAccessorHandler,
	paths []string,
	excludeSourceCodeInfo bool,
) []*buildResult {
	ctx, span := trace.StartSpan(ctx, "parse")
	defer span.End()

	var buildResults []*buildResult
	chunkSize := 0
	if parallelism := thread.Parallelism(); parallelism > 1 {
		chunkSize = len(paths) / parallelism
	}
	chunks := stringutil.SliceToChunks(paths, chunkSize)
	buildResultC := make(chan *buildResult, len(chunks))
	for _, iPaths := range chunks {
		iPaths := iPaths
		go func() {
			var buildResult *buildResult
			defer func() {
				select {
				case buildResultC <- buildResult:
				case <-ctx.Done():
				}
			}()
			defer func() {
				// Recover any panics here since we run in a goroutine
				v := recover()
				if v != nil {
					buildResult = newBuildResult(
						nil,
						nil,
						fmt.Errorf("panic: %v, stack:\n%s", v, string(debug.Stack())),
					)
				}
			}()
			buildResult = getBuildResult(
				ctx,
				parserAccessorHandler,
				iPaths,
				excludeSourceCodeInfo,
			)
		}()
	}
	for i := 0; i < len(chunks); i++ {
		select {
		case <-ctx.Done():
			return []*buildResult{newBuildResult(nil, nil, ctx.Err())}
		case buildResult := <-buildResultC:
			buildResults = append(buildResults, buildResult)
		}
	}
	return buildResults
}

func getBuildResult(
	ctx context.Context,
	parserAccessorHandler *parserAccessorHandler,
	paths []string,
	excludeSourceCodeInfo bool,
) *buildResult {
	var errorsWithPos []protoparse.ErrorWithPos
	var lock sync.Mutex
	parser := protoparse.Parser{
		IncludeSourceCodeInfo: !excludeSourceCodeInfo,
		Accessor:              parserAccessorHandler.Open,
		ErrorReporter: func(errorWithPos protoparse.ErrorWithPos) error {
			// protoparse isn't concurrent right now but just to be safe
			// for the future
			lock.Lock()
			errorsWithPos = append(errorsWithPos, errorWithPos)
			lock.Unlock()
			// continue parsing
			return nil
		},
	}
	// fileDescriptors are in the same order as paths per the documentation
	descFileDescriptors, err := parser.ParseFiles(paths...)
	if err != nil {
		if err == protoparse.ErrInvalidSource {
			if len(errorsWithPos) == 0 {
				return newBuildResult(
					nil,
					nil,
					errors.New("got invalid source error from parse but no errors reported"),
				)
			}
			fileAnnotations, err := getFileAnnotations(
				ctx,
				parserAccessorHandler,
				errorsWithPos,
			)
			if err != nil {
				return newBuildResult(nil, nil, err)
			}
			return newBuildResult(nil, fileAnnotations, nil)
		}
		return newBuildResult(nil, nil, err)
	} else if len(errorsWithPos) > 0 {
		// https://github.com/jhump/protoreflect/pull/331
		return newBuildResult(
			nil,
			nil,
			errors.New("got no error from parse but errors reported"),
		)
	}
	if len(descFileDescriptors) != len(paths) {
		return newBuildResult(
			nil,
			nil,
			fmt.Errorf("expected FileDescriptors to be of length %d but was %d", len(paths), len(descFileDescriptors)),
		)
	}
	for i, descFileDescriptor := range descFileDescriptors {
		path := paths[i]
		filename := descFileDescriptor.GetName()
		// doing another rough verification
		// NO LONGER NEED TO DO SUFFIX SINCE WE KNOW THE ROOT FILE NAME
		if path != filename {
			return newBuildResult(
				nil,
				nil,
				fmt.Errorf("expected fileDescriptor name %s to be a equal to %s", filename, path),
			)
		}
	}
	return newBuildResult(descFileDescriptors, nil, nil)
}

func getFileAnnotations(
	ctx context.Context,
	parserAccessorHandler *parserAccessorHandler,
	errorsWithPos []protoparse.ErrorWithPos,
) ([]bufanalysis.FileAnnotation, error) {
	fileAnnotations := make([]bufanalysis.FileAnnotation, 0, len(errorsWithPos))
	for _, errorWithPos := range errorsWithPos {
		fileAnnotation, err := getFileAnnotation(
			ctx,
			parserAccessorHandler,
			errorWithPos,
		)
		if err != nil {
			return nil, err
		}
		fileAnnotations = append(fileAnnotations, fileAnnotation)
	}
	return fileAnnotations, nil
}

func getFileAnnotation(
	ctx context.Context,
	parserAccessorHandler *parserAccessorHandler,
	errorWithPos protoparse.ErrorWithPos,
) (bufanalysis.FileAnnotation, error) {
	var fileInfo bufcore.FileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	typeString := "COMPILE"
	message := "Compile error."
	// this should never happen
	// maybe we should error
	if errorWithPos.Unwrap() != nil {
		message = errorWithPos.Unwrap().Error()
	}
	sourcePos := protoparse.SourcePos{}
	if errorWithSourcePos, ok := errorWithPos.(protoparse.ErrorWithSourcePos); ok {
		if pos := errorWithSourcePos.Pos; pos != nil {
			sourcePos = *pos
		}
	}
	if sourcePos.Filename != "" {
		path, err := normalpath.NormalizeAndValidate(sourcePos.Filename)
		if err != nil {
			return nil, err
		}
		fileInfo, err = bufcore.NewFileInfo(
			path,
			parserAccessorHandler.ExternalPath(path),
			parserAccessorHandler.IsImport(path),
		)
		if err != nil {
			return nil, err
		}
	}
	if sourcePos.Line > 0 {
		startLine = sourcePos.Line
		endLine = sourcePos.Line
	}
	if sourcePos.Col > 0 {
		startColumn = sourcePos.Col
		endColumn = sourcePos.Col
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		message,
	), nil
}

func getDescFileDescriptorsFromBuildResults(
	buildResults []*buildResult,
	rootRelFilePaths []string,
) ([]*desc.FileDescriptor, error) {
	var descFileDescriptors []*desc.FileDescriptor
	for _, buildResult := range buildResults {
		descFileDescriptors = append(descFileDescriptors, buildResult.DescFileDescriptors...)
	}
	return checkAndSortDescFileDescriptors(descFileDescriptors, rootRelFilePaths)
}

// We need to sort the FileDescriptors as they may/probably are out of order
// relative to input order after concurrent builds. This mimics the output
// order of protoc.
func checkAndSortDescFileDescriptors(
	descFileDescriptors []*desc.FileDescriptor,
	rootRelFilePaths []string,
) ([]*desc.FileDescriptor, error) {
	if len(descFileDescriptors) != len(rootRelFilePaths) {
		return nil, fmt.Errorf("rootRelFilePath length was %d but FileDescriptor length was %d", len(rootRelFilePaths), len(descFileDescriptors))
	}
	nameToDescFileDescriptor := make(map[string]*desc.FileDescriptor, len(descFileDescriptors))
	for _, descFileDescriptor := range descFileDescriptors {
		// This is equal to descFileDescriptor.AsFileDescriptorProto().GetName()
		// but we double-check just in case
		//
		// https://github.com/jhump/protoreflect/blob/master/desc/descriptor.go#L82
		name := descFileDescriptor.GetName()
		if name == "" {
			return nil, errors.New("no name on FileDescriptor")
		}
		if name != descFileDescriptor.AsFileDescriptorProto().GetName() {
			return nil, errors.New("name not equal on FileDescriptorProto")
		}
		if _, ok := nameToDescFileDescriptor[name]; ok {
			return nil, fmt.Errorf("duplicate FileDescriptor: %s", name)
		}
		nameToDescFileDescriptor[name] = descFileDescriptor
	}
	// We now know that all FileDescriptors had unique names and the number of FileDescriptors
	// is equal to the number of rootRelFilePaths. We also verified earlier that rootRelFilePaths
	// has only unique values. Now we can put them in order.
	sortedDescFileDescriptors := make([]*desc.FileDescriptor, 0, len(descFileDescriptors))
	for _, rootRelFilePath := range rootRelFilePaths {
		descFileDescriptor, ok := nameToDescFileDescriptor[rootRelFilePath]
		if !ok {
			return nil, fmt.Errorf("no FileDescriptor for rootRelFilePath: %q", rootRelFilePath)
		}
		sortedDescFileDescriptors = append(sortedDescFileDescriptors, descFileDescriptor)
	}
	return sortedDescFileDescriptors, nil
}

// getFiles gets the Files for the desc.FileDescriptors.
//
// This mimics protoc's output order.
// This assumes checkAndSortDescFileDescriptors was called.
func (b *builder) getImage(
	ctx context.Context,
	excludeSourceCodeInfo bool,
	sortedFileDescriptors []*desc.FileDescriptor,
	parserAccessorHandler *parserAccessorHandler,
) (bufimage.Image, error) {
	ctx, span := trace.StartSpan(ctx, "get_image")
	defer span.End()

	// if we aren't including imports, then we need a set of file names that
	// are included so we can create a topologically sorted list w/out
	// including imports that should not be present.
	//
	// if we are including imports, then we need to know what filenames
	// are imports are what filenames are not
	// all input desc.FileDescriptors are not imports, we derive the imports
	// from GetDependencies.
	nonImportFilenames := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		nonImportFilenames[fileDescriptor.GetName()] = struct{}{}
	}

	var imageFiles []bufimage.ImageFile
	var err error
	alreadySeen := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		imageFiles, err = getImageFilesRec(
			ctx,
			excludeSourceCodeInfo,
			fileDescriptor,
			parserAccessorHandler,
			alreadySeen,
			nonImportFilenames,
			imageFiles,
		)
		if err != nil {
			return nil, err
		}
	}
	return bufimage.NewImage(imageFiles)
}

func getImageFilesRec(
	ctx context.Context,
	excludeSourceCodeInfo bool,
	descFileDescriptor *desc.FileDescriptor,
	parserAccessorHandler *parserAccessorHandler,
	alreadySeen map[string]struct{},
	nonImportFilenames map[string]struct{},
	imageFiles []bufimage.ImageFile,
) ([]bufimage.ImageFile, error) {
	if descFileDescriptor == nil {
		return nil, errors.New("nil FileDescriptor")
	}
	path := descFileDescriptor.GetName()
	if _, ok := alreadySeen[path]; ok {
		return imageFiles, nil
	}
	alreadySeen[path] = struct{}{}

	var err error
	for _, dependency := range descFileDescriptor.GetDependencies() {
		imageFiles, err = getImageFilesRec(
			ctx,
			excludeSourceCodeInfo,
			dependency,
			parserAccessorHandler,
			alreadySeen,
			nonImportFilenames,
			imageFiles,
		)
		if err != nil {
			return nil, err
		}
	}

	fileDescriptorProto := descFileDescriptor.AsFileDescriptorProto()
	if fileDescriptorProto == nil {
		return nil, errors.New("nil FileDescriptorProto")
	}
	if excludeSourceCodeInfo {
		// need to do this anyways as Parser does not respect this for FileDescriptorProtos
		fileDescriptorProto.SourceCodeInfo = nil
	}
	_, isNotImport := nonImportFilenames[path]
	imageFile, err := bufimage.NewImageFile(
		fileDescriptorProto,
		// if empty, defaults to path
		parserAccessorHandler.ExternalPath(path),
		!isNotImport,
	)
	if err != nil {
		return nil, err
	}
	return append(imageFiles, imageFile), nil
}

type buildResult struct {
	DescFileDescriptors []*desc.FileDescriptor
	FileAnnotations     []bufanalysis.FileAnnotation
	Err                 error
}

func newBuildResult(
	descFileDescriptors []*desc.FileDescriptor,
	fileAnnotations []bufanalysis.FileAnnotation,
	err error,
) *buildResult {
	return &buildResult{
		DescFileDescriptors: descFileDescriptors,
		FileAnnotations:     fileAnnotations,
		Err:                 err,
	}
}

type buildOptions struct {
	excludeSourceCodeInfo bool
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}
