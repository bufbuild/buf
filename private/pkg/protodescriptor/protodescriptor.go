// Copyright 2020-2023 Buf Technologies, Inc.
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
	"strconv"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

// FileDescriptor is an interface that matches the methods on a *descriptorpb.FileDescriptorProto.
//
// Note that a FileDescriptor is not necessarily validated, unlike other interfaces in buf.
type FileDescriptor interface {
	proto.Message
	GetName() string
	GetPackage() string
	GetDependency() []string
	GetPublicDependency() []int32
	GetWeakDependency() []int32
	GetMessageType() []*descriptorpb.DescriptorProto
	GetEnumType() []*descriptorpb.EnumDescriptorProto
	GetService() []*descriptorpb.ServiceDescriptorProto
	GetExtension() []*descriptorpb.FieldDescriptorProto
	GetOptions() *descriptorpb.FileOptions
	GetSourceCodeInfo() *descriptorpb.SourceCodeInfo
	GetSyntax() string
	GetEdition() string
}

// FileDescriptorsForFileDescriptorProtos is a convenience function since Go does not have generics.
func FileDescriptorsForFileDescriptorProtos(fileDescriptorProtos ...*descriptorpb.FileDescriptorProto) []FileDescriptor {
	fileDescriptors := make([]FileDescriptor, len(fileDescriptorProtos))
	for i, fileDescriptorProto := range fileDescriptorProtos {
		fileDescriptors[i] = fileDescriptorProto
	}
	return fileDescriptors
}

// FileDescriptorsForFileDescriptorSet is a convenience function since Go does not have generics.
func FileDescriptorsForFileDescriptorSet(fileDescriptorSet *descriptorpb.FileDescriptorSet) []FileDescriptor {
	return FileDescriptorsForFileDescriptorProtos(fileDescriptorSet.File...)
}

// FileDescriptorProtoForFileDescriptor creates a new *descriptorpb.FileDescriptorProto for the fileDescriptor.
//
// If the FileDescriptor is already a *descriptorpb.FileDescriptorProto, this returns the input value.
//
// Note that this will not round trip exactly. If a *descriptorpb.FileDescriptorProto is turned into another
// object that is a FileDescriptor, and then passed to this function, the return value will not be equal
// if name, package, or syntax are set but empty. Instead, the return value will have these values unset.
// For our/most purposes, this is fine.
func FileDescriptorProtoForFileDescriptor(fileDescriptor FileDescriptor) *descriptorpb.FileDescriptorProto {
	if fileDescriptorProto, ok := fileDescriptor.(*descriptorpb.FileDescriptorProto); ok {
		return fileDescriptorProto
	}
	fileDescriptorProto := &descriptorpb.FileDescriptorProto{
		Dependency:       fileDescriptor.GetDependency(),
		PublicDependency: fileDescriptor.GetPublicDependency(),
		WeakDependency:   fileDescriptor.GetWeakDependency(),
		MessageType:      fileDescriptor.GetMessageType(),
		EnumType:         fileDescriptor.GetEnumType(),
		Service:          fileDescriptor.GetService(),
		Extension:        fileDescriptor.GetExtension(),
		Options:          fileDescriptor.GetOptions(),
		SourceCodeInfo:   fileDescriptor.GetSourceCodeInfo(),
	}
	// Note that if a *descriptorpb.FileDescriptorProto has a set but empty name, package,
	// or syntax, this won't be an exact round trip. But for our use, we say this is fine.
	if name := fileDescriptor.GetName(); name != "" {
		fileDescriptorProto.Name = proto.String(name)
	}
	if pkg := fileDescriptor.GetPackage(); pkg != "" {
		fileDescriptorProto.Package = proto.String(pkg)
	}
	if syntax := fileDescriptor.GetSyntax(); syntax != "" {
		fileDescriptorProto.Syntax = proto.String(syntax)
	}
	if edition := fileDescriptor.GetEdition(); edition != "" {
		fileDescriptorProto.Edition = proto.String(edition)
	}
	fileDescriptorProto.ProtoReflect().SetUnknown(fileDescriptor.ProtoReflect().GetUnknown())
	return fileDescriptorProto
}

