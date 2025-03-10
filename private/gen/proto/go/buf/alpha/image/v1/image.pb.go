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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        (unknown)
// source: buf/alpha/image/v1/image.proto

package imagev1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Image is an ext FileDescriptorSet.
//
// See https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto
type Image struct {
	state           protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_File *[]*ImageFile          `protobuf:"bytes,1,rep,name=file"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *Image) Reset() {
	*x = Image{}
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Image) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Image) ProtoMessage() {}

func (x *Image) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Image) GetFile() []*ImageFile {
	if x != nil {
		if x.xxx_hidden_File != nil {
			return *x.xxx_hidden_File
		}
	}
	return nil
}

func (x *Image) SetFile(v []*ImageFile) {
	x.xxx_hidden_File = &v
}

type Image_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	File []*ImageFile
}

func (b0 Image_builder) Build() *Image {
	m0 := &Image{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_File = &b.File
	return m0
}

// ImageFile is an extended FileDescriptorProto.
//
// Since FileDescriptorProto does not have extensions, we copy the fields from
// FileDescriptorProto, and then add our own extensions via the buf_extension
// field. This is compatible with a FileDescriptorProto.
//
// See https://github.com/protocolbuffers/protobuf/blob/master/src/google/protobuf/descriptor.proto
type ImageFile struct {
	state                       protoimpl.MessageState                  `protogen:"opaque.v1"`
	xxx_hidden_Name             *string                                 `protobuf:"bytes,1,opt,name=name"`
	xxx_hidden_Package          *string                                 `protobuf:"bytes,2,opt,name=package"`
	xxx_hidden_Dependency       []string                                `protobuf:"bytes,3,rep,name=dependency"`
	xxx_hidden_PublicDependency []int32                                 `protobuf:"varint,10,rep,name=public_dependency,json=publicDependency"`
	xxx_hidden_WeakDependency   []int32                                 `protobuf:"varint,11,rep,name=weak_dependency,json=weakDependency"`
	xxx_hidden_MessageType      *[]*descriptorpb.DescriptorProto        `protobuf:"bytes,4,rep,name=message_type,json=messageType"`
	xxx_hidden_EnumType         *[]*descriptorpb.EnumDescriptorProto    `protobuf:"bytes,5,rep,name=enum_type,json=enumType"`
	xxx_hidden_Service          *[]*descriptorpb.ServiceDescriptorProto `protobuf:"bytes,6,rep,name=service"`
	xxx_hidden_Extension        *[]*descriptorpb.FieldDescriptorProto   `protobuf:"bytes,7,rep,name=extension"`
	xxx_hidden_Options          *descriptorpb.FileOptions               `protobuf:"bytes,8,opt,name=options"`
	xxx_hidden_SourceCodeInfo   *descriptorpb.SourceCodeInfo            `protobuf:"bytes,9,opt,name=source_code_info,json=sourceCodeInfo"`
	xxx_hidden_Syntax           *string                                 `protobuf:"bytes,12,opt,name=syntax"`
	xxx_hidden_Edition          descriptorpb.Edition                    `protobuf:"varint,14,opt,name=edition,enum=google.protobuf.Edition"`
	xxx_hidden_BufExtension     *ImageFileExtension                     `protobuf:"bytes,8042,opt,name=buf_extension,json=bufExtension"`
	XXX_raceDetectHookData      protoimpl.RaceDetectHookData
	XXX_presence                [1]uint32
	unknownFields               protoimpl.UnknownFields
	sizeCache                   protoimpl.SizeCache
}

func (x *ImageFile) Reset() {
	*x = ImageFile{}
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ImageFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ImageFile) ProtoMessage() {}

func (x *ImageFile) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ImageFile) GetName() string {
	if x != nil {
		if x.xxx_hidden_Name != nil {
			return *x.xxx_hidden_Name
		}
		return ""
	}
	return ""
}

func (x *ImageFile) GetPackage() string {
	if x != nil {
		if x.xxx_hidden_Package != nil {
			return *x.xxx_hidden_Package
		}
		return ""
	}
	return ""
}

func (x *ImageFile) GetDependency() []string {
	if x != nil {
		return x.xxx_hidden_Dependency
	}
	return nil
}

func (x *ImageFile) GetPublicDependency() []int32 {
	if x != nil {
		return x.xxx_hidden_PublicDependency
	}
	return nil
}

func (x *ImageFile) GetWeakDependency() []int32 {
	if x != nil {
		return x.xxx_hidden_WeakDependency
	}
	return nil
}

func (x *ImageFile) GetMessageType() []*descriptorpb.DescriptorProto {
	if x != nil {
		if x.xxx_hidden_MessageType != nil {
			return *x.xxx_hidden_MessageType
		}
	}
	return nil
}

func (x *ImageFile) GetEnumType() []*descriptorpb.EnumDescriptorProto {
	if x != nil {
		if x.xxx_hidden_EnumType != nil {
			return *x.xxx_hidden_EnumType
		}
	}
	return nil
}

func (x *ImageFile) GetService() []*descriptorpb.ServiceDescriptorProto {
	if x != nil {
		if x.xxx_hidden_Service != nil {
			return *x.xxx_hidden_Service
		}
	}
	return nil
}

func (x *ImageFile) GetExtension() []*descriptorpb.FieldDescriptorProto {
	if x != nil {
		if x.xxx_hidden_Extension != nil {
			return *x.xxx_hidden_Extension
		}
	}
	return nil
}

func (x *ImageFile) GetOptions() *descriptorpb.FileOptions {
	if x != nil {
		return x.xxx_hidden_Options
	}
	return nil
}

func (x *ImageFile) GetSourceCodeInfo() *descriptorpb.SourceCodeInfo {
	if x != nil {
		return x.xxx_hidden_SourceCodeInfo
	}
	return nil
}

func (x *ImageFile) GetSyntax() string {
	if x != nil {
		if x.xxx_hidden_Syntax != nil {
			return *x.xxx_hidden_Syntax
		}
		return ""
	}
	return ""
}

func (x *ImageFile) GetEdition() descriptorpb.Edition {
	if x != nil {
		if protoimpl.X.Present(&(x.XXX_presence[0]), 12) {
			return x.xxx_hidden_Edition
		}
	}
	return descriptorpb.Edition(0)
}

func (x *ImageFile) GetBufExtension() *ImageFileExtension {
	if x != nil {
		return x.xxx_hidden_BufExtension
	}
	return nil
}

func (x *ImageFile) SetName(v string) {
	x.xxx_hidden_Name = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 14)
}

func (x *ImageFile) SetPackage(v string) {
	x.xxx_hidden_Package = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 14)
}

func (x *ImageFile) SetDependency(v []string) {
	x.xxx_hidden_Dependency = v
}

func (x *ImageFile) SetPublicDependency(v []int32) {
	x.xxx_hidden_PublicDependency = v
}

func (x *ImageFile) SetWeakDependency(v []int32) {
	x.xxx_hidden_WeakDependency = v
}

func (x *ImageFile) SetMessageType(v []*descriptorpb.DescriptorProto) {
	x.xxx_hidden_MessageType = &v
}

func (x *ImageFile) SetEnumType(v []*descriptorpb.EnumDescriptorProto) {
	x.xxx_hidden_EnumType = &v
}

func (x *ImageFile) SetService(v []*descriptorpb.ServiceDescriptorProto) {
	x.xxx_hidden_Service = &v
}

func (x *ImageFile) SetExtension(v []*descriptorpb.FieldDescriptorProto) {
	x.xxx_hidden_Extension = &v
}

func (x *ImageFile) SetOptions(v *descriptorpb.FileOptions) {
	x.xxx_hidden_Options = v
}

func (x *ImageFile) SetSourceCodeInfo(v *descriptorpb.SourceCodeInfo) {
	x.xxx_hidden_SourceCodeInfo = v
}

func (x *ImageFile) SetSyntax(v string) {
	x.xxx_hidden_Syntax = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 11, 14)
}

func (x *ImageFile) SetEdition(v descriptorpb.Edition) {
	x.xxx_hidden_Edition = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 12, 14)
}

func (x *ImageFile) SetBufExtension(v *ImageFileExtension) {
	x.xxx_hidden_BufExtension = v
}

func (x *ImageFile) HasName() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *ImageFile) HasPackage() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *ImageFile) HasOptions() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Options != nil
}

func (x *ImageFile) HasSourceCodeInfo() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_SourceCodeInfo != nil
}

func (x *ImageFile) HasSyntax() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 11)
}

func (x *ImageFile) HasEdition() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 12)
}

func (x *ImageFile) HasBufExtension() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_BufExtension != nil
}

func (x *ImageFile) ClearName() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_Name = nil
}

func (x *ImageFile) ClearPackage() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_Package = nil
}

func (x *ImageFile) ClearOptions() {
	x.xxx_hidden_Options = nil
}

func (x *ImageFile) ClearSourceCodeInfo() {
	x.xxx_hidden_SourceCodeInfo = nil
}

func (x *ImageFile) ClearSyntax() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 11)
	x.xxx_hidden_Syntax = nil
}

func (x *ImageFile) ClearEdition() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 12)
	x.xxx_hidden_Edition = descriptorpb.Edition_EDITION_UNKNOWN
}

func (x *ImageFile) ClearBufExtension() {
	x.xxx_hidden_BufExtension = nil
}

type ImageFile_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Name             *string
	Package          *string
	Dependency       []string
	PublicDependency []int32
	WeakDependency   []int32
	MessageType      []*descriptorpb.DescriptorProto
	EnumType         []*descriptorpb.EnumDescriptorProto
	Service          []*descriptorpb.ServiceDescriptorProto
	Extension        []*descriptorpb.FieldDescriptorProto
	Options          *descriptorpb.FileOptions
	SourceCodeInfo   *descriptorpb.SourceCodeInfo
	Syntax           *string
	Edition          *descriptorpb.Edition
	// buf_extension contains buf-specific extensions to FileDescriptorProtos.
	//
	// The prefixed name and high tag value is used to all but guarantee there
	// will never be any conflict with Google's FileDescriptorProto definition.
	// The definition of a FileDescriptorProto has not changed in years, so
	// we're not too worried about a conflict here.
	BufExtension *ImageFileExtension
}

func (b0 ImageFile_builder) Build() *ImageFile {
	m0 := &ImageFile{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Name != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 14)
		x.xxx_hidden_Name = b.Name
	}
	if b.Package != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 14)
		x.xxx_hidden_Package = b.Package
	}
	x.xxx_hidden_Dependency = b.Dependency
	x.xxx_hidden_PublicDependency = b.PublicDependency
	x.xxx_hidden_WeakDependency = b.WeakDependency
	x.xxx_hidden_MessageType = &b.MessageType
	x.xxx_hidden_EnumType = &b.EnumType
	x.xxx_hidden_Service = &b.Service
	x.xxx_hidden_Extension = &b.Extension
	x.xxx_hidden_Options = b.Options
	x.xxx_hidden_SourceCodeInfo = b.SourceCodeInfo
	if b.Syntax != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 11, 14)
		x.xxx_hidden_Syntax = b.Syntax
	}
	if b.Edition != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 12, 14)
		x.xxx_hidden_Edition = *b.Edition
	}
	x.xxx_hidden_BufExtension = b.BufExtension
	return m0
}

// ImageFileExtension contains extensions to ImageFiles.
//
// The fields are not included directly on the ImageFile so that we can both
// detect if extensions exist, which signifies this was created by buf and not
// by protoc, and so that we can add fields in a freeform manner without
// worrying about conflicts with FileDescriptorProto.
type ImageFileExtension struct {
	state                          protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_IsImport            bool                   `protobuf:"varint,1,opt,name=is_import,json=isImport"`
	xxx_hidden_ModuleInfo          *ModuleInfo            `protobuf:"bytes,2,opt,name=module_info,json=moduleInfo"`
	xxx_hidden_IsSyntaxUnspecified bool                   `protobuf:"varint,3,opt,name=is_syntax_unspecified,json=isSyntaxUnspecified"`
	xxx_hidden_UnusedDependency    []int32                `protobuf:"varint,4,rep,name=unused_dependency,json=unusedDependency"`
	XXX_raceDetectHookData         protoimpl.RaceDetectHookData
	XXX_presence                   [1]uint32
	unknownFields                  protoimpl.UnknownFields
	sizeCache                      protoimpl.SizeCache
}

func (x *ImageFileExtension) Reset() {
	*x = ImageFileExtension{}
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ImageFileExtension) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ImageFileExtension) ProtoMessage() {}

func (x *ImageFileExtension) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ImageFileExtension) GetIsImport() bool {
	if x != nil {
		return x.xxx_hidden_IsImport
	}
	return false
}

func (x *ImageFileExtension) GetModuleInfo() *ModuleInfo {
	if x != nil {
		return x.xxx_hidden_ModuleInfo
	}
	return nil
}

func (x *ImageFileExtension) GetIsSyntaxUnspecified() bool {
	if x != nil {
		return x.xxx_hidden_IsSyntaxUnspecified
	}
	return false
}

func (x *ImageFileExtension) GetUnusedDependency() []int32 {
	if x != nil {
		return x.xxx_hidden_UnusedDependency
	}
	return nil
}

func (x *ImageFileExtension) SetIsImport(v bool) {
	x.xxx_hidden_IsImport = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 4)
}

func (x *ImageFileExtension) SetModuleInfo(v *ModuleInfo) {
	x.xxx_hidden_ModuleInfo = v
}

func (x *ImageFileExtension) SetIsSyntaxUnspecified(v bool) {
	x.xxx_hidden_IsSyntaxUnspecified = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 2, 4)
}

func (x *ImageFileExtension) SetUnusedDependency(v []int32) {
	x.xxx_hidden_UnusedDependency = v
}

func (x *ImageFileExtension) HasIsImport() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *ImageFileExtension) HasModuleInfo() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_ModuleInfo != nil
}

func (x *ImageFileExtension) HasIsSyntaxUnspecified() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 2)
}

func (x *ImageFileExtension) ClearIsImport() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_IsImport = false
}

func (x *ImageFileExtension) ClearModuleInfo() {
	x.xxx_hidden_ModuleInfo = nil
}

func (x *ImageFileExtension) ClearIsSyntaxUnspecified() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 2)
	x.xxx_hidden_IsSyntaxUnspecified = false
}

type ImageFileExtension_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

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
	IsImport *bool
	// ModuleInfo contains information about the Buf module this file belongs to.
	//
	// This field is optional and will not be set if the module is not known.
	ModuleInfo *ModuleInfo
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
	IsSyntaxUnspecified *bool
	// unused_dependency are the indexes within the dependency field on
	// FileDescriptorProto for those dependencies that are not used.
	//
	// This matches the shape of the public_dependency and weak_dependency
	// fields.
	UnusedDependency []int32
}

func (b0 ImageFileExtension_builder) Build() *ImageFileExtension {
	m0 := &ImageFileExtension{}
	b, x := &b0, m0
	_, _ = b, x
	if b.IsImport != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 4)
		x.xxx_hidden_IsImport = *b.IsImport
	}
	x.xxx_hidden_ModuleInfo = b.ModuleInfo
	if b.IsSyntaxUnspecified != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 2, 4)
		x.xxx_hidden_IsSyntaxUnspecified = *b.IsSyntaxUnspecified
	}
	x.xxx_hidden_UnusedDependency = b.UnusedDependency
	return m0
}

// ModuleInfo contains information about a Buf module that an ImageFile
// belongs to.
type ModuleInfo struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Name        *ModuleName            `protobuf:"bytes,1,opt,name=name"`
	xxx_hidden_Commit      *string                `protobuf:"bytes,2,opt,name=commit"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *ModuleInfo) Reset() {
	*x = ModuleInfo{}
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ModuleInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleInfo) ProtoMessage() {}

