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
// source: buf/alpha/registry/v1alpha1/push.proto

package registryv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
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

// PushRequest specifies the module to push to the BSR.
type PushRequest struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner      string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository string                 `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Branch     string                 `protobuf:"bytes,3,opt,name=branch,proto3"`
	xxx_hidden_Module     *v1alpha1.Module       `protobuf:"bytes,4,opt,name=module,proto3"`
	xxx_hidden_Tags       []string               `protobuf:"bytes,5,rep,name=tags,proto3"`
	xxx_hidden_Tracks     []string               `protobuf:"bytes,6,rep,name=tracks,proto3"`
	xxx_hidden_DraftName  string                 `protobuf:"bytes,7,opt,name=draft_name,json=draftName,proto3"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *PushRequest) Reset() {
	*x = PushRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PushRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PushRequest) ProtoMessage() {}

func (x *PushRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *PushRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *PushRequest) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
func (x *PushRequest) GetBranch() string {
	if x != nil {
		return x.xxx_hidden_Branch
	}
	return ""
}

func (x *PushRequest) GetModule() *v1alpha1.Module {
	if x != nil {
		return x.xxx_hidden_Module
	}
	return nil
}

func (x *PushRequest) GetTags() []string {
	if x != nil {
		return x.xxx_hidden_Tags
	}
	return nil
}

// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
func (x *PushRequest) GetTracks() []string {
	if x != nil {
		return x.xxx_hidden_Tracks
	}
	return nil
}

func (x *PushRequest) GetDraftName() string {
	if x != nil {
		return x.xxx_hidden_DraftName
	}
	return ""
}

func (x *PushRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *PushRequest) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
func (x *PushRequest) SetBranch(v string) {
	x.xxx_hidden_Branch = v
}

func (x *PushRequest) SetModule(v *v1alpha1.Module) {
	x.xxx_hidden_Module = v
}

func (x *PushRequest) SetTags(v []string) {
	x.xxx_hidden_Tags = v
}

// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
func (x *PushRequest) SetTracks(v []string) {
	x.xxx_hidden_Tracks = v
}

func (x *PushRequest) SetDraftName(v string) {
	x.xxx_hidden_DraftName = v
}

func (x *PushRequest) HasModule() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Module != nil
}

func (x *PushRequest) ClearModule() {
	x.xxx_hidden_Module = nil
}

type PushRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner      string
	Repository string
	// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
	Branch string
	Module *v1alpha1.Module
	// Optional; if provided, the provided tags
	// are created for the pushed commit.
	Tags []string
	// Optional; if provided, the pushed commit
	// will be appended to these tracks. If the
	// tracks do not exist, they will be created.
	//
	// Deprecated: Marked as deprecated in buf/alpha/registry/v1alpha1/push.proto.
	Tracks []string
	// If non-empty, the push creates a draft commit with this name.
	DraftName string
}

func (b0 PushRequest_builder) Build() *PushRequest {
	m0 := &PushRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Branch = b.Branch
	x.xxx_hidden_Module = b.Module
	x.xxx_hidden_Tags = b.Tags
	x.xxx_hidden_Tracks = b.Tracks
	x.xxx_hidden_DraftName = b.DraftName
	return m0
}

