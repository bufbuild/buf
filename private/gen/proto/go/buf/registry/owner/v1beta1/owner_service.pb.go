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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: buf/registry/owner/v1beta1/owner_service.proto

package ownerv1beta1

import (
	_ "github.com/bufbuild/buf/private/gen/proto/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetOwnersRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The Users or Organizations to request.
	OwnerRefs []*OwnerRef `protobuf:"bytes,1,rep,name=owner_refs,json=ownerRefs,proto3" json:"owner_refs,omitempty"`
}

func (x *GetOwnersRequest) Reset() {
	*x = GetOwnersRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetOwnersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOwnersRequest) ProtoMessage() {}

func (x *GetOwnersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOwnersRequest.ProtoReflect.Descriptor instead.
func (*GetOwnersRequest) Descriptor() ([]byte, []int) {
	return file_buf_registry_owner_v1beta1_owner_service_proto_rawDescGZIP(), []int{0}
}

func (x *GetOwnersRequest) GetOwnerRefs() []*OwnerRef {
	if x != nil {
		return x.OwnerRefs
	}
	return nil
}

type GetOwnersResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The retreived Users or Organizations in the same order as requested.
	Owners []*Owner `protobuf:"bytes,1,rep,name=owners,proto3" json:"owners,omitempty"`
}

func (x *GetOwnersResponse) Reset() {
	*x = GetOwnersResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetOwnersResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOwnersResponse) ProtoMessage() {}

func (x *GetOwnersResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOwnersResponse.ProtoReflect.Descriptor instead.
func (*GetOwnersResponse) Descriptor() ([]byte, []int) {
	return file_buf_registry_owner_v1beta1_owner_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetOwnersResponse) GetOwners() []*Owner {
	if x != nil {
		return x.Owners
	}
	return nil
}

var File_buf_registry_owner_v1beta1_owner_service_proto protoreflect.FileDescriptor

var file_buf_registry_owner_v1beta1_owner_service_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x62, 0x75, 0x66, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x6f, 0x77, 0x6e,
	0x65, 0x72, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x1a, 0x62, 0x75, 0x66, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x1a, 0x26, 0x62, 0x75,
	0x66, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x6f, 0x77, 0x6e, 0x65, 0x72,
	0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x61, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x73, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x4d, 0x0a, 0x0a, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x5f, 0x72,
	0x65, 0x66, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x52, 0x65, 0x66, 0x42,
	0x08, 0xba, 0x48, 0x05, 0x92, 0x01, 0x02, 0x08, 0x01, 0x52, 0x09, 0x6f, 0x77, 0x6e, 0x65, 0x72,
	0x52, 0x65, 0x66, 0x73, 0x22, 0x58, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x43, 0x0a, 0x06, 0x6f, 0x77, 0x6e,
	0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x42, 0x08, 0xba, 0x48,
	0x05, 0x92, 0x01, 0x02, 0x08, 0x01, 0x52, 0x06, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x73, 0x32, 0x7d,
	0x0a, 0x0c, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6d,
	0x0a, 0x09, 0x47, 0x65, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x73, 0x12, 0x2c, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x6f, 0x77, 0x6e, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4f, 0x77, 0x6e, 0x65,
	0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x73,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x42, 0x94, 0x02,
	0x0a, 0x1e, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31,
	0x42, 0x11, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x54, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70,
	0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x3b, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x52,
	0x4f, 0xaa, 0x02, 0x1a, 0x42, 0x75, 0x66, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x2e, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xca, 0x02,
	0x1a, 0x42, 0x75, 0x66, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x4f, 0x77,
	0x6e, 0x65, 0x72, 0x5c, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0xe2, 0x02, 0x26, 0x42, 0x75,
	0x66, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x4f, 0x77, 0x6e, 0x65, 0x72,
	0x5c, 0x56, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1d, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x3a, 0x3a, 0x56, 0x31, 0x62,
	0x65, 0x74, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_registry_owner_v1beta1_owner_service_proto_rawDescOnce sync.Once
	file_buf_registry_owner_v1beta1_owner_service_proto_rawDescData = file_buf_registry_owner_v1beta1_owner_service_proto_rawDesc
)

func file_buf_registry_owner_v1beta1_owner_service_proto_rawDescGZIP() []byte {
	file_buf_registry_owner_v1beta1_owner_service_proto_rawDescOnce.Do(func() {
		file_buf_registry_owner_v1beta1_owner_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_registry_owner_v1beta1_owner_service_proto_rawDescData)
	})
	return file_buf_registry_owner_v1beta1_owner_service_proto_rawDescData
}

var file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_registry_owner_v1beta1_owner_service_proto_goTypes = []interface{}{
	(*GetOwnersRequest)(nil),  // 0: buf.registry.owner.v1beta1.GetOwnersRequest
	(*GetOwnersResponse)(nil), // 1: buf.registry.owner.v1beta1.GetOwnersResponse
	(*OwnerRef)(nil),          // 2: buf.registry.owner.v1beta1.OwnerRef
	(*Owner)(nil),             // 3: buf.registry.owner.v1beta1.Owner
}
var file_buf_registry_owner_v1beta1_owner_service_proto_depIdxs = []int32{
	2, // 0: buf.registry.owner.v1beta1.GetOwnersRequest.owner_refs:type_name -> buf.registry.owner.v1beta1.OwnerRef
	3, // 1: buf.registry.owner.v1beta1.GetOwnersResponse.owners:type_name -> buf.registry.owner.v1beta1.Owner
	0, // 2: buf.registry.owner.v1beta1.OwnerService.GetOwners:input_type -> buf.registry.owner.v1beta1.GetOwnersRequest
	1, // 3: buf.registry.owner.v1beta1.OwnerService.GetOwners:output_type -> buf.registry.owner.v1beta1.GetOwnersResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_buf_registry_owner_v1beta1_owner_service_proto_init() }
func file_buf_registry_owner_v1beta1_owner_service_proto_init() {
	if File_buf_registry_owner_v1beta1_owner_service_proto != nil {
		return
	}
	file_buf_registry_owner_v1beta1_owner_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetOwnersRequest); i {
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
		file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetOwnersResponse); i {
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
			RawDescriptor: file_buf_registry_owner_v1beta1_owner_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_registry_owner_v1beta1_owner_service_proto_goTypes,
		DependencyIndexes: file_buf_registry_owner_v1beta1_owner_service_proto_depIdxs,
		MessageInfos:      file_buf_registry_owner_v1beta1_owner_service_proto_msgTypes,
	}.Build()
	File_buf_registry_owner_v1beta1_owner_service_proto = out.File
	file_buf_registry_owner_v1beta1_owner_service_proto_rawDesc = nil
	file_buf_registry_owner_v1beta1_owner_service_proto_goTypes = nil
	file_buf_registry_owner_v1beta1_owner_service_proto_depIdxs = nil
}
