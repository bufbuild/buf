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
	"sync"

	"github.com/bufbuild/buf/internal/buf/ext/extfile"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	"github.com/bufbuild/buf/internal/gen/embed/wkt"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/util/utillog"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type builder struct {
	logger      *zap.Logger
	parallelism int
}

func newBuilder(logger *zap.Logger, parallelism int) *builder {
	return &builder{
		logger:      logger,
		parallelism: parallelism,
	}
}

// Build runs compilation.
//
// If an error is returned, it is a system error.
// Only one of BuildResolt and FileAnnotations will be returned.
//
// FileAnnotations will be sorted, but Paths will not have the roots as a prefix, instead
// they will be relative to the roots. This should be fixed for linter outputs if image
// mode is not used.
func (b *builder) Build(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
	rootFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (_ *BuildResult, _ []*filev1beta1.FileAnnotation, retErr error) {
	defer utillog.DeferWithError(b.logger, "run", &retErr, zap.Int("num_files", len(rootFilePaths)))()

	if len(roots) == 0 {
		return nil, nil, errors.New("no roots specified")
	}
	if len(rootFilePaths) == 0 {
		return nil, nil, errors.New("no input files specified")
	}
	if uniqueLen := len(utilstring.SliceToMap(rootFilePaths)); uniqueLen != len(rootFilePaths) {
		// this is a system error, we should have verified this elsewhere
		return nil, nil, errors.New("rootFilePaths has duplicate values")
	}

	results := b.parse(
		ctx,
		readBucket,
		roots,
		rootFilePaths,
		includeImports,
		includeSourceInfo,
	)
	var resultErr error
	for _, result := range results {
		resultErr = multierr.Append(resultErr, result.Err)
	}
	if resultErr != nil {
		return nil, nil, resultErr
	}
	var fileAnnotations []*filev1beta1.FileAnnotation
	for _, result := range results {
		fileAnnotations = append(fileAnnotations, result.FileAnnotations...)
	}
	if len(fileAnnotations) > 0 {
		extfile.SortFileAnnotations(fileAnnotations)
		return nil, fileAnnotations, nil
	}

	descFileDescriptors, err := getDescFileDescriptors(results, rootFilePaths)
	if err != nil {
		return nil, nil, err
	}
	image, err := getImage(b.logger, descFileDescriptors, includeImports, includeSourceInfo)
	if err != nil {
		return nil, nil, err
	}
	imageWithImports := image
	if !includeImports {
		// note that the FileDescriptorProtos are shared! so if SourceCodeInfo was cleared on one, it will be cleared on both
		imageWithImports, err = getImage(b.logger, descFileDescriptors, true, includeSourceInfo)
		if err != nil {
			return nil, nil, err
		}
	}
	return &BuildResult{
		Image:            image,
		ImageWithImports: imageWithImports,
	}, nil, nil
}

func (b *builder) parse(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
	rootFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) []*result {
	defer utillog.Defer(b.logger, "parse", zap.Int("num_files", len(rootFilePaths)))()

	accessor := newParserAccessor(ctx, readBucket, roots)
	var results []*result
	chunkSize := 0
	if b.parallelism > 1 {
		chunkSize = len(rootFilePaths) / b.parallelism
	}
	chunks := utilstring.SliceToChunks(rootFilePaths, chunkSize)
	resultC := make(chan *result, len(chunks))
	for _, rootFilePaths := range chunks {
		rootFilePaths := rootFilePaths
		go func() {
			resultC <- b.getResult(
				ctx,
				accessor,
				roots,
				rootFilePaths,
				includeSourceInfo,
			)
		}()
	}
	for i := 0; i < len(chunks); i++ {
		select {
		case <-ctx.Done():
			return []*result{newResult(nil, nil, nil, ctx.Err())}
		case result := <-resultC:
			results = append(results, result)
		}
	}
	return results
}

func (b *builder) getResult(
	ctx context.Context,
	accessor protoparse.FileAccessor,
	roots []string,
	rootFilePaths []string,
	includeSourceInfo bool,
) *result {
	var errorsWithPos []protoparse.ErrorWithPos
	var lock sync.Mutex

	parser := protoparse.Parser{
		ImportPaths:           roots,
		IncludeSourceCodeInfo: includeSourceInfo,
		Accessor:              accessor,
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
	// fileDescriptors are in the same order as rootFilePaths per the documentation
	descFileDescriptors, err := parser.ParseFiles(rootFilePaths...)
	if err != nil {
		if err == protoparse.ErrInvalidSource {
			if len(errorsWithPos) == 0 {
				return newResult(rootFilePaths, nil, nil, errors.New("got invalid source error but no errors reported"))
			}
			fileAnnotations := make([]*filev1beta1.FileAnnotation, 0, len(errorsWithPos))
			for _, errorWithPos := range errorsWithPos {
				fileAnnotation, err := getFileAnnotation(errorWithPos)
				if err != nil {
					return newResult(rootFilePaths, nil, nil, err)
				}
				fileAnnotations = append(fileAnnotations, fileAnnotation)
			}
			return newResult(rootFilePaths, nil, fileAnnotations, nil)
		}
		return newResult(rootFilePaths, nil, nil, err)
	}
	return newResult(rootFilePaths, descFileDescriptors, nil, nil)
}

func getFileAnnotation(errorWithPos protoparse.ErrorWithPos) (*filev1beta1.FileAnnotation, error) {
	fileAnnotation := &filev1beta1.FileAnnotation{
		Type: "COMPILE",
	}
	// this should never happen
	// maybe we should error
	if errorWithPos.Unwrap() != nil {
		fileAnnotation.Message = errorWithPos.Unwrap().Error()
	} else {
		fileAnnotation.Message = "Compile error."
	}
	sourcePos := errorWithPos.GetPosition()
	if sourcePos.Filename != "" {
		// TODO: make sure this is normalized
		fileAnnotation.Path = sourcePos.Filename
	}
	if sourcePos.Line > 0 {
		fileAnnotation.StartLine = uint32(sourcePos.Line)
		fileAnnotation.EndLine = uint32(sourcePos.Line)
	}
	if sourcePos.Col > 0 {
		fileAnnotation.StartColumn = uint32(sourcePos.Col)
		fileAnnotation.EndColumn = uint32(sourcePos.Col)
	}
	return fileAnnotation, nil
}

// getImage gets the imagev1beta1.Image for the desc.FileDescriptor.
//
// This mimics protoc's output order.
// This assumes checkAndSortDescFileDescriptors was called)
//
// This sets all BufbuildExtension fields on the imagev1beta1.Image and imagev1beta1.Files.
func getImage(
	logger *zap.Logger,
	sortedFileDescriptors []*desc.FileDescriptor,
	includeImports bool,
	includeSourceInfo bool,
) (*imagev1beta1.Image, error) {
	defer utillog.Defer(logger, "get_image")()

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

	image := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}
	alreadySeen := map[string]struct{}{}
	for _, fileDescriptor := range sortedFileDescriptors {
		if err := getImageRec(
			alreadySeen,
			nonImportFilenames,
			image,
			fileDescriptor,
			includeImports,
			includeSourceInfo,
		); err != nil {
			return nil, err
		}
	}
	if err := extimage.ValidateImage(image); err != nil {
		return nil, err
	}
	return image, nil
}

func getImageRec(
	alreadySeen map[string]struct{},
	nonImportFilenames map[string]struct{},
	image *imagev1beta1.Image,
	descFileDescriptor *desc.FileDescriptor,
	includeImports bool,
	includeSourceInfo bool,
) error {
	if descFileDescriptor == nil {
		return errors.New("nil FileDescriptor")
	}
	if _, ok := alreadySeen[descFileDescriptor.GetName()]; ok {
		return nil
	}
	alreadySeen[descFileDescriptor.GetName()] = struct{}{}

	for _, dependency := range descFileDescriptor.GetDependencies() {
		if !includeImports {
			// we only include deps that were explicitly in the set of file names given
			if _, ok := nonImportFilenames[dependency.GetName()]; !ok {
				continue
			}
		}
		if err := getImageRec(
			alreadySeen,
			nonImportFilenames,
			image,
			dependency,
			includeImports,
			includeSourceInfo,
		); err != nil {
			return err
		}
	}

	file := descFileDescriptor.AsFileDescriptorProto()
	if file == nil {
		return errors.New("nil File")
	}
	if !includeSourceInfo {
		file.SourceCodeInfo = nil
	}
	image.File = append(image.File, file)
	_, isNotImport := nonImportFilenames[file.GetName()]
	if !isNotImport {
		fileIndex := uint32(len(image.File) - 1)
		image.BufbuildImageExtension.ImageImportRefs = append(
			image.BufbuildImageExtension.ImageImportRefs,
			&imagev1beta1.ImageImportRef{
				FileIndex: proto.Uint32(fileIndex),
			},
		)
	}
	return nil
}

func getDescFileDescriptors(results []*result, rootFilePaths []string) ([]*desc.FileDescriptor, error) {
	var descFileDescriptors []*desc.FileDescriptor
	for _, result := range results {
		iRootFilePaths := result.RootFilePaths
		iDescFileDescriptors := result.DescFileDescriptors
		// do a rough verification that rootFilePaths <-> fileDescriptors
		// parser.ParseFiles is documented to return the same number of FileDescriptors
		// as the number of input files
		// https://godoc.org/github.com/jhump/protoreflect/desc/protoparse#Parser.ParseFiles
		if len(iDescFileDescriptors) != len(iRootFilePaths) {
			return nil, fmt.Errorf("expected FileDescriptors to be of length %d but was %d", len(iRootFilePaths), len(iDescFileDescriptors))
		}
		for i, iDescFileDescriptor := range iDescFileDescriptors {
			iRootFilePath := iRootFilePaths[i]
			iFilename := iDescFileDescriptor.GetName()
			// doing another rough verification
			// NO LONGER NEED TO DO SUFFIX SINCE WE KNOW THE ROOT FILE NAME
			if iRootFilePath != iFilename {
				return nil, fmt.Errorf("expected fileDescriptor name %s to be a equal to %s", iFilename, iRootFilePath)
			}
		}
		descFileDescriptors = append(descFileDescriptors, iDescFileDescriptors...)
	}
	return checkAndSortDescFileDescriptors(descFileDescriptors, rootFilePaths)
}

// We need to sort the FileDescriptors as they may/probably are out of order
// relative to input order after concurrent builds. This mimics the output
// order of protoc.
func checkAndSortDescFileDescriptors(
	descFileDescriptors []*desc.FileDescriptor,
	rootFilePaths []string,
) ([]*desc.FileDescriptor, error) {
	if len(descFileDescriptors) != len(rootFilePaths) {
		return nil, fmt.Errorf("rootFilePath length was %d but FileDescriptor length was %d", len(rootFilePaths), len(descFileDescriptors))
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
	// is equal to the number of rootFilePaths. We also verified earlier that rootFilePaths
	// has only unique values. Now we can put them in order.
	sortedDescFileDescriptors := make([]*desc.FileDescriptor, 0, len(descFileDescriptors))
	for _, rootFilePath := range rootFilePaths {
		descFileDescriptor, ok := nameToDescFileDescriptor[rootFilePath]
		if !ok {
			return nil, fmt.Errorf("no FileDescriptor for rootFilePath: %q", rootFilePath)
		}
		sortedDescFileDescriptors = append(sortedDescFileDescriptors, descFileDescriptor)
	}
	return sortedDescFileDescriptors, nil
}

func newParserAccessor(
	ctx context.Context,
	readBucket storage.ReadBucket,
	roots []string,
) func(string) (io.ReadCloser, error) {
	return func(rootFilePath string) (io.ReadCloser, error) {
		readCloser, err := readBucket.Get(ctx, rootFilePath)
		if err != nil {
			if !storage.IsNotExist(err) {
				return nil, err
			}
			for _, root := range roots {
				relFilePath, relErr := storagepath.Rel(root, rootFilePath)
				if relErr != nil {
					return nil, relErr
				}
				if wktReadCloser, wktErr := wkt.ReadBucket.Get(ctx, relFilePath); wktErr == nil {
					return wktReadCloser, nil
				}
			}
			return nil, err
		}
		return readCloser, nil
	}
}

type result struct {
	RootFilePaths       []string
	DescFileDescriptors []*desc.FileDescriptor
	FileAnnotations     []*filev1beta1.FileAnnotation
	Err                 error
}

func newResult(
	rootFilePaths []string,
	descFileDescriptors []*desc.FileDescriptor,
	fileAnnotations []*filev1beta1.FileAnnotation,
	err error,
) *result {
	return &result{
		RootFilePaths:       rootFilePaths,
		DescFileDescriptors: descFileDescriptors,
		FileAnnotations:     fileAnnotations,
		Err:                 err,
	}
}