// PushResponse is the pushed module pin, local to the used remote.
type PushResponse struct {
	state                     protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_LocalModulePin *LocalModulePin        `protobuf:"bytes,5,opt,name=local_module_pin,json=localModulePin,proto3"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *PushResponse) Reset() {
	*x = PushResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PushResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PushResponse) ProtoMessage() {}

func (x *PushResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *PushResponse) GetLocalModulePin() *LocalModulePin {
	if x != nil {
		return x.xxx_hidden_LocalModulePin
	}
	return nil
}

func (x *PushResponse) SetLocalModulePin(v *LocalModulePin) {
	x.xxx_hidden_LocalModulePin = v
}

func (x *PushResponse) HasLocalModulePin() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_LocalModulePin != nil
}

func (x *PushResponse) ClearLocalModulePin() {
	x.xxx_hidden_LocalModulePin = nil
}

type PushResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	LocalModulePin *LocalModulePin
}

func (b0 PushResponse_builder) Build() *PushResponse {
	m0 := &PushResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_LocalModulePin = b.LocalModulePin
	return m0
}

// PushManifestAndBlobsRequest holds the module to push in the manifest+blobs
// encoding format.
type PushManifestAndBlobsRequest struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner      string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository string                 `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Manifest   *v1alpha1.Blob         `protobuf:"bytes,3,opt,name=manifest,proto3"`
	xxx_hidden_Blobs      *[]*v1alpha1.Blob      `protobuf:"bytes,4,rep,name=blobs,proto3"`
	xxx_hidden_Tags       []string               `protobuf:"bytes,5,rep,name=tags,proto3"`
	xxx_hidden_DraftName  string                 `protobuf:"bytes,6,opt,name=draft_name,json=draftName,proto3"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *PushManifestAndBlobsRequest) Reset() {
	*x = PushManifestAndBlobsRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PushManifestAndBlobsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PushManifestAndBlobsRequest) ProtoMessage() {}

func (x *PushManifestAndBlobsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *PushManifestAndBlobsRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *PushManifestAndBlobsRequest) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

func (x *PushManifestAndBlobsRequest) GetManifest() *v1alpha1.Blob {
	if x != nil {
		return x.xxx_hidden_Manifest
	}
	return nil
}

func (x *PushManifestAndBlobsRequest) GetBlobs() []*v1alpha1.Blob {
	if x != nil {
		if x.xxx_hidden_Blobs != nil {
			return *x.xxx_hidden_Blobs
		}
	}
	return nil
}

func (x *PushManifestAndBlobsRequest) GetTags() []string {
	if x != nil {
		return x.xxx_hidden_Tags
	}
	return nil
}

func (x *PushManifestAndBlobsRequest) GetDraftName() string {
	if x != nil {
		return x.xxx_hidden_DraftName
	}
	return ""
}

func (x *PushManifestAndBlobsRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *PushManifestAndBlobsRequest) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

func (x *PushManifestAndBlobsRequest) SetManifest(v *v1alpha1.Blob) {
	x.xxx_hidden_Manifest = v
}

func (x *PushManifestAndBlobsRequest) SetBlobs(v []*v1alpha1.Blob) {
	x.xxx_hidden_Blobs = &v
}

func (x *PushManifestAndBlobsRequest) SetTags(v []string) {
	x.xxx_hidden_Tags = v
}

func (x *PushManifestAndBlobsRequest) SetDraftName(v string) {
	x.xxx_hidden_DraftName = v
}

func (x *PushManifestAndBlobsRequest) HasManifest() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Manifest != nil
}

func (x *PushManifestAndBlobsRequest) ClearManifest() {
	x.xxx_hidden_Manifest = nil
}

type PushManifestAndBlobsRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner      string
	Repository string
	// Manifest with all the module files being pushed.
	// The content of the manifest blob is a text encoding of an ordered list of unique paths, each path encoded as:
	//
	//	<digest_type>:<digest>[SP][SP]<path>[LF]
	//
	// The only current supported digest type is 'shake256'. The shake256 digest consists of 64 bytes of lowercase hex
	// encoded output of SHAKE256. See buf.alpha.module.v1alpha1.Digest for more details.
	Manifest *v1alpha1.Blob
	// Referenced blobs in the manifest. Keep in mind there is not necessarily one
	// blob per file, but one blob per digest, so for files with exactly the same
	// content, you can send just one blob.
	Blobs []*v1alpha1.Blob
	// Optional; if provided, the provided tags
	// are created for the pushed commit.
	Tags []string
	// If non-empty, the push creates a draft commit with this name.
	DraftName string
}

func (b0 PushManifestAndBlobsRequest_builder) Build() *PushManifestAndBlobsRequest {
	m0 := &PushManifestAndBlobsRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Manifest = b.Manifest
	x.xxx_hidden_Blobs = &b.Blobs
	x.xxx_hidden_Tags = b.Tags
	x.xxx_hidden_DraftName = b.DraftName
	return m0
}

// PushManifestAndBlobsResponse is the pushed module pin, local to the used
// remote.
type PushManifestAndBlobsResponse struct {
	state                     protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_LocalModulePin *LocalModulePin        `protobuf:"bytes,1,opt,name=local_module_pin,json=localModulePin,proto3"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *PushManifestAndBlobsResponse) Reset() {
	*x = PushManifestAndBlobsResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PushManifestAndBlobsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PushManifestAndBlobsResponse) ProtoMessage() {}

