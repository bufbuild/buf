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
// 	protoc-gen-go v1.36.1
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/reference.proto

//go:build protoopaque

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

type Reference struct {
	state                protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Reference isReference_Reference  `protobuf_oneof:"reference"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *Reference) Reset() {
	*x = Reference{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Reference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Reference) ProtoMessage() {}

func (x *Reference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Reference) GetBranch() *RepositoryBranch {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_Branch); ok {
			return x.Branch
		}
	}
	return nil
}

func (x *Reference) GetTag() *RepositoryTag {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_Tag); ok {
			return x.Tag
		}
	}
	return nil
}

func (x *Reference) GetCommit() *RepositoryCommit {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_Commit); ok {
			return x.Commit
		}
	}
	return nil
}

func (x *Reference) GetMain() *RepositoryMainReference {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_Main); ok {
			return x.Main
		}
	}
	return nil
}

func (x *Reference) GetDraft() *RepositoryDraft {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_Draft); ok {
			return x.Draft
		}
	}
	return nil
}

func (x *Reference) GetVcsCommit() *RepositoryVCSCommit {
	if x != nil {
		if x, ok := x.xxx_hidden_Reference.(*reference_VcsCommit); ok {
			return x.VcsCommit
		}
	}
	return nil
}

func (x *Reference) SetBranch(v *RepositoryBranch) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_Branch{v}
}

func (x *Reference) SetTag(v *RepositoryTag) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_Tag{v}
}

func (x *Reference) SetCommit(v *RepositoryCommit) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_Commit{v}
}

func (x *Reference) SetMain(v *RepositoryMainReference) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_Main{v}
}

func (x *Reference) SetDraft(v *RepositoryDraft) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_Draft{v}
}

func (x *Reference) SetVcsCommit(v *RepositoryVCSCommit) {
	if v == nil {
		x.xxx_hidden_Reference = nil
		return
	}
	x.xxx_hidden_Reference = &reference_VcsCommit{v}
}

func (x *Reference) HasReference() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Reference != nil
}

func (x *Reference) HasBranch() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_Branch)
	return ok
}

func (x *Reference) HasTag() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_Tag)
	return ok
}

func (x *Reference) HasCommit() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_Commit)
	return ok
}

func (x *Reference) HasMain() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_Main)
	return ok
}

func (x *Reference) HasDraft() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_Draft)
	return ok
}

func (x *Reference) HasVcsCommit() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_Reference.(*reference_VcsCommit)
	return ok
}

func (x *Reference) ClearReference() {
	x.xxx_hidden_Reference = nil
}

func (x *Reference) ClearBranch() {
	if _, ok := x.xxx_hidden_Reference.(*reference_Branch); ok {
		x.xxx_hidden_Reference = nil
	}
}

func (x *Reference) ClearTag() {
	if _, ok := x.xxx_hidden_Reference.(*reference_Tag); ok {
		x.xxx_hidden_Reference = nil
	}
}

func (x *Reference) ClearCommit() {
	if _, ok := x.xxx_hidden_Reference.(*reference_Commit); ok {
		x.xxx_hidden_Reference = nil
	}
}

func (x *Reference) ClearMain() {
	if _, ok := x.xxx_hidden_Reference.(*reference_Main); ok {
		x.xxx_hidden_Reference = nil
	}
}

func (x *Reference) ClearDraft() {
	if _, ok := x.xxx_hidden_Reference.(*reference_Draft); ok {
		x.xxx_hidden_Reference = nil
	}
}

func (x *Reference) ClearVcsCommit() {
	if _, ok := x.xxx_hidden_Reference.(*reference_VcsCommit); ok {
		x.xxx_hidden_Reference = nil
	}
}

const Reference_Reference_not_set_case case_Reference_Reference = 0
const Reference_Branch_case case_Reference_Reference = 1
const Reference_Tag_case case_Reference_Reference = 2
const Reference_Commit_case case_Reference_Reference = 3
const Reference_Main_case case_Reference_Reference = 5
const Reference_Draft_case case_Reference_Reference = 6
const Reference_VcsCommit_case case_Reference_Reference = 7

func (x *Reference) WhichReference() case_Reference_Reference {
	if x == nil {
		return Reference_Reference_not_set_case
	}
	switch x.xxx_hidden_Reference.(type) {
	case *reference_Branch:
		return Reference_Branch_case
	case *reference_Tag:
		return Reference_Tag_case
	case *reference_Commit:
		return Reference_Commit_case
	case *reference_Main:
		return Reference_Main_case
	case *reference_Draft:
		return Reference_Draft_case
	case *reference_VcsCommit:
		return Reference_VcsCommit_case
	default:
		return Reference_Reference_not_set_case
	}
}

type Reference_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Fields of oneof xxx_hidden_Reference:
	// The requested reference is a branch.
	Branch *RepositoryBranch
	// The requested reference is a tag.
	Tag *RepositoryTag
	// The requested reference is a commit.
	Commit *RepositoryCommit
	// The requested reference is the default reference.
	Main *RepositoryMainReference
	// The requested reference is a draft commit.
	Draft *RepositoryDraft
	// The requested reference is a VCS commit.
	VcsCommit *RepositoryVCSCommit
	// -- end of xxx_hidden_Reference
}

func (b0 Reference_builder) Build() *Reference {
	m0 := &Reference{}
	b, x := &b0, m0
	_, _ = b, x
	if b.Branch != nil {
		x.xxx_hidden_Reference = &reference_Branch{b.Branch}
	}
	if b.Tag != nil {
		x.xxx_hidden_Reference = &reference_Tag{b.Tag}
	}
	if b.Commit != nil {
		x.xxx_hidden_Reference = &reference_Commit{b.Commit}
	}
	if b.Main != nil {
		x.xxx_hidden_Reference = &reference_Main{b.Main}
	}
	if b.Draft != nil {
		x.xxx_hidden_Reference = &reference_Draft{b.Draft}
	}
	if b.VcsCommit != nil {
		x.xxx_hidden_Reference = &reference_VcsCommit{b.VcsCommit}
	}
	return m0
}

type case_Reference_Reference protoreflect.FieldNumber

func (x case_Reference_Reference) String() string {
	md := file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[0].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isReference_Reference interface {
	isReference_Reference()
}

type reference_Branch struct {
	// The requested reference is a branch.
	Branch *RepositoryBranch `protobuf:"bytes,1,opt,name=branch,proto3,oneof"`
}

type reference_Tag struct {
	// The requested reference is a tag.
	Tag *RepositoryTag `protobuf:"bytes,2,opt,name=tag,proto3,oneof"`
}

type reference_Commit struct {
	// The requested reference is a commit.
	Commit *RepositoryCommit `protobuf:"bytes,3,opt,name=commit,proto3,oneof"`
}

type reference_Main struct {
	// The requested reference is the default reference.
	Main *RepositoryMainReference `protobuf:"bytes,5,opt,name=main,proto3,oneof"`
}

type reference_Draft struct {
	// The requested reference is a draft commit.
	Draft *RepositoryDraft `protobuf:"bytes,6,opt,name=draft,proto3,oneof"`
}

type reference_VcsCommit struct {
	// The requested reference is a VCS commit.
	VcsCommit *RepositoryVCSCommit `protobuf:"bytes,7,opt,name=vcs_commit,json=vcsCommit,proto3,oneof"`
}

func (*reference_Branch) isReference_Reference() {}

func (*reference_Tag) isReference_Reference() {}

func (*reference_Commit) isReference_Reference() {}

func (*reference_Main) isReference_Reference() {}

func (*reference_Draft) isReference_Reference() {}

func (*reference_VcsCommit) isReference_Reference() {}

type RepositoryMainReference struct {
	state             protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Name   string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_Commit *RepositoryCommit      `protobuf:"bytes,2,opt,name=commit,proto3" json:"commit,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *RepositoryMainReference) Reset() {
	*x = RepositoryMainReference{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepositoryMainReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryMainReference) ProtoMessage() {}

func (x *RepositoryMainReference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *RepositoryMainReference) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *RepositoryMainReference) GetCommit() *RepositoryCommit {
	if x != nil {
		return x.xxx_hidden_Commit
	}
	return nil
}