func (x *ModuleInfo) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ModuleInfo) GetName() *ModuleName {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return nil
}

func (x *ModuleInfo) GetCommit() string {
	if x != nil {
		if x.xxx_hidden_Commit != nil {
			return *x.xxx_hidden_Commit
		}
		return ""
	}
	return ""
}

func (x *ModuleInfo) SetName(v *ModuleName) {
	x.xxx_hidden_Name = v
}

func (x *ModuleInfo) SetCommit(v string) {
	x.xxx_hidden_Commit = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 2)
}

func (x *ModuleInfo) HasName() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Name != nil
}

func (x *ModuleInfo) HasCommit() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *ModuleInfo) ClearName() {
	x.xxx_hidden_Name = nil
}

func (x *ModuleInfo) ClearCommit() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_Commit = nil
}

type ModuleInfo_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// name is the name of the Buf module.
	//
	// This will always be set.
	Name *ModuleName
	// commit is the repository commit.
	//
	// This field is optional and will not be set if the commit is not known.
	Commit *string
}

func (b0 ModuleInfo_builder) Build() *ModuleInfo {
	m0 := &ModuleInfo{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Name = b.Name
	if b.Commit != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 2)
		x.xxx_hidden_Commit = b.Commit
	}
	return m0
}

// ModuleName is a module name.
//
// All fields will always be set.
type ModuleName struct {
	state                  protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Remote      *string                `protobuf:"bytes,1,opt,name=remote"`
	xxx_hidden_Owner       *string                `protobuf:"bytes,2,opt,name=owner"`
	xxx_hidden_Repository  *string                `protobuf:"bytes,3,opt,name=repository"`
	XXX_raceDetectHookData protoimpl.RaceDetectHookData
	XXX_presence           [1]uint32
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *ModuleName) Reset() {
	*x = ModuleName{}
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ModuleName) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleName) ProtoMessage() {}

