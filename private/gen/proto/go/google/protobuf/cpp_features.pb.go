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
// 	protoc-gen-go v1.36.3
// 	protoc        (unknown)
// source: google/protobuf/cpp_features.proto

package protobuf

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CppFeatures_StringType int32

const (
	CppFeatures_STRING_TYPE_UNKNOWN CppFeatures_StringType = 0
	CppFeatures_VIEW                CppFeatures_StringType = 1
	CppFeatures_CORD                CppFeatures_StringType = 2
	CppFeatures_STRING              CppFeatures_StringType = 3
)

// Enum value maps for CppFeatures_StringType.
var (
	CppFeatures_StringType_name = map[int32]string{
		0: "STRING_TYPE_UNKNOWN",
		1: "VIEW",
		2: "CORD",
		3: "STRING",
	}
	CppFeatures_StringType_value = map[string]int32{
		"STRING_TYPE_UNKNOWN": 0,
		"VIEW":                1,
		"CORD":                2,
		"STRING":              3,
	}
)

func (x CppFeatures_StringType) Enum() *CppFeatures_StringType {
	p := new(CppFeatures_StringType)
	*p = x
	return p
}

func (x CppFeatures_StringType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CppFeatures_StringType) Descriptor() protoreflect.EnumDescriptor {
	return file_google_protobuf_cpp_features_proto_enumTypes[0].Descriptor()
}

func (CppFeatures_StringType) Type() protoreflect.EnumType {
	return &file_google_protobuf_cpp_features_proto_enumTypes[0]
}

func (x CppFeatures_StringType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

type CppFeatures struct {
	state                             protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_LegacyClosedEnum       bool                   `protobuf:"varint,1,opt,name=legacy_closed_enum,json=legacyClosedEnum"`
	xxx_hidden_StringType             CppFeatures_StringType `protobuf:"varint,2,opt,name=string_type,json=stringType,enum=pb.CppFeatures_StringType"`
	xxx_hidden_EnumNameUsesStringView bool                   `protobuf:"varint,3,opt,name=enum_name_uses_string_view,json=enumNameUsesStringView"`
	XXX_raceDetectHookData            protoimpl.RaceDetectHookData
	XXX_presence                      [1]uint32
	unknownFields                     protoimpl.UnknownFields
	sizeCache                         protoimpl.SizeCache
}

func (x *CppFeatures) Reset() {
	*x = CppFeatures{}
	mi := &file_google_protobuf_cpp_features_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CppFeatures) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CppFeatures) ProtoMessage() {}

func (x *CppFeatures) ProtoReflect() protoreflect.Message {
	mi := &file_google_protobuf_cpp_features_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *CppFeatures) GetLegacyClosedEnum() bool {
	if x != nil {
		return x.xxx_hidden_LegacyClosedEnum
	}
	return false
}

func (x *CppFeatures) GetStringType() CppFeatures_StringType {
	if x != nil {
		if protoimpl.X.Present(&(x.XXX_presence[0]), 1) {
			return x.xxx_hidden_StringType
		}
	}
	return CppFeatures_STRING_TYPE_UNKNOWN
}

func (x *CppFeatures) GetEnumNameUsesStringView() bool {
	if x != nil {
		return x.xxx_hidden_EnumNameUsesStringView
	}
	return false
}

func (x *CppFeatures) SetLegacyClosedEnum(v bool) {
	x.xxx_hidden_LegacyClosedEnum = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 3)
}

func (x *CppFeatures) SetStringType(v CppFeatures_StringType) {
	x.xxx_hidden_StringType = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 3)
}

func (x *CppFeatures) SetEnumNameUsesStringView(v bool) {
	x.xxx_hidden_EnumNameUsesStringView = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 2, 3)
}

func (x *CppFeatures) HasLegacyClosedEnum() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *CppFeatures) HasStringType() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *CppFeatures) HasEnumNameUsesStringView() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 2)
}

func (x *CppFeatures) ClearLegacyClosedEnum() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_LegacyClosedEnum = false
}

func (x *CppFeatures) ClearStringType() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_StringType = CppFeatures_STRING_TYPE_UNKNOWN
}

func (x *CppFeatures) ClearEnumNameUsesStringView() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 2)
	x.xxx_hidden_EnumNameUsesStringView = false
}

