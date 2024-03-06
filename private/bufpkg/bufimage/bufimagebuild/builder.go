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

package bufimagebuild

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleprotocompile"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/data/datawkt"
	"github.com/bufbuild/buf/private/pkg/thread"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/reporter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	loggerName = "bufimagebuild"
	tracerName = "bufbuild/buf"
)

type builder struct {
	logger               *zap.Logger
	moduleFileSetBuilder bufmodulebuild.ModuleFileSetBuilder
	tracer               trace.Tracer
}

func newBuilder(logger *zap.Logger, moduleReader bufmodule.ModuleReader) *builder {
	return &builder{
		logger: logger.Named(loggerName),
		moduleFileSetBuilder: bufmodulebuild.NewModuleFileSetBuilder(
			logger,
			moduleReader,
		),
		tracer: otel.GetTracerProvider().Tracer(tracerName),
	}
}

func (b *builder) Build(
	ctx context.Context,
	module bufmodule.Module,
	options ...BuildOption,
) (bufimage.Image, []bufanalysis.FileAnnotation, error) {
	buildOptions := newBuildOptions()
	for _, option := range options {
		option(buildOptions)
	}
	return b.build(
		ctx,
		module,
		buildOptions.excludeSourceCodeInfo,
		buildOptions.expectedDirectDependencies,
		buildOptions.workspace,
	)
}

func (b *builder) build(
	ctx context.Context,
	module bufmodule.Module,
	excludeSourceCodeInfo bool,
	expectedDirectDeps []bufmoduleref.ModuleReference,
	workspace bufmodule.Workspace,
) (_ bufimage.Image, _ []bufanalysis.FileAnnotation, retErr error) {
	ctx, span := b.tracer.Start(ctx, "build")
	defer span.End()
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
	}()

	// TODO: remove this once bufmodule.ModuleFileSet is deleted or no longer inherits from Module
	// We still need to handle the ModuleFileSet case for buf export, as we actually need the
	// ModuleFileSet there.
	moduleFileSet, ok := module.(bufmodule.ModuleFileSet)
	if !ok {
		// If we just had a Module, convert it to a ModuleFileSet.
		var err error
		moduleFileSet, err = b.moduleFileSetBuilder.Build(
			ctx,
			module,
			bufmodulebuild.WithWorkspace(workspace),
		)
		if err != nil {
			return nil, nil, err
		}
	}

	parserAccessorHandler := bufmoduleprotocompile.NewParserAccessorHandler(ctx, moduleFileSet)
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

	buildResult := getBuildResult(
		ctx,
		parserAccessorHandler,
		paths,
		excludeSourceCodeInfo,
	)
	if buildResult.Err != nil {
		return nil, nil, buildResult.Err
	}
	if len(buildResult.FileAnnotations) > 0 {
		return nil, bufanalysis.DeduplicateAndSortFileAnnotations(buildResult.FileAnnotations), nil
	}

	fileDescriptors, err := checkAndSortFileDescriptors(buildResult.FileDescriptors, paths)
	if err != nil {
		return nil, nil, err
	}
	image, err := getImage(
		ctx,
		excludeSourceCodeInfo,
		fileDescriptors,
		parserAccessorHandler,
		buildResult.SyntaxUnspecifiedFilenames,
		buildResult.FilenameToUnusedDependencyFilenames,
		b.tracer,
	)
	if err != nil {
		return nil, nil, err
	}
	if err := b.warnInvalidImports(ctx, image, expectedDirectDeps, workspace); err != nil {
		b.logger.Error("warn_invalid_imports", zap.Error(err))
	}
	return image, nil, nil
}

