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
// source: buf/alpha/registry/v1alpha1/resource.proto

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

type Resource struct {
	state               protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Resource isResource_Resource    `protobuf_oneof:"resource"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

func (x *Resource) Reset() {
	*x = Resource{}
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Resource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Resource) ProtoMessage() {}

func (x *Resource) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Resource) GetRepository() *Repository {
	if x != nil {
		if x, ok := x.xxx_hidden_Resource.(*resource_Repository); ok {
			return x.Repository
		}
	}
	return nil
}

func (x *Resource) GetPlugin() *CuratedPlugin {
	if x != nil {
		if x, ok := x.xxx_hidden_Resource.(*resource_Plugin); ok {
			return x.Plugin
		}
	}
	return nil
}

func (x *Resource) SetRepository(v *Repository) {
	if v == nil {
		x.xxx_hidden_Resource = nil
		return
	}
	x.xxx_hidden_Resource = &resource_Repository{v}
}

func (x *Resource) SetPlugin(v *CuratedPlugin) {
	if v == nil {
		x.xxx_hidden_Resource = nil
		return
	}
	x.xxx_hidden_Resource = &resource_Plugin{v}
}

func (x *Resource) HasResource() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Resource != nil
}

func (x *Resource) HasRepository() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Resource.(*resource_Repository)
	return ok
}

func (x *Resource) HasPlugin() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Resource.(*resource_Plugin)
	return ok
}

func (x *Resource) ClearResource() {
	x.xxx_hidden_Resource = nil
}

func (x *Resource) ClearRepository() {
	if _, ok := x.xxx_hidden_Resource.(*resource_Repository); ok {
		x.xxx_hidden_Resource = nil
	}
}

func (x *Resource) ClearPlugin() {
	if _, ok := x.xxx_hidden_Resource.(*resource_Plugin); ok {
		x.xxx_hidden_Resource = nil
	}
}

const Resource_Resource_not_set_case case_Resource_Resource = 0
const Resource_Repository_case case_Resource_Resource = 1
const Resource_Plugin_case case_Resource_Resource = 2

func (x *Resource) WhichResource() case_Resource_Resource {
	if x == nil {
		return Resource_Resource_not_set_case
	}
	switch x.xxx_hidden_Resource.(type) {
	case *resource_Repository:
		return Resource_Repository_case
	case *resource_Plugin:
		return Resource_Plugin_case
	default:
		return Resource_Resource_not_set_case
	}
}

type Resource_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Fields of oneof xxx_hidden_Resource:
	Repository *Repository
	Plugin     *CuratedPlugin
	// -- end of xxx_hidden_Resource
}

func (b0 Resource_builder) Build() *Resource {
	m0 := &Resource{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Repository != nil {
		x.xxx_hidden_Resource = &resource_Repository{b.Repository}
	}
	if b.Plugin != nil {
		x.xxx_hidden_Resource = &resource_Plugin{b.Plugin}
	}
	return m0
}

type case_Resource_Resource protoreflect.FieldNumber

func (x case_Resource_Resource) String() string {
	md := file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[0].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isResource_Resource interface {
	isResource_Resource()
}

type resource_Repository struct {
	Repository *Repository `protobuf:"bytes,1,opt,name=repository,proto3,oneof"`
}

type resource_Plugin struct {
	Plugin *CuratedPlugin `protobuf:"bytes,2,opt,name=plugin,proto3,oneof"`
}

func (*resource_Repository) isResource_Resource() {}

func (*resource_Plugin) isResource_Resource() {}

type GetResourceByNameRequest struct {
	state            protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Name  string                 `protobuf:"bytes,2,opt,name=name,proto3"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *GetResourceByNameRequest) Reset() {
	*x = GetResourceByNameRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetResourceByNameRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetResourceByNameRequest) ProtoMessage() {}

func (x *GetResourceByNameRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetResourceByNameRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *GetResourceByNameRequest) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *GetResourceByNameRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *GetResourceByNameRequest) SetName(v string) {
	x.xxx_hidden_Name = v
}

type GetResourceByNameRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Owner of the requested resource.
	Owner string
	// Name of the requested resource.
	Name string
}

func (b0 GetResourceByNameRequest_builder) Build() *GetResourceByNameRequest {
	m0 := &GetResourceByNameRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Name = b.Name
	return m0
}

