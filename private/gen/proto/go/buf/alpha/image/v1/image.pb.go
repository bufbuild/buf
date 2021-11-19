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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        (unknown)
// source: buf/alpha/image/v1/image.proto

package imagev1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Image is an extended FileDescriptorSet.
//
// See https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto
type Image struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	File []*ImageFile `protobuf:"bytes,1,rep,name=file" json:"file,omitempty"`
}

func (x *Image) Reset() {
	*x = Image{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_image_v1_image_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Image) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Image) ProtoMessage() {}

func (x *Image) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Image.ProtoReflect.Descriptor instead.
func (*Image) Descriptor() ([]byte, []int) {
	return file_buf_alpha_image_v1_image_proto_rawDescGZIP(), []int{0}
}

func (x *Image) GetFile() []*ImageFile {
	if x != nil {
		return x.File
	}
	return nil
}

// ImageFile is an extended FileDescriptorProto.
//
// Since FileDescriptorProto does not have extensions, we copy the fields from
// FileDescriptorProto, and then add our own extensions via the
// buf_image_file_extension field. This is compatible with a
// FileDescriptorProto.
//
// See https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto
type ImageFile struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name             *string                                `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Package          *string                                `protobuf:"bytes,2,opt,name=package" json:"package,omitempty"`
	Dependency       []string                               `protobuf:"bytes,3,rep,name=dependency" json:"dependency,omitempty"`
	PublicDependency []int32                                `protobuf:"varint,10,rep,name=public_dependency,json=publicDependency" json:"public_dependency,omitempty"`
	WeakDependency   []int32                                `protobuf:"varint,11,rep,name=weak_dependency,json=weakDependency" json:"weak_dependency,omitempty"`
	MessageType      []*descriptorpb.DescriptorProto        `protobuf:"bytes,4,rep,name=message_type,json=messageType" json:"message_type,omitempty"`
	EnumType         []*descriptorpb.EnumDescriptorProto    `protobuf:"bytes,5,rep,name=enum_type,json=enumType" json:"enum_type,omitempty"`
	Service          []*descriptorpb.ServiceDescriptorProto `protobuf:"bytes,6,rep,name=service" json:"service,omitempty"`
	Extension        []*descriptorpb.FieldDescriptorProto   `protobuf:"bytes,7,rep,name=extension" json:"extension,omitempty"`
	Options          *descriptorpb.FileOptions              `protobuf:"bytes,8,opt,name=options" json:"options,omitempty"`
	SourceCodeInfo   *descriptorpb.SourceCodeInfo           `protobuf:"bytes,9,opt,name=source_code_info,json=sourceCodeInfo" json:"source_code_info,omitempty"`
	Syntax           *string                                `protobuf:"bytes,12,opt,name=syntax" json:"syntax,omitempty"`
	// buf_extension contains buf-specific extensions to FileDescriptorProtos.
	//
	// The prefixed name and high tag value is used to all but guarantee there
	// will never be any conflict with Google's FileDescriptorProto definition.
	// The definition of a FileDescriptorProto has not changed in years, so
	// we're not too worried about a conflict here.
	BufExtension *ImageFileExtension `protobuf:"bytes,8042,opt,name=buf_extension,json=bufExtension" json:"buf_extension,omitempty"`
}

func (x *ImageFile) Reset() {
	*x = ImageFile{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_image_v1_image_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ImageFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ImageFile) ProtoMessage() {}

func (x *ImageFile) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ImageFile.ProtoReflect.Descriptor instead.
func (*ImageFile) Descriptor() ([]byte, []int) {
	return file_buf_alpha_image_v1_image_proto_rawDescGZIP(), []int{1}
}

func (x *ImageFile) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *ImageFile) GetPackage() string {
	if x != nil && x.Package != nil {
		return *x.Package
	}
	return ""
}

func (x *ImageFile) GetDependency() []string {
	if x != nil {
		return x.Dependency
	}
	return nil
}

func (x *ImageFile) GetPublicDependency() []int32 {
	if x != nil {
		return x.PublicDependency
	}
	return nil
}

func (x *ImageFile) GetWeakDependency() []int32 {
	if x != nil {
		return x.WeakDependency
	}
	return nil
}

func (x *ImageFile) GetMessageType() []*descriptorpb.DescriptorProto {
	if x != nil {
		return x.MessageType
	}
	return nil
}

func (x *ImageFile) GetEnumType() []*descriptorpb.EnumDescriptorProto {
	if x != nil {
		return x.EnumType
	}
	return nil
}

func (x *ImageFile) GetService() []*descriptorpb.ServiceDescriptorProto {
	if x != nil {
		return x.Service
	}
	return nil
}

func (x *ImageFile) GetExtension() []*descriptorpb.FieldDescriptorProto {
	if x != nil {
		return x.Extension
	}
	return nil
}

func (x *ImageFile) GetOptions() *descriptorpb.FileOptions {
	if x != nil {
		return x.Options
	}
	return nil
}

func (x *ImageFile) GetSourceCodeInfo() *descriptorpb.SourceCodeInfo {
	if x != nil {
		return x.SourceCodeInfo
	}
	return nil
}

func (x *ImageFile) GetSyntax() string {
	if x != nil && x.Syntax != nil {
		return *x.Syntax
	}
	return ""
}

func (x *ImageFile) GetBufExtension() *ImageFileExtension {
	if x != nil {
		return x.BufExtension
	}
	return nil
}

// ImageFileExtension contains extensions to ImageFiles.
//
// The fields are not included directly on the ImageFile so that we can both
// detect if extensions exist, which signifies this was created by buf and not
// by protoc, and so that we can add fields in a freeform manner without
// worrying about conflicts with FileDescriptorProto.
type ImageFileExtension struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// is_import denotes whether this file is considered an "import".
	//
	// An import is a file which was not derived from the local source files.
	// There are two cases where this could be true:
	//
	// 1. A Well-Known Type included from the compiler.
	// 2. A file that was included from a Buf module dependency.
	//
	// We use "import" as this matches with the protoc concept of
	// --include_imports, however import is a bit of an overloaded term.
	//
	// This will always be set.
	IsImport *bool `protobuf:"varint,1,opt,name=is_import,json=isImport" json:"is_import,omitempty"`
	// ModuleInfo contains information about the Buf module this file belongs to.
	//
	// This field is optional and will not be set if the module is not known.
	ModuleInfo *ModuleInfo `protobuf:"bytes,2,opt,name=module_info,json=moduleInfo" json:"module_info,omitempty"`
	// is_syntax_unspecified denotes whether the file did not have a syntax
	// explicitly specified.
	//
	// Per the FileDescriptorProto spec, it would be fine in this case to just
	// leave the syntax field unset to denote this and to set the syntax field
	// to "proto2" if it is specified. However, protoc does not set the syntax
	// field if it was "proto2", and plugins may (incorrectly) depend on this.
	// We also want to maintain consistency with protoc as much as possible.
	// So instead, we have this field which will denote whether syntax was not
	// specified.
	//
	// This will always be set.
	IsSyntaxUnspecified *bool `protobuf:"varint,3,opt,name=is_syntax_unspecified,json=isSyntaxUnspecified" json:"is_syntax_unspecified,omitempty"`
	// unused_dependency are the indexes within the dependency field on
	// FileDescriptorProto for those dependencies that are not used.
	//
	// This matches the shape of the public_dependency and weak_dependency
	// fields.
	UnusedDependency []int32 `protobuf:"varint,4,rep,name=unused_dependency,json=unusedDependency" json:"unused_dependency,omitempty"`
}

func (x *ImageFileExtension) Reset() {
	*x = ImageFileExtension{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_image_v1_image_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ImageFileExtension) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ImageFileExtension) ProtoMessage() {}

func (x *ImageFileExtension) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ImageFileExtension.ProtoReflect.Descriptor instead.
func (*ImageFileExtension) Descriptor() ([]byte, []int) {
	return file_buf_alpha_image_v1_image_proto_rawDescGZIP(), []int{2}
}

func (x *ImageFileExtension) GetIsImport() bool {
	if x != nil && x.IsImport != nil {
		return *x.IsImport
	}
	return false
}

func (x *ImageFileExtension) GetModuleInfo() *ModuleInfo {
	if x != nil {
		return x.ModuleInfo
	}
	return nil
}

func (x *ImageFileExtension) GetIsSyntaxUnspecified() bool {
	if x != nil && x.IsSyntaxUnspecified != nil {
		return *x.IsSyntaxUnspecified
	}
	return false
}

func (x *ImageFileExtension) GetUnusedDependency() []int32 {
	if x != nil {
		return x.UnusedDependency
	}
	return nil
}

// ModuleInfo contains information about a Buf module that an ImageFile
// belongs to.
type ModuleInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// name is the name of the Buf module.
	//
	// This will always be set.
	Name *ModuleName `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	// commit is the repository commit.
	//
	// This field is optional and will not be set if the commit is not known.
	Commit *string `protobuf:"bytes,2,opt,name=commit" json:"commit,omitempty"`
}