func (x *RepositoryMainReference) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *RepositoryMainReference) SetCommit(v *RepositoryCommit) {
	x.xxx_hidden_Commit = v
}

func (x *RepositoryMainReference) HasCommit() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Commit != nil
}

func (x *RepositoryMainReference) ClearCommit() {
	x.xxx_hidden_Commit = nil
}

type RepositoryMainReference_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Name is the configured default_branch for the repository (default: 'main').
	Name string
	// The latest commit in this repository. If the repository has no commits,
	// this will be empty.
	Commit *RepositoryCommit
}

func (b0 RepositoryMainReference_builder) Build() *RepositoryMainReference {
	m0 := &RepositoryMainReference{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_Commit = b.Commit
	return m0
}

type RepositoryDraft struct {
	state             protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Name   string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_Commit *RepositoryCommit      `protobuf:"bytes,2,opt,name=commit,proto3" json:"commit,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *RepositoryDraft) Reset() {
	*x = RepositoryDraft{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepositoryDraft) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryDraft) ProtoMessage() {}

func (x *RepositoryDraft) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *RepositoryDraft) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *RepositoryDraft) GetCommit() *RepositoryCommit {
	if x != nil {
		return x.xxx_hidden_Commit
	}
	return nil
}

func (x *RepositoryDraft) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *RepositoryDraft) SetCommit(v *RepositoryCommit) {
	x.xxx_hidden_Commit = v
}

func (x *RepositoryDraft) HasCommit() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Commit != nil
}

func (x *RepositoryDraft) ClearCommit() {
	x.xxx_hidden_Commit = nil
}

type RepositoryDraft_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The name of the draft
	Name string
	// The commit this draft points to.
	Commit *RepositoryCommit
}

func (b0 RepositoryDraft_builder) Build() *RepositoryDraft {
	m0 := &RepositoryDraft{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_Commit = b.Commit
	return m0
}

type RepositoryVCSCommit struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Id         string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	xxx_hidden_CreateTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	xxx_hidden_Name       string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_CommitName string                 `protobuf:"bytes,4,opt,name=commit_name,json=commitName,proto3" json:"commit_name,omitempty"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *RepositoryVCSCommit) Reset() {
	*x = RepositoryVCSCommit{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RepositoryVCSCommit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepositoryVCSCommit) ProtoMessage() {}

func (x *RepositoryVCSCommit) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *RepositoryVCSCommit) GetId() string {
	if x != nil {
		return x.xxx_hidden_Id
	}
	return ""
}

func (x *RepositoryVCSCommit) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.xxx_hidden_CreateTime
	}
	return nil
}

