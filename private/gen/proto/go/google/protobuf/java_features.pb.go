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
// source: google/protobuf/java_features.proto

package protobuf

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
	xxx_hidden_LegacyClosedEnum bool                        `protobuf:"varint,1,opt,name=legacy_closed_enum,json=legacyClosedEnum"`
	xxx_hidden_Utf8Validation   JavaFeatures_Utf8Validation `protobuf:"varint,2,opt,name=utf8_validation,json=utf8Validation,enum=pb.JavaFeatures_Utf8Validation"`
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

const file_google_protobuf_java_features_proto_rawDesc = "" +
	"\n" +
	"#google/protobuf/java_features.proto\x12\x02pb\x1a google/protobuf/descriptor.proto\"\x9b\x05\n" +
	"\fJavaFeatures\x12\x90\x02\n" +
	"\x12legacy_closed_enum\x18\x01 \x01(\bB\xe1\x01\x88\x01\x01\x98\x01\x04\x98\x01\x01\xa2\x01\t\x12\x04true\x18\x84\a\xa2\x01\n" +
	"\x12\x05false\x18\xe7\a\xb2\x01\xbb\x01\b\xe8\a\x10\xe8\a\x1a\xb2\x01The legacy closed enum behavior in Java is deprecated and is scheduled to be removed in edition 2025.  See http://protobuf.dev/programming-guides/enum/#java for more information.R\x10legacyClosedEnum\x12\xaf\x02\n" +
	"\x0futf8_validation\x18\x02 \x01(\x0e2\x1f.pb.JavaFeatures.Utf8ValidationB\xe4\x01\x88\x01\x01\x98\x01\x04\x98\x01\x01\xa2\x01\f\x12\aDEFAULT\x18\x84\a\xb2\x01\xc8\x01\b\xe8\a\x10\xe9\a\x1a\xbf\x01The Java-specific utf8 validation feature is deprecated and is scheduled to be removed in edition 2025.  Utf8 validation behavior should use the global cross-language utf8_validation feature.R\x0eutf8Validation\"F\n" +
	"\x0eUtf8Validation\x12\x1b\n" +
	"\x17UTF8_VALIDATION_UNKNOWN\x10\x00\x12\v\n" +
	"\aDEFAULT\x10\x01\x12\n" +
	"\n" +
	"\x06VERIFY\x10\x02:B\n" +
	"\x04java\x12\x1b.google.protobuf.FeatureSet\x18\xe9\a \x01(\v2\x10.pb.JavaFeaturesR\x04javaB(\n" +
	"\x13com.google.protobufB\x11JavaFeaturesProto"

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
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_google_protobuf_java_features_proto_rawDesc), len(file_google_protobuf_java_features_proto_rawDesc)),
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
	file_google_protobuf_java_features_proto_goTypes = nil
	file_google_protobuf_java_features_proto_depIdxs = nil
}
