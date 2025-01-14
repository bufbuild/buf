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
// 	protoc-gen-go v1.36.2
// 	protoc        (unknown)
// source: google/protobuf/java_features.proto

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

// The UTF8 validation strategy to use.  See go/editions-utf8-validation for
// more information on this feature.
type JavaFeatures_Utf8Validation int32

const (
	// Invalid default, which should never be used.
	JavaFeatures_UTF8_VALIDATION_UNKNOWN JavaFeatures_Utf8Validation = 0
	// Respect the UTF8 validation behavior specified by the global
	// utf8_validation feature.
	JavaFeatures_DEFAULT JavaFeatures_Utf8Validation = 1
	// Verifies UTF8 validity overriding the global utf8_validation
	// feature. This represents the legacy java_string_check_utf8 option.
	JavaFeatures_VERIFY JavaFeatures_Utf8Validation = 2
)

// Enum value maps for JavaFeatures_Utf8Validation.
var (
	JavaFeatures_Utf8Validation_name = map[int32]string{
		0: "UTF8_VALIDATION_UNKNOWN",
		1: "DEFAULT",
		2: "VERIFY",
	}
	JavaFeatures_Utf8Validation_value = map[string]int32{
		"UTF8_VALIDATION_UNKNOWN": 0,
		"DEFAULT":                 1,
		"VERIFY":                  2,
	}
)

func (x JavaFeatures_Utf8Validation) Enum() *JavaFeatures_Utf8Validation {
	p := new(JavaFeatures_Utf8Validation)
	*p = x
	return p
}

func (x JavaFeatures_Utf8Validation) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (JavaFeatures_Utf8Validation) Descriptor() protoreflect.EnumDescriptor {
	return file_google_protobuf_java_features_proto_enumTypes[0].Descriptor()
}

func (JavaFeatures_Utf8Validation) Type() protoreflect.EnumType {
	return &file_google_protobuf_java_features_proto_enumTypes[0]
}

func (x JavaFeatures_Utf8Validation) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

type JavaFeatures struct {
	state                       protoimpl.MessageState      `protogen:"opaque.v1"`
	xxx_hidden_LegacyClosedEnum bool                        `protobuf:"varint,1,opt,name=legacy_closed_enum,json=legacyClosedEnum" json:"legacy_closed_enum,omitempty"`
	xxx_hidden_Utf8Validation   JavaFeatures_Utf8Validation `protobuf:"varint,2,opt,name=utf8_validation,json=utf8Validation,enum=pb.JavaFeatures_Utf8Validation" json:"utf8_validation,omitempty"`
	XXX_raceDetectHookData      protoimpl.RaceDetectHookData
	XXX_presence                [1]uint32
	unknownFields               protoimpl.UnknownFields
	sizeCache                   protoimpl.SizeCache
}

func (x *JavaFeatures) Reset() {
	*x = JavaFeatures{}
	mi := &file_google_protobuf_java_features_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *JavaFeatures) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JavaFeatures) ProtoMessage() {}

func (x *JavaFeatures) ProtoReflect() protoreflect.Message {
	mi := &file_google_protobuf_java_features_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *JavaFeatures) GetLegacyClosedEnum() bool {
	if x != nil {
		return x.xxx_hidden_LegacyClosedEnum
	}
	return false
}

func (x *JavaFeatures) GetUtf8Validation() JavaFeatures_Utf8Validation {
	if x != nil {
		if protoimpl.X.Present(&(x.XXX_presence[0]), 1) {
			return x.xxx_hidden_Utf8Validation
		}
	}
	return JavaFeatures_UTF8_VALIDATION_UNKNOWN
}

func (x *JavaFeatures) SetLegacyClosedEnum(v bool) {
	x.xxx_hidden_LegacyClosedEnum = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 0, 2)
}

func (x *JavaFeatures) SetUtf8Validation(v JavaFeatures_Utf8Validation) {
	x.xxx_hidden_Utf8Validation = v
	protoimpl.X.SetPresent(&(x.XXX_presence[0]), 1, 2)
}

func (x *JavaFeatures) HasLegacyClosedEnum() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 0)
}

func (x *JavaFeatures) HasUtf8Validation() bool {
	if x == nil {
		return false
	}
	return protoimpl.X.Present(&(x.XXX_presence[0]), 1)
}

func (x *JavaFeatures) ClearLegacyClosedEnum() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 0)
	x.xxx_hidden_LegacyClosedEnum = false
}

func (x *JavaFeatures) ClearUtf8Validation() {
	protoimpl.X.ClearPresent(&(x.XXX_presence[0]), 1)
	x.xxx_hidden_Utf8Validation = JavaFeatures_UTF8_VALIDATION_UNKNOWN
}

type JavaFeatures_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// Whether or not to treat an enum field as closed.  This option is only
	// applicable to enum fields, and will be removed in the future.  It is
	// consistent with the legacy behavior of using proto3 enum types for proto2
	// fields.
	LegacyClosedEnum *bool
	Utf8Validation   *JavaFeatures_Utf8Validation
}