type CppFeatures_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Whether or not to treat an enum field as closed.  This option is only
	// applicable to enum fields, and will be removed in the future.  It is
	// consistent with the legacy behavior of using proto3 enum types for proto2
	// fields.
	LegacyClosedEnum       *bool
	StringType             *CppFeatures_StringType
	EnumNameUsesStringView *bool
}

func (b0 CppFeatures_builder) Build() *CppFeatures {
	m0 := &CppFeatures{}
	b, x := &b0, m0
	_, _ = b, x
	if b.LegacyClosedEnum != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 3)
		x.xxx_hidden_LegacyClosedEnum = *b.LegacyClosedEnum
	}
	if b.StringType != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 3)
		x.xxx_hidden_StringType = *b.StringType
	}
	if b.EnumNameUsesStringView != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 2, 3)
		x.xxx_hidden_EnumNameUsesStringView = *b.EnumNameUsesStringView
	}
	return m0
}

var file_google_protobuf_cpp_features_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.FeatureSet)(nil),
		ExtensionType: (*CppFeatures)(nil),
		Field:         1000,
		Name:          "pb.cpp",
		Tag:           "bytes,1000,opt,name=cpp",
		Filename:      "google/protobuf/cpp_features.proto",
	},
}

// Extension fields to descriptorpb.FeatureSet.
var (
	// optional pb.CppFeatures cpp = 1000;
	E_Cpp = &file_google_protobuf_cpp_features_proto_extTypes[0]
)

var File_google_protobuf_cpp_features_proto protoreflect.FileDescriptor

var file_google_protobuf_cpp_features_proto_rawDesc = []byte{
	0x0a, 0x22, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x63, 0x70, 0x70, 0x5f, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb2, 0x04, 0x0a, 0x0b, 0x43,
	0x70, 0x70, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x12, 0x8d, 0x02, 0x0a, 0x12, 0x6c,
	0x65, 0x67, 0x61, 0x63, 0x79, 0x5f, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x5f, 0x65, 0x6e, 0x75,
	0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x42, 0xde, 0x01, 0x88, 0x01, 0x01, 0x98, 0x01, 0x04,
	0x98, 0x01, 0x01, 0xa2, 0x01, 0x09, 0x12, 0x04, 0x74, 0x72, 0x75, 0x65, 0x18, 0x84, 0x07, 0xa2,
	0x01, 0x0a, 0x12, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x18, 0xe7, 0x07, 0xb2, 0x01, 0xb8, 0x01,
	0x08, 0xe8, 0x07, 0x10, 0xe8, 0x07, 0x1a, 0xaf, 0x01, 0x54, 0x68, 0x65, 0x20, 0x6c, 0x65, 0x67,
	0x61, 0x63, 0x79, 0x20, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x20, 0x65, 0x6e, 0x75, 0x6d, 0x20,
	0x62, 0x65, 0x68, 0x61, 0x76, 0x69, 0x6f, 0x72, 0x20, 0x69, 0x6e, 0x20, 0x43, 0x2b, 0x2b, 0x20,
	0x69, 0x73, 0x20, 0x64, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x20, 0x61, 0x6e,
	0x64, 0x20, 0x69, 0x73, 0x20, 0x73, 0x63, 0x68, 0x65, 0x64, 0x75, 0x6c, 0x65, 0x64, 0x20, 0x74,
	0x6f, 0x20, 0x62, 0x65, 0x20, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x20, 0x69, 0x6e, 0x20,
	0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x32, 0x30, 0x32, 0x35, 0x2e, 0x20, 0x20, 0x53,
	0x65, 0x65, 0x20, 0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x64, 0x65, 0x76, 0x2f, 0x70, 0x72, 0x6f, 0x67, 0x72, 0x61, 0x6d, 0x6d, 0x69,
	0x6e, 0x67, 0x2d, 0x67, 0x75, 0x69, 0x64, 0x65, 0x73, 0x2f, 0x65, 0x6e, 0x75, 0x6d, 0x2f, 0x23,
	0x63, 0x70, 0x70, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x6d, 0x6f, 0x72, 0x65, 0x20, 0x69, 0x6e, 0x66,
	0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x10, 0x6c, 0x65, 0x67, 0x61, 0x63, 0x79,
	0x43, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x66, 0x0a, 0x0b, 0x73, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x1a, 0x2e, 0x70, 0x62, 0x2e, 0x43, 0x70, 0x70, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73,
	0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x54, 0x79, 0x70, 0x65, 0x42, 0x29, 0x88, 0x01, 0x01,
	0x98, 0x01, 0x04, 0x98, 0x01, 0x01, 0xa2, 0x01, 0x0b, 0x12, 0x06, 0x53, 0x54, 0x52, 0x49, 0x4e,
	0x47, 0x18, 0x84, 0x07, 0xa2, 0x01, 0x09, 0x12, 0x04, 0x56, 0x49, 0x45, 0x57, 0x18, 0xe9, 0x07,
	0xb2, 0x01, 0x03, 0x08, 0xe8, 0x07, 0x52, 0x0a, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x64, 0x0a, 0x1a, 0x65, 0x6e, 0x75, 0x6d, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x5f,
	0x75, 0x73, 0x65, 0x73, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x5f, 0x76, 0x69, 0x65, 0x77,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x42, 0x28, 0x88, 0x01, 0x02, 0x98, 0x01, 0x06, 0x98, 0x01,
	0x01, 0xa2, 0x01, 0x0a, 0x12, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x18, 0x84, 0x07, 0xa2, 0x01,
	0x09, 0x12, 0x04, 0x74, 0x72, 0x75, 0x65, 0x18, 0xe9, 0x07, 0xb2, 0x01, 0x03, 0x08, 0xe9, 0x07,
	0x52, 0x16, 0x65, 0x6e, 0x75, 0x6d, 0x4e, 0x61, 0x6d, 0x65, 0x55, 0x73, 0x65, 0x73, 0x53, 0x74,
	0x72, 0x69, 0x6e, 0x67, 0x56, 0x69, 0x65, 0x77, 0x22, 0x45, 0x0a, 0x0a, 0x53, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x54, 0x79, 0x70, 0x65, 0x12, 0x17, 0x0a, 0x13, 0x53, 0x54, 0x52, 0x49, 0x4e, 0x47,
	0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12,
	0x08, 0x0a, 0x04, 0x56, 0x49, 0x45, 0x57, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x43, 0x4f, 0x52,
	0x44, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x53, 0x54, 0x52, 0x49, 0x4e, 0x47, 0x10, 0x03, 0x3a,
	0x3f, 0x0a, 0x03, 0x63, 0x70, 0x70, 0x12, 0x1b, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x53, 0x65, 0x74, 0x18, 0xe8, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x62, 0x2e,
	0x43, 0x70, 0x70, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x52, 0x03, 0x63, 0x70, 0x70,
}

