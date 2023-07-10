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
// source: buf/alpha/registry/v1alpha1/labels.proto

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

type LabelNamespace int32

const (
	LabelNamespace_LABEL_NAMESPACE_UNSPECIFIED LabelNamespace = 0
	LabelNamespace_LABEL_NAMESPACE_TAG         LabelNamespace = 1
	LabelNamespace_LABEL_NAMESPACE_BRANCH      LabelNamespace = 2
	LabelNamespace_LABEL_NAMESPACE_GIT_COMMIT  LabelNamespace = 3
	LabelNamespace_LABEL_NAMESPACE_REVIEW      LabelNamespace = 4
	LabelNamespace_LABEL_NAMESPACE_BSR_HEAD    LabelNamespace = 5
)

// Enum value maps for LabelNamespace.
var (
	LabelNamespace_name = map[int32]string{
		0: "LABEL_NAMESPACE_UNSPECIFIED",
		1: "LABEL_NAMESPACE_TAG",
		2: "LABEL_NAMESPACE_BRANCH",
		3: "LABEL_NAMESPACE_GIT_COMMIT",
		4: "LABEL_NAMESPACE_REVIEW",
		5: "LABEL_NAMESPACE_BSR_HEAD",
	}
	LabelNamespace_value = map[string]int32{
		"LABEL_NAMESPACE_UNSPECIFIED": 0,
		"LABEL_NAMESPACE_TAG":         1,
		"LABEL_NAMESPACE_BRANCH":      2,
		"LABEL_NAMESPACE_GIT_COMMIT":  3,
		"LABEL_NAMESPACE_REVIEW":      4,
		"LABEL_NAMESPACE_BSR_HEAD":    5,
	}
)

func (x LabelNamespace) Enum() *LabelNamespace {
	p := new(LabelNamespace)
	*p = x
	return p
}

func (x LabelNamespace) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LabelNamespace) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_labels_proto_enumTypes[0].Descriptor()
}

func (LabelNamespace) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_labels_proto_enumTypes[0]
}

func (x LabelNamespace) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use LabelNamespace.Descriptor instead.
func (LabelNamespace) EnumDescriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{0}
}

type Label struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LabelName  *LabelName  `protobuf:"bytes,1,opt,name=label_name,json=labelName,proto3" json:"label_name,omitempty"`
	LabelValue *LabelValue `protobuf:"bytes,2,opt,name=label_value,json=labelValue,proto3" json:"label_value,omitempty"`
}

func (x *Label) Reset() {
	*x = Label{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Label) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Label) ProtoMessage() {}

func (x *Label) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Label.ProtoReflect.Descriptor instead.
func (*Label) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{0}
}

func (x *Label) GetLabelName() *LabelName {
	if x != nil {
		return x.LabelName
	}
	return nil
}

func (x *Label) GetLabelValue() *LabelValue {
	if x != nil {
		return x.LabelValue
	}
	return nil
}

type LabelName struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace LabelNamespace `protobuf:"varint,1,opt,name=namespace,proto3,enum=buf.alpha.registry.v1alpha1.LabelNamespace" json:"namespace,omitempty"`
	Name      string         `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *LabelName) Reset() {
	*x = LabelName{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LabelName) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelName) ProtoMessage() {}

func (x *LabelName) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelName.ProtoReflect.Descriptor instead.
func (*LabelName) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{1}
}

func (x *LabelName) GetNamespace() LabelNamespace {
	if x != nil {
		return x.Namespace
	}
	return LabelNamespace_LABEL_NAMESPACE_UNSPECIFIED
}

func (x *LabelName) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type LabelValue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommitId string `protobuf:"bytes,1,opt,name=commit_id,json=commitId,proto3" json:"commit_id,omitempty"`
}

func (x *LabelValue) Reset() {
	*x = LabelValue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LabelValue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelValue) ProtoMessage() {}

func (x *LabelValue) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelValue.ProtoReflect.Descriptor instead.
func (*LabelValue) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{2}
}

func (x *LabelValue) GetCommitId() string {
	if x != nil {
		return x.CommitId
	}
	return ""
}

type CreateLabelRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LabelName  *LabelName             `protobuf:"bytes,1,opt,name=label_name,json=labelName,proto3" json:"label_name,omitempty"`
	LabelValue *LabelValue            `protobuf:"bytes,2,opt,name=label_value,json=labelValue,proto3" json:"label_value,omitempty"`
	Author     *string                `protobuf:"bytes,3,opt,name=author,proto3,oneof" json:"author,omitempty"`
	CreateTime *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=create_time,json=createTime,proto3,oneof" json:"create_time,omitempty"`
}