func (b0 JavaFeatures_builder) Build() *JavaFeatures {
	m0 := &JavaFeatures{}
	b, x := &b0, m0
	_, _ = b, x
	if b.LegacyClosedEnum != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 0, 2)
		x.xxx_hidden_LegacyClosedEnum = *b.LegacyClosedEnum
	}
	if b.Utf8Validation != nil {
		protoimpl.X.SetPresentNonAtomic(&(x.XXX_presence[0]), 1, 2)
		x.xxx_hidden_Utf8Validation = *b.Utf8Validation
	}
	return m0
}

var file_google_protobuf_java_features_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.FeatureSet)(nil),
		ExtensionType: (*JavaFeatures)(nil),
		Field:         1001,
		Name:          "pb.java",
		Tag:           "bytes,1001,opt,name=java",
		Filename:      "google/protobuf/java_features.proto",
	},
}

// Extension fields to descriptorpb.FeatureSet.
var (
	// optional pb.JavaFeatures java = 1001;
	E_Java = &file_google_protobuf_java_features_proto_extTypes[0]
)

var File_google_protobuf_java_features_proto protoreflect.FileDescriptor

var file_google_protobuf_java_features_proto_rawDesc = []byte{
	0x0a, 0x23, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x6a, 0x61, 0x76, 0x61, 0x5f, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72,
	0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x9b, 0x05, 0x0a, 0x0c,
	0x4a, 0x61, 0x76, 0x61, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x12, 0x90, 0x02, 0x0a,
	0x12, 0x6c, 0x65, 0x67, 0x61, 0x63, 0x79, 0x5f, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x5f, 0x65,
	0x6e, 0x75, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x42, 0xe1, 0x01, 0x88, 0x01, 0x01, 0x98,
	0x01, 0x04, 0x98, 0x01, 0x01, 0xa2, 0x01, 0x09, 0x12, 0x04, 0x74, 0x72, 0x75, 0x65, 0x18, 0x84,
	0x07, 0xa2, 0x01, 0x0a, 0x12, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x18, 0xe7, 0x07, 0xb2, 0x01,
	0xbb, 0x01, 0x08, 0xe8, 0x07, 0x10, 0xe8, 0x07, 0x1a, 0xb2, 0x01, 0x54, 0x68, 0x65, 0x20, 0x6c,
	0x65, 0x67, 0x61, 0x63, 0x79, 0x20, 0x63, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x20, 0x65, 0x6e, 0x75,
	0x6d, 0x20, 0x62, 0x65, 0x68, 0x61, 0x76, 0x69, 0x6f, 0x72, 0x20, 0x69, 0x6e, 0x20, 0x4a, 0x61,
	0x76, 0x61, 0x20, 0x69, 0x73, 0x20, 0x64, 0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64,
	0x20, 0x61, 0x6e, 0x64, 0x20, 0x69, 0x73, 0x20, 0x73, 0x63, 0x68, 0x65, 0x64, 0x75, 0x6c, 0x65,
	0x64, 0x20, 0x74, 0x6f, 0x20, 0x62, 0x65, 0x20, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x20,
	0x69, 0x6e, 0x20, 0x65, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x32, 0x30, 0x32, 0x35, 0x2e,
	0x20, 0x20, 0x53, 0x65, 0x65, 0x20, 0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x64, 0x65, 0x76, 0x2f, 0x70, 0x72, 0x6f, 0x67, 0x72, 0x61,
	0x6d, 0x6d, 0x69, 0x6e, 0x67, 0x2d, 0x67, 0x75, 0x69, 0x64, 0x65, 0x73, 0x2f, 0x65, 0x6e, 0x75,
	0x6d, 0x2f, 0x23, 0x6a, 0x61, 0x76, 0x61, 0x20, 0x66, 0x6f, 0x72, 0x20, 0x6d, 0x6f, 0x72, 0x65,
	0x20, 0x69, 0x6e, 0x66, 0x6f, 0x72, 0x6d, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x52, 0x10, 0x6c,
	0x65, 0x67, 0x61, 0x63, 0x79, 0x43, 0x6c, 0x6f, 0x73, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x12,
	0xaf, 0x02, 0x0a, 0x0f, 0x75, 0x74, 0x66, 0x38, 0x5f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x70, 0x62, 0x2e, 0x4a,
	0x61, 0x76, 0x61, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x2e, 0x55, 0x74, 0x66, 0x38,
	0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0xe4, 0x01, 0x88, 0x01, 0x01,
	0x98, 0x01, 0x04, 0x98, 0x01, 0x01, 0xa2, 0x01, 0x0c, 0x12, 0x07, 0x44, 0x45, 0x46, 0x41, 0x55,
	0x4c, 0x54, 0x18, 0x84, 0x07, 0xb2, 0x01, 0xc8, 0x01, 0x08, 0xe8, 0x07, 0x10, 0xe9, 0x07, 0x1a,
	0xbf, 0x01, 0x54, 0x68, 0x65, 0x20, 0x4a, 0x61, 0x76, 0x61, 0x2d, 0x73, 0x70, 0x65, 0x63, 0x69,
	0x66, 0x69, 0x63, 0x20, 0x75, 0x74, 0x66, 0x38, 0x20, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x20, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x20, 0x69, 0x73, 0x20, 0x64,
	0x65, 0x70, 0x72, 0x65, 0x63, 0x61, 0x74, 0x65, 0x64, 0x20, 0x61, 0x6e, 0x64, 0x20, 0x69, 0x73,
	0x20, 0x73, 0x63, 0x68, 0x65, 0x64, 0x75, 0x6c, 0x65, 0x64, 0x20, 0x74, 0x6f, 0x20, 0x62, 0x65,
	0x20, 0x72, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x64, 0x20, 0x69, 0x6e, 0x20, 0x65, 0x64, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x20, 0x32, 0x30, 0x32, 0x35, 0x2e, 0x20, 0x20, 0x55, 0x74, 0x66, 0x38, 0x20,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x62, 0x65, 0x68, 0x61, 0x76,
	0x69, 0x6f, 0x72, 0x20, 0x73, 0x68, 0x6f, 0x75, 0x6c, 0x64, 0x20, 0x75, 0x73, 0x65, 0x20, 0x74,
	0x68, 0x65, 0x20, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x20, 0x63, 0x72, 0x6f, 0x73, 0x73, 0x2d,
	0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x20, 0x75, 0x74, 0x66, 0x38, 0x5f, 0x76, 0x61,
	0x6c, 0x69, 0x64, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x20, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65,
	0x2e, 0x52, 0x0e, 0x75, 0x74, 0x66, 0x38, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x22, 0x46, 0x0a, 0x0e, 0x55, 0x74, 0x66, 0x38, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x1b, 0x0a, 0x17, 0x55, 0x54, 0x46, 0x38, 0x5f, 0x56, 0x41, 0x4c, 0x49,
	0x44, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00,
	0x12, 0x0b, 0x0a, 0x07, 0x44, 0x45, 0x46, 0x41, 0x55, 0x4c, 0x54, 0x10, 0x01, 0x12, 0x0a, 0x0a,
	0x06, 0x56, 0x45, 0x52, 0x49, 0x46, 0x59, 0x10, 0x02, 0x3a, 0x42, 0x0a, 0x04, 0x6a, 0x61, 0x76,
	0x61, 0x12, 0x1b, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53, 0x65, 0x74, 0x18, 0xe9,
	0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x62, 0x2e, 0x4a, 0x61, 0x76, 0x61, 0x46,
	0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x52, 0x04, 0x6a, 0x61, 0x76, 0x61, 0x42, 0x28, 0x0a,
	0x13, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x42, 0x11, 0x4a, 0x61, 0x76, 0x61, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x73, 0x50, 0x72, 0x6f, 0x74, 0x6f,
}

