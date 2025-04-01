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
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/schema.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	unsafe "unsafe"
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

type GetSchemaRequest struct {
	state                             protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Owner                  string                 `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository             string                 `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Version                string                 `protobuf:"bytes,3,opt,name=version,proto3"`
	xxx_hidden_Types                  []string               `protobuf:"bytes,4,rep,name=types,proto3"`
	xxx_hidden_IfNotCommit            string                 `protobuf:"bytes,5,opt,name=if_not_commit,json=ifNotCommit,proto3"`
	xxx_hidden_ExcludeCustomOptions   bool                   `protobuf:"varint,6,opt,name=exclude_custom_options,json=excludeCustomOptions,proto3"`
	xxx_hidden_ExcludeKnownExtensions bool                   `protobuf:"varint,7,opt,name=exclude_known_extensions,json=excludeKnownExtensions,proto3"`
	unknownFields                     protoimpl.UnknownFields
	sizeCache                         protoimpl.SizeCache
}

func (x *GetSchemaRequest) Reset() {
	*x = GetSchemaRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetSchemaRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSchemaRequest) ProtoMessage() {}

func (x *GetSchemaRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetSchemaRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *GetSchemaRequest) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

func (x *GetSchemaRequest) GetVersion() string {
	if x != nil {
		return x.xxx_hidden_Version
	}
	return ""
}

func (x *GetSchemaRequest) GetTypes() []string {
	if x != nil {
		return x.xxx_hidden_Types
	}
	return nil
}

func (x *GetSchemaRequest) GetIfNotCommit() string {
	if x != nil {
		return x.xxx_hidden_IfNotCommit
	}
	return ""
}

func (x *GetSchemaRequest) GetExcludeCustomOptions() bool {
	if x != nil {
		return x.xxx_hidden_ExcludeCustomOptions
	}
	return false
}

func (x *GetSchemaRequest) GetExcludeKnownExtensions() bool {
	if x != nil {
		return x.xxx_hidden_ExcludeKnownExtensions
	}
	return false
}

func (x *GetSchemaRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *GetSchemaRequest) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

func (x *GetSchemaRequest) SetVersion(v string) {
	x.xxx_hidden_Version = v
}

func (x *GetSchemaRequest) SetTypes(v []string) {
	x.xxx_hidden_Types = v
}

func (x *GetSchemaRequest) SetIfNotCommit(v string) {
	x.xxx_hidden_IfNotCommit = v
}

func (x *GetSchemaRequest) SetExcludeCustomOptions(v bool) {
	x.xxx_hidden_ExcludeCustomOptions = v
}

func (x *GetSchemaRequest) SetExcludeKnownExtensions(v bool) {
	x.xxx_hidden_ExcludeKnownExtensions = v
}

type GetSchemaRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The owner of the repo that contains the schema to retrieve (a user name or
	// organization name).
	Owner string
	// The name of the repo that contains the schema to retrieve.
	Repository string
	// Optional version of the repo. If unspecified, defaults to latest version on
	// the repo's "main" branch.
	Version string
	// Zero or more types names. The names may refer to messages, enums, services,
	// methods, or extensions. All names must be fully-qualified. If any name
	// is unknown, the request will fail and no schema will be returned.
	//
	// If no names are provided, the full schema for the module is returned.
	// Otherwise, the resulting schema contains only the named elements and all of
	// their dependencies. This is enough information for the caller to construct
	// a dynamic message for any requested message types or to dynamically invoke
	// an RPC for any requested methods or services.
	Types []string
	// If present, this is a commit that the client already has cached. So if the
	// given module version resolves to this same commit, the server should not
	// send back any descriptors since the client already has them.
	//
	// This allows a client to efficiently poll for updates: after the initial RPC
	// to get a schema, the client can cache the descriptors and the resolved
	// commit. It then includes that commit in subsequent requests in this field,
	// and the server will only reply with a schema (and new commit) if/when the
	// resolved commit changes.
	IfNotCommit string
	// If true, the returned schema will not include extension definitions for custom
	// options that appear on schema elements. When filtering the schema based on the
	// given element names, options on all encountered elements are usually examined
	// as well. But that is not the case if excluding custom options.
	//
	// This flag is ignored if element_names is empty as the entire schema is always
	// returned in that case.
	ExcludeCustomOptions bool
	// If true, the returned schema will not include known extensions for extendable
	// messages for schema elements. If exclude_custom_options is true, such extensions
	// may still be returned if the applicable descriptor options type is part of the
	// requested schema.
	//
	// This flag is ignored if element_names is empty as the entire schema is always
	// returned in that case.
	ExcludeKnownExtensions bool
}

