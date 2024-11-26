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

package buflsp

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/slogext"
	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/google/uuid"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// fileOpener is a function that opens files as they are named in the import
// statements of .proto files.
//
// This is the context given to [buildImage] to control what text to look up for
// specific files, so that we can e.g. use file contents that are still unsaved
// in the editor, or use files from a different commit for building an --against
// image.
type fileOpener func(string) (io.ReadCloser, error)

// buildImage builds a Buf Image for the given path. This does not use the controller to build
// the image, because we need delicate control over the input files: namely, for the case
// when we depend on a file that has been opened and modified in the editor.
func buildImage(
	ctx context.Context,
	path string,
	logger *slog.Logger,
	opener fileOpener,
) (bufimage.Image, []protocol.Diagnostic) {
	var report report
	var symbols linker.Symbols
	compiler := protocompile.Compiler{
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
		Resolver:       &protocompile.SourceResolver{Accessor: opener},
		Symbols:        &symbols,
		Reporter:       &report,
	}

	compiled, err := compiler.Compile(ctx, path)
	if err != nil {
		logger.Warn("error building image", slog.String("path", path), slogext.ErrorAttr(err))
	}
	if compiled[0] == nil {
		return nil, report.diagnostics
	}

	var imageFiles []bufimage.ImageFile
	seen := map[string]bool{}

	queue := []protoreflect.FileDescriptor{compiled[0]}
	for len(queue) > 0 {
		descriptor := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		if seen[descriptor.Path()] {
			continue
		}
		seen[descriptor.Path()] = true

		unused, ok := report.pathToUnusedImports[descriptor.Path()]
		var unusedIndices []int32
		if ok {
			unusedIndices = make([]int32, 0, len(unused))
		}

		imports := descriptor.Imports()
		for i := 0; i < imports.Len(); i++ {
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
			report.syntaxMissing[descriptor.Path()],
			unusedIndices,
		)
		if err != nil {
			break
		}

		imageFiles = append(imageFiles, imageFile)
		logger.Debug(fmt.Sprintf("added image file for %s", descriptor.Path()))
	}

	if err != nil {
		logger.Warn("could not build image", slog.String("path", path), slogext.ErrorAttr(err))
		return nil, report.diagnostics
	}

	image, err := bufimage.NewImage(imageFiles)
	if err != nil {
		logger.Warn("could not build image", slog.String("path", path), slogext.ErrorAttr(err))
		return nil, report.diagnostics
	}

	return image, report.diagnostics
}
