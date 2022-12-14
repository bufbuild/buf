// Copyright 2020-2022 Buf Technologies, Inc.
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FileDescriptor is an interface that matches the methods on a *descriptorpb.FileDescriptorProto.
//
// Note that a FileDescriptor is not necessarily validated, unlike other interfaces in buf.
type FileDescriptor interface {
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
}

// FileDescriptorsForFileDescriptorSet is a convenience function since Go does not have generics.
func FileDescriptorsForFileDescriptorSet(fileDescriptorSet *descriptorpb.FileDescriptorSet) []FileDescriptor {
	return FileDescriptorsForFileDescriptorProtos(fileDescriptorSet.File...)
}

// FileDescriptorsForFileDescriptorProtos is a convenience function since Go does not have generics.
func FileDescriptorsForFileDescriptorProtos(fileDescriptorProtos ...*descriptorpb.FileDescriptorProto) []FileDescriptor {
	fileDescriptors := make([]FileDescriptor, len(fileDescriptorProtos))
	for i, fileDescriptorProto := range fileDescriptorProtos {
		fileDescriptors[i] = fileDescriptorProto
	}
	return fileDescriptors
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
	return fileDescriptorProto
}