func (x *ModuleName) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_image_v1_image_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ModuleName) GetRemote() string {
	if x != nil {
		if x.xxx_hidden_Remote != nil {
			return *x.xxx_hidden_Remote
		}
		return ""
	}
	return ""
}

func (x *ModuleName) GetOwner() string {
	if x != nil {
		if x.xxx_hidden_Owner != nil {
			return *x.xxx_hidden_Owner
		}
		return ""
	}
	return ""
}

func (x *ModuleName) GetRepository() string {
	if x != nil {
		if x.xxx_hidden_Repository != nil {
			return *x.xxx_hidden_Repository
		}
		return ""
	}
	return ""
}

func (x *ModuleName) SetRemote(v string) {
	x.xxx_hidden_Remote = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 3)
}

func (x *ModuleName) SetOwner(v string) {
	x.xxx_hidden_Owner = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 3)
}

func (x *ModuleName) SetRepository(v string) {
	x.xxx_hidden_Repository = &v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 2, 3)
}

func (x *ModuleName) HasRemote() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *ModuleName) HasOwner() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *ModuleName) HasRepository() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 2)
}

func (x *ModuleName) ClearRemote() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_Remote = nil
}

func (x *ModuleName) ClearOwner() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_Owner = nil
}

func (x *ModuleName) ClearRepository() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 2)
	x.xxx_hidden_Repository = nil
}

