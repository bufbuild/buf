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
// source: buf/alpha/registry/v1alpha1/sync.proto

package registryv1alpha1

import (
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
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

// GitSyncPoint is the sync point for a particular module contained in a Git repository.
type GitSyncPoint struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Owner         string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	Repository    string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	Branch        string `protobuf:"bytes,3,opt,name=branch,proto3" json:"branch,omitempty"`
	GitCommitHash string `protobuf:"bytes,4,opt,name=git_commit_hash,json=gitCommitHash,proto3" json:"git_commit_hash,omitempty"`
	BsrCommitName string `protobuf:"bytes,5,opt,name=bsr_commit_name,json=bsrCommitName,proto3" json:"bsr_commit_name,omitempty"`
}

func (x *GitSyncPoint) Reset() {
	*x = GitSyncPoint{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GitSyncPoint) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitSyncPoint) ProtoMessage() {}

func (x *GitSyncPoint) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitSyncPoint.ProtoReflect.Descriptor instead.
func (*GitSyncPoint) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{0}
}

func (x *GitSyncPoint) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *GitSyncPoint) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *GitSyncPoint) GetBranch() string {
	if x != nil {
		return x.Branch
	}
	return ""
}

func (x *GitSyncPoint) GetGitCommitHash() string {
	if x != nil {
		return x.GitCommitHash
	}
	return ""
}

func (x *GitSyncPoint) GetBsrCommitName() string {
	if x != nil {
		return x.BsrCommitName
	}
	return ""
}

type GetGitSyncPointRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Owner is the owner of the BSR repository.
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Repository is the name of the BSR repository.
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	// Branch is the Git branch for which to look up the commit.
	Branch string `protobuf:"bytes,3,opt,name=branch,proto3" json:"branch,omitempty"`
}

func (x *GetGitSyncPointRequest) Reset() {
	*x = GetGitSyncPointRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGitSyncPointRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGitSyncPointRequest) ProtoMessage() {}

func (x *GetGitSyncPointRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGitSyncPointRequest.ProtoReflect.Descriptor instead.
func (*GetGitSyncPointRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{1}
}

func (x *GetGitSyncPointRequest) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *GetGitSyncPointRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *GetGitSyncPointRequest) GetBranch() string {
	if x != nil {
		return x.Branch
	}
	return ""
}

type GetGitSyncPointResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// SyncPoint is the latest syncpoint for the specified owner/repo/branch.
	SyncPoint *GitSyncPoint `protobuf:"bytes,1,opt,name=sync_point,json=syncPoint,proto3" json:"sync_point,omitempty"`
}

func (x *GetGitSyncPointResponse) Reset() {
	*x = GetGitSyncPointResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetGitSyncPointResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetGitSyncPointResponse) ProtoMessage() {}

