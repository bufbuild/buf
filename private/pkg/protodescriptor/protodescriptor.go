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

package protodescriptor

import (
	"errors"
	"fmt"
	"sort"
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
	GetEdition() descriptorpb.Edition
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
	if edition := fileDescriptor.GetEdition(); edition != descriptorpb.Edition_EDITION_UNKNOWN {
		fileDescriptorProto.Edition = &edition
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
func FileDescriptorProtosForFileDescriptors[F FileDescriptor](fileDescriptors ...F) []*descriptorpb.FileDescriptorProto {
	fileDescriptorsAny := any(fileDescriptors) // must assign to interface var to do type assertion below
	if fileDescriptorProtos, ok := fileDescriptorsAny.([]*descriptorpb.FileDescriptorProto); ok {
		return fileDescriptorProtos
	}

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
func FileDescriptorSetForFileDescriptors[F FileDescriptor](fileDescriptors ...F) *descriptorpb.FileDescriptorSet {
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
	for i, fileDescriptorProto := range request.ProtoFile {
		if err := ValidateFileDescriptor(fileDescriptorProto); err != nil {
			return fmt.Errorf("CodeGeneratorRequest.ProtoFile[%d]: %w", i, err)
		}
	}
	for i, fileDescriptorProto := range request.SourceFileDescriptors {
		if err := ValidateFileDescriptor(fileDescriptorProto); err != nil {
			return fmt.Errorf("CodeGeneratorRequest.SourceFileDescriptors[%d]: %w", i, err)
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
	paths := make(map[string]*descriptorpb.FileDescriptorProto, len(request.ProtoFile))
	for _, fileDescriptorProto := range request.ProtoFile {
		path := fileDescriptorProto.GetName()
		if _, alreadyExists := paths[path]; alreadyExists {
			return fmt.Errorf("CodeGeneratorRequest.ProtoFile contains duplicate entry for %q", path)
		}
		paths[path] = fileDescriptorProto
	}
	alreadyChecked := make(map[string]struct{}, len(paths))
	if err := validatePathsPresent("file to generate", paths, request.FileToGenerate, alreadyChecked); err != nil {
		return err
	}
	filesToGenerate := make(map[string]struct{}, len(request.FileToGenerate))
	for _, path := range request.FileToGenerate {
		if _, alreadyExists := filesToGenerate[path]; alreadyExists {
			return fmt.Errorf("CodeGeneratorRequest.FileToGenerate contains duplicate entry for %q", path)
		}
		filesToGenerate[path] = struct{}{}
	}
	if len(request.SourceFileDescriptors) > 0 {
		// If request has SourceFileDescriptors, make sure they correctly
		// correspond to items in FileToGenerate. Sadly, these are not necessarily
		// in the same order as FileToGenerate, so checking they have the same
		// contents involves some set arithmetic...
		sourceFiles := make(map[string]struct{}, len(request.SourceFileDescriptors))
		for _, fileDescriptorProto := range request.SourceFileDescriptors {
			path := fileDescriptorProto.GetName()
			if _, alreadyExists := sourceFiles[path]; alreadyExists {
				return fmt.Errorf("CodeGeneratorRequest.SourceFileDescriptors contains duplicate entry for %q", path)
			}
			sourceFiles[path] = struct{}{}
		}
		for path := range filesToGenerate {
			if _, present := sourceFiles[path]; !present {
				return fmt.Errorf("CodeGeneratorRequest.SourceFileDescriptors is missing entry for file to generate %q", path)
			}
			delete(sourceFiles, path) // remove as we go so any remaining entries are extra
		}
		if len(sourceFiles) > 0 {
			// SourceFileDescriptors contained superfluous files!
			superfluousPaths := make([]string, 0, len(sourceFiles))
			for path := range sourceFiles {
				superfluousPaths = append(superfluousPaths, strconv.Quote(path))
			}
			sort.Strings(superfluousPaths)
			return fmt.Errorf("CodeGeneratorRequest.SourceFileDescriptors contains incorrect entries for files not being generated: %v", superfluousPaths)
		}
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

func validatePathsPresent(
	what string,
	present map[string]*descriptorpb.FileDescriptorProto,
	paths []string,
	alreadyChecked map[string]struct{},
) error {
	for _, path := range paths {
		if _, checked := alreadyChecked[path]; checked {
			continue
		}
		alreadyChecked[path] = struct{}{}

		fileDescriptorProto := present[path]
		if fileDescriptorProto == nil {
			return fmt.Errorf("CodeGeneratorRequest.ProtoFile is missing %s %q", what, path)
		}

		if err := validatePathsPresent("dependency", present, fileDescriptorProto.Dependency, alreadyChecked); err != nil {
			return err
		}
	}
	return nil
}