// warnInvalidImports checks that all the target image files have valid imports statements that
// point to files in the local module, in a direct dependency, or in a workspace local unnamed
// module. It outputs WARN messages otherwise, one per invalid import statement.
//
// TODO: Understand this code before doing anything
// TODO: switch to use bufimage.ImageModuleDependencies
func (b *builder) warnInvalidImports(
	ctx context.Context,
	builtImage bufimage.Image,
	expectedDirectDeps []bufmoduleref.ModuleReference,
	localWorkspace bufmodule.Workspace,
) error {
	if expectedDirectDeps == nil && localWorkspace == nil {
		// Bail out early in case the caller didn't send explicitly send direct module dependencies nor
		// workspace. TODO: We should always send direct deps, so this imports warning can always
		// happen.
		return nil
	}
	expectedDirectDepsIdentities := make(map[string]struct{}, len(expectedDirectDeps))
	for _, expectedDirectDep := range expectedDirectDeps {
		expectedDirectDepsIdentities[expectedDirectDep.IdentityString()] = struct{}{}
	}
	workspaceIdentities := make(map[string]struct{})
	if localWorkspace != nil {
		wsModules := localWorkspace.GetModules()
		for _, mod := range wsModules {
			if mod == nil {
				b.logger.Debug("nil_module_in_workspace")
				continue
			}
			targetFileInfos, err := mod.TargetFileInfos(ctx)
			if err != nil {
				return fmt.Errorf("workspace module target file infos: %w", err)
			}
			for _, fileInfo := range targetFileInfos {
				if fileInfo.ModuleIdentity() != nil {
					workspaceIdentities[fileInfo.ModuleIdentity().IdentityString()] = struct{}{}
					break
				}
			}
		}
	}
	b.logger.Debug(
		"module_identities",
		zap.Any("from_direct_dependencies", expectedDirectDepsIdentities),
		zap.Any("from_workspace", workspaceIdentities),
	)

	// categorize image files into direct vs transitive dependencies
	allImgFiles := make(map[string]map[string][]string)        // for logging purposes only, modIdentity:filepath:imports
	targetFiles := make(map[string]struct{})                   // filepath
	expectedDirectDepsFilesToModule := make(map[string]string) // filepath:modIdentity
	workspaceFilesToModule := make(map[string]string)          // filepath:modIdentity
	transitiveDepsFilesToModule := make(map[string]string)     // filepath:modIdentity
	for _, imageFile := range builtImage.Files() {
		{ // populate allImgFiles
			modIdentity := "local"
			if imageFile.ModuleIdentity() != nil {
				modIdentity = imageFile.ModuleIdentity().IdentityString()
			}
			if _, ok := allImgFiles[modIdentity]; !ok {
				allImgFiles[modIdentity] = make(map[string][]string)
			}
			allImgFiles[modIdentity][imageFile.Path()] = imageFile.FileDescriptorProto().GetDependency()
		}
		if !imageFile.IsImport() {
			targetFiles[imageFile.Path()] = struct{}{}
			continue
		}
		if imageFile.ModuleIdentity() == nil {
			workspaceFilesToModule[imageFile.Path()] = "" // local workspace unnamed module
			continue
		}
		// file is import and comes from a named module. It's either a direct dep, a workspace module,
		// or a transitive dep.
		modIdentity := imageFile.ModuleIdentity().IdentityString()
		if _, ok := expectedDirectDepsIdentities[modIdentity]; ok {
			expectedDirectDepsFilesToModule[imageFile.Path()] = modIdentity
		} else if _, ok := workspaceIdentities[modIdentity]; ok {
			workspaceFilesToModule[imageFile.Path()] = modIdentity
		} else {
			transitiveDepsFilesToModule[imageFile.Path()] = modIdentity
		}
	}
	b.logger.Debug(
		"image_files",
		zap.Any("all_with_imports", allImgFiles),
		zap.Any("local_target", targetFiles),
		zap.Any("from_workspace", workspaceFilesToModule),
		zap.Any("from_expected_direct_dependencies", expectedDirectDepsFilesToModule),
		zap.Any("from_transitive_dependencies", transitiveDepsFilesToModule),
	)

	// validate import statements of target files against dependencies categorization above
	for _, imageFile := range builtImage.Files() {
		if imageFile.IsImport() {
			continue // only check import statements in local files
		}
		// .GetDependency() returns an array of file path imports in the file descriptor
		for _, importFilePath := range imageFile.FileDescriptorProto().GetDependency() {
			if _, ok := targetFiles[importFilePath]; ok {
				continue // import comes from local
			}
			if _, ok := expectedDirectDepsFilesToModule[importFilePath]; ok {
				continue // import comes from direct dep
			}
			if datawkt.Exists(importFilePath) {
				continue // wkt files are shipped with protoc, and we ship them in datawkt, so it's always safe to import them
			}
			warnMsg := fmt.Sprintf(
				"File %q imports %q, which is not found in your local files or direct dependencies",
				imageFile.Path(), importFilePath,
			)
			if workspaceModule, ok := workspaceFilesToModule[importFilePath]; ok {
				if workspaceModule == "" {
					// If dependency comes from an unnamed module, that is probably a local dependency, and
					// that module won't be pushed to the BSR. We can skip this warning.
					continue
				}
				// If dependency comes from a named module, we _could_ skip this warning as it _might_
				// fail when pushing trying to build, but we better keep it in case it is a transitive
				// dependency too, and both direct and transitive dependencies live in the same workspace.
				warnMsg += fmt.Sprintf(
					", but is found in local workspace module %q. Declare dependency %q in the deps key in buf.yaml.",
					workspaceModule,
					workspaceModule,
				)
			} else if transitiveDepModule, ok := transitiveDepsFilesToModule[importFilePath]; ok {
				warnMsg += fmt.Sprintf(
					", but is found in the transitive dependency %q. Declare dependency %q in the deps key in buf.yaml.",
					transitiveDepModule,
					transitiveDepModule,
				)
			} else {
				warnMsg += ", or any of your workspace modules or transitive dependencies. Check that external imports are declared as dependencies in the deps key in buf.yaml."
			}
			b.logger.Warn(warnMsg)
		}
	}
	return nil
}