type ModuleName_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Remote     *string
	Owner      *string
	Repository *string
}

func (b0 ModuleName_builder) Build() *ModuleName {
	m0 := &ModuleName{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Remote != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 3)
		x.xxx_hidden_Remote = b.Remote
	}
	if b.Owner != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 3)
		x.xxx_hidden_Owner = b.Owner
	}
	if b.Repository != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 2, 3)
		x.xxx_hidden_Repository = b.Repository
	}
	return m0
}

var File_buf_alpha_image_v1_image_proto protoreflect.FileDescriptor

var file_buf_alpha_image_v1_image_proto_rawDesc = string([]byte{
	0x0a, 0x1e, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x12, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2e, 0x76, 0x31, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3a, 0x0a, 0x05, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x12,
	0x31, 0x0a, 0x04, 0x66, 0x69, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x04, 0x66, 0x69,
	0x6c, 0x65, 0x22, 0xdc, 0x05, 0x0a, 0x09, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65,
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
	0x32, 0x0a, 0x07, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x18, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x07, 0x65, 0x64, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x4c, 0x0a, 0x0d, 0x62, 0x75, 0x66, 0x5f, 0x65, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x18, 0xea, 0x3e, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31,
	0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x52, 0x0c, 0x62, 0x75, 0x66, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f,
	0x6e, 0x22, 0xd3, 0x01, 0x0a, 0x12, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x45,
	0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x73, 0x5f, 0x69,
	0x6d, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x69, 0x73, 0x49,
	0x6d, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x3f, 0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x5f,
	0x69, 0x6e, 0x66, 0x6f, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0a, 0x6d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x32, 0x0a, 0x15, 0x69, 0x73, 0x5f, 0x73, 0x79, 0x6e,
	0x74, 0x61, 0x78, 0x5f, 0x75, 0x6e, 0x73, 0x70, 0x65, 0x63, 0x69, 0x66, 0x69, 0x65, 0x64, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x13, 0x69, 0x73, 0x53, 0x79, 0x6e, 0x74, 0x61, 0x78, 0x55,
	0x6e, 0x73, 0x70, 0x65, 0x63, 0x69, 0x66, 0x69, 0x65, 0x64, 0x12, 0x2b, 0x0a, 0x11, 0x75, 0x6e,
	0x75, 0x73, 0x65, 0x64, 0x5f, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x18,
	0x04, 0x20, 0x03, 0x28, 0x05, 0x52, 0x10, 0x75, 0x6e, 0x75, 0x73, 0x65, 0x64, 0x44, 0x65, 0x70,
	0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x79, 0x22, 0x58, 0x0a, 0x0a, 0x4d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x32, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e,
	0x61, 0x6d, 0x65, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x22, 0x5a, 0x0a, 0x0a, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a,
	0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0xdd, 0x01,
	0x0a, 0x16, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x42, 0x0a, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x48, 0x01, 0x50, 0x01, 0x5a, 0x47, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75,
	0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2f, 0x76, 0x31, 0x3b, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x76,
	0x31, 0xf8, 0x01, 0x01, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x49, 0xaa, 0x02, 0x12, 0x42, 0x75, 0x66,
	0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x56, 0x31, 0xca,
	0x02, 0x12, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x49, 0x6d, 0x61, 0x67,
	0x65, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61,
	0x5c, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x15, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70,
	0x68, 0x61, 0x3a, 0x3a, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x3a, 0x3a, 0x56, 0x31,
})

