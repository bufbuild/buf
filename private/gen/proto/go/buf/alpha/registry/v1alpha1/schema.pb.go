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
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/schema.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Format int32

const (
	Format_FORMAT_UNSPECIFIED Format = 0
	Format_FORMAT_BINARY      Format = 1
	Format_FORMAT_JSON        Format = 2
	Format_FORMAT_TEXT        Format = 3
)

// Enum value maps for Format.
var (
	Format_name = map[int32]string{
		0: "FORMAT_UNSPECIFIED",
		1: "FORMAT_BINARY",
		2: "FORMAT_JSON",
		3: "FORMAT_TEXT",
	}
	Format_value = map[string]int32{
		"FORMAT_UNSPECIFIED": 0,
		"FORMAT_BINARY":      1,
		"FORMAT_JSON":        2,
		"FORMAT_TEXT":        3,
	}
)

func (x Format) Enum() *Format {
	p := new(Format)
	*p = x
	return p
}

func (x Format) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Format) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_schema_proto_enumTypes[0].Descriptor()
}

func (Format) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_schema_proto_enumTypes[0]
}

func (x Format) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Format.Descriptor instead.
func (Format) EnumDescriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{0}
}

type GetSchemaRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The owner of the repo that contains the schema to retrieve (a user name or
	// organization name).
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// The name of the repo that contains the schema to retrieve.
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	// Optional version of the repo. If unspecified, defaults to latest version on
	// the repo's "main" branch.
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// Zero or more types names. The names may refer to messages, enums, services,
	// methods, or extensions. All names must be fully-qualified. If any name
	// is unknown, the request will fail and no schema will be returned.
	//
	// If no names are provided, the full schema for the module is returned.
	// Otherwise, the resulting schema contains only the named elements and all of
	// their dependencies. This is enough information for the caller to construct
	// a dynamic message for any requested message types or to dynamically invoke
	// an RPC for any requested methods or services.
	Types []string `protobuf:"bytes,4,rep,name=types,proto3" json:"types,omitempty"`
	// If present, this is a commit that the client already has cached. So if the
	// given module version resolves to this same commit, the server should not
	// send back any descriptors since the client already has them.
	//
	// This allows a client to efficiently poll for updates: after the initial RPC
	// to get a schema, the client can cache the descriptors and the resolved
	// commit. It then includes that commit in subsequent requests in this field,
	// and the server will only reply with a schema (and new commit) if/when the
	// resolved commit changes.
	IfNotCommit string `protobuf:"bytes,5,opt,name=if_not_commit,json=ifNotCommit,proto3" json:"if_not_commit,omitempty"`
	// If true, the returned schema will not include extension definitions for custom
	// options that appear on schema elements. When filtering the schema based on the
	// given element names, options on all encountered elements are usually examined
	// as well. But that is not the case if excluding custom options.
	//
	// This flag is ignored if element_names is empty as the entire schema is always
	// returned in that case.
	ExcludeCustomOptions bool `protobuf:"varint,6,opt,name=exclude_custom_options,json=excludeCustomOptions,proto3" json:"exclude_custom_options,omitempty"`
	// If true, the returned schema will not include known extensions for extendable
	// messages for schema elements. If exclude_custom_options is true, such extensions
	// may still be returned if the applicable descriptor options type is part of the
	// requested schema.
	//
	// This flag is ignored if element_names is empty as the entire schema is always
	// returned in that case.
	ExcludeKnownExtensions bool `protobuf:"varint,7,opt,name=exclude_known_extensions,json=excludeKnownExtensions,proto3" json:"exclude_known_extensions,omitempty"`
}

func (x *GetSchemaRequest) Reset() {
	*x = GetSchemaRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetSchemaRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSchemaRequest) ProtoMessage() {}

func (x *GetSchemaRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetSchemaRequest.ProtoReflect.Descriptor instead.
func (*GetSchemaRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{0}
}

func (x *GetSchemaRequest) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *GetSchemaRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *GetSchemaRequest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *GetSchemaRequest) GetTypes() []string {
	if x != nil {
		return x.Types
	}
	return nil
}

