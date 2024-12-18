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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.0
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/recommendation.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// RecommendedRepository is the information about a repository needed to link to
// its owner page.
type RecommendedRepository struct {
	state                   protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner        string                 `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	xxx_hidden_Name         string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_CreateTime   *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	xxx_hidden_Description  string                 `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	xxx_hidden_RepositoryId string                 `protobuf:"bytes,5,opt,name=repository_id,json=repositoryId,proto3" json:"repository_id,omitempty"`
	unknownFields           protoimpl.UnknownFields
	sizeCache               protoimpl.SizeCache
}

func (x *RecommendedRepository) Reset() {
	*x = RecommendedRepository{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RecommendedRepository) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecommendedRepository) ProtoMessage() {}

func (x *RecommendedRepository) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *RecommendedRepository) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *RecommendedRepository) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *RecommendedRepository) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.xxx_hidden_CreateTime
	}
	return nil
}

func (x *RecommendedRepository) GetDescription() string {
	if x != nil {
		return x.xxx_hidden_Description
	}
	return ""
}

func (x *RecommendedRepository) GetRepositoryId() string {
	if x != nil {
		return x.xxx_hidden_RepositoryId
	}
	return ""
}

func (x *RecommendedRepository) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *RecommendedRepository) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *RecommendedRepository) SetCreateTime(v *timestamppb.Timestamp) {
	x.xxx_hidden_CreateTime = v
}

func (x *RecommendedRepository) SetDescription(v string) {
	x.xxx_hidden_Description = v
}

func (x *RecommendedRepository) SetRepositoryId(v string) {
	x.xxx_hidden_RepositoryId = v
}

func (x *RecommendedRepository) HasCreateTime() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_CreateTime != nil
}

func (x *RecommendedRepository) ClearCreateTime() {
	x.xxx_hidden_CreateTime = nil
}

type RecommendedRepository_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner        string
	Name         string
	CreateTime   *timestamppb.Timestamp
	Description  string
	RepositoryId string
}

func (b0 RecommendedRepository_builder) Build() *RecommendedRepository {
	m0 := &RecommendedRepository{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_CreateTime = b.CreateTime
	x.xxx_hidden_Description = b.Description
	x.xxx_hidden_RepositoryId = b.RepositoryId
	return m0
}

// SetRecommendedResource is the information needed to configure a resource recommendation
type SetRecommendedResource struct {
	state            protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner string                 `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	xxx_hidden_Name  string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *SetRecommendedResource) Reset() {
	*x = SetRecommendedResource{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetRecommendedResource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetRecommendedResource) ProtoMessage() {}

func (x *SetRecommendedResource) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *SetRecommendedResource) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *SetRecommendedResource) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *SetRecommendedResource) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *SetRecommendedResource) SetName(v string) {
	x.xxx_hidden_Name = v
}

type SetRecommendedResource_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Owner string
	Name  string
}

func (b0 SetRecommendedResource_builder) Build() *SetRecommendedResource {
	m0 := &SetRecommendedResource{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Name = b.Name
	return m0
}

type RecommendedRepositoriesRequest struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RecommendedRepositoriesRequest) Reset() {
	*x = RecommendedRepositoriesRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RecommendedRepositoriesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecommendedRepositoriesRequest) ProtoMessage() {}

func (x *RecommendedRepositoriesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type RecommendedRepositoriesRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 RecommendedRepositoriesRequest_builder) Build() *RecommendedRepositoriesRequest {
	m0 := &RecommendedRepositoriesRequest{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

type RecommendedRepositoriesResponse struct {
	state                   protoimpl.MessageState    `protogen:"opaque.v1"`
	xxx_hidden_Repositories *[]*RecommendedRepository `protobuf:"bytes,1,rep,name=repositories,proto3" json:"repositories,omitempty"`
	unknownFields           protoimpl.UnknownFields
	sizeCache               protoimpl.SizeCache
}

func (x *RecommendedRepositoriesResponse) Reset() {
	*x = RecommendedRepositoriesResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RecommendedRepositoriesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RecommendedRepositoriesResponse) ProtoMessage() {}

func (x *RecommendedRepositoriesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *RecommendedRepositoriesResponse) GetRepositories() []*RecommendedRepository {
	if x != nil {
		if x.xxx_hidden_Repositories != nil {
			return *x.xxx_hidden_Repositories
		}
	}
	return nil
}

func (x *RecommendedRepositoriesResponse) SetRepositories(v []*RecommendedRepository) {
	x.xxx_hidden_Repositories = &v
}

type RecommendedRepositoriesResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Repositories []*RecommendedRepository
}

func (b0 RecommendedRepositoriesResponse_builder) Build() *RecommendedRepositoriesResponse {
	m0 := &RecommendedRepositoriesResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Repositories = &b.Repositories
	return m0
}

type ListRecommendedResourcesRequest struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListRecommendedResourcesRequest) Reset() {
	*x = ListRecommendedResourcesRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListRecommendedResourcesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRecommendedResourcesRequest) ProtoMessage() {}

func (x *ListRecommendedResourcesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type ListRecommendedResourcesRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 ListRecommendedResourcesRequest_builder) Build() *ListRecommendedResourcesRequest {
	m0 := &ListRecommendedResourcesRequest{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

type ListRecommendedResourcesResponse struct {
	state                protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Resources *[]*Resource           `protobuf:"bytes,1,rep,name=resources,proto3" json:"resources,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *ListRecommendedResourcesResponse) Reset() {
	*x = ListRecommendedResourcesResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListRecommendedResourcesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRecommendedResourcesResponse) ProtoMessage() {}

func (x *ListRecommendedResourcesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ListRecommendedResourcesResponse) GetResources() []*Resource {
	if x != nil {
		if x.xxx_hidden_Resources != nil {
			return *x.xxx_hidden_Resources
		}
	}
	return nil
}

func (x *ListRecommendedResourcesResponse) SetResources(v []*Resource) {
	x.xxx_hidden_Resources = &v
}

type ListRecommendedResourcesResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Resources []*Resource
}

