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
// source: buf/alpha/registry/v1alpha1/repository_branch.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RepositoryBranch struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// primary key, unique, immutable
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The name of the repository branch.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// The name of the latest commit on the branch.
	LatestCommitName string `protobuf:"bytes,3,opt,name=latest_commit_name,json=latestCommitName,proto3" json:"latest_commit_name,omitempty"`
	// is_main_branch denotes whether this branch is considered the main branch of the repository.
	IsMainBranch bool `protobuf:"varint,4,opt,name=is_main_branch,json=isMainBranch,proto3" json:"is_main_branch,omitempty"`
	// The last update time of the branch.
	LastUpdateTime *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=last_update_time,json=lastUpdateTime,proto3" json:"last_update_time,omitempty"`
}

func (x *RepositoryBranch) Reset() {
	*x = RepositoryBranch{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RepositoryBranch) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryBranch) ProtoMessage() {}

func (x *RepositoryBranch) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepositoryBranch.ProtoReflect.Descriptor instead.
func (*RepositoryBranch) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP(), []int{0}
}

func (x *RepositoryBranch) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *RepositoryBranch) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *RepositoryBranch) GetLatestCommitName() string {
	if x != nil {
		return x.LatestCommitName
	}
	return ""
}

func (x *RepositoryBranch) GetIsMainBranch() bool {
	if x != nil {
		return x.IsMainBranch
	}
	return false
}

func (x *RepositoryBranch) GetLastUpdateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.LastUpdateTime
	}
	return nil
}

type ListRepositoryBranchesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The ID of the repository whose branches should be listed.
	RepositoryId string `protobuf:"bytes,1,opt,name=repository_id,json=repositoryId,proto3" json:"repository_id,omitempty"`
	PageSize     uint32 `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	// The first page is returned if this is empty.
	PageToken string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
}

func (x *ListRepositoryBranchesRequest) Reset() {
	*x = ListRepositoryBranchesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListRepositoryBranchesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRepositoryBranchesRequest) ProtoMessage() {}

func (x *ListRepositoryBranchesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRepositoryBranchesRequest.ProtoReflect.Descriptor instead.
func (*ListRepositoryBranchesRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP(), []int{1}
}

func (x *ListRepositoryBranchesRequest) GetRepositoryId() string {
	if x != nil {
		return x.RepositoryId
	}
	return ""
}

func (x *ListRepositoryBranchesRequest) GetPageSize() uint32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}

func (x *ListRepositoryBranchesRequest) GetPageToken() string {
	if x != nil {
		return x.PageToken
	}
	return ""
}

type ListRepositoryBranchesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryBranches []*RepositoryBranch `protobuf:"bytes,1,rep,name=repository_branches,json=repositoryBranches,proto3" json:"repository_branches,omitempty"`
	// There are no more pages if this is empty.
	NextPageToken string `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

func (x *ListRepositoryBranchesResponse) Reset() {
	*x = ListRepositoryBranchesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListRepositoryBranchesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRepositoryBranchesResponse) ProtoMessage() {}

func (x *ListRepositoryBranchesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRepositoryBranchesResponse.ProtoReflect.Descriptor instead.
func (*ListRepositoryBranchesResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP(), []int{2}
}

func (x *ListRepositoryBranchesResponse) GetRepositoryBranches() []*RepositoryBranch {
	if x != nil {
		return x.RepositoryBranches
	}
	return nil
}

func (x *ListRepositoryBranchesResponse) GetNextPageToken() string {
	if x != nil {
		return x.NextPageToken
	}
	return ""
}

type GetCurrentDefaultBranchRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The ID of the repository whose current default branch is returned.
	RepositoryId string `protobuf:"bytes,1,opt,name=repository_id,json=repositoryId,proto3" json:"repository_id,omitempty"`
}

func (x *GetCurrentDefaultBranchRequest) Reset() {
	*x = GetCurrentDefaultBranchRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCurrentDefaultBranchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCurrentDefaultBranchRequest) ProtoMessage() {}

func (x *GetCurrentDefaultBranchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCurrentDefaultBranchRequest.ProtoReflect.Descriptor instead.
func (*GetCurrentDefaultBranchRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP(), []int{3}
}

func (x *GetCurrentDefaultBranchRequest) GetRepositoryId() string {
	if x != nil {
		return x.RepositoryId
	}
	return ""
}

type GetCurrentDefaultBranchResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CurrentDefaultBranch *RepositoryBranch `protobuf:"bytes,1,opt,name=current_default_branch,json=currentDefaultBranch,proto3" json:"current_default_branch,omitempty"`
}

func (x *GetCurrentDefaultBranchResponse) Reset() {
	*x = GetCurrentDefaultBranchResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCurrentDefaultBranchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCurrentDefaultBranchResponse) ProtoMessage() {}

func (x *GetCurrentDefaultBranchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCurrentDefaultBranchResponse.ProtoReflect.Descriptor instead.
func (*GetCurrentDefaultBranchResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP(), []int{4}
}

func (x *GetCurrentDefaultBranchResponse) GetCurrentDefaultBranch() *RepositoryBranch {
	if x != nil {
		return x.CurrentDefaultBranch
	}
	return nil
}

var File_buf_alpha_registry_v1alpha1_repository_branch_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDesc = []byte{
	0x0a, 0x33, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0xd0, 0x01, 0x0a, 0x10, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2c, 0x0a, 0x12,
	0x6c, 0x61, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x6c, 0x61, 0x74, 0x65, 0x73, 0x74,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x0e, 0x69, 0x73,
	0x5f, 0x6d, 0x61, 0x69, 0x6e, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x0c, 0x69, 0x73, 0x4d, 0x61, 0x69, 0x6e, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68,
	0x12, 0x44, 0x0a, 0x10, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0e, 0x6c, 0x61, 0x73, 0x74, 0x55, 0x70, 0x64, 0x61,
	0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x22, 0x80, 0x01, 0x0a, 0x1d, 0x4c, 0x69, 0x73, 0x74, 0x52,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0c, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x49, 0x64, 0x12, 0x1b, 0x0a,
	0x09, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d,
	0x52, 0x08, 0x70, 0x61, 0x67, 0x65, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x61,
	0x67, 0x65, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x70, 0x61, 0x67, 0x65, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0xa8, 0x01, 0x0a, 0x1e, 0x4c, 0x69,
	0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e,
	0x63, 0x68, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x5e, 0x0a, 0x13,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x63,
	0x68, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52, 0x12, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x73, 0x12, 0x26, 0x0a, 0x0f,
	0x6e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6e, 0x65, 0x78, 0x74, 0x50, 0x61, 0x67, 0x65, 0x54,
	0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x45, 0x0a, 0x1e, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65,
	0x6e, 0x74, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x49, 0x64, 0x22, 0x86, 0x01, 0x0a, 0x1f,
	0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c,
	0x74, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x63, 0x0a, 0x16, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x64, 0x65, 0x66, 0x61, 0x75,
	0x6c, 0x74, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52, 0x14,
	0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x42, 0x72,
	0x61, 0x6e, 0x63, 0x68, 0x32, 0xce, 0x02, 0x0a, 0x17, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x12, 0x96, 0x01, 0x0a, 0x16, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x73, 0x12, 0x3a, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x12, 0x99, 0x01, 0x0a, 0x17, 0x47, 0x65,
	0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x42,
	0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x3b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x44, 0x65,
	0x66, 0x61, 0x75, 0x6c, 0x74, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x3c, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2e, 0x47, 0x65, 0x74, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x44, 0x65, 0x66, 0x61, 0x75,
	0x6c, 0x74, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x03, 0x90, 0x02, 0x01, 0x42, 0xa2, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x15, 0x52, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62,
	0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76,
	0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f,
	0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xa2, 0x02, 0x03,
	0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2,
	0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50,
	0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x3a,
	0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescData = file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_buf_alpha_registry_v1alpha1_repository_branch_proto_goTypes = []interface{}{
	(*RepositoryBranch)(nil),                // 0: buf.alpha.registry.v1alpha1.RepositoryBranch
	(*ListRepositoryBranchesRequest)(nil),   // 1: buf.alpha.registry.v1alpha1.ListRepositoryBranchesRequest
	(*ListRepositoryBranchesResponse)(nil),  // 2: buf.alpha.registry.v1alpha1.ListRepositoryBranchesResponse
	(*GetCurrentDefaultBranchRequest)(nil),  // 3: buf.alpha.registry.v1alpha1.GetCurrentDefaultBranchRequest
	(*GetCurrentDefaultBranchResponse)(nil), // 4: buf.alpha.registry.v1alpha1.GetCurrentDefaultBranchResponse
	(*timestamppb.Timestamp)(nil),           // 5: google.protobuf.Timestamp
}
var file_buf_alpha_registry_v1alpha1_repository_branch_proto_depIdxs = []int32{
	5, // 0: buf.alpha.registry.v1alpha1.RepositoryBranch.last_update_time:type_name -> google.protobuf.Timestamp
	0, // 1: buf.alpha.registry.v1alpha1.ListRepositoryBranchesResponse.repository_branches:type_name -> buf.alpha.registry.v1alpha1.RepositoryBranch
	0, // 2: buf.alpha.registry.v1alpha1.GetCurrentDefaultBranchResponse.current_default_branch:type_name -> buf.alpha.registry.v1alpha1.RepositoryBranch
	1, // 3: buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches:input_type -> buf.alpha.registry.v1alpha1.ListRepositoryBranchesRequest
	3, // 4: buf.alpha.registry.v1alpha1.RepositoryBranchService.GetCurrentDefaultBranch:input_type -> buf.alpha.registry.v1alpha1.GetCurrentDefaultBranchRequest
	2, // 5: buf.alpha.registry.v1alpha1.RepositoryBranchService.ListRepositoryBranches:output_type -> buf.alpha.registry.v1alpha1.ListRepositoryBranchesResponse
	4, // 6: buf.alpha.registry.v1alpha1.RepositoryBranchService.GetCurrentDefaultBranch:output_type -> buf.alpha.registry.v1alpha1.GetCurrentDefaultBranchResponse
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_repository_branch_proto_init() }
func file_buf_alpha_registry_v1alpha1_repository_branch_proto_init() {
	if File_buf_alpha_registry_v1alpha1_repository_branch_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RepositoryBranch); i {
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
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListRepositoryBranchesRequest); i {
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
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListRepositoryBranchesResponse); i {
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
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCurrentDefaultBranchRequest); i {
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
		file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCurrentDefaultBranchResponse); i {
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
			RawDescriptor: file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_repository_branch_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_repository_branch_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_repository_branch_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_repository_branch_proto = out.File
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_depIdxs = nil
}
