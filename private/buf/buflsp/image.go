// Copyright 2020-2025 Buf Technologies, Inc.
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

package buflsp

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"

	"buf.build/go/standard/xlog/xslog"
	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/google/uuid"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// fileOpener is a type that opens files as they are named in the import
// statements of .proto files.
//
// This is the context given to [buildImage] to control what text to look up for
// specific files, so that we can e.g. use file contents that are still unsaved
// in the editor, or use files from a different commit for building an --against
// image.
type fileOpener map[string]string

func (p fileOpener) Open(path string) (io.ReadCloser, error) {
	text, ok := p[path]
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return io.NopCloser(strings.NewReader(text)), nil
}

// buildImage builds a Buf Image for the given path. This does not use the controller to build
// the image, because we need delicate control over the input files: namely, for the case
// when we depend on a file that has been opened and modified in the editor.
func buildImage(
	ctx context.Context,
	path string,
	logger *slog.Logger,
	opener fileOpener,
) (bufimage.Image, []protocol.Diagnostic) {
	var errorsWithPos []reporter.ErrorWithPos
	var warningErrorsWithPos []reporter.ErrorWithPos
	var symbols linker.Symbols
	compiler := protocompile.Compiler{
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
		Resolver:       &protocompile.SourceResolver{Accessor: opener.Open},
		Symbols:        &symbols,
		Reporter: reporter.NewReporter(
			func(errorWithPos reporter.ErrorWithPos) error {
				errorsWithPos = append(errorsWithPos, errorWithPos)
				return nil
			},
			func(warningErrorWithPos reporter.ErrorWithPos) {
				warningErrorsWithPos = append(warningErrorsWithPos, warningErrorWithPos)
			},
		),
	}

	var diagnostics []protocol.Diagnostic
	compiled, err := compiler.Compile(ctx, path)
	if err != nil {
		logger.Warn("error building image", slog.String("path", path), xslog.ErrorAttr(err))
		if len(errorsWithPos) > 0 {
			diagnostics = xslices.Map(errorsWithPos, func(errorWithPos reporter.ErrorWithPos) protocol.Diagnostic {
				return newDiagnostic(errorWithPos, false, opener, logger)
			})
		}
	}
	if len(compiled) == 0 || compiled[0] == nil {
		return nil, nil // Image failed to build.
	}
	compiledFile := compiled[0]

	syntaxMissing := make(map[string]bool)
	pathToUnusedImports := make(map[string]map[string]bool)
	for _, warningErrorWithPos := range warningErrorsWithPos {
		if warningErrorWithPos.Unwrap() == parser.ErrNoSyntax {
			syntaxMissing[warningErrorWithPos.GetPosition().Filename] = true
		} else if unusedImport, ok := warningErrorWithPos.Unwrap().(linker.ErrorUnusedImport); ok {
			path := warningErrorWithPos.GetPosition().Filename
			unused, ok := pathToUnusedImports[path]
			if !ok {
				unused = map[string]bool{}
				pathToUnusedImports[path] = unused
			}
			unused[unusedImport.UnusedImport()] = true
		}
	}

	var imageFiles []bufimage.ImageFile
	seen := map[string]bool{}

	queue := []protoreflect.FileDescriptor{compiledFile}
	for len(queue) > 0 {
		descriptor := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		if seen[descriptor.Path()] {
			continue
		}
		seen[descriptor.Path()] = true

		unused, ok := pathToUnusedImports[descriptor.Path()]
		var unusedIndices []int32
		if ok {
			unusedIndices = make([]int32, 0, len(unused))
		}

		imports := descriptor.Imports()
		for i := range imports.Len() {
			dep := imports.Get(i).FileDescriptor
			if dep == nil {
				logger.Warn(fmt.Sprintf("found nil FileDescriptor for import %s", imports.Get(i).Path()))
				continue
			}

			queue = append(queue, dep)

			if unused != nil {
				if _, ok := unused[dep.Path()]; ok {
					unusedIndices = append(unusedIndices, int32(i))
				}
			}
		}

		descriptorProto := protoutil.ProtoFromFileDescriptor(descriptor)
		if descriptorProto == nil {
			err = fmt.Errorf("protoutil.ProtoFromFileDescriptor() returned nil for %q", descriptor.Path())
			break
		}

		var imageFile bufimage.ImageFile
		imageFile, err = bufimage.NewImageFile(
			descriptorProto,
			nil,
			uuid.UUID{},
			"",
			descriptor.Path(),
			descriptor.Path() != path,
			syntaxMissing[descriptor.Path()],
			unusedIndices,
		)
		if err != nil {
			break
		}

		imageFiles = append(imageFiles, imageFile)
		logger.Debug(fmt.Sprintf("added image file for %s", descriptor.Path()))
	}

	if err != nil {
		logger.Warn("could not build image", slog.String("path", path), xslog.ErrorAttr(err))
		return nil, diagnostics
	}

	image, err := bufimage.NewImage(imageFiles)
	if err != nil {
		logger.Warn("could not build image", slog.String("path", path), xslog.ErrorAttr(err))
		return nil, diagnostics
	}

	return image, diagnostics
}

// newDiagnostic converts a protocompile error into a diagnostic.
//
// Unfortunately, protocompile's errors are currently too meagre to provide full code
// spans; that will require a fix in the compiler.
func newDiagnostic(err reporter.ErrorWithPos, isWarning bool, opener fileOpener, logger *slog.Logger) protocol.Diagnostic {
	// Read the file text for the error's filename to convert byte offset to UTF-16 column.
	position := err.GetPosition()
	filename := position.Filename
	// Fallback to byte-based column (will be wrong for non-ASCII).
	utf16Col := err.GetPosition().Col - 1

	// TODO: this is a temporary workaround for old diagnostic errors.
	// When using the new compiler these conversions will be already handled.
	if text, ok := opener[filename]; ok {
		file := source.NewFile(filename, text)
		loc := file.Location(position.Offset, positionalEncoding)
		utf16Col = loc.Column - 1
	} else {
		logger.Warn(
			"failed to open file for diagnostic position encoding",
			slog.String("filename", filename),
		)
	}

	pos := protocol.Position{
		Line:      uint32(err.GetPosition().Line - 1),
		Character: uint32(utf16Col),
	}

	severity := protocol.DiagnosticSeverityError
	if isWarning {
		severity = protocol.DiagnosticSeverityWarning
	}

	return protocol.Diagnostic{
		// TODO: The compiler currently does not record spans for diagnostics. This is
		// essentially a bug that will result in worse diagnostics until fixed.
		Range:    protocol.Range{Start: pos, End: pos},
		Severity: severity,
		Message:  err.Unwrap().Error(),
		Source:   serverName,
	}
}