func (x *ModuleInfo) Reset() {
	*x = ModuleInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_image_v1_image_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleInfo) ProtoMessage() {}

func (x *ModuleInfo) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleInfo.ProtoReflect.Descriptor instead.
func (*ModuleInfo) Descriptor() ([]byte, []int) {
	return file_buf_alpha_image_v1_image_proto_rawDescGZIP(), []int{3}
}

func (x *ModuleInfo) GetName() *ModuleName {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *ModuleInfo) GetCommit() string {
	if x != nil && x.Commit != nil {
		return *x.Commit
	}
	return ""
}

// ModuleName is a module name.
//
// All fields will always be set.
type ModuleName struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Remote     *string `protobuf:"bytes,1,opt,name=remote" json:"remote,omitempty"`
	Owner      *string `protobuf:"bytes,2,opt,name=owner" json:"owner,omitempty"`
	Repository *string `protobuf:"bytes,3,opt,name=repository" json:"repository,omitempty"`
}

func (x *ModuleName) Reset() {
	*x = ModuleName{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_image_v1_image_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleName) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleName) ProtoMessage() {}

func (x *ModuleName) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleName.ProtoReflect.Descriptor instead.
func (*ModuleName) Descriptor() ([]byte, []int) {
	return file_buf_alpha_image_v1_image_proto_rawDescGZIP(), []int{4}
}