func (b0 GetSchemaRequest_builder) Build() *GetSchemaRequest {
	m0 := &GetSchemaRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Version = b.Version
	x.xxx_hidden_Types = b.Types
	x.xxx_hidden_IfNotCommit = b.IfNotCommit
	x.xxx_hidden_ExcludeCustomOptions = b.ExcludeCustomOptions
	x.xxx_hidden_ExcludeKnownExtensions = b.ExcludeKnownExtensions
	return m0
}

type GetSchemaResponse struct {
	state                  protoimpl.MessageState          `protogen:"opaque.v1"`
	xxx_hidden_Commit      string                          `protobuf:"bytes,1,opt,name=commit,proto3"`
	xxx_hidden_SchemaFiles *descriptorpb.FileDescriptorSet `protobuf:"bytes,2,opt,name=schema_files,json=schemaFiles,proto3"`
	unknownFields          protoimpl.UnknownFields
	sizeCache              protoimpl.SizeCache
}

func (x *GetSchemaResponse) Reset() {
	*x = GetSchemaResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetSchemaResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSchemaResponse) ProtoMessage() {}

func (x *GetSchemaResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *GetSchemaResponse) GetCommit() string {
	if x != nil {
		return x.xxx_hidden_Commit
	}
	return ""
}

func (x *GetSchemaResponse) GetSchemaFiles() *descriptorpb.FileDescriptorSet {
	if x != nil {
		return x.xxx_hidden_SchemaFiles
	}
	return nil
}

func (x *GetSchemaResponse) SetCommit(v string) {
	x.xxx_hidden_Commit = v
}

func (x *GetSchemaResponse) SetSchemaFiles(v *descriptorpb.FileDescriptorSet) {
	x.xxx_hidden_SchemaFiles = v
}

func (x *GetSchemaResponse) HasSchemaFiles() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_SchemaFiles != nil
}

func (x *GetSchemaResponse) ClearSchemaFiles() {
	x.xxx_hidden_SchemaFiles = nil
}

type GetSchemaResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The resolved version of the schema. If the requested version was a commit,
	// this value is the same as that. If the requested version referred to a tag
	// or branch, this is the commit for that tag or latest commit for that
	// branch. If the request did not include any version, this is the latest
	// version for the module's main branch.
	Commit string
	// The schema, which is a set of file descriptors that include the requested elements
	// and their dependencies.
	SchemaFiles *descriptorpb.FileDescriptorSet
}

func (b0 GetSchemaResponse_builder) Build() *GetSchemaResponse {
	m0 := &GetSchemaResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Commit = b.Commit
	x.xxx_hidden_SchemaFiles = b.SchemaFiles
	return m0
}

