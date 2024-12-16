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
// source: buf/alpha/registry/v1alpha1/git_metadata.proto

//go:build !protoopaque

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

// GitIdentity is a Git user identity, typically either an author or a committer.
type GitIdentity struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// Name is the name of the Git identity. This is not the BSR user's username.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Email is the email of the Git identity. This is not the BSR user's email.
	Email string `protobuf:"bytes,2,opt,name=email,proto3" json:"email,omitempty"`
	// Time is the time at which this identity was captured.
	Time          *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=time,proto3" json:"time,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GitIdentity) Reset() {
	*x = GitIdentity{}
	mi := &file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GitIdentity) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitIdentity) ProtoMessage() {}

func (x *GitIdentity) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GitIdentity) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *GitIdentity) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *GitIdentity) GetTime() *timestamppb.Timestamp {
	if x != nil {
		return x.Time
	}
	return nil
}

func (x *GitIdentity) SetName(v string) {
	x.Name = v
}

func (x *GitIdentity) SetEmail(v string) {
	x.Email = v
}

func (x *GitIdentity) SetTime(v *timestamppb.Timestamp) {
	x.Time = v
}

func (x *GitIdentity) HasTime() bool {
	if x == nil {
		return false
	}
	return x.Time != nil
}

func (x *GitIdentity) ClearTime() {
	x.Time = nil
}

type GitIdentity_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Name is the name of the Git identity. This is not the BSR user's username.
	Name string
	// Email is the email of the Git identity. This is not the BSR user's email.
	Email string
	// Time is the time at which this identity was captured.
	Time *timestamppb.Timestamp
}

func (b0 GitIdentity_builder) Build() *GitIdentity {
	m0 := &GitIdentity{}
	b, x := &b0, m0
	_, _ = b, x
	x.Name = b.Name
	x.Email = b.Email
	x.Time = b.Time
	return m0
}

// GitCommitInformation is the information associated with a Git commit.
// This always includes the hash.
// The author and/or committer user identities are included when available.
type GitCommitInformation struct {
	state protoimpl.MessageState `protogen:"hybrid.v1"`
	// Hash is the SHA1 hash of the git commit.
	Hash string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	// Author is metadata associated with the author of the git commit.
	// This may not always be available, so it is not always populated.
	Author *GitIdentity `protobuf:"bytes,2,opt,name=author,proto3" json:"author,omitempty"`
	// Committer is the metadata associated with the committer of the git commit.
	// This may not always be available, so it is not always populated.
	Committer     *GitIdentity `protobuf:"bytes,3,opt,name=committer,proto3" json:"committer,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GitCommitInformation) Reset() {
	*x = GitCommitInformation{}
	mi := &file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GitCommitInformation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitCommitInformation) ProtoMessage() {}

func (x *GitCommitInformation) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GitCommitInformation) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

func (x *GitCommitInformation) GetAuthor() *GitIdentity {
	if x != nil {
		return x.Author
	}
	return nil
}

func (x *GitCommitInformation) GetCommitter() *GitIdentity {
	if x != nil {
		return x.Committer
	}
	return nil
}

func (x *GitCommitInformation) SetHash(v string) {
	x.Hash = v
}

func (x *GitCommitInformation) SetAuthor(v *GitIdentity) {
	x.Author = v
}

func (x *GitCommitInformation) SetCommitter(v *GitIdentity) {
	x.Committer = v
}

func (x *GitCommitInformation) HasAuthor() bool {
	if x == nil {
		return false
	}
	return x.Author != nil
}

func (x *GitCommitInformation) HasCommitter() bool {
	if x == nil {
		return false
	}
	return x.Committer != nil
}

func (x *GitCommitInformation) ClearAuthor() {
	x.Author = nil
}

func (x *GitCommitInformation) ClearCommitter() {
	x.Committer = nil
}

type GitCommitInformation_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Hash is the SHA1 hash of the git commit.
	Hash string
	// Author is metadata associated with the author of the git commit.
	// This may not always be available, so it is not always populated.
	Author *GitIdentity
	// Committer is the metadata associated with the committer of the git commit.
	// This may not always be available, so it is not always populated.
	Committer *GitIdentity
}

func (b0 GitCommitInformation_builder) Build() *GitCommitInformation {
	m0 := &GitCommitInformation{}
	b, x := &b0, m0
	_, _ = b, x
	x.Hash = b.Hash
	x.Author = b.Author
	x.Committer = b.Committer
	return m0
}

var File_buf_alpha_registry_v1alpha1_git_metadata_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_git_metadata_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x67, 0x69,
	0x74, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x67,
	0x0a, 0x0b, 0x47, 0x69, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x2e, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x22, 0xb4, 0x01, 0x0a, 0x14, 0x47, 0x69, 0x74, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x12, 0x0a, 0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x68, 0x61, 0x73, 0x68, 0x12, 0x40, 0x0a, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x47, 0x69, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x06,
	0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x12, 0x46, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x74, 0x65, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x69, 0x74, 0x49, 0x64, 0x65, 0x6e, 0x74,
	0x69, 0x74, 0x79, 0x52, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x74, 0x65, 0x72, 0x42, 0x9d,
	0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x42, 0x10, 0x47, 0x69, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x50,
	0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f,
	0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x3b, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68,
	0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c,
	0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e,
	0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_alpha_registry_v1alpha1_git_metadata_proto_goTypes = []any{
	(*GitIdentity)(nil),           // 0: buf.alpha.registry.v1alpha1.GitIdentity
	(*GitCommitInformation)(nil),  // 1: buf.alpha.registry.v1alpha1.GitCommitInformation
	(*timestamppb.Timestamp)(nil), // 2: google.protobuf.Timestamp
}
var file_buf_alpha_registry_v1alpha1_git_metadata_proto_depIdxs = []int32{
	2, // 0: buf.alpha.registry.v1alpha1.GitIdentity.time:type_name -> google.protobuf.Timestamp
	0, // 1: buf.alpha.registry.v1alpha1.GitCommitInformation.author:type_name -> buf.alpha.registry.v1alpha1.GitIdentity
	0, // 2: buf.alpha.registry.v1alpha1.GitCommitInformation.committer:type_name -> buf.alpha.registry.v1alpha1.GitIdentity
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_git_metadata_proto_init() }
func file_buf_alpha_registry_v1alpha1_git_metadata_proto_init() {
	if File_buf_alpha_registry_v1alpha1_git_metadata_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_git_metadata_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_git_metadata_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_git_metadata_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_git_metadata_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_git_metadata_proto = out.File
	file_buf_alpha_registry_v1alpha1_git_metadata_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_git_metadata_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_git_metadata_proto_depIdxs = nil
}
