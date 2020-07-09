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

package bufcore

import (
	"errors"
	"fmt"

	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
	"github.com/bufbuild/buf/internal/pkg/protodescriptor"
)

func validateFileInfoPath(path string) error {
	return protodescriptor.ValidateProtoPath("root relative file path", path)
}

func validateFileInfoPaths(paths []string) error {
	return protodescriptor.ValidateProtoPaths("root relative file path", paths)
}

// we validate the FileDescriptorProtos as part of NewFile
func validateProtoImageExceptFileDescriptorProtos(protoImage *imagev1.Image) error {
	if protoImage == nil {
		return errors.New("nil Image")
	}
	if len(protoImage.File) == 0 {
		return errors.New("empty Image Files")
	}
	if protoImage.BufbuildImageExtension != nil {
		return validateProtoImageExtension(protoImage.BufbuildImageExtension, uint32(len(protoImage.File)))
	}
	return nil
}

func validateProtoImageExtension(
	protoImageExtension *imagev1.ImageExtension,
	numFiles uint32,
) error {
	seenFileIndexes := make(map[uint32]struct{}, len(protoImageExtension.ImageImportRefs))
	for _, imageImportRef := range protoImageExtension.ImageImportRefs {
		if imageImportRef == nil {
			return errors.New("nil ImageImportRef")
		}
		if imageImportRef.FileIndex == nil {
			return errors.New("nil ImageImportRef.FileIndex")
		}
		fileIndex := *imageImportRef.FileIndex
		if fileIndex >= numFiles {
			return fmt.Errorf("invalid file index: %d", fileIndex)
		}
		if _, ok := seenFileIndexes[fileIndex]; ok {
			return fmt.Errorf("duplicate file index: %d", fileIndex)
		}
		seenFileIndexes[fileIndex] = struct{}{}
	}
	return nil
}
