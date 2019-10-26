package bufbuild

import (
	"context"
	"io"
	"runtime"
	"sync"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/protodescpb"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"go.uber.org/zap"
)

type runner struct {
	logger *zap.Logger
}

func newRunner(logger *zap.Logger) *runner {
	return &runner{
		logger: logger.Named("build"),
	}
}

func (r *runner) Run(
	ctx context.Context,
	bucket storage.ReadBucket,
	protoFileSet ProtoFileSet,
	opts ...RunOption,
) (bufpb.Image, []*analysis.Annotation, error) {
	options := &runOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return r.run(
		ctx,
		bucket,
		protoFileSet.Roots(),
		protoFileSet.RootFilePaths(),
		options.IncludeImports,
		options.IncludeSourceInfo,
	)
}

func (r *runner) run(
	ctx context.Context,
	bucket storage.ReadBucket,
	roots []string,
	rootFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (_ bufpb.Image, _ []*analysis.Annotation, retErr error) {
	defer logutil.DeferWithError(r.logger, "run", &retErr, zap.Int("num_files", len(rootFilePaths)))()

	if len(roots) == 0 {
		return nil, nil, errs.NewInvalidArgument("no roots specified")
	}
	if len(rootFilePaths) == 0 {
		return nil, nil, errs.NewInvalidArgument("no input files specified")
	}
	if uniqueLen := len(stringutil.SliceToMap(rootFilePaths)); uniqueLen != len(rootFilePaths) {
		// this is a system error, we should have verified this elsewhere
		return nil, nil, errs.NewInternal("rootFilePaths has duplicate values")
	}

	results := r.parse(
		ctx,
		bucket,
		roots,
		rootFilePaths,
		includeImports,
		includeSourceInfo,
	)

	var resultErr error
	for _, result := range results {
		resultErr = errs.Append(resultErr, result.Err)
	}
	if resultErr != nil {
		return nil, nil, resultErr
	}
	var annotations []*analysis.Annotation
	for _, result := range results {
		annotations = append(annotations, result.Annotations...)
	}
	if len(annotations) > 0 {
		analysis.SortAnnotations(annotations)
		return nil, annotations, nil
	}

	var descFileDescriptors []*desc.FileDescriptor
	for _, result := range results {
		iRootFilePaths := result.RootFilePaths
		iDescFileDescriptors := result.DescFileDescriptors
		// do a rough verification that rootFilePaths <-> fileDescriptors
		// parser.ParseFiles is documented to return the same number of FileDescriptors
		// as the number of input files
		// https://godoc.org/github.com/jhump/protoreflect/desc/protoparse#Parser.ParseFiles
		if len(iDescFileDescriptors) != len(iRootFilePaths) {
			return nil, nil, errs.NewInternalf("expected FileDescriptors to be of length %d but was %d", len(iRootFilePaths), len(iDescFileDescriptors))
		}
		for i, iDescFileDescriptor := range iDescFileDescriptors {
			iRootFilePath := iRootFilePaths[i]
			iFilename := iDescFileDescriptor.GetName()
			// doing another rough verification
			// NO LONGER NEED TO DO SUFFIX SINCE WE KNOW THE ROOT FILE NAME
			//if !strings.HasSuffix(iRootFilePath, iFilename) {
			if iRootFilePath != iFilename {
				return nil, nil, errs.NewInternalf("expected fileDescriptor name %s to be a equal to %s", iFilename, iRootFilePath)
			}
		}
		descFileDescriptors = append(descFileDescriptors, iDescFileDescriptors...)
	}

	image, err := getImage(descFileDescriptors, rootFilePaths, includeImports, includeSourceInfo)
	if err != nil {
		return nil, nil, err
	}
	return image, nil, nil
}

func (r *runner) parse(
	ctx context.Context,
	bucket storage.ReadBucket,
	roots []string,
	rootFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) []*result {
	defer logutil.Defer(r.logger, "parse", zap.Int("num_files", len(rootFilePaths)))()

	accessor := func(filename string) (io.ReadCloser, error) {
		return bucket.Get(ctx, filename)
	}
	var results []*result
	chunks := stringutil.SliceToChunks(rootFilePaths, len(rootFilePaths)/runtime.NumCPU())
	resultC := make(chan *result, len(chunks))
	for _, rootFilePaths := range chunks {
		rootFilePaths := rootFilePaths
		go func() {
			resultC <- r.getResult(
				ctx,
				bucket,
				accessor,
				roots,
				rootFilePaths,
				includeSourceInfo,
			)
		}()
	}
	for i := 0; i < len(chunks); i++ {
		results = append(results, <-resultC)
	}
	return results
}

func (r *runner) getResult(
	ctx context.Context,
	bucket storage.ReadBucket,
	accessor protoparse.FileAccessor,
	roots []string,
	rootFilePaths []string,
	includeSourceInfo bool,
) *result {
	// DO NOT NEED THIS ANYMORE
	// TODO: test ResolveFilenames in protofile against the output
	//filenames, err := protoparse.ResolveFilenames(roots, filePaths...)
	//if err != nil {
	//return newResult(filePaths, nil, nil, err)
	//}

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
				return newResult(rootFilePaths, nil, nil, errs.NewInternal("got invalid source error but no errors reported"))
			}
			annotations := make([]*analysis.Annotation, 0, len(errorsWithPos))
			for _, errorWithPos := range errorsWithPos {
				annotation, err := getAnnotation(errorWithPos)
				if err != nil {
					return newResult(rootFilePaths, nil, nil, err)
				}
				annotations = append(annotations, annotation)
			}
			return newResult(rootFilePaths, nil, annotations, nil)
		}
		return newResult(rootFilePaths, nil, nil, err)
	}
	return newResult(rootFilePaths, descFileDescriptors, nil, nil)
}