var file_google_protobuf_java_features_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_google_protobuf_java_features_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_google_protobuf_java_features_proto_goTypes = []any{
	(JavaFeatures_Utf8Validation)(0), // 0: pb.JavaFeatures.Utf8Validation
	(*JavaFeatures)(nil),             // 1: pb.JavaFeatures
	(*descriptorpb.FeatureSet)(nil),  // 2: google.protobuf.FeatureSet
}
var file_google_protobuf_java_features_proto_depIdxs = []int32{
	0, // 0: pb.JavaFeatures.utf8_validation:type_name -> pb.JavaFeatures.Utf8Validation
	2, // 1: pb.java:extendee -> google.protobuf.FeatureSet
	1, // 2: pb.java:type_name -> pb.JavaFeatures
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	2, // [2:3] is the sub-list for extension type_name
	1, // [1:2] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_google_protobuf_java_features_proto_init() }
func file_google_protobuf_java_features_proto_init() {
	if File_google_protobuf_java_features_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_google_protobuf_java_features_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 1,
			NumServices:   0,
		},
		GoTypes:           file_google_protobuf_java_features_proto_goTypes,
		DependencyIndexes: file_google_protobuf_java_features_proto_depIdxs,
		EnumInfos:         file_google_protobuf_java_features_proto_enumTypes,
		MessageInfos:      file_google_protobuf_java_features_proto_msgTypes,
		ExtensionInfos:    file_google_protobuf_java_features_proto_extTypes,
	}.Build()
	File_google_protobuf_java_features_proto = out.File
	file_google_protobuf_java_features_proto_rawDesc = nil
	file_google_protobuf_java_features_proto_goTypes = nil
	file_google_protobuf_java_features_proto_depIdxs = nil
}