func (x *GetSchemaRequest) GetIfNotCommit() string {
	if x != nil {
		return x.IfNotCommit
	}
	return ""
}

func (x *GetSchemaRequest) GetExcludeCustomOptions() bool {
	if x != nil {
		return x.ExcludeCustomOptions
	}
	return false
}

func (x *GetSchemaRequest) GetExcludeKnownExtensions() bool {
	if x != nil {
		return x.ExcludeKnownExtensions
	}
	return false
}

type GetSchemaResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The resolved version of the schema. If the requested version was a commit,
	// this value is the same as that. If the requested version referred to a tag
	// or branch, this is the commit for that tag or latest commit for that
	// branch. If the request did not include any version, this is the latest
	// version for the module's main branch.
	Commit string `protobuf:"bytes,1,opt,name=commit,proto3" json:"commit,omitempty"`
	// The schema, which is a set of file descriptors that include the requested elements
	// and their dependencies.
	SchemaFiles *descriptorpb.FileDescriptorSet `protobuf:"bytes,2,opt,name=schema_files,json=schemaFiles,proto3" json:"schema_files,omitempty"`
}

func (x *GetSchemaResponse) Reset() {
	*x = GetSchemaResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetSchemaResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSchemaResponse) ProtoMessage() {}

func (x *GetSchemaResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetSchemaResponse.ProtoReflect.Descriptor instead.
func (*GetSchemaResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{1}
}

func (x *GetSchemaResponse) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *GetSchemaResponse) GetSchemaFiles() *descriptorpb.FileDescriptorSet {
	if x != nil {
		return x.SchemaFiles
	}
	return nil
}

type ConvertMessageRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The owner of the repo that contains the schema to retrieve (a user name or
	// organization name).
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// The name of the repo that contains the schema to retrieve.
	Repository string `protobuf:"bytes,2,opt,name=repository,proto3" json:"repository,omitempty"`
	// Optional version of the repo. This can be a tag or branch name or a commit.
	// If unspecified, defaults to latest version on the repo's "main" branch.
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// The fully-qualified name of the message. Required.
	MessageName string `protobuf:"bytes,4,opt,name=message_name,json=messageName,proto3" json:"message_name,omitempty"`
	// The format of the input data. Required.
	InputFormat Format `protobuf:"varint,5,opt,name=input_format,json=inputFormat,proto3,enum=buf.alpha.registry.v1alpha1.Format" json:"input_format,omitempty"`
	// The input data that is to be converted. Required. This must be
	// a valid encoding of type indicated by message_name in the format
	// indicated by input_format.
	InputData []byte `protobuf:"bytes,6,opt,name=input_data,json=inputData,proto3" json:"input_data,omitempty"`
	// If true, any unresolvable fields in the input are discarded. For
	// formats other than FORMAT_BINARY, this means that the operation
	// will fail if the input contains unrecognized field names. For
	// FORMAT_BINARY, unrecognized fields can be retained and possibly
	// included in the reformatted output (depending on the requested
	// output format).
	DiscardUnknown bool `protobuf:"varint,7,opt,name=discard_unknown,json=discardUnknown,proto3" json:"discard_unknown,omitempty"`
	// Types that are assignable to OutputFormat:
	//
	//	*ConvertMessageRequest_OutputBinary
	//	*ConvertMessageRequest_OutputJson
	//	*ConvertMessageRequest_OutputText
	OutputFormat isConvertMessageRequest_OutputFormat `protobuf_oneof:"output_format"`
}

func (x *ConvertMessageRequest) Reset() {
	*x = ConvertMessageRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConvertMessageRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertMessageRequest) ProtoMessage() {}

func (x *ConvertMessageRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConvertMessageRequest.ProtoReflect.Descriptor instead.
func (*ConvertMessageRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{2}
}

func (x *ConvertMessageRequest) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *ConvertMessageRequest) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *ConvertMessageRequest) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *ConvertMessageRequest) GetMessageName() string {
	if x != nil {
		return x.MessageName
	}
	return ""
}

