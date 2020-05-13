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

package protodescriptorutil

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/internal/pkg/normalpath"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// ValidateFileDescriptorSet validates a FileDescriptorSet.
func ValidateFileDescriptorSet(fileDescriptorSet *descriptorpb.FileDescriptorSet) error {
	if fileDescriptorSet == nil {
		return errors.New("validate error: nil FileDescriptorSet")
	}
	return ValidateFileDescriptorProtos(fileDescriptorSet.File)
}

// ValidateFileDescriptorProtos validates multiple FileDescriptorProtos.
func ValidateFileDescriptorProtos(fileDescriptorProtos []*descriptorpb.FileDescriptorProto) error {
	if len(fileDescriptorProtos) == 0 {
		return errors.New("validate error: empty FileDescriptorProtos")
	}
	seenNames := make(map[string]struct{}, len(fileDescriptorProtos))
	for _, fileDescriptorProto := range fileDescriptorProtos {
		if err := ValidateFileDescriptorProto(fileDescriptorProto); err != nil {
			return err
		}
		name := fileDescriptorProto.GetName()
		if _, ok := seenNames[name]; ok {
			return fmt.Errorf("validate error: duplicate FileDescriptorProto.Name: %q", name)
		}
		seenNames[name] = struct{}{}
	}
	return nil
}

// ValidateFileDescriptorProto validates a FileDescriptorProto.
func ValidateFileDescriptorProto(fileDescriptorProto *descriptorpb.FileDescriptorProto) error {
	if fileDescriptorProto == nil {
		return errors.New("validate error: nil FileDescriptorProto")
	}
	if fileDescriptorProto.Name == nil {
		return errors.New("validate error: nil FileDescriptorProto.Name")
	}
	name := fileDescriptorProto.GetName()
	if name == "" {
		return errors.New("validate error: empty FileDescriptorProto.Name")
	}
	normalizedName, err := normalpath.NormalizeAndValidate(name)
	if err != nil {
		return fmt.Errorf("validate error: %v", err)
	}
	if name != normalizedName {
		return fmt.Errorf("validate error: FileDescriptorProto.Name %q has normalized name %q", name, normalizedName)
	}
	return nil
}

// ValidateCodeGeneratorRequest validates the CodeGeneratorReqquest.
func ValidateCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest) error {
	if request == nil {
		return errors.New("validate error: nil CodeGeneratorRequest")
	}
	if err := ValidateFileDescriptorProtos(request.ProtoFile); err != nil {
		return err
	}
	if len(request.FileToGenerate) == 0 {
		return fmt.Errorf("no file to generate")
	}
	normalizedFileToGenerate := make([]string, len(request.FileToGenerate))
	for i, elem := range request.FileToGenerate {
		normalized, err := normalpath.NormalizeAndValidate(elem)
		if err != nil {
			return err
		}
		normalizedFileToGenerate[i] = normalized
	}
	namesMap := make(map[string]struct{}, len(request.ProtoFile))
	for _, fileDescriptorProto := range request.ProtoFile {
		namesMap[fileDescriptorProto.GetName()] = struct{}{}
	}
	for _, normalized := range normalizedFileToGenerate {
		if _, ok := namesMap[normalized]; !ok {
			return fmt.Errorf("file to generate %q is not within the fileDescriptorSet", normalized)
		}
	}
	return nil
}
