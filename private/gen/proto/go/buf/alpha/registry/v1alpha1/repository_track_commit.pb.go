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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/repository_track_commit.proto

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

// RepositoryTrackCommit is the existence of a RepositoryCommit on a RepositoryTrack. Currently its only purpose is
// for querying whether a RepositoryCommit is on a RepositoryTrack and determining it's sequence id.
type RepositoryTrackCommit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// immutable
	CreateTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	// immutable
	RepositoryTrackId string `protobuf:"bytes,4,opt,name=repository_track_id,json=repositoryTrackId,proto3" json:"repository_track_id,omitempty"`
	// immutable
	RepositoryCommitId string `protobuf:"bytes,5,opt,name=repository_commit_id,json=repositoryCommitId,proto3" json:"repository_commit_id,omitempty"`
	// unique for repository_track, immutable
	SequenceId int64 `protobuf:"varint,6,opt,name=sequence_id,json=sequenceId,proto3" json:"sequence_id,omitempty"`
}

func (x *RepositoryTrackCommit) Reset() {
	*x = RepositoryTrackCommit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RepositoryTrackCommit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryTrackCommit) ProtoMessage() {}

func (x *RepositoryTrackCommit) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepositoryTrackCommit.ProtoReflect.Descriptor instead.
func (*RepositoryTrackCommit) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{0}
}

func (x *RepositoryTrackCommit) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

func (x *RepositoryTrackCommit) GetRepositoryTrackId() string {
	if x != nil {
		return x.RepositoryTrackId
	}
	return ""
}

func (x *RepositoryTrackCommit) GetRepositoryCommitId() string {
	if x != nil {
		return x.RepositoryCommitId
	}
	return ""
}

func (x *RepositoryTrackCommit) GetSequenceId() int64 {
	if x != nil {
		return x.SequenceId
	}
	return 0
}

type GetRepositoryTrackCommitByRepositoryCommitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryTrackId  string `protobuf:"bytes,1,opt,name=repository_track_id,json=repositoryTrackId,proto3" json:"repository_track_id,omitempty"`
	RepositoryCommitId string `protobuf:"bytes,2,opt,name=repository_commit_id,json=repositoryCommitId,proto3" json:"repository_commit_id,omitempty"`
}

func (x *GetRepositoryTrackCommitByRepositoryCommitRequest) Reset() {
	*x = GetRepositoryTrackCommitByRepositoryCommitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRepositoryTrackCommitByRepositoryCommitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRepositoryTrackCommitByRepositoryCommitRequest) ProtoMessage() {}

func (x *GetRepositoryTrackCommitByRepositoryCommitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRepositoryTrackCommitByRepositoryCommitRequest.ProtoReflect.Descriptor instead.
func (*GetRepositoryTrackCommitByRepositoryCommitRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{1}
}

func (x *GetRepositoryTrackCommitByRepositoryCommitRequest) GetRepositoryTrackId() string {
	if x != nil {
		return x.RepositoryTrackId
	}
	return ""
}

func (x *GetRepositoryTrackCommitByRepositoryCommitRequest) GetRepositoryCommitId() string {
	if x != nil {
		return x.RepositoryCommitId
	}
	return ""
}

type GetRepositoryTrackCommitByRepositoryCommitResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryTrackCommit *RepositoryTrackCommit `protobuf:"bytes,1,opt,name=repository_track_commit,json=repositoryTrackCommit,proto3" json:"repository_track_commit,omitempty"`
}

func (x *GetRepositoryTrackCommitByRepositoryCommitResponse) Reset() {
	*x = GetRepositoryTrackCommitByRepositoryCommitResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRepositoryTrackCommitByRepositoryCommitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRepositoryTrackCommitByRepositoryCommitResponse) ProtoMessage() {}