type ConvertMessageRequest struct {
	state                     protoimpl.MessageState               `protogen:"opaque.v1"`
	xxx_hidden_Owner          string                               `protobuf:"bytes,1,opt,name=owner,proto3"`
	xxx_hidden_Repository     string                               `protobuf:"bytes,2,opt,name=repository,proto3"`
	xxx_hidden_Version        string                               `protobuf:"bytes,3,opt,name=version,proto3"`
	xxx_hidden_MessageName    string                               `protobuf:"bytes,4,opt,name=message_name,json=messageName,proto3"`
	xxx_hidden_InputFormat    Format                               `protobuf:"varint,5,opt,name=input_format,json=inputFormat,proto3,enum=buf.alpha.registry.v1alpha1.Format"`
	xxx_hidden_InputData      []byte                               `protobuf:"bytes,6,opt,name=input_data,json=inputData,proto3"`
	xxx_hidden_DiscardUnknown bool                                 `protobuf:"varint,7,opt,name=discard_unknown,json=discardUnknown,proto3"`
	xxx_hidden_OutputFormat   isConvertMessageRequest_OutputFormat `protobuf_oneof:"output_format"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *ConvertMessageRequest) Reset() {
	*x = ConvertMessageRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConvertMessageRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertMessageRequest) ProtoMessage() {}

func (x *ConvertMessageRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ConvertMessageRequest) GetOwner() string {
	if x != nil {
		return x.xxx_hidden_Owner
	}
	return ""
}

func (x *ConvertMessageRequest) GetRepository() string {
	if x != nil {
		return x.xxx_hidden_Repository
	}
	return ""
}

func (x *ConvertMessageRequest) GetVersion() string {
	if x != nil {
		return x.xxx_hidden_Version
	}
	return ""
}

func (x *ConvertMessageRequest) GetMessageName() string {
	if x != nil {
		return x.xxx_hidden_MessageName
	}
	return ""
}

func (x *ConvertMessageRequest) GetInputFormat() Format {
	if x != nil {
		return x.xxx_hidden_InputFormat
	}
	return Format_FORMAT_UNSPECIFIED
}

func (x *ConvertMessageRequest) GetInputData() []byte {
	if x != nil {
		return x.xxx_hidden_InputData
	}
	return nil
}

func (x *ConvertMessageRequest) GetDiscardUnknown() bool {
	if x != nil {
		return x.xxx_hidden_DiscardUnknown
	}
	return false
}

func (x *ConvertMessageRequest) GetOutputBinary() *BinaryOutputOptions {
	if x != nil {
		if x, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputBinary); ok {
			return x.OutputBinary
		}
	}
	return nil
}

func (x *ConvertMessageRequest) GetOutputJson() *JSONOutputOptions {
	if x != nil {
		if x, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputJson); ok {
			return x.OutputJson
		}
	}
	return nil
}

func (x *ConvertMessageRequest) GetOutputText() *TextOutputOptions {
	if x != nil {
		if x, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputText); ok {
			return x.OutputText
		}
	}
	return nil
}

func (x *ConvertMessageRequest) SetOwner(v string) {
	x.xxx_hidden_Owner = v
}

func (x *ConvertMessageRequest) SetRepository(v string) {
	x.xxx_hidden_Repository = v
}

func (x *ConvertMessageRequest) SetVersion(v string) {
	x.xxx_hidden_Version = v
}

func (x *ConvertMessageRequest) SetMessageName(v string) {
	x.xxx_hidden_MessageName = v
}

func (x *ConvertMessageRequest) SetInputFormat(v Format) {
	x.xxx_hidden_InputFormat = v
}

func (x *ConvertMessageRequest) SetInputData(v []byte) {
	if v == nil {
		v = []byte{}
	}
	x.xxx_hidden_InputData = v
}

func (x *ConvertMessageRequest) SetDiscardUnknown(v bool) {
	x.xxx_hidden_DiscardUnknown = v
}

func (x *ConvertMessageRequest) SetOutputBinary(v *BinaryOutputOptions) {
	if v == nil {
		x.xxx_hidden_OutputFormat = nil
		return
	}
	x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputBinary{v}
}

func (x *ConvertMessageRequest) SetOutputJson(v *JSONOutputOptions) {
	if v == nil {
		x.xxx_hidden_OutputFormat = nil
		return
	}
	x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputJson{v}
}

func (x *ConvertMessageRequest) SetOutputText(v *TextOutputOptions) {
	if v == nil {
		x.xxx_hidden_OutputFormat = nil
		return
	}
	x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputText{v}
}

func (x *ConvertMessageRequest) HasOutputFormat() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_OutputFormat != nil
}

func (x *ConvertMessageRequest) HasOutputBinary() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputBinary)
	return ok
}

func (x *ConvertMessageRequest) HasOutputJson() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputJson)
	return ok
}

func (x *ConvertMessageRequest) HasOutputText() bool {
	if x == nil {
		return false
	}
	_, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputText)
	return ok
}

func (x *ConvertMessageRequest) ClearOutputFormat() {
	x.xxx_hidden_OutputFormat = nil
}

func (x *ConvertMessageRequest) ClearOutputBinary() {
	if _, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputBinary); ok {
		x.xxx_hidden_OutputFormat = nil
	}
}

func (x *ConvertMessageRequest) ClearOutputJson() {
	if _, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputJson); ok {
		x.xxx_hidden_OutputFormat = nil
	}
}

func (x *ConvertMessageRequest) ClearOutputText() {
	if _, ok := x.xxx_hidden_OutputFormat.(*convertMessageRequest_OutputText); ok {
		x.xxx_hidden_OutputFormat = nil
	}
}

const ConvertMessageRequest_OutputFormat_not_set_case case_ConvertMessageRequest_OutputFormat = 0
const ConvertMessageRequest_OutputBinary_case case_ConvertMessageRequest_OutputFormat = 8
const ConvertMessageRequest_OutputJson_case case_ConvertMessageRequest_OutputFormat = 9
const ConvertMessageRequest_OutputText_case case_ConvertMessageRequest_OutputFormat = 10

func (x *ConvertMessageRequest) WhichOutputFormat() case_ConvertMessageRequest_OutputFormat {
	if x == nil {
		return ConvertMessageRequest_OutputFormat_not_set_case
	}
	switch x.xxx_hidden_OutputFormat.(type) {
	case *convertMessageRequest_OutputBinary:
		return ConvertMessageRequest_OutputBinary_case
	case *convertMessageRequest_OutputJson:
		return ConvertMessageRequest_OutputJson_case
	case *convertMessageRequest_OutputText:
		return ConvertMessageRequest_OutputText_case
	default:
		return ConvertMessageRequest_OutputFormat_not_set_case
	}
}

type ConvertMessageRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The owner of the repo that contains the schema to retrieve (a user name or
	// organization name).
	Owner string
	// The name of the repo that contains the schema to retrieve.
	Repository string
	// Optional version of the repo. This can be a tag or branch name or a commit.
	// If unspecified, defaults to latest version on the repo's "main" branch.
	Version string
	// The fully-qualified name of the message. Required.
	MessageName string
	// The format of the input data. Required.
	InputFormat Format
	// The input data that is to be converted. Required. This must be
	// a valid encoding of type indicated by message_name in the format
	// indicated by input_format.
	InputData []byte
	// If true, any unresolvable fields in the input are discarded. For
	// formats other than FORMAT_BINARY, this means that the operation
	// will fail if the input contains unrecognized field names. For
	// FORMAT_BINARY, unrecognized fields can be retained and possibly
	// included in the reformatted output (depending on the requested
	// output format).
	DiscardUnknown bool
	// Fields of oneof xxx_hidden_OutputFormat:
	OutputBinary *BinaryOutputOptions
	OutputJson   *JSONOutputOptions
	OutputText   *TextOutputOptions
	// -- end of xxx_hidden_OutputFormat
}

func (b0 ConvertMessageRequest_builder) Build() *ConvertMessageRequest {
	m0 := &ConvertMessageRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Owner = b.Owner
	x.xxx_hidden_Repository = b.Repository
	x.xxx_hidden_Version = b.Version
	x.xxx_hidden_MessageName = b.MessageName
	x.xxx_hidden_InputFormat = b.InputFormat
	x.xxx_hidden_InputData = b.InputData
	x.xxx_hidden_DiscardUnknown = b.DiscardUnknown
	if b.OutputBinary != nil {
		x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputBinary{b.OutputBinary}
	}
	if b.OutputJson != nil {
		x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputJson{b.OutputJson}
	}
	if b.OutputText != nil {
		x.xxx_hidden_OutputFormat = &convertMessageRequest_OutputText{b.OutputText}
	}
	return m0
}

type case_ConvertMessageRequest_OutputFormat protoreflect.FieldNumber

func (x case_ConvertMessageRequest_OutputFormat) String() string {
	md := file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2].Descriptor()
	if x == 0 {
		return "not set"
	}
	return protoimpl.X.MessageFieldStringOf(md, protoreflect.FieldNumber(x))
}

type isConvertMessageRequest_OutputFormat interface {
	isConvertMessageRequest_OutputFormat()
}

type convertMessageRequest_OutputBinary struct {
	OutputBinary *BinaryOutputOptions `protobuf:"bytes,8,opt,name=output_binary,json=outputBinary,proto3,oneof"`
}

type convertMessageRequest_OutputJson struct {
	OutputJson *JSONOutputOptions `protobuf:"bytes,9,opt,name=output_json,json=outputJson,proto3,oneof"`
}

type convertMessageRequest_OutputText struct {
	OutputText *TextOutputOptions `protobuf:"bytes,10,opt,name=output_text,json=outputText,proto3,oneof"`
}

func (*convertMessageRequest_OutputBinary) isConvertMessageRequest_OutputFormat() {}

func (*convertMessageRequest_OutputJson) isConvertMessageRequest_OutputFormat() {}

func (*convertMessageRequest_OutputText) isConvertMessageRequest_OutputFormat() {}

type BinaryOutputOptions struct {
	state         protoimpl.MessageState `protogen:"opaque.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BinaryOutputOptions) Reset() {
	*x = BinaryOutputOptions{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BinaryOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BinaryOutputOptions) ProtoMessage() {}

func (x *BinaryOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

type BinaryOutputOptions_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

}

func (b0 BinaryOutputOptions_builder) Build() *BinaryOutputOptions {
	m0 := &BinaryOutputOptions{}
	b, x := &b0, m0
	_, _ = b, x
	return m0
}

type JSONOutputOptions struct {
	state                      protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_UseEnumNumbers  bool                   `protobuf:"varint,3,opt,name=use_enum_numbers,json=useEnumNumbers,proto3"`
	xxx_hidden_IncludeDefaults bool                   `protobuf:"varint,4,opt,name=include_defaults,json=includeDefaults,proto3"`
	unknownFields              protoimpl.UnknownFields
	sizeCache                  protoimpl.SizeCache
}

func (x *JSONOutputOptions) Reset() {
	*x = JSONOutputOptions{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *JSONOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JSONOutputOptions) ProtoMessage() {}

func (x *JSONOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *JSONOutputOptions) GetUseEnumNumbers() bool {
	if x != nil {
		return x.xxx_hidden_UseEnumNumbers
	}
	return false
}

func (x *JSONOutputOptions) GetIncludeDefaults() bool {
	if x != nil {
		return x.xxx_hidden_IncludeDefaults
	}
	return false
}

func (x *JSONOutputOptions) SetUseEnumNumbers(v bool) {
	x.xxx_hidden_UseEnumNumbers = v
}

func (x *JSONOutputOptions) SetIncludeDefaults(v bool) {
	x.xxx_hidden_IncludeDefaults = v
}

type JSONOutputOptions_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Enum fields will be emitted as numeric values. If false (the default), enum
	// fields are emitted as strings that are the enum values' names.
	UseEnumNumbers bool
	// Includes fields that have their default values. This applies only to fields
	// defined in proto3 syntax that have no explicit "optional" keyword. Other
	// optional fields will be included if present in the input data.
	IncludeDefaults bool
}

func (b0 JSONOutputOptions_builder) Build() *JSONOutputOptions {
	m0 := &JSONOutputOptions{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_UseEnumNumbers = b.UseEnumNumbers
	x.xxx_hidden_IncludeDefaults = b.IncludeDefaults
	return m0
}

type TextOutputOptions struct {
	state                          protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_IncludeUnrecognized bool                   `protobuf:"varint,2,opt,name=include_unrecognized,json=includeUnrecognized,proto3"`
	unknownFields                  protoimpl.UnknownFields
	sizeCache                      protoimpl.SizeCache
}

func (x *TextOutputOptions) Reset() {
	*x = TextOutputOptions{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TextOutputOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TextOutputOptions) ProtoMessage() {}

func (x *TextOutputOptions) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *TextOutputOptions) GetIncludeUnrecognized() bool {
	if x != nil {
		return x.xxx_hidden_IncludeUnrecognized
	}
	return false
}

func (x *TextOutputOptions) SetIncludeUnrecognized(v bool) {
	x.xxx_hidden_IncludeUnrecognized = v
}

type TextOutputOptions_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// If true and the input data includes unrecognized fields, the unrecognized
	// fields will be preserved in the text output (using field numbers and raw
	// values).
	IncludeUnrecognized bool
}

func (b0 TextOutputOptions_builder) Build() *TextOutputOptions {
	m0 := &TextOutputOptions{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_IncludeUnrecognized = b.IncludeUnrecognized
	return m0
}

type ConvertMessageResponse struct {
	state                 protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Commit     string                 `protobuf:"bytes,1,opt,name=commit,proto3"`
	xxx_hidden_OutputData []byte                 `protobuf:"bytes,2,opt,name=output_data,json=outputData,proto3"`
	unknownFields         protoimpl.UnknownFields
	sizeCache             protoimpl.SizeCache
}

func (x *ConvertMessageResponse) Reset() {
	*x = ConvertMessageResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConvertMessageResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertMessageResponse) ProtoMessage() {}

func (x *ConvertMessageResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ConvertMessageResponse) GetCommit() string {
	if x != nil {
		return x.xxx_hidden_Commit
	}
	return ""
}

func (x *ConvertMessageResponse) GetOutputData() []byte {
	if x != nil {
		return x.xxx_hidden_OutputData
	}
	return nil
}

func (x *ConvertMessageResponse) SetCommit(v string) {
	x.xxx_hidden_Commit = v
}

func (x *ConvertMessageResponse) SetOutputData(v []byte) {
	if v == nil {
		v = []byte{}
	}
	x.xxx_hidden_OutputData = v
}

type ConvertMessageResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// The resolved version of the schema. If the requested version was a commit,
	// this value is the same as that. If the requested version referred to a tag
	// or branch, this is the commit for that tag or latest commit for that
	// branch. If the request did not include any version, this is the latest
	// version for the module's main branch.
	Commit string
	// The reformatted data.
	OutputData []byte
}

func (b0 ConvertMessageResponse_builder) Build() *ConvertMessageResponse {
	m0 := &ConvertMessageResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Commit = b.Commit
	x.xxx_hidden_OutputData = b.OutputData
	return m0
}

var File_buf_alpha_registry_v1alpha1_schema_proto protoreflect.FileDescriptor

const file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc = "" +
	"\n" +
	"(buf/alpha/registry/v1alpha1/schema.proto\x12\x1bbuf.alpha.registry.v1alpha1\x1a google/protobuf/descriptor.proto\"\x8c\x02\n" +
	"\x10GetSchemaRequest\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12\x18\n" +
	"\aversion\x18\x03 \x01(\tR\aversion\x12\x14\n" +
	"\x05types\x18\x04 \x03(\tR\x05types\x12\"\n" +
	"\rif_not_commit\x18\x05 \x01(\tR\vifNotCommit\x124\n" +
	"\x16exclude_custom_options\x18\x06 \x01(\bR\x14excludeCustomOptions\x128\n" +
	"\x18exclude_known_extensions\x18\a \x01(\bR\x16excludeKnownExtensions\"r\n" +
	"\x11GetSchemaResponse\x12\x16\n" +
	"\x06commit\x18\x01 \x01(\tR\x06commit\x12E\n" +
	"\fschema_files\x18\x02 \x01(\v2\".google.protobuf.FileDescriptorSetR\vschemaFiles\"\xaa\x04\n" +
	"\x15ConvertMessageRequest\x12\x14\n" +
	"\x05owner\x18\x01 \x01(\tR\x05owner\x12\x1e\n" +
	"\n" +
	"repository\x18\x02 \x01(\tR\n" +
	"repository\x12\x18\n" +
	"\aversion\x18\x03 \x01(\tR\aversion\x12!\n" +
	"\fmessage_name\x18\x04 \x01(\tR\vmessageName\x12F\n" +
	"\finput_format\x18\x05 \x01(\x0e2#.buf.alpha.registry.v1alpha1.FormatR\vinputFormat\x12\x1d\n" +
	"\n" +
	"input_data\x18\x06 \x01(\fR\tinputData\x12'\n" +
	"\x0fdiscard_unknown\x18\a \x01(\bR\x0ediscardUnknown\x12W\n" +
	"\routput_binary\x18\b \x01(\v20.buf.alpha.registry.v1alpha1.BinaryOutputOptionsH\x00R\foutputBinary\x12Q\n" +
	"\voutput_json\x18\t \x01(\v2..buf.alpha.registry.v1alpha1.JSONOutputOptionsH\x00R\n" +
	"outputJson\x12Q\n" +
	"\voutput_text\x18\n" +
	" \x01(\v2..buf.alpha.registry.v1alpha1.TextOutputOptionsH\x00R\n" +
	"outputTextB\x0f\n" +
	"\routput_format\"\x15\n" +
	"\x13BinaryOutputOptions\"h\n" +
	"\x11JSONOutputOptions\x12(\n" +
	"\x10use_enum_numbers\x18\x03 \x01(\bR\x0euseEnumNumbers\x12)\n" +
	"\x10include_defaults\x18\x04 \x01(\bR\x0fincludeDefaults\"F\n" +
	"\x11TextOutputOptions\x121\n" +
	"\x14include_unrecognized\x18\x02 \x01(\bR\x13includeUnrecognized\"Q\n" +
	"\x16ConvertMessageResponse\x12\x16\n" +
	"\x06commit\x18\x01 \x01(\tR\x06commit\x12\x1f\n" +
	"\voutput_data\x18\x02 \x01(\fR\n" +
	"outputData*U\n" +
	"\x06Format\x12\x16\n" +
	"\x12FORMAT_UNSPECIFIED\x10\x00\x12\x11\n" +
	"\rFORMAT_BINARY\x10\x01\x12\x0f\n" +
	"\vFORMAT_JSON\x10\x02\x12\x0f\n" +
	"\vFORMAT_TEXT\x10\x032\xfb\x01\n" +
	"\rSchemaService\x12o\n" +
	"\tGetSchema\x12-.buf.alpha.registry.v1alpha1.GetSchemaRequest\x1a..buf.alpha.registry.v1alpha1.GetSchemaResponse\"\x03\x90\x02\x01\x12y\n" +
	"\x0eConvertMessage\x122.buf.alpha.registry.v1alpha1.ConvertMessageRequest\x1a3.buf.alpha.registry.v1alpha1.ConvertMessageResponseB\x98\x02\n" +
	"\x1fcom.buf.alpha.registry.v1alpha1B\vSchemaProtoP\x01ZYgithub.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1;registryv1alpha1\xa2\x02\x03BAR\xaa\x02\x1bBuf.Alpha.Registry.V1alpha1\xca\x02\x1bBuf\\Alpha\\Registry\\V1alpha1\xe2\x02'Buf\\Alpha\\Registry\\V1alpha1\\GPBMetadata\xea\x02\x1eBuf::Alpha::Registry::V1alpha1b\x06proto3"

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
	file_buf_alpha_registry_v1alpha1_schema_proto_msgTypes[2].OneofWrappers = []any{
		(*convertMessageRequest_OutputBinary)(nil),
		(*convertMessageRequest_OutputJson)(nil),
		(*convertMessageRequest_OutputText)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc), len(file_buf_alpha_registry_v1alpha1_schema_proto_rawDesc)),
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
	file_buf_alpha_registry_v1alpha1_schema_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_schema_proto_depIdxs = nil
}