type GetResourceByNameResponse struct {
	state               protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Resource *Resource              `protobuf:"bytes,1,opt,name=resource,proto3"`
	unknownFields       protoimpl.UnknownFields
	sizeCache           protoimpl.SizeCache
}

func (x *GetResourceByNameResponse) Reset() {
	*x = GetResourceByNameResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetResourceByNameResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetResourceByNameResponse) ProtoMessage() {}

func (x *GetResourceByNameResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetResourceByNameResponse) GetResource() *Resource {
	if x != nil {
		return x.xxx_hidden_Resource
	}
	return nil
}

func (x *GetResourceByNameResponse) SetResource(v *Resource) {
	x.xxx_hidden_Resource = v
}

func (x *GetResourceByNameResponse) HasResource() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Resource != nil
}

func (x *GetResourceByNameResponse) ClearResource() {
	x.xxx_hidden_Resource = nil
}

type GetResourceByNameResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Resource *Resource
}

func (b0 GetResourceByNameResponse_builder) Build() *GetResourceByNameResponse {
	m0 := &GetResourceByNameResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Resource = b.Resource
	return m0
}

var File_buf_alpha_registry_v1alpha1_resource_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_resource_proto_rawDesc = string([]byte{
	0x0a, 0x2a, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x31, 0x62, 0x75, 0x66, 0x2f, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x5f, 0x63, 0x75,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2c, 0x62, 0x75,
	0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xa7, 0x01, 0x0a, 0x08, 0x52,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x49, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x48, 0x00, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x12, 0x44, 0x0a, 0x06, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2e, 0x43, 0x75, 0x72, 0x61, 0x74, 0x65, 0x64, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x48, 0x00,
	0x52, 0x06, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x42, 0x0a, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x22, 0x44, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x5e, 0x0a, 0x19, 0x47, 0x65,
	0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x41, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x52, 0x08, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x32, 0x9b, 0x01, 0x0a, 0x0f, 0x52,
	0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x87,
	0x01, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x35, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79,
	0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x36, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x42, 0x9a, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0d, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69,
	0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67,
	0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02,
	0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42,
	0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66,
	0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c,
	0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68,
	0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_buf_alpha_registry_v1alpha1_resource_proto_goTypes = []any{
	(*Resource)(nil),                  // 0: buf.alpha.registry.v1alpha1.Resource
	(*GetResourceByNameRequest)(nil),  // 1: buf.alpha.registry.v1alpha1.GetResourceByNameRequest
	(*GetResourceByNameResponse)(nil), // 2: buf.alpha.registry.v1alpha1.GetResourceByNameResponse
	(*Repository)(nil),                // 3: buf.alpha.registry.v1alpha1.Repository
	(*CuratedPlugin)(nil),             // 4: buf.alpha.registry.v1alpha1.CuratedPlugin
}
var file_buf_alpha_registry_v1alpha1_resource_proto_depIdxs = []int32{
	3, // 0: buf.alpha.registry.v1alpha1.Resource.repository:type_name -> buf.alpha.registry.v1alpha1.Repository
	4, // 1: buf.alpha.registry.v1alpha1.Resource.plugin:type_name -> buf.alpha.registry.v1alpha1.CuratedPlugin
	0, // 2: buf.alpha.registry.v1alpha1.GetResourceByNameResponse.resource:type_name -> buf.alpha.registry.v1alpha1.Resource
	1, // 3: buf.alpha.registry.v1alpha1.ResourceService.GetResourceByName:input_type -> buf.alpha.registry.v1alpha1.GetResourceByNameRequest
	2, // 4: buf.alpha.registry.v1alpha1.ResourceService.GetResourceByName:output_type -> buf.alpha.registry.v1alpha1.GetResourceByNameResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_resource_proto_init() }
func file_buf_alpha_registry_v1alpha1_resource_proto_init() {
	if File_buf_alpha_registry_v1alpha1_resource_proto != nil {
		return
	}
	file_buf_alpha_registry_v1alpha1_plugin_curation_proto_init()
	file_buf_alpha_registry_v1alpha1_repository_proto_init()
	file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes[0].OneofWrappers = []any{
		(*resource_Repository)(nil),
		(*resource_Plugin)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_registry_v1alpha1_resource_proto_rawDesc), len(file_buf_alpha_registry_v1alpha1_resource_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_resource_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_resource_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_resource_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_resource_proto = out.File
	file_buf_alpha_registry_v1alpha1_resource_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_resource_proto_depIdxs = nil
}