func (x *PushManifestAndBlobsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_push_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *PushManifestAndBlobsResponse) GetLocalModulePin() *LocalModulePin {
	if x != nil {
		return x.xxx_hidden_LocalModulePin
	}
	return nil
}

func (x *PushManifestAndBlobsResponse) SetLocalModulePin(v *LocalModulePin) {
	x.xxx_hidden_LocalModulePin = v
}

func (x *PushManifestAndBlobsResponse) HasLocalModulePin() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_LocalModulePin != nil
}

func (x *PushManifestAndBlobsResponse) ClearLocalModulePin() {
	x.xxx_hidden_LocalModulePin = nil
}

type PushManifestAndBlobsResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	LocalModulePin *LocalModulePin
}

func (b0 PushManifestAndBlobsResponse_builder) Build() *PushManifestAndBlobsResponse {
	m0 := &PushManifestAndBlobsResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_LocalModulePin = b.LocalModulePin
	return m0
}

var File_buf_alpha_registry_v1alpha1_push_proto protoreflect.FileDescriptor

const file_buf_alpha_registry_v1alpha1_push_proto_rawDesc = "" +
	"\n" +
	"&buf/alpha/registry/v1alpha1/push.proto\x12\x1bbuf.alpha.registry.v1alpha1\x1a&buf/alpha/module/v1alpha1/module.proto\x1a(buf/alpha/registry/v1alpha1/module.proto\"\xe9\x01\n" +
	"\vPushRequest\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12\x1a\n" +
	"\x06branch\x18\x03 \x01(\tB\x02\x18\x01R\x06branch\x129\n" +
	"\x06module\x18\x04 \x01(\v2!.buf.alpha.module.v1alpha1.ModuleR\x06module\x12\x12\n" +
	"\x04tags\x18\x05 \x03(\tR\x04tags\x12\x1a\n" +
	"\x06tracks\x18\x06 \x03(\tB\x02\x18\x01R\x06tracks\x12\x1d\n" +
	"\n" +
	"draft_name\x18\a \x01(\tR\tdraftName\"e\n" +
	"\fPushResponse\x12U\n" +
	"\x10local_module_pin\x18\x05 \x01(\v2+.buf.alpha.registry.v1alpha1.LocalModulePinR\x0elocalModulePin\"\xfa\x01\n" +
	"\x1bPushManifestAndBlobsRequest\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12;\n" +
	"\bmanifest\x18\x03 \x01(\v2\x1f.buf.alpha.module.v1alpha1.BlobR\bmanifest\x125\n" +
	"\x05blobs\x18\x04 \x03(\v2\x1f.buf.alpha.module.v1alpha1.BlobR\x05blobs\x12\x12\n" +
	"\x04tags\x18\x05 \x03(\tR\x04tags\x12\x1d\n" +
	"\n" +
	"draft_name\x18\x06 \x01(\tR\tdraftName\"u\n" +
	"\x1cPushManifestAndBlobsResponse\x12U\n" +
	"\x10local_module_pin\x18\x01 \x01(\v2+.buf.alpha.registry.v1alpha1.LocalModulePinR\x0elocalModulePin2\x82\x02\n" +
	"\vPushService\x12`\n" +
	"\x04Push\x12(.buf.alpha.registry.v1alpha1.PushRequest\x1a).buf.alpha.registry.v1alpha1.PushResponse\"\x03\x90\x02\x02\x12\x90\x01\n" +
	"\x14PushManifestAndBlobs\x128.buf.alpha.registry.v1alpha1.PushManifestAndBlobsRequest\x1a9.buf.alpha.registry.v1alpha1.PushManifestAndBlobsResponse\"\x03\x90\x02\x02B\x96\x02\n" +
	"\x1fcom.buf.alpha.registry.v1alpha1B\tPushProtoP\x01ZYgithub.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1;registryv1alpha1\xa2\x02\x03BAR\xaa\x02\x1bBuf.Alpha.Registry.V1alpha1\xca\x02\x1bBuf\\Alpha\\Registry\\V1alpha1\xe2\x02'Buf\\Alpha\\Registry\\V1alpha1\\GPBMetadata\xea\x02\x1eBuf::Alpha::Registry::V1alpha1b\x06proto3"