func (x *ConvertMessageRequest) GetInputFormat() Format {
	if x != nil {
		return x.InputFormat
	}
	return Format_FORMAT_UNSPECIFIED
}

func (x *ConvertMessageRequest) GetInputData() []byte {
	if x != nil {
		return x.InputData
	}
	return nil
}

func (x *ConvertMessageRequest) GetDiscardUnknown() bool {
	if x != nil {
		return x.DiscardUnknown
	}
	return false
}

func (m *ConvertMessageRequest) GetOutputFormat() isConvertMessageRequest_OutputFormat {
	if m != nil {
		return m.OutputFormat
	}
	return nil
}

func (x *ConvertMessageRequest) GetOutputBinary() *BinaryOutputOptions {
	if x, ok := x.GetOutputFormat().(*ConvertMessageRequest_OutputBinary); ok {
		return x.OutputBinary
	}
	return nil
}

func (x *ConvertMessageRequest) GetOutputJson() *JSONOutputOptions {
	if x, ok := x.GetOutputFormat().(*ConvertMessageRequest_OutputJson); ok {
		return x.OutputJson
	}
	return nil
}

func (x *ConvertMessageRequest) GetOutputText() *TextOutputOptions {
	if x, ok := x.GetOutputFormat().(*ConvertMessageRequest_OutputText); ok {
		return x.OutputText
	}
	return nil
}

type isConvertMessageRequest_OutputFormat interface {
	isConvertMessageRequest_OutputFormat()
}

type ConvertMessageRequest_OutputBinary struct {
	OutputBinary *BinaryOutputOptions `protobuf:"bytes,8,opt,name=output_binary,json=outputBinary,proto3,oneof"`
}

type ConvertMessageRequest_OutputJson struct {
	OutputJson *JSONOutputOptions `protobuf:"bytes,9,opt,name=output_json,json=outputJson,proto3,oneof"`
}

type ConvertMessageRequest_OutputText struct {
	OutputText *TextOutputOptions `protobuf:"bytes,10,opt,name=output_text,json=outputText,proto3,oneof"`
}

func (*ConvertMessageRequest_OutputBinary) isConvertMessageRequest_OutputFormat() {}

func (*ConvertMessageRequest_OutputJson) isConvertMessageRequest_OutputFormat() {}

func (*ConvertMessageRequest_OutputText) isConvertMessageRequest_OutputFormat() {}

type BinaryOutputOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *BinaryOutputOptions) Reset() {
	*x = BinaryOutputOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BinaryOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BinaryOutputOptions) ProtoMessage() {}

func (x *BinaryOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BinaryOutputOptions.ProtoReflect.Descriptor instead.
func (*BinaryOutputOptions) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{3}
}

type JSONOutputOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Enum fields will be emitted as numeric values. If false (the default), enum
	// fields are emitted as strings that are the enum values' names.
	UseEnumNumbers bool `protobuf:"varint,3,opt,name=use_enum_numbers,json=useEnumNumbers,proto3" json:"use_enum_numbers,omitempty"`
	// Includes fields that have their default values. This applies only to fields
	// defined in proto3 syntax that have no explicit "optional" keyword. Other
	// optional fields will be included if present in the input data.
	IncludeDefaults bool `protobuf:"varint,4,opt,name=include_defaults,json=includeDefaults,proto3" json:"include_defaults,omitempty"`
}

func (x *JSONOutputOptions) Reset() {
	*x = JSONOutputOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JSONOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JSONOutputOptions) ProtoMessage() {}

func (x *JSONOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JSONOutputOptions.ProtoReflect.Descriptor instead.
func (*JSONOutputOptions) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{4}
}

func (x *JSONOutputOptions) GetUseEnumNumbers() bool {
	if x != nil {
		return x.UseEnumNumbers
	}
	return false
}

func (x *JSONOutputOptions) GetIncludeDefaults() bool {
	if x != nil {
		return x.IncludeDefaults
	}
	return false
}

type TextOutputOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// If true and the input data includes unrecognized fields, the unrecognized
	// fields will be preserved in the text output (using field numbers and raw
	// values).
	IncludeUnrecognized bool `protobuf:"varint,2,opt,name=include_unrecognized,json=includeUnrecognized,proto3" json:"include_unrecognized,omitempty"`
}