func (b0 ListRecommendedResourcesResponse_builder) Build() *ListRecommendedResourcesResponse {
	m0 := &ListRecommendedResourcesResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Resources = &b.Resources
	return m0
}

type SetRecommendedResourcesRequest struct {
	state                protoimpl.MessageState     `protogen:"opaque.v1"`
	xxx_hidden_Resources *[]*SetRecommendedResource `protobuf:"bytes,1,rep,name=resources,proto3" json:"resources,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *SetRecommendedResourcesRequest) Reset() {
	*x = SetRecommendedResourcesRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetRecommendedResourcesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetRecommendedResourcesRequest) ProtoMessage() {}

func (x *SetRecommendedResourcesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *SetRecommendedResourcesRequest) GetResources() []*SetRecommendedResource {
	if x != nil {
		if x.xxx_hidden_Resources != nil {
			return *x.xxx_hidden_Resources
		}
	}
	return nil
}

func (x *SetRecommendedResourcesRequest) SetResources(v []*SetRecommendedResource) {
	x.xxx_hidden_Resources = &v
}

type SetRecommendedResourcesRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Resources []*SetRecommendedResource
}

func (b0 SetRecommendedResourcesRequest_builder) Build() *SetRecommendedResourcesRequest {
	m0 := &SetRecommendedResourcesRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Resources = &b.Resources
	return m0
}

type SetRecommendedResourcesResponse struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SetRecommendedResourcesResponse) Reset() {
	*x = SetRecommendedResourcesResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SetRecommendedResourcesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetRecommendedResourcesResponse) ProtoMessage() {}

func (x *SetRecommendedResourcesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type SetRecommendedResourcesResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 SetRecommendedResourcesResponse_builder) Build() *SetRecommendedResourcesResponse {
	m0 := &SetRecommendedResourcesResponse{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

var File_buf_alpha_registry_v1alpha1_recommendation_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_recommendation_proto_rawDesc = []byte{
	0x0a, 0x30, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65,
	0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a,
	0x2a, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc5, 0x01, 0x0a,
	0x15, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x12, 0x3b, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x20, 0x0a,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x23, 0x0a, 0x0d, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x69, 0x64,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x49, 0x64, 0x22, 0x42, 0x0a, 0x16, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d,
	0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x20, 0x0a, 0x1e, 0x52, 0x65, 0x63, 0x6f,
	0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x79, 0x0a, 0x1f, 0x52, 0x65,
	0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x56, 0x0a,
	0x0c, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x52, 0x0c, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x69, 0x65, 0x73, 0x22, 0x21, 0x0a, 0x1f, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x63,
	0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x67, 0x0a, 0x20, 0x4c, 0x69, 0x73, 0x74,
	0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x43, 0x0a, 0x09,
	0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x25, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x22, 0x73, 0x0a, 0x1e, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x51, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x33, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x52, 0x09, 0x72, 0x65, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x22, 0x21, 0x0a, 0x1f, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63,
	0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0xee, 0x03, 0x0a, 0x15, 0x52, 0x65,
	0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x99, 0x01, 0x0a, 0x17, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x65, 0x73, 0x12,
	0x3b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65,
	0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3c, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x63, 0x6f, 0x6d,
	0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x69,
	0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x12,
	0x9c, 0x01, 0x0a, 0x18, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e,
	0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x12, 0x3c, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x52,
	0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3d, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x63,
	0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x12, 0x99,
	0x01, 0x0a, 0x17, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65,
	0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x12, 0x3b, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f,
	0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3c, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65,
	0x6e, 0x64, 0x65, 0x64, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x02, 0x42, 0xa0, 0x02, 0x0a, 0x1f, 0x63,
	0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x13,
	0x52, 0x65, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70,
	0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61,
	0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e, 0x42,
	0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_buf_alpha_registry_v1alpha1_recommendation_proto_goTypes = []any{
	(*RecommendedRepository)(nil),            // 0: buf.alpha.registry.v1alpha1.RecommendedRepository
	(*SetRecommendedResource)(nil),           // 1: buf.alpha.registry.v1alpha1.SetRecommendedResource
	(*RecommendedRepositoriesRequest)(nil),   // 2: buf.alpha.registry.v1alpha1.RecommendedRepositoriesRequest
	(*RecommendedRepositoriesResponse)(nil),  // 3: buf.alpha.registry.v1alpha1.RecommendedRepositoriesResponse
	(*ListRecommendedResourcesRequest)(nil),  // 4: buf.alpha.registry.v1alpha1.ListRecommendedResourcesRequest
	(*ListRecommendedResourcesResponse)(nil), // 5: buf.alpha.registry.v1alpha1.ListRecommendedResourcesResponse
	(*SetRecommendedResourcesRequest)(nil),   // 6: buf.alpha.registry.v1alpha1.SetRecommendedResourcesRequest
	(*SetRecommendedResourcesResponse)(nil),  // 7: buf.alpha.registry.v1alpha1.SetRecommendedResourcesResponse
	(*timestamppb.Timestamp)(nil),            // 8: google.protobuf.Timestamp
	(*Resource)(nil),                         // 9: buf.alpha.registry.v1alpha1.Resource
}
var file_buf_alpha_registry_v1alpha1_recommendation_proto_depIdxs = []int32{
	8, // 0: buf.alpha.registry.v1alpha1.RecommendedRepository.create_time:type_name -> google.protobuf.Timestamp
	0, // 1: buf.alpha.registry.v1alpha1.RecommendedRepositoriesResponse.repositories:type_name -> buf.alpha.registry.v1alpha1.RecommendedRepository
	9, // 2: buf.alpha.registry.v1alpha1.ListRecommendedResourcesResponse.resources:type_name -> buf.alpha.registry.v1alpha1.Resource
	1, // 3: buf.alpha.registry.v1alpha1.SetRecommendedResourcesRequest.resources:type_name -> buf.alpha.registry.v1alpha1.SetRecommendedResource
	2, // 4: buf.alpha.registry.v1alpha1.RecommendationService.RecommendedRepositories:input_type -> buf.alpha.registry.v1alpha1.RecommendedRepositoriesRequest
	4, // 5: buf.alpha.registry.v1alpha1.RecommendationService.ListRecommendedResources:input_type -> buf.alpha.registry.v1alpha1.ListRecommendedResourcesRequest
	6, // 6: buf.alpha.registry.v1alpha1.RecommendationService.SetRecommendedResources:input_type -> buf.alpha.registry.v1alpha1.SetRecommendedResourcesRequest
	3, // 7: buf.alpha.registry.v1alpha1.RecommendationService.RecommendedRepositories:output_type -> buf.alpha.registry.v1alpha1.RecommendedRepositoriesResponse
	5, // 8: buf.alpha.registry.v1alpha1.RecommendationService.ListRecommendedResources:output_type -> buf.alpha.registry.v1alpha1.ListRecommendedResourcesResponse
	7, // 9: buf.alpha.registry.v1alpha1.RecommendationService.SetRecommendedResources:output_type -> buf.alpha.registry.v1alpha1.SetRecommendedResourcesResponse
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_recommendation_proto_init() }
func file_buf_alpha_registry_v1alpha1_recommendation_proto_init() {
	if File_buf_alpha_registry_v1alpha1_recommendation_proto != nil {
		return
	}
	file_buf_alpha_registry_v1alpha1_resource_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_recommendation_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_recommendation_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_recommendation_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_recommendation_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_recommendation_proto = out.File
	file_buf_alpha_registry_v1alpha1_recommendation_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_recommendation_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_recommendation_proto_depIdxs = nil
}