var file_buf_alpha_registry_v1alpha1_push_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_buf_alpha_registry_v1alpha1_push_proto_goTypes = []any{
	(*PushRequest)(nil),                  // 0: buf.alpha.registry.v1alpha1.PushRequest
	(*PushResponse)(nil),                 // 1: buf.alpha.registry.v1alpha1.PushResponse
	(*PushManifestAndBlobsRequest)(nil),  // 2: buf.alpha.registry.v1alpha1.PushManifestAndBlobsRequest
	(*PushManifestAndBlobsResponse)(nil), // 3: buf.alpha.registry.v1alpha1.PushManifestAndBlobsResponse
	(*v1alpha1.Module)(nil),              // 4: buf.alpha.module.v1alpha1.Module
	(*LocalModulePin)(nil),               // 5: buf.alpha.registry.v1alpha1.LocalModulePin
	(*v1alpha1.Blob)(nil),                // 6: buf.alpha.module.v1alpha1.Blob
}
var file_buf_alpha_registry_v1alpha1_push_proto_depIdxs = []int32{
	4, // 0: buf.alpha.registry.v1alpha1.PushRequest.module:type_name -> buf.alpha.module.v1alpha1.Module
	5, // 1: buf.alpha.registry.v1alpha1.PushResponse.local_module_pin:type_name -> buf.alpha.registry.v1alpha1.LocalModulePin
	6, // 2: buf.alpha.registry.v1alpha1.PushManifestAndBlobsRequest.manifest:type_name -> buf.alpha.module.v1alpha1.Blob
	6, // 3: buf.alpha.registry.v1alpha1.PushManifestAndBlobsRequest.blobs:type_name -> buf.alpha.module.v1alpha1.Blob
	5, // 4: buf.alpha.registry.v1alpha1.PushManifestAndBlobsResponse.local_module_pin:type_name -> buf.alpha.registry.v1alpha1.LocalModulePin
	0, // 5: buf.alpha.registry.v1alpha1.PushService.Push:input_type -> buf.alpha.registry.v1alpha1.PushRequest
	2, // 6: buf.alpha.registry.v1alpha1.PushService.PushManifestAndBlobs:input_type -> buf.alpha.registry.v1alpha1.PushManifestAndBlobsRequest
	1, // 7: buf.alpha.registry.v1alpha1.PushService.Push:output_type -> buf.alpha.registry.v1alpha1.PushResponse
	3, // 8: buf.alpha.registry.v1alpha1.PushService.PushManifestAndBlobs:output_type -> buf.alpha.registry.v1alpha1.PushManifestAndBlobsResponse
	7, // [7:9] is the sub-list for method output_type
	5, // [5:7] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_push_proto_init() }
func file_buf_alpha_registry_v1alpha1_push_proto_init() {
	if File_buf_alpha_registry_v1alpha1_push_proto != nil {
		return
	}
	file_buf_alpha_registry_v1alpha1_module_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_registry_v1alpha1_push_proto_rawDesc), len(file_buf_alpha_registry_v1alpha1_push_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_push_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_push_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_push_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_push_proto = out.File
	file_buf_alpha_registry_v1alpha1_push_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_push_proto_depIdxs = nil
}
