// Copyright 2020-2021 Buf Technologies, Inc.
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

package protodescriptor

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ValidateFileDescriptorProto validates the FileDescriptorProto.
func ValidateFileDescriptorProto(fileDescriptorProto *descriptorpb.FileDescriptorProto) error {
	if fileDescriptorProto == nil {
		return errors.New("nil FileDescriptorProto")
	}
	if err := ValidateProtoPath("FileDescriptorProto.Name", fileDescriptorProto.GetName()); err != nil {
		return err
	}
	if err := ValidateProtoPaths("FileDescriptorProto.Dependency", fileDescriptorProto.GetDependency()); err != nil {
		return err
	}
	return nil
}

// ValidateFileDescriptorProtos validates the FileDescriptorProtos.
func ValidateFileDescriptorProtos(fileDescriptorProtos []*descriptorpb.FileDescriptorProto) error {
	for _, fileDescriptorProto := range fileDescriptorProtos {
		if err := ValidateFileDescriptorProto(fileDescriptorProto); err != nil {
			return err
		}
	}
	return nil
}

// ValidateCodeGeneratorRequest validates the CodeGeneratorRequest.
func ValidateCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest) error {
	if err := ValidateCodeGeneratorRequestExceptFileDescriptorProtos(request); err != nil {
		return err
	}
	return ValidateFileDescriptorProtos(request.ProtoFile)
}

// ValidateCodeGeneratorRequestExceptFileDescriptorProtos validates the CodeGeneratorRequest
// minus the FileDescriptorProtos.
func ValidateCodeGeneratorRequestExceptFileDescriptorProtos(request *pluginpb.CodeGeneratorRequest) error {
	if request == nil {
		return errors.New("nil CodeGeneratorRequest")
	}
	if len(request.ProtoFile) == 0 {
		return errors.New("empty CodeGeneratorRequest.ProtoFile")
	}
	if len(request.FileToGenerate) == 0 {
		return errors.New("empty CodeGeneratorRequest.FileToGenerate")
	}
	if err := ValidateProtoPaths("CodeGeneratorRequest.FileToGenerate", request.FileToGenerate); err != nil {
		return err
	}
	return nil
}

// ValidateCodeGeneratorResponse validates the CodeGeneratorResponse.
//
// This validates that names are set.
//
// It is actually OK per the plugin.proto specs to not have the name set, and
// if this is empty, the content should be combined with the previous file.
// However, for our handlers, we do not support this, and for our
// binary handlers, we combine CodeGeneratorResponse.File contents.
//
// https://github.com/protocolbuffers/protobuf/blob/b99994d994e399174fe688a5efbcb6d91f36952a/src/google/protobuf/compiler/plugin.proto#L127
func ValidateCodeGeneratorResponse(response *pluginpb.CodeGeneratorResponse) error {
	if response == nil {
		return errors.New("nil CodeGeneratorResponse")
	}
	for _, file := range response.File {
		if file.GetName() == "" {
			return errors.New("empty CodeGeneratorResponse.File.Name")
		}
	}
	return nil
}

// ValidateProtoPath validates the proto path.
//
// This checks that the path is normalized and ends in .proto.
func ValidateProtoPath(name string, path string) error {
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
	if normalpath.Ext(path) != ".proto" {
		return fmt.Errorf("%s %s does not have a .proto extension", name, path)
	}
	return nil
}

// ValidateProtoPaths validates the proto paths.
//
// This checks that the paths are normalized and end in .proto.
func ValidateProtoPaths(name string, paths []string) error {
	for _, path := range paths {
		if err := ValidateProtoPath(name, path); err != nil {
			return err
		}
	}
	return nil
}