func (x *ModuleName) GetRemote() string {
	if x != nil && x.Remote != nil {
		return *x.Remote
	}
	return ""
}

func (x *ModuleName) GetOwner() string {
	if x != nil && x.Owner != nil {
		return *x.Owner
	}
	return ""
}

func (x *ModuleName) GetRepository() string {
	if x != nil && x.Repository != nil {
		return *x.Repository
	}
	return ""
}

var File_buf_alpha_image_v1_image_proto protoreflect.FileDescriptor

var file_buf_alpha_image_v1_image_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x12, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2e, 0x76, 0x31, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3a, 0x0a, 0x05, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x12,
	0x31, 0x0a, 0x04, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x04, 0x66, 0x69,
	0x6c, 0x65, 0x22, 0xa8, 0x05, 0x0a, 0x09, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x12, 0x1e,
	0x0a, 0x0a, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x0a, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x2b,
	0x0a, 0x11, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65,
	0x6e, 0x63, 0x79, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x05, 0x52, 0x10, 0x70, 0x75, 0x62, 0x6c, 0x69,
	0x63, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x12, 0x27, 0x0a, 0x0f, 0x77,
	0x65, 0x61, 0x6b, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x0b,
	0x20, 0x03, 0x28, 0x05, 0x52, 0x0e, 0x77, 0x65, 0x61, 0x6b, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64,
	0x65, 0x6e, 0x63, 0x79, 0x12, 0x43, 0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x65, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x0b, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x41, 0x0a, 0x09, 0x65, 0x6e, 0x75,
	0x6d, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45,
	0x6e, 0x75, 0x6d, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x50, 0x72, 0x6f,
	0x74, 0x6f, 0x52, 0x08, 0x65, 0x6e, 0x75, 0x6d, 0x54, 0x79, 0x70, 0x65, 0x12, 0x41, 0x0a, 0x07,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f,
	0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x07, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x43, 0x0a, 0x09, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x6f, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x52, 0x09, 0x65, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x12, 0x36, 0x0a, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18,
	0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x4f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x52, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x49, 0x0a, 0x10,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x6e, 0x66, 0x6f,
	0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43,
	0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43,
	0x6f, 0x64, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x79, 0x6e, 0x74, 0x61,
	0x78, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x79, 0x6e, 0x74, 0x61, 0x78, 0x12,
	0x4c, 0x0a, 0x0d, 0x62, 0x75, 0x66, 0x5f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e,
	0x18, 0xea, 0x3e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6d, 0x61,
	0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x52,
	0x0c, 0x62, 0x75, 0x66, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0xd3, 0x01,
	0x0a, 0x12, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x45, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x73, 0x5f, 0x69, 0x6d, 0x70, 0x6f, 0x72,
	0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x49, 0x6d, 0x70, 0x6f, 0x72,
	0x74, 0x12, 0x3f, 0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f, 0x69, 0x6e, 0x66, 0x6f,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0a, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x49, 0x6e,
	0x66, 0x6f, 0x12, 0x32, 0x0a, 0x15, 0x69, 0x73, 0x5f, 0x73, 0x79, 0x6e, 0x74, 0x61, 0x78, 0x5f,
	0x75, 0x6e, 0x73, 0x70, 0x65, 0x63, 0x69, 0x66, 0x69, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x13, 0x69, 0x73, 0x53, 0x79, 0x6e, 0x74, 0x61, 0x78, 0x55, 0x6e, 0x73, 0x70, 0x65,
	0x63, 0x69, 0x66, 0x69, 0x65, 0x64, 0x12, 0x2b, 0x0a, 0x11, 0x75, 0x6e, 0x75, 0x73, 0x65, 0x64,
	0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x05, 0x52, 0x10, 0x75, 0x6e, 0x75, 0x73, 0x65, 0x64, 0x44, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65,
	0x6e, 0x63, 0x79, 0x22, 0x58, 0x0a, 0x0a, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x49, 0x6e, 0x66,
	0x6f, 0x12, 0x32, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x22, 0x5a, 0x0a,
	0x0a, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x72,
	0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x6d,
	0x6f, 0x74, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0xdd, 0x01, 0x0a, 0x16, 0x63, 0x6f,
	0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2e, 0x76, 0x31, 0x42, 0x0a, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x48, 0x01, 0x50, 0x01, 0x5a, 0x47, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72,
	0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x69, 0x6d, 0x61,
	0x67, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x76, 0x31, 0xf8, 0x01, 0x01,
	0xa2, 0x02, 0x03, 0x42, 0x41, 0x49, 0xaa, 0x02, 0x12, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x12, 0x42, 0x75,
	0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x5c, 0x56, 0x31,
	0xe2, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x49, 0x6d, 0x61,
	0x67, 0x65, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x15, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a,
	0x49, 0x6d, 0x61, 0x67, 0x65, 0x3a, 0x3a, 0x56, 0x31,
}