func (x *CreateLabelRequest) Reset() {
	*x = CreateLabelRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateLabelRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateLabelRequest) ProtoMessage() {}

func (x *CreateLabelRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateLabelRequest.ProtoReflect.Descriptor instead.
func (*CreateLabelRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{3}
}

func (x *CreateLabelRequest) GetLabelName() *LabelName {
	if x != nil {
		return x.LabelName
	}
	return nil
}

func (x *CreateLabelRequest) GetLabelValue() *LabelValue {
	if x != nil {
		return x.LabelValue
	}
	return nil
}

func (x *CreateLabelRequest) GetAuthor() string {
	if x != nil && x.Author != nil {
		return *x.Author
	}
	return ""
}

func (x *CreateLabelRequest) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

type CreateLabelResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CommitId *LabelValue `protobuf:"bytes,1,opt,name=commit_id,json=commitId,proto3" json:"commit_id,omitempty"`
}

func (x *CreateLabelResponse) Reset() {
	*x = CreateLabelResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateLabelResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateLabelResponse) ProtoMessage() {}

func (x *CreateLabelResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateLabelResponse.ProtoReflect.Descriptor instead.
func (*CreateLabelResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{4}
}

func (x *CreateLabelResponse) GetCommitId() *LabelValue {
	if x != nil {
		return x.CommitId
	}
	return nil
}

type MoveLabelRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LabelName *LabelName  `protobuf:"bytes,1,opt,name=label_name,json=labelName,proto3" json:"label_name,omitempty"`
	From      *LabelValue `protobuf:"bytes,2,opt,name=from,proto3,oneof" json:"from,omitempty"`
	To        *LabelValue `protobuf:"bytes,3,opt,name=to,proto3" json:"to,omitempty"`
}

func (x *MoveLabelRequest) Reset() {
	*x = MoveLabelRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MoveLabelRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MoveLabelRequest) ProtoMessage() {}

func (x *MoveLabelRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MoveLabelRequest.ProtoReflect.Descriptor instead.
func (*MoveLabelRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{5}
}

func (x *MoveLabelRequest) GetLabelName() *LabelName {
	if x != nil {
		return x.LabelName
	}
	return nil
}

func (x *MoveLabelRequest) GetFrom() *LabelValue {
	if x != nil {
		return x.From
	}
	return nil
}

func (x *MoveLabelRequest) GetTo() *LabelValue {
	if x != nil {
		return x.To
	}
	return nil
}

type MoveLabelResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *MoveLabelResponse) Reset() {
	*x = MoveLabelResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MoveLabelResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MoveLabelResponse) ProtoMessage() {}

func (x *MoveLabelResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MoveLabelResponse.ProtoReflect.Descriptor instead.
func (*MoveLabelResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{6}
}

type GetLabelsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	RepositoryOwner string      `protobuf:"bytes,1,opt,name=repository_owner,json=repositoryOwner,proto3" json:"repository_owner,omitempty"`
	RepositoryName  string      `protobuf:"bytes,2,opt,name=repository_name,json=repositoryName,proto3" json:"repository_name,omitempty"`
	LabelName       *LabelName  `protobuf:"bytes,3,opt,name=label_name,json=labelName,proto3,oneof" json:"label_name,omitempty"`
	LabelValue      *LabelValue `protobuf:"bytes,4,opt,name=label_value,json=labelValue,proto3,oneof" json:"label_value,omitempty"`
}