var file_google_protobuf_cpp_features_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_google_protobuf_cpp_features_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_google_protobuf_cpp_features_proto_goTypes = []any{
	(CppFeatures_StringType)(0),     // 0: pb.CppFeatures.StringType
	(*CppFeatures)(nil),             // 1: pb.CppFeatures
	(*descriptorpb.FeatureSet)(nil), // 2: google.protobuf.FeatureSet
}
var file_google_protobuf_cpp_features_proto_depIdxs = []int32{
	0, // 0: pb.CppFeatures.string_type:type_name -> pb.CppFeatures.StringType
	2, // 1: pb.cpp:extendee -> google.protobuf.FeatureSet
	1, // 2: pb.cpp:type_name -> pb.CppFeatures
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	2, // [2:3] is the sub-list for extension type_name
	1, // [1:2] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_google_protobuf_cpp_features_proto_init() }
func file_google_protobuf_cpp_features_proto_init() {
	if File_google_protobuf_cpp_features_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_google_protobuf_cpp_features_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_google_protobuf_cpp_features_proto_goTypes,
		DependencyIndexes: file_google_protobuf_cpp_features_proto_depIdxs,
		EnumInfos:         file_google_protobuf_cpp_features_proto_enumTypes,
		MessageInfos:      file_google_protobuf_cpp_features_proto_msgTypes,
		ExtensionInfos:    file_google_protobuf_cpp_features_proto_extTypes,
	}.Build()
	File_google_protobuf_cpp_features_proto = out.File
	file_google_protobuf_cpp_features_proto_rawDesc = nil
	file_google_protobuf_cpp_features_proto_goTypes = nil
	file_google_protobuf_cpp_features_proto_depIdxs = nil
}