func (x *GetGitSyncPointResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetGitSyncPointResponse.ProtoReflect.Descriptor instead.
func (*GetGitSyncPointResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{2}
}

func (x *GetGitSyncPointResponse) GetSyncPoint() *GitSyncPoint {
	if x != nil {
		return x.SyncPoint
	}
	return nil
}

type SyncGitCommitRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Owner is the owner of the BSR repository.
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Repository is the name of the BSR repository.
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	// Branch is the Git branch that this commit belongs to.
	Branch string `protobuf:"bytes,3,opt,name=branch,proto3" json:"branch,omitempty"`
	// Manifest with all the module files being pushed.
	Manifest *v1alpha1.Blob `protobuf:"bytes,4,opt,name=manifest,proto3" json:"manifest,omitempty"`
	// Referenced blobs in the manifest. Keep in mind there is not necessarily one
	// blob per file, but one blob per digest, so for files with exactly the same
	// content, you can send just one blob.
	Blobs []*v1alpha1.Blob `protobuf:"bytes,5,rep,name=blobs,proto3" json:"blobs,omitempty"`
	// Hash is the SHA1 hash of the Git commit.
	Hash string `protobuf:"bytes,6,opt,name=hash,proto3" json:"hash,omitempty"`
	// Author is the author of the Git commit. This is typically an end-user.
	Author *GitIdentity `protobuf:"bytes,7,opt,name=author,proto3" json:"author,omitempty"`
	// Commiter is the commiter of the Git commit. This typically a CI system.
	Commiter *GitIdentity `protobuf:"bytes,8,opt,name=commiter,proto3" json:"commiter,omitempty"`
	// Tags are the Git tags which point to this commit.
	Tags []string `protobuf:"bytes,9,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *SyncGitCommitRequest) Reset() {
	*x = SyncGitCommitRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SyncGitCommitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncGitCommitRequest) ProtoMessage() {}

func (x *SyncGitCommitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncGitCommitRequest.ProtoReflect.Descriptor instead.
func (*SyncGitCommitRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{3}
}

func (x *SyncGitCommitRequest) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *SyncGitCommitRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *SyncGitCommitRequest) GetBranch() string {
	if x != nil {
		return x.Branch
	}
	return ""
}

func (x *SyncGitCommitRequest) GetManifest() *v1alpha1.Blob {
	if x != nil {
		return x.Manifest
	}
	return nil
}

func (x *SyncGitCommitRequest) GetBlobs() []*v1alpha1.Blob {
	if x != nil {
		return x.Blobs
	}
	return nil
}

func (x *SyncGitCommitRequest) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *SyncGitCommitRequest) GetAuthor() *GitIdentity {
	if x != nil {
		return x.Author
	}
	return nil
}

func (x *SyncGitCommitRequest) GetCommiter() *GitIdentity {
	if x != nil {
		return x.Commiter
	}
	return nil
}

func (x *SyncGitCommitRequest) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

type SyncGitCommitResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// SyncPoint is the latest syncpoint for the SyncGitCommit request.
	SyncPoint *GitSyncPoint `protobuf:"bytes,1,opt,name=sync_point,json=syncPoint,proto3" json:"sync_point,omitempty"`
}

func (x *SyncGitCommitResponse) Reset() {
	*x = SyncGitCommitResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SyncGitCommitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncGitCommitResponse) ProtoMessage() {}

func (x *SyncGitCommitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncGitCommitResponse.ProtoReflect.Descriptor instead.
func (*SyncGitCommitResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{4}
}

func (x *SyncGitCommitResponse) GetSyncPoint() *GitSyncPoint {
	if x != nil {
		return x.SyncPoint
	}
	return nil
}

type AttachGitTagsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Owner is the owner of the BSR repository.
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Repository is the name of the BSR repository.
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	// Hash is the SHA1 hash of the Git commit that is tagged. The BSR has the ability to resolve this
	// git hash to a BSR commit.
	Hash string `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
	// Tags are the Git tags which point to this commit, and that will be synced to the BSR commit.
	Tags []string `protobuf:"bytes,4,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *AttachGitTagsRequest) Reset() {
	*x = AttachGitTagsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttachGitTagsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttachGitTagsRequest) ProtoMessage() {}

func (x *AttachGitTagsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttachGitTagsRequest.ProtoReflect.Descriptor instead.
func (*AttachGitTagsRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{5}
}

func (x *AttachGitTagsRequest) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *AttachGitTagsRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *AttachGitTagsRequest) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *AttachGitTagsRequest) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

type AttachGitTagsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The BSR commit that was resolved from the given hash, to which the tags were attached.
	BsrCommitName string `protobuf:"bytes,1,opt,name=bsr_commit_name,json=bsrCommitName,proto3" json:"bsr_commit_name,omitempty"`
}

func (x *AttachGitTagsResponse) Reset() {
	*x = AttachGitTagsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AttachGitTagsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AttachGitTagsResponse) ProtoMessage() {}

func (x *AttachGitTagsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AttachGitTagsResponse.ProtoReflect.Descriptor instead.
func (*AttachGitTagsResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP(), []int{6}
}

func (x *AttachGitTagsResponse) GetBsrCommitName() string {
	if x != nil {
		return x.BsrCommitName
	}
	return ""
}

var File_buf_alpha_registry_v1alpha1_sync_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_sync_proto_rawDesc = []byte{
	0x0a, 0x26, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x73, 0x79,
	0x6e, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x26, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x2e, 0x62,
	0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x67, 0x69, 0x74, 0x5f, 0x6d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xac, 0x01,
	0x0a, 0x0c, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x12, 0x14,
	0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x26, 0x0a, 0x0f,
	0x67, 0x69, 0x74, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x68, 0x61, 0x73, 0x68, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x67, 0x69, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x48, 0x61, 0x73, 0x68, 0x12, 0x26, 0x0a, 0x0f, 0x62, 0x73, 0x72, 0x5f, 0x63, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x62,
	0x73, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x66, 0x0a, 0x16,
	0x47, 0x65, 0x74, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06,
	0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x62, 0x72,
	0x61, 0x6e, 0x63, 0x68, 0x22, 0x63, 0x0a, 0x17, 0x47, 0x65, 0x74, 0x47, 0x69, 0x74, 0x53, 0x79,
	0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x48, 0x0a, 0x0a, 0x73, 0x79, 0x6e, 0x63, 0x5f, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x09,
	0x73, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x22, 0x88, 0x03, 0x0a, 0x14, 0x53, 0x79,
	0x6e, 0x63, 0x47, 0x69, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x72, 0x61, 0x6e,
	0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68,
	0x12, 0x3b, 0x0a, 0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6d,
	0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x42,
	0x6c, 0x6f, 0x62, 0x52, 0x08, 0x6d, 0x61, 0x6e, 0x69, 0x66, 0x65, 0x73, 0x74, 0x12, 0x35, 0x0a,
	0x05, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x05, 0x62,
	0x6c, 0x6f, 0x62, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x40, 0x0a, 0x06, 0x61, 0x75, 0x74, 0x68,
	0x6f, 0x72, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x69, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69,
	0x74, 0x79, 0x52, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x12, 0x44, 0x0a, 0x08, 0x63, 0x6f,
	0x6d, 0x6d, 0x69, 0x74, 0x65, 0x72, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x69, 0x74, 0x49, 0x64,
	0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x65, 0x72,
	0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x09, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04,
	0x74, 0x61, 0x67, 0x73, 0x22, 0x61, 0x0a, 0x15, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x69, 0x74, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x48, 0x0a,
	0x0a, 0x73, 0x79, 0x6e, 0x63, 0x5f, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x29, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x09, 0x73, 0x79,
	0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74, 0x22, 0x74, 0x0a, 0x14, 0x41, 0x74, 0x74, 0x61, 0x63,
	0x68, 0x47, 0x69, 0x74, 0x54, 0x61, 0x67, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73, 0x68, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67,
	0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x22, 0x3f, 0x0a,
	0x15, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68, 0x47, 0x69, 0x74, 0x54, 0x61, 0x67, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x26, 0x0a, 0x0f, 0x62, 0x73, 0x72, 0x5f, 0x63, 0x6f,
	0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0d, 0x62, 0x73, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x32, 0x86,
	0x03, 0x0a, 0x0b, 0x53, 0x79, 0x6e, 0x63, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x81,
	0x01, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69,
	0x6e, 0x74, 0x12, 0x33, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2e, 0x47, 0x65, 0x74, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63, 0x50, 0x6f, 0x69, 0x6e, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x34, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x47, 0x69, 0x74, 0x53, 0x79, 0x6e, 0x63,
	0x50, 0x6f, 0x69, 0x6e, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90,
	0x02, 0x01, 0x12, 0x7b, 0x0a, 0x0d, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x69, 0x74, 0x43, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x12, 0x31, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x69, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x47, 0x69, 0x74, 0x43, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x02, 0x12,
	0x76, 0x0a, 0x0d, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68, 0x47, 0x69, 0x74, 0x54, 0x61, 0x67, 0x73,
	0x12, 0x31, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x41,
	0x74, 0x74, 0x61, 0x63, 0x68, 0x47, 0x69, 0x74, 0x54, 0x61, 0x67, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x41, 0x74, 0x74, 0x61, 0x63, 0x68, 0x47, 0x69, 0x74, 0x54, 0x61, 0x67, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x96, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x09, 0x53, 0x79, 0x6e,
	0x63, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75,
	0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x3b, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e,
	0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c,
	0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68,
	0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea,
	0x02, 0x1e, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_registry_v1alpha1_sync_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_sync_proto_rawDescData = file_buf_alpha_registry_v1alpha1_sync_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_sync_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_sync_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_sync_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_sync_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_sync_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_buf_alpha_registry_v1alpha1_sync_proto_goTypes = []interface{}{
	(*GitSyncPoint)(nil),            // 0: buf.alpha.registry.v1alpha1.GitSyncPoint
	(*GetGitSyncPointRequest)(nil),  // 1: buf.alpha.registry.v1alpha1.GetGitSyncPointRequest
	(*GetGitSyncPointResponse)(nil), // 2: buf.alpha.registry.v1alpha1.GetGitSyncPointResponse
	(*SyncGitCommitRequest)(nil),    // 3: buf.alpha.registry.v1alpha1.SyncGitCommitRequest
	(*SyncGitCommitResponse)(nil),   // 4: buf.alpha.registry.v1alpha1.SyncGitCommitResponse
	(*AttachGitTagsRequest)(nil),    // 5: buf.alpha.registry.v1alpha1.AttachGitTagsRequest
	(*AttachGitTagsResponse)(nil),   // 6: buf.alpha.registry.v1alpha1.AttachGitTagsResponse
	(*v1alpha1.Blob)(nil),           // 7: buf.alpha.module.v1alpha1.Blob
	(*GitIdentity)(nil),             // 8: buf.alpha.registry.v1alpha1.GitIdentity
}
var file_buf_alpha_registry_v1alpha1_sync_proto_depIdxs = []int32{
	0, // 0: buf.alpha.registry.v1alpha1.GetGitSyncPointResponse.sync_point:type_name -> buf.alpha.registry.v1alpha1.GitSyncPoint
	7, // 1: buf.alpha.registry.v1alpha1.SyncGitCommitRequest.manifest:type_name -> buf.alpha.module.v1alpha1.Blob
	7, // 2: buf.alpha.registry.v1alpha1.SyncGitCommitRequest.blobs:type_name -> buf.alpha.module.v1alpha1.Blob
	8, // 3: buf.alpha.registry.v1alpha1.SyncGitCommitRequest.author:type_name -> buf.alpha.registry.v1alpha1.GitIdentity
	8, // 4: buf.alpha.registry.v1alpha1.SyncGitCommitRequest.commiter:type_name -> buf.alpha.registry.v1alpha1.GitIdentity
	0, // 5: buf.alpha.registry.v1alpha1.SyncGitCommitResponse.sync_point:type_name -> buf.alpha.registry.v1alpha1.GitSyncPoint
	1, // 6: buf.alpha.registry.v1alpha1.SyncService.GetGitSyncPoint:input_type -> buf.alpha.registry.v1alpha1.GetGitSyncPointRequest
	3, // 7: buf.alpha.registry.v1alpha1.SyncService.SyncGitCommit:input_type -> buf.alpha.registry.v1alpha1.SyncGitCommitRequest
	5, // 8: buf.alpha.registry.v1alpha1.SyncService.AttachGitTags:input_type -> buf.alpha.registry.v1alpha1.AttachGitTagsRequest
	2, // 9: buf.alpha.registry.v1alpha1.SyncService.GetGitSyncPoint:output_type -> buf.alpha.registry.v1alpha1.GetGitSyncPointResponse
	4, // 10: buf.alpha.registry.v1alpha1.SyncService.SyncGitCommit:output_type -> buf.alpha.registry.v1alpha1.SyncGitCommitResponse
	6, // 11: buf.alpha.registry.v1alpha1.SyncService.AttachGitTags:output_type -> buf.alpha.registry.v1alpha1.AttachGitTagsResponse
	9, // [9:12] is the sub-list for method output_type
	6, // [6:9] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_sync_proto_init() }
func file_buf_alpha_registry_v1alpha1_sync_proto_init() {
	if File_buf_alpha_registry_v1alpha1_sync_proto != nil {
		return
	}
	file_buf_alpha_registry_v1alpha1_git_metadata_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GitSyncPoint); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGitSyncPointRequest); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetGitSyncPointResponse); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SyncGitCommitRequest); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SyncGitCommitResponse); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttachGitTagsRequest); i {
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
		file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AttachGitTagsResponse); i {
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
			RawDescriptor: file_buf_alpha_registry_v1alpha1_sync_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_sync_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_sync_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_sync_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_sync_proto = out.File
	file_buf_alpha_registry_v1alpha1_sync_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_sync_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_sync_proto_depIdxs = nil
}