var file_buf_alpha_image_v1_image_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_buf_alpha_image_v1_image_proto_goTypes = []any{
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
	(descriptorpb.Edition)(0),                   // 11: google.protobuf.Edition
}
var file_buf_alpha_image_v1_image_proto_depIdxs = []int32{
	1,  // 0: buf.alpha.image.v1.Image.file:type_name -> buf.alpha.image.v1.ImageFile
	5,  // 1: buf.alpha.image.v1.ImageFile.message_type:type_name -> google.protobuf.DescriptorProto
	6,  // 2: buf.alpha.image.v1.ImageFile.enum_type:type_name -> google.protobuf.EnumDescriptorProto
	7,  // 3: buf.alpha.image.v1.ImageFile.service:type_name -> google.protobuf.ServiceDescriptorProto
	8,  // 4: buf.alpha.image.v1.ImageFile.extension:type_name -> google.protobuf.FieldDescriptorProto
	9,  // 5: buf.alpha.image.v1.ImageFile.options:type_name -> google.protobuf.FileOptions
	10, // 6: buf.alpha.image.v1.ImageFile.source_code_info:type_name -> google.protobuf.SourceCodeInfo
	11, // 7: buf.alpha.image.v1.ImageFile.edition:type_name -> google.protobuf.Edition
	2,  // 8: buf.alpha.image.v1.ImageFile.buf_extension:type_name -> buf.alpha.image.v1.ImageFileExtension
	3,  // 9: buf.alpha.image.v1.ImageFileExtension.module_info:type_name -> buf.alpha.image.v1.ModuleInfo
	4,  // 10: buf.alpha.image.v1.ModuleInfo.name:type_name -> buf.alpha.image.v1.ModuleName
	11, // [11:11] is the sub-list for method output_type
	11, // [11:11] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_buf_alpha_image_v1_image_proto_init() }
func file_buf_alpha_image_v1_image_proto_init() {
	if File_buf_alpha_image_v1_image_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_image_v1_image_proto_rawDesc), len(file_buf_alpha_image_v1_image_proto_rawDesc)),
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
	file_buf_alpha_image_v1_image_proto_goTypes = nil
	file_buf_alpha_image_v1_image_proto_depIdxs = nil
}