var (
	file_buf_alpha_image_v1_image_proto_rawDescOnce sync.Once
	file_buf_alpha_image_v1_image_proto_rawDescData = file_buf_alpha_image_v1_image_proto_rawDesc
)

func file_buf_alpha_image_v1_image_proto_rawDescGZIP() []byte {
	file_buf_alpha_image_v1_image_proto_rawDescOnce.Do(func() {
		file_buf_alpha_image_v1_image_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_image_v1_image_proto_rawDescData)
	})
	return file_buf_alpha_image_v1_image_proto_rawDescData
}

var file_buf_alpha_image_v1_image_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_buf_alpha_image_v1_image_proto_goTypes = []interface{}{
	(*Image)(nil),                               // 0: buf.alpha.image.v1.Image
	(*ImageFile)(nil),                           // 1: buf.alpha.image.v1.ImageFile
	(*ImageFileExtension)(nil),                  // 2: buf.alpha.image.v1.ImageFileExtension
	(*ModuleInfo)(nil),                          // 3: buf.alpha.image.v1.ModuleInfo
	(*ModuleName)(nil),                          // 4: buf.alpha.image.v1.ModuleName
	(*descriptorpb.DescriptorProto)(nil),        // 5: google.protobuf.DescriptorProto
	(*descriptorpb.EnumDescriptorProto)(nil),    // 6: google.protobuf.EnumDescriptorProto
	(*descriptorpb.ServiceDescriptorProto)(nil), // 7: google.protobuf.ServiceDescriptorProto
	(*descriptorpb.FieldDescriptorProto)(nil),   // 8: google.protobuf.FieldDescriptorProto
	(*descriptorpb.FileOptions)(nil),            // 9: google.protobuf.FileOptions
	(*descriptorpb.SourceCodeInfo)(nil),         // 10: google.protobuf.SourceCodeInfo
}
var file_buf_alpha_image_v1_image_proto_depIdxs = []int32{
	1,  // 0: buf.alpha.image.v1.Image.file:type_name -> buf.alpha.image.v1.ImageFile
	5,  // 1: buf.alpha.image.v1.ImageFile.message_type:type_name -> google.protobuf.DescriptorProto
	6,  // 2: buf.alpha.image.v1.ImageFile.enum_type:type_name -> google.protobuf.EnumDescriptorProto
	7,  // 3: buf.alpha.image.v1.ImageFile.service:type_name -> google.protobuf.ServiceDescriptorProto
	8,  // 4: buf.alpha.image.v1.ImageFile.extension:type_name -> google.protobuf.FieldDescriptorProto
	9,  // 5: buf.alpha.image.v1.ImageFile.options:type_name -> google.protobuf.FileOptions
	10, // 6: buf.alpha.image.v1.ImageFile.source_code_info:type_name -> google.protobuf.SourceCodeInfo
	2,  // 7: buf.alpha.image.v1.ImageFile.buf_extension:type_name -> buf.alpha.image.v1.ImageFileExtension
	3,  // 8: buf.alpha.image.v1.ImageFileExtension.module_info:type_name -> buf.alpha.image.v1.ModuleInfo
	4,  // 9: buf.alpha.image.v1.ModuleInfo.name:type_name -> buf.alpha.image.v1.ModuleName
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_buf_alpha_image_v1_image_proto_init() }
func file_buf_alpha_image_v1_image_proto_init() {
	if File_buf_alpha_image_v1_image_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_image_v1_image_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Image); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_image_v1_image_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ImageFile); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_image_v1_image_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ImageFileExtension); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_image_v1_image_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModuleInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_image_v1_image_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModuleName); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_image_v1_image_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_image_v1_image_proto_goTypes,
		DependencyIndexes: file_buf_alpha_image_v1_image_proto_depIdxs,
		MessageInfos:      file_buf_alpha_image_v1_image_proto_msgTypes,
	}.Build()
	File_buf_alpha_image_v1_image_proto = out.File
	file_buf_alpha_image_v1_image_proto_rawDesc = nil
	file_buf_alpha_image_v1_image_proto_goTypes = nil
	file_buf_alpha_image_v1_image_proto_depIdxs = nil
}