func (x *TextOutputOptions) Reset() {
	*x = TextOutputOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TextOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TextOutputOptions) ProtoMessage() {}

func (x *TextOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TextOutputOptions.ProtoReflect.Descriptor instead.
func (*TextOutputOptions) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{5}
}

func (x *TextOutputOptions) GetIncludeUnrecognized() bool {
	if x != nil {
		return x.IncludeUnrecognized
	}
	return false
}

type ConvertMessageResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The resolved version of the schema. If the requested version was a commit,
	// this value is the same as that. If the requested version referred to a tag
	// or branch, this is the commit for that tag or latest commit for that
	// branch. If the request did not include any version, this is the latest
	// version for the module's main branch.
	Commit string `protobuf:"bytes,1,opt,name=commit,proto3" json:"commit,omitempty"`
	// The reformatted data.
	OutputData []byte `protobuf:"bytes,2,opt,name=output_data,json=outputData,proto3" json:"output_data,omitempty"`
}

func (x *ConvertMessageResponse) Reset() {
	*x = ConvertMessageResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConvertMessageResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertMessageResponse) ProtoMessage() {}

func (x *ConvertMessageResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConvertMessageResponse.ProtoReflect.Descriptor instead.
func (*ConvertMessageResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP(), []int{6}
}

func (x *ConvertMessageResponse) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *ConvertMessageResponse) GetOutputData() []byte {
	if x != nil {
		return x.OutputData
	}
	return nil
}