func (x *GetRepositoryTrackCommitByRepositoryCommitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRepositoryTrackCommitByRepositoryCommitResponse.ProtoReflect.Descriptor instead.
func (*GetRepositoryTrackCommitByRepositoryCommitResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{2}
}

func (x *GetRepositoryTrackCommitByRepositoryCommitResponse) GetRepositoryTrackCommit() *RepositoryTrackCommit {
	if x != nil {
		return x.RepositoryTrackCommit
	}
	return nil
}

type ListRepositoryTrackCommitsByRepositoryTrackRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryTrackId string `protobuf:"bytes,1,opt,name=repository_track_id,json=repositoryTrackId,proto3" json:"repository_track_id,omitempty"`
	PageSize          uint32 `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	PageToken         string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
	Reverse           bool   `protobuf:"varint,4,opt,name=reverse,proto3" json:"reverse,omitempty"`
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) Reset() {
	*x = ListRepositoryTrackCommitsByRepositoryTrackRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRepositoryTrackCommitsByRepositoryTrackRequest) ProtoMessage() {}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRepositoryTrackCommitsByRepositoryTrackRequest.ProtoReflect.Descriptor instead.
func (*ListRepositoryTrackCommitsByRepositoryTrackRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{3}
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) GetRepositoryTrackId() string {
	if x != nil {
		return x.RepositoryTrackId
	}
	return ""
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) GetPageSize() uint32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) GetPageToken() string {
	if x != nil {
		return x.PageToken
	}
	return ""
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackRequest) GetReverse() bool {
	if x != nil {
		return x.Reverse
	}
	return false
}

type ListRepositoryTrackCommitsByRepositoryTrackResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryTrackCommits []*RepositoryTrackCommit `protobuf:"bytes,1,rep,name=repository_track_commits,json=repositoryTrackCommits,proto3" json:"repository_track_commits,omitempty"`
	NextPageToken          string                   `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackResponse) Reset() {
	*x = ListRepositoryTrackCommitsByRepositoryTrackResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRepositoryTrackCommitsByRepositoryTrackResponse) ProtoMessage() {}

func (x *ListRepositoryTrackCommitsByRepositoryTrackResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRepositoryTrackCommitsByRepositoryTrackResponse.ProtoReflect.Descriptor instead.
func (*ListRepositoryTrackCommitsByRepositoryTrackResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{4}
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackResponse) GetRepositoryTrackCommits() []*RepositoryTrackCommit {
	if x != nil {
		return x.RepositoryTrackCommits
	}
	return nil
}

func (x *ListRepositoryTrackCommitsByRepositoryTrackResponse) GetNextPageToken() string {
	if x != nil {
		return x.NextPageToken
	}
	return ""
}

type GetRepositoryTrackCommitByReferenceRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryOwner string `protobuf:"bytes,1,opt,name=repository_owner,json=repositoryOwner,proto3" json:"repository_owner,omitempty"`
	RepositoryName  string `protobuf:"bytes,2,opt,name=repository_name,json=repositoryName,proto3" json:"repository_name,omitempty"`
	Track           string `protobuf:"bytes,3,opt,name=track,proto3" json:"track,omitempty"`
	Reference       string `protobuf:"bytes,4,opt,name=reference,proto3" json:"reference,omitempty"`
}

func (x *GetRepositoryTrackCommitByReferenceRequest) Reset() {
	*x = GetRepositoryTrackCommitByReferenceRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRepositoryTrackCommitByReferenceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRepositoryTrackCommitByReferenceRequest) ProtoMessage() {}

func (x *GetRepositoryTrackCommitByReferenceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRepositoryTrackCommitByReferenceRequest.ProtoReflect.Descriptor instead.
func (*GetRepositoryTrackCommitByReferenceRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{5}
}

func (x *GetRepositoryTrackCommitByReferenceRequest) GetRepositoryOwner() string {
	if x != nil {
		return x.RepositoryOwner
	}
	return ""
}

func (x *GetRepositoryTrackCommitByReferenceRequest) GetRepositoryName() string {
	if x != nil {
		return x.RepositoryName
	}
	return ""
}

func (x *GetRepositoryTrackCommitByReferenceRequest) GetTrack() string {
	if x != nil {
		return x.Track
	}
	return ""
}

func (x *GetRepositoryTrackCommitByReferenceRequest) GetReference() string {
	if x != nil {
		return x.Reference
	}
	return ""
}

type GetRepositoryTrackCommitByReferenceResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryTrackCommit *RepositoryTrackCommit `protobuf:"bytes,1,opt,name=repository_track_commit,json=repositoryTrackCommit,proto3" json:"repository_track_commit,omitempty"`
}

func (x *GetRepositoryTrackCommitByReferenceResponse) Reset() {
	*x = GetRepositoryTrackCommitByReferenceResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRepositoryTrackCommitByReferenceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRepositoryTrackCommitByReferenceResponse) ProtoMessage() {}

func (x *GetRepositoryTrackCommitByReferenceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRepositoryTrackCommitByReferenceResponse.ProtoReflect.Descriptor instead.
func (*GetRepositoryTrackCommitByReferenceResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP(), []int{6}
}

func (x *GetRepositoryTrackCommitByReferenceResponse) GetRepositoryTrackCommit() *RepositoryTrackCommit {
	if x != nil {
		return x.RepositoryTrackCommit
	}
	return nil
}

var File_buf_alpha_registry_v1alpha1_repository_track_commit_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDesc = []byte{
	0x0a, 0x39, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd7, 0x01, 0x0a, 0x15, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x12, 0x3b, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65,
	0x12, 0x2e, 0x0a, 0x13, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x74,
	0x72, 0x61, 0x63, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x49, 0x64,
	0x12, 0x30, 0x0a, 0x14, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x12,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x73, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63,
	0x65, 0x49, 0x64, 0x22, 0x95, 0x01, 0x0a, 0x31, 0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2e, 0x0a, 0x13, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x49, 0x64, 0x12, 0x30, 0x0a, 0x14, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x12, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x64, 0x22, 0xa0, 0x01, 0x0a, 0x32,
	0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61,
	0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x6a, 0x0a, 0x17, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79,
	0x5f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63,
	0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x15, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x22, 0xba,
	0x01, 0x0a, 0x32, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x42, 0x79, 0x52,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2e, 0x0a, 0x13, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x11, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72,
	0x61, 0x63, 0x6b, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x73, 0x69,
	0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x70, 0x61, 0x67, 0x65, 0x53, 0x69,
	0x7a, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x61, 0x67, 0x65, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x12, 0x18, 0x0a, 0x07, 0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x72, 0x65, 0x76, 0x65, 0x72, 0x73, 0x65, 0x22, 0xcb, 0x01, 0x0a, 0x33,
	0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72,
	0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x6c, 0x0a, 0x18, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x5f, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72,
	0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x16, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x73, 0x12, 0x26, 0x0a, 0x0f, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x74,
	0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x6e, 0x65, 0x78, 0x74,
	0x50, 0x61, 0x67, 0x65, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0xb4, 0x01, 0x0a, 0x2a, 0x47, 0x65,
	0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x29, 0x0a, 0x10, 0x72, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x4f, 0x77,
	0x6e, 0x65, 0x72, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05,
	0x74, 0x72, 0x61, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x72, 0x61,
	0x63, 0x6b, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65,
	0x22, 0x99, 0x01, 0x0a, 0x2b, 0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52,
	0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x6a, 0x0a, 0x17, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x74,
	0x72, 0x61, 0x63, 0x6b, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x15, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x32, 0xfc, 0x04, 0x0a,
	0x1c, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0xcd, 0x01,
	0x0a, 0x2a, 0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54,
	0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x4e, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x4f, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0xd0, 0x01,
	0x0a, 0x2b, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79,
	0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x42, 0x79, 0x52, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x12, 0x4f, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74,
	0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x50,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73,
	0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x42, 0x79, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0xb8, 0x01, 0x0a, 0x23, 0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52,
	0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x47, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42,
	0x79, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x48, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x47, 0x65, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61,
	0x63, 0x6b, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x79, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65,
	0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0xa7, 0x02, 0x0a, 0x1f,
	0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42,
	0x1a, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x54, 0x72, 0x61, 0x63, 0x6b,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67,
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
}

var (
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescData = file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_goTypes = []interface{}{
	(*RepositoryTrackCommit)(nil),                               // 0: buf.alpha.registry.v1alpha1.RepositoryTrackCommit
	(*GetRepositoryTrackCommitByRepositoryCommitRequest)(nil),   // 1: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByRepositoryCommitRequest
	(*GetRepositoryTrackCommitByRepositoryCommitResponse)(nil),  // 2: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByRepositoryCommitResponse
	(*ListRepositoryTrackCommitsByRepositoryTrackRequest)(nil),  // 3: buf.alpha.registry.v1alpha1.ListRepositoryTrackCommitsByRepositoryTrackRequest
	(*ListRepositoryTrackCommitsByRepositoryTrackResponse)(nil), // 4: buf.alpha.registry.v1alpha1.ListRepositoryTrackCommitsByRepositoryTrackResponse
	(*GetRepositoryTrackCommitByReferenceRequest)(nil),          // 5: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByReferenceRequest
	(*GetRepositoryTrackCommitByReferenceResponse)(nil),         // 6: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByReferenceResponse
	(*timestamppb.Timestamp)(nil),                               // 7: google.protobuf.Timestamp
}
var file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_depIdxs = []int32{
	7, // 0: buf.alpha.registry.v1alpha1.RepositoryTrackCommit.create_time:type_name -> google.protobuf.Timestamp
	0, // 1: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByRepositoryCommitResponse.repository_track_commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryTrackCommit
	0, // 2: buf.alpha.registry.v1alpha1.ListRepositoryTrackCommitsByRepositoryTrackResponse.repository_track_commits:type_name -> buf.alpha.registry.v1alpha1.RepositoryTrackCommit
	0, // 3: buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByReferenceResponse.repository_track_commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryTrackCommit
	1, // 4: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.GetRepositoryTrackCommitByRepositoryCommit:input_type -> buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByRepositoryCommitRequest
	3, // 5: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.ListRepositoryTrackCommitsByRepositoryTrack:input_type -> buf.alpha.registry.v1alpha1.ListRepositoryTrackCommitsByRepositoryTrackRequest
	5, // 6: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.GetRepositoryTrackCommitByReference:input_type -> buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByReferenceRequest
	2, // 7: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.GetRepositoryTrackCommitByRepositoryCommit:output_type -> buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByRepositoryCommitResponse
	4, // 8: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.ListRepositoryTrackCommitsByRepositoryTrack:output_type -> buf.alpha.registry.v1alpha1.ListRepositoryTrackCommitsByRepositoryTrackResponse
	6, // 9: buf.alpha.registry.v1alpha1.RepositoryTrackCommitService.GetRepositoryTrackCommitByReference:output_type -> buf.alpha.registry.v1alpha1.GetRepositoryTrackCommitByReferenceResponse
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_init() }
func file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_init() {
	if File_buf_alpha_registry_v1alpha1_repository_track_commit_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RepositoryTrackCommit); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRepositoryTrackCommitByRepositoryCommitRequest); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRepositoryTrackCommitByRepositoryCommitResponse); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListRepositoryTrackCommitsByRepositoryTrackRequest); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListRepositoryTrackCommitsByRepositoryTrackResponse); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRepositoryTrackCommitByReferenceRequest); i {
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
		file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRepositoryTrackCommitByReferenceResponse); i {
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
			RawDescriptor: file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_repository_track_commit_proto = out.File
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_repository_track_commit_proto_depIdxs = nil
}