func (x *RepositoryVCSCommit) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *RepositoryVCSCommit) GetCommitName() string {
	if x != nil {
		return x.xxx_hidden_CommitName
	}
	return ""
}

func (x *RepositoryVCSCommit) SetId(v string) {
	x.xxx_hidden_Id = v
}

func (x *RepositoryVCSCommit) SetCreateTime(v *timestamppb.Timestamp) {
	x.xxx_hidden_CreateTime = v
}

func (x *RepositoryVCSCommit) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *RepositoryVCSCommit) SetCommitName(v string) {
	x.xxx_hidden_CommitName = v
}

func (x *RepositoryVCSCommit) HasCreateTime() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_CreateTime != nil
}

func (x *RepositoryVCSCommit) ClearCreateTime() {
	x.xxx_hidden_CreateTime = nil
}

type RepositoryVCSCommit_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// primary key, unique.
	Id string
	// immutable
	CreateTime *timestamppb.Timestamp
	// The name of the VCS commit, e.g. for Git, it would be the Git hash.
	Name string
	// The name of the BSR commit this VCS commit belongs to.
	CommitName string
}

func (b0 RepositoryVCSCommit_builder) Build() *RepositoryVCSCommit {
	m0 := &RepositoryVCSCommit{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Id = b.Id
	x.xxx_hidden_CreateTime = b.CreateTime
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_CommitName = b.CommitName
	return m0
}

type GetReferenceByNameRequest struct {
	state                     protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Name           string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	xxx_hidden_Owner          string                 `protobuf:"bytes,2,opt,name=owner,proto3" json:"owner,omitempty"`
	xxx_hidden_RepositoryName string                 `protobuf:"bytes,3,opt,name=repository_name,json=repositoryName,proto3" json:"repository_name,omitempty"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *GetReferenceByNameRequest) Reset() {
	*x = GetReferenceByNameRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetReferenceByNameRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetReferenceByNameRequest) ProtoMessage() {}

func (x *GetReferenceByNameRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetReferenceByNameRequest) GetName() string {
	if x != nil {
		return x.xxx_hidden_Name
	}
	return ""
}

func (x *GetReferenceByNameRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *GetReferenceByNameRequest) GetRepositoryName() string {
	if x != nil {
		return x.xxx_hidden_RepositoryName
	}
	return ""
}

func (x *GetReferenceByNameRequest) SetName(v string) {
	x.xxx_hidden_Name = v
}

func (x *GetReferenceByNameRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *GetReferenceByNameRequest) SetRepositoryName(v string) {
	x.xxx_hidden_RepositoryName = v
}

type GetReferenceByNameRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Optional name (if unspecified, will use the repository's default_branch).
	Name string
	// Owner of the repository the reference belongs to.
	Owner string
	// Name of the repository the reference belongs to.
	RepositoryName string
}

func (b0 GetReferenceByNameRequest_builder) Build() *GetReferenceByNameRequest {
	m0 := &GetReferenceByNameRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Name = b.Name
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_RepositoryName = b.RepositoryName
	return m0
}

type GetReferenceByNameResponse struct {
	state                protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Reference *Reference             `protobuf:"bytes,1,opt,name=reference,proto3" json:"reference,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *GetReferenceByNameResponse) Reset() {
	*x = GetReferenceByNameResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetReferenceByNameResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetReferenceByNameResponse) ProtoMessage() {}

func (x *GetReferenceByNameResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetReferenceByNameResponse) GetReference() *Reference {
	if x != nil {
		return x.xxx_hidden_Reference
	}
	return nil
}

func (x *GetReferenceByNameResponse) SetReference(v *Reference) {
	x.xxx_hidden_Reference = v
}

func (x *GetReferenceByNameResponse) HasReference() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Reference != nil
}