var File_buf_alpha_registry_v1alpha1_schema_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc = []byte{
	0x0a, 0x28, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x73, 0x63,
	0x68, 0x65, 0x6d, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8c, 0x02, 0x0a, 0x10, 0x47, 0x65,
	0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14,
	0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f,
	0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f,
	0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69,
	0x74, 0x6f, 0x72, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x14,
	0x0a, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x74,
	0x79, 0x70, 0x65, 0x73, 0x12, 0x22, 0x0a, 0x0d, 0x69, 0x66, 0x5f, 0x6e, 0x6f, 0x74, 0x5f, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x69, 0x66, 0x4e,
	0x6f, 0x74, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x34, 0x0a, 0x16, 0x65, 0x78, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x5f, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x14, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64,
	0x65, 0x43, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x38,
	0x0a, 0x18, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x5f, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x5f,
	0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x16, 0x65, 0x78, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x4b, 0x6e, 0x6f, 0x77, 0x6e, 0x45, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0x72, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x53,
	0x63, 0x68, 0x65, 0x6d, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a,
	0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x45, 0x0a, 0x0c, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x5f,
	0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69,
	0x6c, 0x65, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x74, 0x52,
	0x0b, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x22, 0xaa, 0x04, 0x0a,
	0x15, 0x43, 0x6f, 0x6e, 0x76, 0x65, 0x72, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a,
	0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x18, 0x0a, 0x07,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x46, 0x0a, 0x0c, 0x69, 0x6e, 0x70,
	0x75, 0x74, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x23, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x46, 0x6f,
	0x72, 0x6d, 0x61, 0x74, 0x52, 0x0b, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x46, 0x6f, 0x72, 0x6d, 0x61,
	0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x44, 0x61, 0x74, 0x61,
	0x12, 0x27, 0x0a, 0x0f, 0x64, 0x69, 0x73, 0x63, 0x61, 0x72, 0x64, 0x5f, 0x75, 0x6e, 0x6b, 0x6e,
	0x6f, 0x77, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e, 0x64, 0x69, 0x73, 0x63, 0x61,
	0x72, 0x64, 0x55, 0x6e, 0x6b, 0x6e, 0x6f, 0x77, 0x6e, 0x12, 0x57, 0x0a, 0x0d, 0x6f, 0x75, 0x74,
	0x70, 0x75, 0x74, 0x5f, 0x62, 0x69, 0x6e, 0x61, 0x72, 0x79, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x30, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x42,
	0x69, 0x6e, 0x61, 0x72, 0x79, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x48, 0x00, 0x52, 0x0c, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x42, 0x69, 0x6e, 0x61,
	0x72, 0x79, 0x12, 0x51, 0x0a, 0x0b, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x6a, 0x73, 0x6f,
	0x6e, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4a, 0x53, 0x4f, 0x4e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x48, 0x00, 0x52, 0x0a, 0x6f, 0x75, 0x74, 0x70, 0x75,
	0x74, 0x4a, 0x73, 0x6f, 0x6e, 0x12, 0x51, 0x0a, 0x0b, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f,
	0x74, 0x65, 0x78, 0x74, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x54, 0x65, 0x78, 0x74, 0x4f, 0x75, 0x74,
	0x70, 0x75, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x48, 0x00, 0x52, 0x0a, 0x6f, 0x75,
	0x74, 0x70, 0x75, 0x74, 0x54, 0x65, 0x78, 0x74, 0x42, 0x0f, 0x0a, 0x0d, 0x6f, 0x75, 0x74, 0x70,
	0x75, 0x74, 0x5f, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x22, 0x15, 0x0a, 0x13, 0x42, 0x69, 0x6e,
	0x61, 0x72, 0x79, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x22, 0x68, 0x0a, 0x11, 0x4a, 0x53, 0x4f, 0x4e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x28, 0x0a, 0x10, 0x75, 0x73, 0x65, 0x5f, 0x65, 0x6e, 0x75,
	0x6d, 0x5f, 0x6e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x0e, 0x75, 0x73, 0x65, 0x45, 0x6e, 0x75, 0x6d, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x73, 0x12,
	0x29, 0x0a, 0x10, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x5f, 0x64, 0x65, 0x66, 0x61, 0x75,
	0x6c, 0x74, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0f, 0x69, 0x6e, 0x63, 0x6c, 0x75,
	0x64, 0x65, 0x44, 0x65, 0x66, 0x61, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x46, 0x0a, 0x11, 0x54, 0x65,
	0x78, 0x74, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12,
	0x31, 0x0a, 0x14, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x5f, 0x75, 0x6e, 0x72, 0x65, 0x63,
	0x6f, 0x67, 0x6e, 0x69, 0x7a, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x13, 0x69,
	0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x55, 0x6e, 0x72, 0x65, 0x63, 0x6f, 0x67, 0x6e, 0x69, 0x7a,
	0x65, 0x64, 0x22, 0x51, 0x0a, 0x16, 0x43, 0x6f, 0x6e, 0x76, 0x65, 0x72, 0x74, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06,
	0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f,
	0x6d, 0x6d, 0x69, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x6f, 0x75, 0x74, 0x70, 0x75,
	0x74, 0x44, 0x61, 0x74, 0x61, 0x2a, 0x55, 0x0a, 0x06, 0x46, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x12,
	0x16, 0x0a, 0x12, 0x46, 0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x11, 0x0a, 0x0d, 0x46, 0x4f, 0x52, 0x4d, 0x41,
	0x54, 0x5f, 0x42, 0x49, 0x4e, 0x41, 0x52, 0x59, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b, 0x46, 0x4f,
	0x52, 0x4d, 0x41, 0x54, 0x5f, 0x4a, 0x53, 0x4f, 0x4e, 0x10, 0x02, 0x12, 0x0f, 0x0a, 0x0b, 0x46,
	0x4f, 0x52, 0x4d, 0x41, 0x54, 0x5f, 0x54, 0x45, 0x58, 0x54, 0x10, 0x03, 0x32, 0xfb, 0x01, 0x0a,
	0x0d, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6f,
	0x0a, 0x09, 0x47, 0x65, 0x74, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x2d, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x63, 0x68,
	0x65, 0x6d, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x63, 0x68, 0x65,
	0x6d, 0x61, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x90, 0x02, 0x01, 0x12,
	0x79, 0x0a, 0x0e, 0x43, 0x6f, 0x6e, 0x76, 0x65, 0x72, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x12, 0x32, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x43, 0x6f, 0x6e, 0x76, 0x65, 0x72, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x33, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x43, 0x6f, 0x6e, 0x76, 0x65, 0x72, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x98, 0x02, 0x0a, 0x1f, 0x63,
	0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0b,
	0x53, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67,
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
	file_buf_alpha_registry_v1alpha1_schema_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_schema_proto_rawDescData = file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_schema_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_schema_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_schema_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_schema_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_schema_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_schema_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_buf_alpha_registry_v1alpha1_schema_proto_goTypes = []any{
	(Format)(0),                            // 0: buf.alpha.registry.v1alpha1.Format
	(*GetSchemaRequest)(nil),               // 1: buf.alpha.registry.v1alpha1.GetSchemaRequest
	(*GetSchemaResponse)(nil),              // 2: buf.alpha.registry.v1alpha1.GetSchemaResponse
	(*ConvertMessageRequest)(nil),          // 3: buf.alpha.registry.v1alpha1.ConvertMessageRequest
	(*BinaryOutputOptions)(nil),            // 4: buf.alpha.registry.v1alpha1.BinaryOutputOptions
	(*JSONOutputOptions)(nil),              // 5: buf.alpha.registry.v1alpha1.JSONOutputOptions
	(*TextOutputOptions)(nil),              // 6: buf.alpha.registry.v1alpha1.TextOutputOptions
	(*ConvertMessageResponse)(nil),         // 7: buf.alpha.registry.v1alpha1.ConvertMessageResponse
	(*descriptorpb.FileDescriptorSet)(nil), // 8: google.protobuf.FileDescriptorSet
}
var file_buf_alpha_registry_v1alpha1_schema_proto_depIdxs = []int32{
	8, // 0: buf.alpha.registry.v1alpha1.GetSchemaResponse.schema_files:type_name -> google.protobuf.FileDescriptorSet
	0, // 1: buf.alpha.registry.v1alpha1.ConvertMessageRequest.input_format:type_name -> buf.alpha.registry.v1alpha1.Format
	4, // 2: buf.alpha.registry.v1alpha1.ConvertMessageRequest.output_binary:type_name -> buf.alpha.registry.v1alpha1.BinaryOutputOptions
	5, // 3: buf.alpha.registry.v1alpha1.ConvertMessageRequest.output_json:type_name -> buf.alpha.registry.v1alpha1.JSONOutputOptions
	6, // 4: buf.alpha.registry.v1alpha1.ConvertMessageRequest.output_text:type_name -> buf.alpha.registry.v1alpha1.TextOutputOptions
	1, // 5: buf.alpha.registry.v1alpha1.SchemaService.GetSchema:input_type -> buf.alpha.registry.v1alpha1.GetSchemaRequest
	3, // 6: buf.alpha.registry.v1alpha1.SchemaService.ConvertMessage:input_type -> buf.alpha.registry.v1alpha1.ConvertMessageRequest
	2, // 7: buf.alpha.registry.v1alpha1.SchemaService.GetSchema:output_type -> buf.alpha.registry.v1alpha1.GetSchemaResponse
	7, // 8: buf.alpha.registry.v1alpha1.SchemaService.ConvertMessage:output_type -> buf.alpha.registry.v1alpha1.ConvertMessageResponse
	7, // [7:9] is the sub-list for method output_type
	5, // [5:7] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_schema_proto_init() }
func file_buf_alpha_registry_v1alpha1_schema_proto_init() {
	if File_buf_alpha_registry_v1alpha1_schema_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*GetSchemaRequest); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*GetSchemaResponse); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*ConvertMessageRequest); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*BinaryOutputOptions); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*JSONOutputOptions); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*TextOutputOptions); i {
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
		file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*ConvertMessageResponse); i {
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
	file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2].OneofWrappers = []any{
		(*ConvertMessageRequest_OutputBinary)(nil),
		(*ConvertMessageRequest_OutputJson)(nil),
		(*ConvertMessageRequest_OutputText)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_schema_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_schema_proto_depIdxs,
		EnumInfos:         file_buf_alpha_registry_v1alpha1_schema_proto_enumTypes,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_schema_proto = out.File
	file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_schema_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_schema_proto_depIdxs = nil
}
