// Copyright 2020 Buf Technologies Inc.
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

package bufbuild

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufimage"
	"github.com/bufbuild/buf/internal/buf/bufpath"
	"github.com/bufbuild/buf/internal/gen/embed/wkt"
	"github.com/bufbuild/buf/internal/pkg/instrument"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/bufbuild/buf/internal/pkg/thread"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// TODO: make special rules for wkt
var wktPathResolver = bufpath.NopPathResolver

type builder struct {
	logger *zap.Logger
}

func newBuilder(logger *zap.Logger, options ...BuilderOption) *builder {
	builder := &builder{
		logger: logger,
	}
	for _, option := range options {
		option(builder)
	}
	return builder

}

func (b *builder) Build(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	fileRefs []bufimage.FileRef,
	options ...BuildOption,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(ctx, readBucket, externalPathResolver, fileRefs, buildOptions.excludeSourceCodeInfo)
}

func (b *builder) build(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	fileRefs []bufimage.FileRef,
	excludeSourceCodeInfo bool,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	defer instrument.Start(b.logger, "build", zap.Int("num_files", len(fileRefs))).End()

	if len(fileRefs) == 0 {
		return nil, nil, errors.New("no input files specified")
	}

	roots, rootRelFilePaths := getRootsAndRootRelFilePathsForFileRefs(fileRefs)
	buildResults := b.getBuildResults(
		ctx,
		readBucket,
		externalPathResolver,
		roots,
		rootRelFilePaths,
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

	descFileDescriptors, err := getDescFileDescriptors(buildResults, rootRelFilePaths)
	if err != nil {
		return nil, nil, err
	}
	image, err := b.getImage(
		ctx,
		readBucket,
		externalPathResolver,
		roots,
		excludeSourceCodeInfo,
		descFileDescriptors,
	)
	if err != nil {
		return nil, nil, err
	}
	return image, nil, nil
}

func (b *builder) getBuildResults(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	rootRelFilePaths []string,
	excludeSourceCodeInfo bool,
) []*buildResult {
	defer instrument.Start(b.logger, "parse", zap.Int("num_files", len(rootRelFilePaths))).End()

	var buildResults []*buildResult
	chunkSize := 0
	if parallelism := thread.Parallelism(); parallelism > 1 {
		chunkSize = len(rootRelFilePaths) / parallelism
	}
	chunks := stringutil.SliceToChunks(rootRelFilePaths, chunkSize)
	buildResultC := make(chan *buildResult, len(chunks))
	for _, rootRelFilePaths := range chunks {
		rootRelFilePaths := rootRelFilePaths
		go func() {
			buildResultC <- getBuildResult(
				ctx,
				readBucket,
				externalPathResolver,
				roots,
				rootRelFilePaths,
				excludeSourceCodeInfo,
			)
		}()
	}
	for i := 0; i < len(chunks); i++ {
		select {
		case <-ctx.Done():
			return []*buildResult{newBuildResult(nil, nil, nil, ctx.Err())}
		case buildResult := <-buildResultC:
			buildResults = append(buildResults, buildResult)
		}
	}
	return buildResults
}

func getRootsAndRootRelFilePathsForFileRefs(fileRefs []bufimage.FileRef) ([]string, []string) {
	rootMap := make(map[string]struct{})
	rootRelFilePaths := make([]string, len(fileRefs))
	for i, fileRef := range fileRefs {
		rootMap[fileRef.RootDirPath()] = struct{}{}
		rootRelFilePaths[i] = fileRef.RootRelFilePath()
	}
	roots := make([]string, 0, len(rootMap))
	for root := range rootMap {
		roots = append(roots, root)
	}
	sort.Strings(roots)
	return roots, rootRelFilePaths
}

func getBuildResult(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	rootRelFilePaths []string,
	excludeSourceCodeInfo bool,
) *buildResult {
	var errorsWithPos []protoparse.ErrorWithPos
	var lock sync.Mutex

	parser := protoparse.Parser{
		ImportPaths:           roots,
		IncludeSourceCodeInfo: !excludeSourceCodeInfo,
		Accessor:              newParserAccessor(ctx, readBucket, roots),
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
	// fileDescriptors are in the same order as rootRelFilePaths per the documentation
	descFileDescriptors, err := parser.ParseFiles(rootRelFilePaths...)
	if err != nil {
		if err == protoparse.ErrInvalidSource {
			if len(errorsWithPos) == 0 {
				return newBuildResult(
					rootRelFilePaths,
					nil,
					nil,
					errors.New("got invalid source error but no errors reported"),
				)
			}
			fileAnnotations := make([]bufanalysis.FileAnnotation, 0, len(errorsWithPos))
			for _, errorWithPos := range errorsWithPos {
				fileAnnotation, err := getFileAnnotation(
					ctx,
					readBucket,
					externalPathResolver,
					roots,
					errorWithPos,
				)
				if err != nil {
					return newBuildResult(rootRelFilePaths, nil, nil, err)
				}
				fileAnnotations = append(fileAnnotations, fileAnnotation)
			}
			return newBuildResult(rootRelFilePaths, nil, fileAnnotations, nil)
		}
		return newBuildResult(rootRelFilePaths, nil, nil, err)
	}
	return newBuildResult(rootRelFilePaths, descFileDescriptors, nil, nil)
}

func getFileAnnotation(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	errorWithPos protoparse.ErrorWithPos,
) (bufanalysis.FileAnnotation, error) {
	var fileRef bufimage.FileRef
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
	//sourcePos := errorWithPos.GetPosition()
	if sourcePos.Filename != "" {
		rootRelFilePath, err := normalpath.NormalizeAndValidate(sourcePos.Filename)
		if err != nil {
			return nil, err
		}
		root, isWKT, err := getRootWKTForRootRelFilePath(ctx, readBucket, roots, rootRelFilePath)
		if err != nil {
			return nil, err
		}
		if isWKT {
			externalPathResolver = wktPathResolver
		}
		fileRef, err = bufimage.NewFileRef(rootRelFilePath, root, externalPathResolver)
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
		fileRef,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		message,
	), nil
}

func getReadCloserForFullRelFilePath(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
	fullRelFilePath string,
) (io.ReadCloser, error) {
	readCloser, readErr := readBucket.Get(ctx, fullRelFilePath)
	if readErr != nil {
		if !storage.IsNotExist(readErr) {
			return nil, readErr
		}
		for _, root := range roots {
			rootRelFilePath, err := normalpath.Rel(root, fullRelFilePath)
			if err != nil {
				return nil, err
			}
			if wktReadCloser, err := wkt.ReadBucket.Get(ctx, rootRelFilePath); err == nil {
				return wktReadCloser, nil
			}
		}
		return nil, readErr
	}
	return readCloser, nil
}

func getRootWKTForRootRelFilePath(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
	rootRelFilePath string,
) (string, bool, error) {
	for _, root := range roots {
		exists, err := storage.Exists(ctx, readBucket, normalpath.Join(root, rootRelFilePath))
		if err != nil {
			return "", false, err
		}
		if exists {
			return root, false, nil
		}
	}
	exists, err := storage.Exists(ctx, wkt.ReadBucket, rootRelFilePath)
	if err != nil {
		return "", false, err
	}
	if exists {
		return ".", true, nil
	}
	return "", false, fmt.Errorf("cannot determine root for file %s", rootRelFilePath)
}

func newParserAccessor(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
) func(string) (io.ReadCloser, error) {
	return func(fullRelFilePath string) (io.ReadCloser, error) {
		return getReadCloserForFullRelFilePath(ctx, readBucket, roots, fullRelFilePath)
	}
}

func getDescFileDescriptors(
	buildResults []*buildResult,
	rootRelFilePaths []string,
) ([]*desc.FileDescriptor, error) {
	var descFileDescriptors []*desc.FileDescriptor
	for _, buildResult := range buildResults {
		iRootRelFilePaths := buildResult.RootRelFilePaths
		iDescFileDescriptors := buildResult.DescFileDescriptors
		// do a rough verification that rootRelFilePaths <-> fileDescriptors
		// parser.ParseFiles is documented to return the same number of FileDescriptors
		// as the number of input files
		// https://godoc.org/github.com/jhump/protoreflect/desc/protoparse#Parser.ParseFiles
		if len(iDescFileDescriptors) != len(iRootRelFilePaths) {
			return nil, fmt.Errorf("expected FileDescriptors to be of length %d but was %d", len(iRootRelFilePaths), len(iDescFileDescriptors))
		}
		for i, iDescFileDescriptor := range iDescFileDescriptors {
			iRootRelFilePath := iRootRelFilePaths[i]
			iFilename := iDescFileDescriptor.GetName()
			// doing another rough verification
			// NO LONGER NEED TO DO SUFFIX SINCE WE KNOW THE ROOT FILE NAME
			if iRootRelFilePath != iFilename {
				return nil, fmt.Errorf("expected fileDescriptor name %s to be a equal to %s", iFilename, iRootRelFilePath)
			}
		}
		descFileDescriptors = append(descFileDescriptors, iDescFileDescriptors...)
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
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	excludeSourceCodeInfo bool,
	sortedFileDescriptors []*desc.FileDescriptor,
) (bufimage.Image, error) {
	defer instrument.Start(b.logger, "get_image").End()

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

	var files []bufimage.File
	var err error
	alreadySeen := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		files, err = getFilesRec(
			ctx,
			readBucket,
			externalPathResolver,
			roots,
			excludeSourceCodeInfo,
			fileDescriptor,
			alreadySeen,
			nonImportFilenames,
			files,
		)
		if err != nil {
			return nil, err
		}
	}
	return bufimage.NewImage(files)
}

func getFilesRec(
	ctx context.Context,
	readBucket storage.ReadBucket,
	externalPathResolver bufpath.ExternalPathResolver,
	roots []string,
	excludeSourceCodeInfo bool,
	descFileDescriptor *desc.FileDescriptor,
	alreadySeen map[string]struct{},
	nonImportFilenames map[string]struct{},
	files []bufimage.File,
) ([]bufimage.File, error) {
	if descFileDescriptor == nil {
		return nil, errors.New("nil FileDescriptor")
	}
	rootRelFilePath := descFileDescriptor.GetName()
	if _, ok := alreadySeen[rootRelFilePath]; ok {
		return files, nil
	}
	alreadySeen[rootRelFilePath] = struct{}{}

	var err error
	for _, dependency := range descFileDescriptor.GetDependencies() {
		files, err = getFilesRec(
			ctx,
			readBucket,
			externalPathResolver,
			roots,
			excludeSourceCodeInfo,
			dependency,
			alreadySeen,
			nonImportFilenames,
			files,
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
	root, isWKT, err := getRootWKTForRootRelFilePath(
		ctx,
		readBucket,
		roots,
		rootRelFilePath,
	)
	if err != nil {
		return nil, err
	}
	if isWKT {
		externalPathResolver = wktPathResolver
	}
	_, isNotImport := nonImportFilenames[rootRelFilePath]
	file, err := bufimage.NewFile(
		fileDescriptorProto,
		root,
		externalPathResolver,
		!isNotImport,
	)
	if err != nil {
		return nil, err
	}
	return append(files, file), nil
}

type buildResult struct {
	RootRelFilePaths    []string
	DescFileDescriptors []*desc.FileDescriptor
	FileAnnotations     []bufanalysis.FileAnnotation
	Err                 error
}

func newBuildResult(
	rootRelFilePaths []string,
	descFileDescriptors []*desc.FileDescriptor,
	fileAnnotations []bufanalysis.FileAnnotation,
	err error,
) *buildResult {
	return &buildResult{
		RootRelFilePaths:    rootRelFilePaths,
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
