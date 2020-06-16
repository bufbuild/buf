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

package bufimage

import (
	"errors"
	"fmt"

	imagev1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1"
	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func validateRootRelFilePath(rootRelFilePath string) error {
	return validatePath("root relative file path", rootRelFilePath)
}

func validateRootRelFilePaths(rootRelFilePaths []string) error {
	return validatePaths("root relative file path", rootRelFilePaths)
}

func validateRootDirPath(rootDirPath string) error {
	return validatePath("root directory path", rootDirPath)
}

func validateFileDescriptorProto(fileDescriptorProto *descriptorpb.FileDescriptorProto) error {
	if fileDescriptorProto == nil {
		return errors.New("nil FileDescriptorProto")
	}
	// this is so we print out a different error message than the error in newFileRef
	if err := validatePath("FileDescriptorProto name", fileDescriptorProto.GetName()); err != nil {
		return err
	}
	if err := validatePaths("FileDescriptorProto dependency", fileDescriptorProto.GetDependency()); err != nil {
		return err
	}
	return nil
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

func validateCodeGeneratorRequestExceptFileDescriptorProtos(request *pluginpb.CodeGeneratorRequest) error {
	if request == nil {
		return errors.New("nil CodeGeneratorRequest")
	}
	if len(request.ProtoFile) == 0 {
		return errors.New("empty CodeGeneratorRequest ProtoFiles")
	}
	if err := validatePaths("file to generate", request.FileToGenerate); err != nil {
		return err
	}
	return nil
}

func validatePaths(name string, paths []string) error {
	for _, path := range paths {
		if err := validatePath(name, path); err != nil {
			return err
		}
	}
	return nil
}

func validatePath(name string, path string) error {
	if path == "" {
		return fmt.Errorf("%s is empty", name)
	}
	normalized, err := normalpath.NormalizeAndValidate(path)
	if err != nil {
		return fmt.Errorf("%s had normalization error: %w", name, err)
	}
	if path != normalized {
		return fmt.Errorf("%s %s was not normalized to %s", name, path, normalized)
	}
	return nil
}