func getAnnotation(errorWithPos protoparse.ErrorWithPos) (*analysis.Annotation, error) {
	annotation := &analysis.Annotation{
		Type: "COMPILE",
	}
	// this should never happen
	// maybe we should error
	if errorWithPos.Unwrap() != nil {
		annotation.Message = errorWithPos.Unwrap().Error()
	} else {
		annotation.Message = "Compile error."
	}
	sourcePos := errorWithPos.GetPosition()
	if sourcePos.Filename != "" {
		annotation.Filename = sourcePos.Filename
	}
	if sourcePos.Line > 0 {
		annotation.StartLine = sourcePos.Line
		annotation.EndLine = sourcePos.Line
	}
	if sourcePos.Col > 0 {
		annotation.StartColumn = sourcePos.Col
		annotation.EndColumn = sourcePos.Col
	}
	return annotation, nil
}

// getImage gets the imagev1beta1.Image for the desc.FileDescriptor.
//
// This mimics protoc's output order.
//
// This sets all BufbuildExtension fields on the imagev1beta1.Image and imagev1beta1.Files.
func getImage(
	fileDescriptors []*desc.FileDescriptor,
	rootFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (bufpb.Image, error) {
	fileDescriptors, err := checkAndSortDescFileDescriptors(fileDescriptors, rootFilePaths)
	if err != nil {
		return nil, err
	}

	// if we aren't including imports, then we need a set of file names that
	// are included so we can create a topologically sorted list w/out
	// including imports that should not be present.
	//
	// if we are including imports, then we need to know what filenames
	// are imports are what filenames are not
	// all input desc.FileDescriptors are not imports, we derive the imports
	// from GetDependencies.
	nonImportFilenames := map[string]struct{}{}
	for _, fileDescriptor := range fileDescriptors {
		nonImportFilenames[fileDescriptor.GetName()] = struct{}{}
	}

	image := &imagev1beta1.Image{
		BufbuildImageExtension: &imagev1beta1.ImageExtension{
			ImageImportRefs: make([]*imagev1beta1.ImageImportRef, 0),
		},
	}
	alreadySeen := map[string]struct{}{}
	for _, fileDescriptor := range fileDescriptors {
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
	return bufpb.NewImage(image)
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
		return errs.NewInternal("nil FileDescriptor")
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
		return errs.NewInternal("nil File")
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
				FileIndex: protodescpb.Uint32(fileIndex),
			},
		)
	}
	return nil
}

// We need to sort the FileDescriptors as they may/probably are out of order
// relative to input order after concurrent builds. This mimics the output
// order of protoc.
func checkAndSortDescFileDescriptors(
	descFileDescriptors []*desc.FileDescriptor,
	rootFilePaths []string,
) ([]*desc.FileDescriptor, error) {
	if len(descFileDescriptors) != len(rootFilePaths) {
		return nil, errs.NewInternalf("rootFilePath length was %d but FileDescriptor length was %d", len(rootFilePaths), len(descFileDescriptors))
	}
	nameToDescFileDescriptor := make(map[string]*desc.FileDescriptor, len(descFileDescriptors))
	for _, descFileDescriptor := range descFileDescriptors {
		// This is equal to descFileDescriptor.AsFileDescriptorProto().GetName()
		// but we double-check just in case
		//
		// https://github.com/jhump/protoreflect/blob/master/desc/descriptor.go#L82
		name := descFileDescriptor.GetName()
		if name == "" {
			return nil, errs.NewInternal("no name on FileDescriptor")
		}
		if name != descFileDescriptor.AsFileDescriptorProto().GetName() {
			return nil, errs.NewInternal("name not equal on FileDescriptorProto")
		}
		if _, ok := nameToDescFileDescriptor[name]; ok {
			return nil, errs.NewInternalf("duplicate FileDescriptor: %s", name)
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
			return nil, errs.NewInternalf("no FileDescriptor for rootFilePath: %q", rootFilePath)
		}
		sortedDescFileDescriptors = append(sortedDescFileDescriptors, descFileDescriptor)
	}
	return sortedDescFileDescriptors, nil
}

type result struct {
	RootFilePaths       []string
	DescFileDescriptors []*desc.FileDescriptor
	Annotations         []*analysis.Annotation
	Err                 error
}

func newResult(
	rootFilePaths []string,
	descFileDescriptors []*desc.FileDescriptor,
	annotations []*analysis.Annotation,
	err error,
) *result {
	return &result{
		RootFilePaths:       rootFilePaths,
		DescFileDescriptors: descFileDescriptors,
		Annotations:         annotations,
		Err:                 err,
	}
}