func getBuildResult(
	ctx context.Context,
	parserAccessorHandler bufmoduleprotocompile.ParserAccessorHandler,
	paths []string,
	excludeSourceCodeInfo bool,
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
	compiler := protocompile.Compiler{
		MaxParallelism: thread.Parallelism(),
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
					nil,
					errors.New("got invalid source error from parse but no errors reported"),
				)
			}
			fileAnnotations, err := bufmoduleprotocompile.GetFileAnnotations(
				ctx,
				parserAccessorHandler,
				errorsWithPos,
			)
			if err != nil {
				return newBuildResult(nil, nil, nil, nil, err)
			}
			return newBuildResult(nil, nil, nil, fileAnnotations, nil)
		}
		if errorWithPos, ok := err.(reporter.ErrorWithPos); ok {
			fileAnnotations, err := bufmoduleprotocompile.GetFileAnnotations(
				ctx,
				parserAccessorHandler,
				[]reporter.ErrorWithPos{errorWithPos},
			)
			if err != nil {
				return newBuildResult(nil, nil, nil, nil, err)
			}
			return newBuildResult(nil, nil, nil, fileAnnotations, nil)
		}
		return newBuildResult(nil, nil, nil, nil, err)
	} else if len(errorsWithPos) > 0 {
		// https://github.com/jhump/protoreflect/pull/331
		return newBuildResult(
			nil,
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
	fileDescriptors := make([]protoreflect.FileDescriptor, len(compiledFiles))
	for i := range compiledFiles {
		fileDescriptors[i] = compiledFiles[i]
	}
	return newBuildResult(
		fileDescriptors,
		syntaxUnspecifiedFilenames,
		filenameToUnusedDependencyFilenames,
		nil,
		nil,
	)
}

// We need to sort the FileDescriptors as they may/probably are out of order
// relative to input order after concurrent builds. This mimics the output
// order of protoc.
func checkAndSortFileDescriptors(
	fileDescriptors []protoreflect.FileDescriptor,
	rootRelFilePaths []string,
) ([]protoreflect.FileDescriptor, error) {
	if len(fileDescriptors) != len(rootRelFilePaths) {
		return nil, fmt.Errorf("rootRelFilePath length was %d but FileDescriptor length was %d", len(rootRelFilePaths), len(fileDescriptors))
	}
	nameToFileDescriptor := make(map[string]protoreflect.FileDescriptor, len(fileDescriptors))
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
	sortedFileDescriptors := make([]protoreflect.FileDescriptor, 0, len(fileDescriptors))
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
	sortedFileDescriptors []protoreflect.FileDescriptor,
	parserAccessorHandler bufmoduleprotocompile.ParserAccessorHandler,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
	tracer trace.Tracer,
) (bufimage.Image, error) {
	ctx, span := tracer.Start(ctx, "get_image")
	defer span.End()

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

	var imageFiles []bufimage.ImageFile
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
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
	}
	image, err := bufimage.NewImage(imageFiles)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return image, err
}

func getImageFilesRec(
	ctx context.Context,
	excludeSourceCodeInfo bool,
	fileDescriptor protoreflect.FileDescriptor,
	parserAccessorHandler bufmoduleprotocompile.ParserAccessorHandler,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
	alreadySeen map[string]struct{},
	nonImportFilenames map[string]struct{},
	imageFiles []bufimage.ImageFile,
) ([]bufimage.ImageFile, error) {
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
	imageFile, err := bufimage.NewImageFile(
		fileDescriptorProto,
		parserAccessorHandler.ModuleIdentity(path),
		parserAccessorHandler.Commit(path),
		// if empty, defaults to path
		parserAccessorHandler.ExternalPath(path),
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
	FileDescriptors                     []protoreflect.FileDescriptor
	SyntaxUnspecifiedFilenames          map[string]struct{}
	FilenameToUnusedDependencyFilenames map[string]map[string]struct{}
	FileAnnotations                     []bufanalysis.FileAnnotation
	Err                                 error
}

func newBuildResult(
	fileDescriptors []protoreflect.FileDescriptor,
	syntaxUnspecifiedFilenames map[string]struct{},
	filenameToUnusedDependencyFilenames map[string]map[string]struct{},
	fileAnnotations []bufanalysis.FileAnnotation,
	err error,
) *buildResult {
	return &buildResult{
		FileDescriptors:                     fileDescriptors,
		SyntaxUnspecifiedFilenames:          syntaxUnspecifiedFilenames,
		FilenameToUnusedDependencyFilenames: filenameToUnusedDependencyFilenames,
		FileAnnotations:                     fileAnnotations,
		Err:                                 err,
	}
}

type buildOptions struct {
	excludeSourceCodeInfo      bool
	expectedDirectDependencies []bufmoduleref.ModuleReference
	workspace                  bufmodule.Workspace
}

func newBuildOptions() *buildOptions {
	return &buildOptions{}
}