func (x *GetLabelsRequest) Reset() {
	*x = GetLabelsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetLabelsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetLabelsRequest) ProtoMessage() {}

func (x *GetLabelsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetLabelsRequest.ProtoReflect.Descriptor instead.
func (*GetLabelsRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{7}
}

func (x *GetLabelsRequest) GetRepositoryOwner() string {
	if x != nil {
		return x.RepositoryOwner
	}
	return ""
}

func (x *GetLabelsRequest) GetRepositoryName() string {
	if x != nil {
		return x.RepositoryName
	}
	return ""
}

func (x *GetLabelsRequest) GetLabelName() *LabelName {
	if x != nil {
		return x.LabelName
	}
	return nil
}

func (x *GetLabelsRequest) GetLabelValue() *LabelValue {
	if x != nil {
		return x.LabelValue
	}
	return nil
}

type GetLabelsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Labels []*Label `protobuf:"bytes,1,rep,name=labels,proto3" json:"labels,omitempty"`
}

func (x *GetLabelsResponse) Reset() {
	*x = GetLabelsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetLabelsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetLabelsResponse) ProtoMessage() {}

func (x *GetLabelsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetLabelsResponse.ProtoReflect.Descriptor instead.
func (*GetLabelsResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP(), []int{8}
}

func (x *GetLabelsResponse) GetLabels() []*Label {
	if x != nil {
		return x.Labels
	}
	return nil
}

var File_buf_alpha_registry_v1alpha1_labels_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_labels_proto_rawDesc = []byte{
	0x0a, 0x28, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x6c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61,
	0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x98, 0x01, 0x0a, 0x05, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x12, 0x45, 0x0a, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x09,
	0x6c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x48, 0x0a, 0x0b, 0x6c, 0x61, 0x62,
	0x65, 0x6c, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x22, 0x6a, 0x0a, 0x09, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x49, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x2b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x22,
	0x29, 0x0a, 0x0a, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x1b, 0x0a,
	0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x64, 0x22, 0x9f, 0x02, 0x0a, 0x12, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x45, 0x0a, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x09, 0x6c,
	0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x48, 0x0a, 0x0b, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65,
	0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x12, 0x1b, 0x0a, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x00, 0x52, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x88, 0x01, 0x01, 0x12,
	0x40, 0x0a, 0x0b, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x48, 0x01, 0x52, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x88, 0x01,
	0x01, 0x42, 0x09, 0x0a, 0x07, 0x5f, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x42, 0x0e, 0x0a, 0x0c,
	0x5f, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x22, 0x5b, 0x0a, 0x13,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x44, 0x0a, 0x09, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52,
	0x08, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x64, 0x22, 0xdd, 0x01, 0x0a, 0x10, 0x4d, 0x6f,
	0x76, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x45,
	0x0a, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x52, 0x09, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x40, 0x0a, 0x04, 0x66, 0x72, 0x6f, 0x6d, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x48, 0x00, 0x52, 0x04,
	0x66, 0x72, 0x6f, 0x6d, 0x88, 0x01, 0x01, 0x12, 0x37, 0x0a, 0x02, 0x74, 0x6f, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x02, 0x74, 0x6f,
	0x42, 0x07, 0x0a, 0x05, 0x5f, 0x66, 0x72, 0x6f, 0x6d, 0x22, 0x13, 0x0a, 0x11, 0x4d, 0x6f, 0x76,
	0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0xa0,
	0x02, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x29, 0x0a, 0x10, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x5f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x27,
	0x0a, 0x0f, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x4a, 0x0a, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e,
	0x61, 0x6d, 0x65, 0x48, 0x00, 0x52, 0x09, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65,
	0x88, 0x01, 0x01, 0x12, 0x4d, 0x0a, 0x0b, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x48, 0x01, 0x52, 0x0a, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x88,
	0x01, 0x01, 0x42, 0x0d, 0x0a, 0x0b, 0x5f, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x42, 0x0e, 0x0a, 0x0c, 0x5f, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x5f, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x22, 0x4f, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3a, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x73, 0x2a, 0xc0, 0x01, 0x0a, 0x0e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x1f, 0x0a, 0x1b, 0x4c, 0x41, 0x42, 0x45, 0x4c, 0x5f, 0x4e,
	0x41, 0x4d, 0x45, 0x53, 0x50, 0x41, 0x43, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49,
	0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x17, 0x0a, 0x13, 0x4c, 0x41, 0x42, 0x45, 0x4c, 0x5f,
	0x4e, 0x41, 0x4d, 0x45, 0x53, 0x50, 0x41, 0x43, 0x45, 0x5f, 0x54, 0x41, 0x47, 0x10, 0x01, 0x12,
	0x1a, 0x0a, 0x16, 0x4c, 0x41, 0x42, 0x45, 0x4c, 0x5f, 0x4e, 0x41, 0x4d, 0x45, 0x53, 0x50, 0x41,
	0x43, 0x45, 0x5f, 0x42, 0x52, 0x41, 0x4e, 0x43, 0x48, 0x10, 0x02, 0x12, 0x1e, 0x0a, 0x1a, 0x4c,
	0x41, 0x42, 0x45, 0x4c, 0x5f, 0x4e, 0x41, 0x4d, 0x45, 0x53, 0x50, 0x41, 0x43, 0x45, 0x5f, 0x47,
	0x49, 0x54, 0x5f, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x03, 0x12, 0x1a, 0x0a, 0x16, 0x4c,
	0x41, 0x42, 0x45, 0x4c, 0x5f, 0x4e, 0x41, 0x4d, 0x45, 0x53, 0x50, 0x41, 0x43, 0x45, 0x5f, 0x52,
	0x45, 0x56, 0x49, 0x45, 0x57, 0x10, 0x04, 0x12, 0x1c, 0x0a, 0x18, 0x4c, 0x41, 0x42, 0x45, 0x4c,
	0x5f, 0x4e, 0x41, 0x4d, 0x45, 0x53, 0x50, 0x41, 0x43, 0x45, 0x5f, 0x42, 0x53, 0x52, 0x5f, 0x48,
	0x45, 0x41, 0x44, 0x10, 0x05, 0x32, 0xe2, 0x02, 0x0a, 0x0c, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x75, 0x0a, 0x0b, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x4c, 0x61, 0x62, 0x65, 0x6c, 0x12, 0x2f, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x02, 0x12, 0x6a, 0x0a,
	0x09, 0x4d, 0x6f, 0x76, 0x65, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x12, 0x2d, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x6f, 0x76, 0x65, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x6f, 0x76, 0x65, 0x4c, 0x61, 0x62, 0x65,
	0x6c, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x6f, 0x0a, 0x09, 0x47, 0x65, 0x74,
	0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x12, 0x2d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x42, 0x98, 0x02, 0x0a, 0x1f, 0x63,
	0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0b,
	0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67,
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
	file_buf_alpha_registry_v1alpha1_labels_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_labels_proto_rawDescData = file_buf_alpha_registry_v1alpha1_labels_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_labels_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_labels_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_labels_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_labels_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_labels_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_labels_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_buf_alpha_registry_v1alpha1_labels_proto_goTypes = []interface{}{
	(LabelNamespace)(0),           // 0: buf.alpha.registry.v1alpha1.LabelNamespace
	(*Label)(nil),                 // 1: buf.alpha.registry.v1alpha1.Label
	(*LabelName)(nil),             // 2: buf.alpha.registry.v1alpha1.LabelName
	(*LabelValue)(nil),            // 3: buf.alpha.registry.v1alpha1.LabelValue
	(*CreateLabelRequest)(nil),    // 4: buf.alpha.registry.v1alpha1.CreateLabelRequest
	(*CreateLabelResponse)(nil),   // 5: buf.alpha.registry.v1alpha1.CreateLabelResponse
	(*MoveLabelRequest)(nil),      // 6: buf.alpha.registry.v1alpha1.MoveLabelRequest
	(*MoveLabelResponse)(nil),     // 7: buf.alpha.registry.v1alpha1.MoveLabelResponse
	(*GetLabelsRequest)(nil),      // 8: buf.alpha.registry.v1alpha1.GetLabelsRequest
	(*GetLabelsResponse)(nil),     // 9: buf.alpha.registry.v1alpha1.GetLabelsResponse
	(*timestamppb.Timestamp)(nil), // 10: google.protobuf.Timestamp
}
var file_buf_alpha_registry_v1alpha1_labels_proto_depIdxs = []int32{
	2,  // 0: buf.alpha.registry.v1alpha1.Label.label_name:type_name -> buf.alpha.registry.v1alpha1.LabelName
	3,  // 1: buf.alpha.registry.v1alpha1.Label.label_value:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	0,  // 2: buf.alpha.registry.v1alpha1.LabelName.namespace:type_name -> buf.alpha.registry.v1alpha1.LabelNamespace
	2,  // 3: buf.alpha.registry.v1alpha1.CreateLabelRequest.label_name:type_name -> buf.alpha.registry.v1alpha1.LabelName
	3,  // 4: buf.alpha.registry.v1alpha1.CreateLabelRequest.label_value:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	10, // 5: buf.alpha.registry.v1alpha1.CreateLabelRequest.create_time:type_name -> google.protobuf.Timestamp
	3,  // 6: buf.alpha.registry.v1alpha1.CreateLabelResponse.commit_id:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	2,  // 7: buf.alpha.registry.v1alpha1.MoveLabelRequest.label_name:type_name -> buf.alpha.registry.v1alpha1.LabelName
	3,  // 8: buf.alpha.registry.v1alpha1.MoveLabelRequest.from:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	3,  // 9: buf.alpha.registry.v1alpha1.MoveLabelRequest.to:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	2,  // 10: buf.alpha.registry.v1alpha1.GetLabelsRequest.label_name:type_name -> buf.alpha.registry.v1alpha1.LabelName
	3,  // 11: buf.alpha.registry.v1alpha1.GetLabelsRequest.label_value:type_name -> buf.alpha.registry.v1alpha1.LabelValue
	1,  // 12: buf.alpha.registry.v1alpha1.GetLabelsResponse.labels:type_name -> buf.alpha.registry.v1alpha1.Label
	4,  // 13: buf.alpha.registry.v1alpha1.LabelService.CreateLabel:input_type -> buf.alpha.registry.v1alpha1.CreateLabelRequest
	6,  // 14: buf.alpha.registry.v1alpha1.LabelService.MoveLabel:input_type -> buf.alpha.registry.v1alpha1.MoveLabelRequest
	8,  // 15: buf.alpha.registry.v1alpha1.LabelService.GetLabels:input_type -> buf.alpha.registry.v1alpha1.GetLabelsRequest
	5,  // 16: buf.alpha.registry.v1alpha1.LabelService.CreateLabel:output_type -> buf.alpha.registry.v1alpha1.CreateLabelResponse
	7,  // 17: buf.alpha.registry.v1alpha1.LabelService.MoveLabel:output_type -> buf.alpha.registry.v1alpha1.MoveLabelResponse
	9,  // 18: buf.alpha.registry.v1alpha1.LabelService.GetLabels:output_type -> buf.alpha.registry.v1alpha1.GetLabelsResponse
	16, // [16:19] is the sub-list for method output_type
	13, // [13:16] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_labels_proto_init() }
func file_buf_alpha_registry_v1alpha1_labels_proto_init() {
	if File_buf_alpha_registry_v1alpha1_labels_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Label); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LabelName); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LabelValue); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateLabelRequest); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateLabelResponse); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MoveLabelRequest); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MoveLabelResponse); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetLabelsRequest); i {
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
		file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetLabelsResponse); i {
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
	file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[3].OneofWrappers = []interface{}{}
	file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[5].OneofWrappers = []interface{}{}
	file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes[7].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_labels_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_labels_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_labels_proto_depIdxs,
		EnumInfos:         file_buf_alpha_registry_v1alpha1_labels_proto_enumTypes,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_labels_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_labels_proto = out.File
	file_buf_alpha_registry_v1alpha1_labels_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_labels_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_labels_proto_depIdxs = nil
}