// FileDescriptorProtosForFileDescriptors is a convenience function since Go does not have generics.
//
// Note that this will not round trip exactly. If a *descriptorpb.FileDescriptorProto is turned into another
// object that is a FileDescriptor, and then passed to this function, the return value will not be equal
// if name, package, or syntax are set but empty. Instead, the return value will have these values unset.
// For our/most purposes, this is fine.
func FileDescriptorProtosForFileDescriptors(fileDescriptors ...FileDescriptor) []*descriptorpb.FileDescriptorProto {
	fileDescriptorProtos := make([]*descriptorpb.FileDescriptorProto, len(fileDescriptors))
	for i, fileDescriptor := range fileDescriptors {
		fileDescriptorProtos[i] = FileDescriptorProtoForFileDescriptor(fileDescriptor)
	}
	return fileDescriptorProtos
}

// FileDescriptorSetForFileDescriptors returns a new *descriptorpb.FileDescriptorSet for the given FileDescriptors.
//
// Note that this will not round trip exactly. If a *descriptorpb.FileDescriptorProto is turned into another
// object that is a FileDescriptor, and then passed to this function, the return value will not be equal
// if name, package, or syntax are set but empty. Instead, the return value will have these values unset.
// For our/most purposes, this is fine.
func FileDescriptorSetForFileDescriptors(fileDescriptors ...FileDescriptor) *descriptorpb.FileDescriptorSet {
	return &descriptorpb.FileDescriptorSet{
		File: FileDescriptorProtosForFileDescriptors(fileDescriptors...),
	}
}

// ValidateFileDescriptor validates the FileDescriptor.
//
// A *descriptorpb.FileDescriptorProto can be passed to this.
func ValidateFileDescriptor(fileDescriptor FileDescriptor) error {
	if fileDescriptor == nil {
		return errors.New("nil FileDescriptor")
	}
	if err := ValidateProtoPath("FileDescriptor.Name", fileDescriptor.GetName()); err != nil {
		return err
	}
	if err := ValidateProtoPaths("FileDescriptor.Dependency", fileDescriptor.GetDependency()); err != nil {
		return err
	}
	return nil
}

// ValidateCodeGeneratorRequest validates the CodeGeneratorRequest.
func ValidateCodeGeneratorRequest(request *pluginpb.CodeGeneratorRequest) error {
	if err := ValidateCodeGeneratorRequestExceptFileDescriptorProtos(request); err != nil {
		return err
	}
	for _, fileDescriptorProto := range request.ProtoFile {
		if err := ValidateFileDescriptor(fileDescriptorProto); err != nil {
			return err
		}
	}
	return nil
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

// FieldDescriptorProtoTypePrettyString prints a pretty string
// representation of the FieldDescriptorProto_Type.
func FieldDescriptorProtoTypePrettyString(t descriptorpb.FieldDescriptorProto_Type) string {
	switch t {
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		return "double"
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		return "float"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64:
		return "int64"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		return "uint64"
	case descriptorpb.FieldDescriptorProto_TYPE_INT32:
		return "int32"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		return "fixed64"
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		return "fixed32"
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		return "bool"
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		return "string"
	case descriptorpb.FieldDescriptorProto_TYPE_GROUP:
		return "group"
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		return "message"
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		return "bytes"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		return "uint32"
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		return "enum"
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		return "sfixed32"
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return "sfixed64"
	case descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		return "sint32"
	case descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		return "sint64"
	default:
		return strconv.Itoa(int(t))
	}
}

// FieldDescriptorProtoLabelPrettyString prints a pretty string
// representation of the FieldDescriptorProto_Label.
func FieldDescriptorProtoLabelPrettyString(l descriptorpb.FieldDescriptorProto_Label) string {
	switch l {
	case descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL:
		return "optional"
	case descriptorpb.FieldDescriptorProto_LABEL_REQUIRED:
		return "required"
	case descriptorpb.FieldDescriptorProto_LABEL_REPEATED:
		return "repeated"
	default:
		return strconv.Itoa(int(l))
	}
}