func (x *GetReferenceByNameResponse) ClearReference() {
	x.xxx_hidden_Reference = nil
}

type GetReferenceByNameResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	Reference *Reference
}

func (b0 GetReferenceByNameResponse_builder) Build() *GetReferenceByNameResponse {
	m0 := &GetReferenceByNameResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Reference = b.Reference
	return m0
}

var File_buf_alpha_registry_v1alpha1_reference_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_reference_proto_rawDesc = []byte{
	0x0a, 0x2b, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65,
	0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x33, 0x62, 0x75, 0x66, 0x2f,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x33, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x30, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x74, 0x61, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xdc, 0x03, 0x0a, 0x09, 0x52, 0x65, 0x66, 0x65,
	0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x47, 0x0a, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x42, 0x72,
	0x61, 0x6e, 0x63, 0x68, 0x48, 0x00, 0x52, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x3e,
	0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2a, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x54, 0x61, 0x67, 0x48, 0x00, 0x52, 0x03, 0x74, 0x61, 0x67, 0x12, 0x47,
	0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x48, 0x00, 0x52,
	0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x4a, 0x0a, 0x04, 0x6d, 0x61, 0x69, 0x6e, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x34, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x4d, 0x61,
	0x69, 0x6e, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x48, 0x00, 0x52, 0x04, 0x6d,
	0x61, 0x69, 0x6e, 0x12, 0x44, 0x0a, 0x05, 0x64, 0x72, 0x61, 0x66, 0x74, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x44, 0x72, 0x61, 0x66, 0x74,
	0x48, 0x00, 0x52, 0x05, 0x64, 0x72, 0x61, 0x66, 0x74, 0x12, 0x51, 0x0a, 0x0a, 0x76, 0x63, 0x73,
	0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x30, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x56, 0x43, 0x53, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x48,
	0x00, 0x52, 0x09, 0x76, 0x63, 0x73, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42, 0x0b, 0x0a, 0x09,
	0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x4a, 0x04, 0x08, 0x04, 0x10, 0x05, 0x52,
	0x05, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x22, 0x74, 0x0a, 0x17, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x4d, 0x61, 0x69, 0x6e, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x45, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f,
	0x6d, 0x6d, 0x69, 0x74, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x22, 0x6c, 0x0a, 0x0f,
	0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x44, 0x72, 0x61, 0x66, 0x74, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x45, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x43, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x22, 0x97, 0x01, 0x0a, 0x13, 0x52,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x56, 0x43, 0x53, 0x43, 0x6f, 0x6d, 0x6d,
	0x69, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x3b, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x4e, 0x61, 0x6d, 0x65, 0x22, 0x6e, 0x0a, 0x19, 0x47, 0x65, 0x74, 0x52, 0x65, 0x66, 0x65, 0x72,
	0x65, 0x6e, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x27, 0x0a, 0x0f, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79,
	0x4e, 0x61, 0x6d, 0x65, 0x22, 0x62, 0x0a, 0x1a, 0x47, 0x65, 0x74, 0x52, 0x65, 0x66, 0x65, 0x72,
	0x65, 0x6e, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x44, 0x0a, 0x09, 0x72, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x52, 0x09, 0x72,
	0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x32, 0x9f, 0x01, 0x0a, 0x10, 0x52, 0x65, 0x66,
	0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x8a, 0x01,
	0x0a, 0x12, 0x47, 0x65, 0x74, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x42, 0x79,
	0x4e, 0x61, 0x6d, 0x65, 0x12, 0x36, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x42,
	0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x37, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x65,
	0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x42, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x42, 0x9b, 0x02, 0x0a, 0x1f, 0x63,
	0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0e,
	0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01,
	0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66,
	0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74,
	0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62,
	0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41,
	0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x52, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca,
	0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x27,
	0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41,
	0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a,
	0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_buf_alpha_registry_v1alpha1_reference_proto_goTypes = []any{
	(*Reference)(nil),                  // 0: buf.alpha.registry.v1alpha1.Reference
	(*RepositoryMainReference)(nil),    // 1: buf.alpha.registry.v1alpha1.RepositoryMainReference
	(*RepositoryDraft)(nil),            // 2: buf.alpha.registry.v1alpha1.RepositoryDraft
	(*RepositoryVCSCommit)(nil),        // 3: buf.alpha.registry.v1alpha1.RepositoryVCSCommit
	(*GetReferenceByNameRequest)(nil),  // 4: buf.alpha.registry.v1alpha1.GetReferenceByNameRequest
	(*GetReferenceByNameResponse)(nil), // 5: buf.alpha.registry.v1alpha1.GetReferenceByNameResponse
	(*RepositoryBranch)(nil),           // 6: buf.alpha.registry.v1alpha1.RepositoryBranch
	(*RepositoryTag)(nil),              // 7: buf.alpha.registry.v1alpha1.RepositoryTag
	(*RepositoryCommit)(nil),           // 8: buf.alpha.registry.v1alpha1.RepositoryCommit
	(*timestamppb.Timestamp)(nil),      // 9: google.protobuf.Timestamp
}
var file_buf_alpha_registry_v1alpha1_reference_proto_depIdxs = []int32{
	6,  // 0: buf.alpha.registry.v1alpha1.Reference.branch:type_name -> buf.alpha.registry.v1alpha1.RepositoryBranch
	7,  // 1: buf.alpha.registry.v1alpha1.Reference.tag:type_name -> buf.alpha.registry.v1alpha1.RepositoryTag
	8,  // 2: buf.alpha.registry.v1alpha1.Reference.commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryCommit
	1,  // 3: buf.alpha.registry.v1alpha1.Reference.main:type_name -> buf.alpha.registry.v1alpha1.RepositoryMainReference
	2,  // 4: buf.alpha.registry.v1alpha1.Reference.draft:type_name -> buf.alpha.registry.v1alpha1.RepositoryDraft
	3,  // 5: buf.alpha.registry.v1alpha1.Reference.vcs_commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryVCSCommit
	8,  // 6: buf.alpha.registry.v1alpha1.RepositoryMainReference.commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryCommit
	8,  // 7: buf.alpha.registry.v1alpha1.RepositoryDraft.commit:type_name -> buf.alpha.registry.v1alpha1.RepositoryCommit
	9,  // 8: buf.alpha.registry.v1alpha1.RepositoryVCSCommit.create_time:type_name -> google.protobuf.Timestamp
	0,  // 9: buf.alpha.registry.v1alpha1.GetReferenceByNameResponse.reference:type_name -> buf.alpha.registry.v1alpha1.Reference
	4,  // 10: buf.alpha.registry.v1alpha1.ReferenceService.GetReferenceByName:input_type -> buf.alpha.registry.v1alpha1.GetReferenceByNameRequest
	5,  // 11: buf.alpha.registry.v1alpha1.ReferenceService.GetReferenceByName:output_type -> buf.alpha.registry.v1alpha1.GetReferenceByNameResponse
	11, // [11:12] is the sub-list for method output_type
	10, // [10:11] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_reference_proto_init() }
func file_buf_alpha_registry_v1alpha1_reference_proto_init() {
	if File_buf_alpha_registry_v1alpha1_reference_proto != nil {
		return
	}
	file_buf_alpha_registry_v1alpha1_repository_branch_proto_init()
	file_buf_alpha_registry_v1alpha1_repository_commit_proto_init()
	file_buf_alpha_registry_v1alpha1_repository_tag_proto_init()
	file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes[0].OneofWrappers = []any{
		(*reference_Branch)(nil),
		(*reference_Tag)(nil),
		(*reference_Commit)(nil),
		(*reference_Main)(nil),
		(*reference_Draft)(nil),
		(*reference_VcsCommit)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_reference_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_reference_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_reference_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_reference_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_reference_proto = out.File
	file_buf_alpha_registry_v1alpha1_reference_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_reference_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_reference_proto_depIdxs = nil
}
