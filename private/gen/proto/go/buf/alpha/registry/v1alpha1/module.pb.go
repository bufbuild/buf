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
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/module.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// LocalModuleReference is a local module reference.
//
// It does not include a remote.
type LocalModuleReference struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner      string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository string                 `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Reference  string                 `protobuf:"bytes,3,opt,name=reference,proto3"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *LocalModuleReference) Reset() {
	*x = LocalModuleReference{}
	mi := &file_buf_alpha_registry_v1alpha1_module_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LocalModuleReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LocalModuleReference) ProtoMessage() {}

func (x *LocalModuleReference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_module_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *LocalModuleReference) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *LocalModuleReference) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

func (x *LocalModuleReference) GetReference() string {
	if x != nil {
		return x.xxx_hidden_Reference
	}
	return ""
}

func (x *LocalModuleReference) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *LocalModuleReference) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

func (x *LocalModuleReference) SetReference(v string) {
	x.xxx_hidden_Reference = v
}

type LocalModuleReference_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner      string
	Repository string
	// either branch or commit
	Reference string
}

func (b0 LocalModuleReference_builder) Build() *LocalModuleReference {
	m0 := &LocalModuleReference{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Reference = b.Reference
	return m0
}

// LocalModulePin is a local module pin.
//
// It does not include a remote.
type LocalModulePin struct {
	state                     protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner          string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository     string                 `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Commit         string                 `protobuf:"bytes,4,opt,name=commit,proto3"`
	xxx_hidden_ManifestDigest string                 `protobuf:"bytes,6,opt,name=manifest_digest,json=manifestDigest,proto3"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *LocalModulePin) Reset() {
	*x = LocalModulePin{}
	mi := &file_buf_alpha_registry_v1alpha1_module_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LocalModulePin) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LocalModulePin) ProtoMessage() {}

func (x *LocalModulePin) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_module_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *LocalModulePin) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *LocalModulePin) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

func (x *LocalModulePin) GetCommit() string {
	if x != nil {
		return x.xxx_hidden_Commit
	}
	return ""
}

func (x *LocalModulePin) GetManifestDigest() string {
	if x != nil {
		return x.xxx_hidden_ManifestDigest
	}
	return ""
}

func (x *LocalModulePin) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *LocalModulePin) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

func (x *LocalModulePin) SetCommit(v string) {
	x.xxx_hidden_Commit = v
}

func (x *LocalModulePin) SetManifestDigest(v string) {
	x.xxx_hidden_ManifestDigest = v
}

type LocalModulePin_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner      string
	Repository string
	Commit     string
	// Module's manifest digest. Replacement for previous b1/b3 digests.
	ManifestDigest string
}

func (b0 LocalModulePin_builder) Build() *LocalModulePin {
	m0 := &LocalModulePin{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Commit = b.Commit
	x.xxx_hidden_ManifestDigest = b.ManifestDigest
	return m0
}

var File_buf_alpha_registry_v1alpha1_module_proto protoreflect.FileDescriptor

const file_buf_alpha_registry_v1alpha1_module_proto_rawDesc = "" +
	"\n" +
	"(buf/alpha/registry/v1alpha1/module.proto\x12\x1bbuf.alpha.registry.v1alpha1\"j\n" +
	"\x14LocalModuleReference\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12\x1c\n" +
	"\treference\x18\x03 \x01(\tR\treference\"\xc8\x01\n" +
	"\x0eLocalModulePin\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12\x16\n" +
	"\x06commit\x18\x04 \x01(\tR\x06commit\x12'\n" +
	"\x0fmanifest_digest\x18\x06 \x01(\tR\x0emanifestDigestJ\x04\b\x03\x10\x04J\x04\b\x05\x10\x06J\x04\b\a\x10\bJ\x04\b\b\x10\tR\x06branchR\vcreate_timeR\x06digestR\n" +
	"draft_nameB\x98\x02\n" +
	"\x1fcom.buf.alpha.registry.v1alpha1B\vModuleProtoP\x01ZYgithub.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1;registryv1alpha1\xa2\x02\x03BAR\xaa\x02\x1bBuf.Alpha.Registry.V1alpha1\xca\x02\x1bBuf\\Alpha\\Registry\\V1alpha1\xe2\x02'Buf\\Alpha\\Registry\\V1alpha1\\GPBMetadata\xea\x02\x1eBuf::Alpha::Registry::V1alpha1b\x06proto3"

var file_buf_alpha_registry_v1alpha1_module_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_alpha_registry_v1alpha1_module_proto_goTypes = []any{
	(*LocalModuleReference)(nil), // 0: buf.alpha.registry.v1alpha1.LocalModuleReference
	(*LocalModulePin)(nil),       // 1: buf.alpha.registry.v1alpha1.LocalModulePin
}
var file_buf_alpha_registry_v1alpha1_module_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_module_proto_init() }
func file_buf_alpha_registry_v1alpha1_module_proto_init() {
	if File_buf_alpha_registry_v1alpha1_module_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_registry_v1alpha1_module_proto_rawDesc), len(file_buf_alpha_registry_v1alpha1_module_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_module_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_module_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_module_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_module_proto = out.File
	file_buf_alpha_registry_v1alpha1_module_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_module_proto_depIdxs = nil
}
