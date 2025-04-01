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
// source: buf/alpha/registry/v1alpha1/convert.proto

package registryv1alpha1

import (
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// The supported formats for the serialized message conversion.
type ConvertFormat int32

const (
	ConvertFormat_CONVERT_FORMAT_UNSPECIFIED ConvertFormat = 0
	ConvertFormat_CONVERT_FORMAT_BIN         ConvertFormat = 1
	ConvertFormat_CONVERT_FORMAT_JSON        ConvertFormat = 2
)

// Enum value maps for ConvertFormat.
var (
	ConvertFormat_name = map[int32]string{
		0: "CONVERT_FORMAT_UNSPECIFIED",
		1: "CONVERT_FORMAT_BIN",
		2: "CONVERT_FORMAT_JSON",
	}
	ConvertFormat_value = map[string]int32{
		"CONVERT_FORMAT_UNSPECIFIED": 0,
		"CONVERT_FORMAT_BIN":         1,
		"CONVERT_FORMAT_JSON":        2,
	}
)

func (x ConvertFormat) Enum() *ConvertFormat {
	p := new(ConvertFormat)
	*p = x
	return p
}

func (x ConvertFormat) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ConvertFormat) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_convert_proto_enumTypes[0].Descriptor()
}

func (ConvertFormat) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_convert_proto_enumTypes[0]
}

func (x ConvertFormat) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

type ConvertRequest struct {
	state                     protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_TypeName       string                 `protobuf:"bytes,1,opt,name=type_name,json=typeName,proto3"`
	xxx_hidden_Image          *v1.Image              `protobuf:"bytes,2,opt,name=image,proto3"`
	xxx_hidden_Payload        []byte                 `protobuf:"bytes,3,opt,name=payload,proto3"`
	xxx_hidden_RequestFormat  ConvertFormat          `protobuf:"varint,4,opt,name=request_format,json=requestFormat,proto3,enum=buf.alpha.registry.v1alpha1.ConvertFormat"`
	xxx_hidden_ResponseFormat ConvertFormat          `protobuf:"varint,5,opt,name=response_format,json=responseFormat,proto3,enum=buf.alpha.registry.v1alpha1.ConvertFormat"`
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *ConvertRequest) Reset() {
	*x = ConvertRequest{}
	mi := &file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConvertRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertRequest) ProtoMessage() {}

func (x *ConvertRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ConvertRequest) GetTypeName() string {
	if x != nil {
		return x.xxx_hidden_TypeName
	}
	return ""
}

func (x *ConvertRequest) GetImage() *v1.Image {
	if x != nil {
		return x.xxx_hidden_Image
	}
	return nil
}

func (x *ConvertRequest) GetPayload() []byte {
	if x != nil {
		return x.xxx_hidden_Payload
	}
	return nil
}

func (x *ConvertRequest) GetRequestFormat() ConvertFormat {
	if x != nil {
		return x.xxx_hidden_RequestFormat
	}
	return ConvertFormat_CONVERT_FORMAT_UNSPECIFIED
}

func (x *ConvertRequest) GetResponseFormat() ConvertFormat {
	if x != nil {
		return x.xxx_hidden_ResponseFormat
	}
	return ConvertFormat_CONVERT_FORMAT_UNSPECIFIED
}

func (x *ConvertRequest) SetTypeName(v string) {
	x.xxx_hidden_TypeName = v
}

func (x *ConvertRequest) SetImage(v *v1.Image) {
	x.xxx_hidden_Image = v
}

func (x *ConvertRequest) SetPayload(v []byte) {
	if v == nil {
		v = []byte{}
	}
	x.xxx_hidden_Payload = v
}

func (x *ConvertRequest) SetRequestFormat(v ConvertFormat) {
	x.xxx_hidden_RequestFormat = v
}

func (x *ConvertRequest) SetResponseFormat(v ConvertFormat) {
	x.xxx_hidden_ResponseFormat = v
}

func (x *ConvertRequest) HasImage() bool {
	if x == nil {
		return false
	}
	return x.xxx_hidden_Image != nil
}

func (x *ConvertRequest) ClearImage() {
	x.xxx_hidden_Image = nil
}

type ConvertRequest_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// type_name is the full type name of the serialized message (like acme.weather.v1.Units).
	TypeName string
	// image is the image source that defines the serialized message.
	Image *v1.Image
	// payload is the serialized Protobuf message.
	Payload []byte
	// request_format is the format of the payload.
	RequestFormat ConvertFormat
	// response_format is the desired format of the output result.
	ResponseFormat ConvertFormat
}

func (b0 ConvertRequest_builder) Build() *ConvertRequest {
	m0 := &ConvertRequest{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_TypeName = b.TypeName
	x.xxx_hidden_Image = b.Image
	x.xxx_hidden_Payload = b.Payload
	x.xxx_hidden_RequestFormat = b.RequestFormat
	x.xxx_hidden_ResponseFormat = b.ResponseFormat
	return m0
}

type ConvertResponse struct {
	state              protoimpl.MessageState `protogen:"opaque.v1"`
	xxx_hidden_Payload []byte                 `protobuf:"bytes,1,opt,name=payload,proto3"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *ConvertResponse) Reset() {
	*x = ConvertResponse{}
	mi := &file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConvertResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConvertResponse) ProtoMessage() {}

func (x *ConvertResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ConvertResponse) GetPayload() []byte {
	if x != nil {
		return x.xxx_hidden_Payload
	}
	return nil
}

func (x *ConvertResponse) SetPayload(v []byte) {
	if v == nil {
		v = []byte{}
	}
	x.xxx_hidden_Payload = v
}

type ConvertResponse_builder struct {
	_ [0]func() // Prevents comparability and use of unkeyed literals for the builder.

	// payload is the converted serialized message in one of the supported formats.
	Payload []byte
}

func (b0 ConvertResponse_builder) Build() *ConvertResponse {
	m0 := &ConvertResponse{}
	b, x := &b0, m0
	_, _ = b, x
	x.xxx_hidden_Payload = b.Payload
	return m0
}

var File_buf_alpha_registry_v1alpha1_convert_proto protoreflect.FileDescriptor

const file_buf_alpha_registry_v1alpha1_convert_proto_rawDesc = "" +
	"\n" +
	")buf/alpha/registry/v1alpha1/convert.proto\x12\x1bbuf.alpha.registry.v1alpha1\x1a\x1ebuf/alpha/image/v1/image.proto\"\xa0\x02\n" +
	"\x0eConvertRequest\x12\x1b\n" +
	"\ttype_name\x18\x01 \x01(\tR\btypeName\x12/\n" +
	"\x05image\x18\x02 \x01(\v2\x19.buf.alpha.image.v1.ImageR\x05image\x12\x18\n" +
	"\apayload\x18\x03 \x01(\fR\apayload\x12Q\n" +
	"\x0erequest_format\x18\x04 \x01(\x0e2*.buf.alpha.registry.v1alpha1.ConvertFormatR\rrequestFormat\x12S\n" +
	"\x0fresponse_format\x18\x05 \x01(\x0e2*.buf.alpha.registry.v1alpha1.ConvertFormatR\x0eresponseFormat\"+\n" +
	"\x0fConvertResponse\x12\x18\n" +
	"\apayload\x18\x01 \x01(\fR\apayload*`\n" +
	"\rConvertFormat\x12\x1e\n" +
	"\x1aCONVERT_FORMAT_UNSPECIFIED\x10\x00\x12\x16\n" +
	"\x12CONVERT_FORMAT_BIN\x10\x01\x12\x17\n" +
	"\x13CONVERT_FORMAT_JSON\x10\x022v\n" +
	"\x0eConvertService\x12d\n" +
	"\aConvert\x12+.buf.alpha.registry.v1alpha1.ConvertRequest\x1a,.buf.alpha.registry.v1alpha1.ConvertResponseB\x99\x02\n" +
	"\x1fcom.buf.alpha.registry.v1alpha1B\fConvertProtoP\x01ZYgithub.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1;registryv1alpha1\xa2\x02\x03BAR\xaa\x02\x1bBuf.Alpha.Registry.V1alpha1\xca\x02\x1bBuf\\Alpha\\Registry\\V1alpha1\xe2\x02'Buf\\Alpha\\Registry\\V1alpha1\\GPBMetadata\xea\x02\x1eBuf::Alpha::Registry::V1alpha1b\x06proto3"

var file_buf_alpha_registry_v1alpha1_convert_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_alpha_registry_v1alpha1_convert_proto_goTypes = []any{
	(ConvertFormat)(0),      // 0: buf.alpha.registry.v1alpha1.ConvertFormat
	(*ConvertRequest)(nil),  // 1: buf.alpha.registry.v1alpha1.ConvertRequest
	(*ConvertResponse)(nil), // 2: buf.alpha.registry.v1alpha1.ConvertResponse
	(*v1.Image)(nil),        // 3: buf.alpha.image.v1.Image
}
var file_buf_alpha_registry_v1alpha1_convert_proto_depIdxs = []int32{
	3, // 0: buf.alpha.registry.v1alpha1.ConvertRequest.image:type_name -> buf.alpha.image.v1.Image
	0, // 1: buf.alpha.registry.v1alpha1.ConvertRequest.request_format:type_name -> buf.alpha.registry.v1alpha1.ConvertFormat
	0, // 2: buf.alpha.registry.v1alpha1.ConvertRequest.response_format:type_name -> buf.alpha.registry.v1alpha1.ConvertFormat
	1, // 3: buf.alpha.registry.v1alpha1.ConvertService.Convert:input_type -> buf.alpha.registry.v1alpha1.ConvertRequest
	2, // 4: buf.alpha.registry.v1alpha1.ConvertService.Convert:output_type -> buf.alpha.registry.v1alpha1.ConvertResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_convert_proto_init() }
func file_buf_alpha_registry_v1alpha1_convert_proto_init() {
	if File_buf_alpha_registry_v1alpha1_convert_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_buf_alpha_registry_v1alpha1_convert_proto_rawDesc), len(file_buf_alpha_registry_v1alpha1_convert_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_convert_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_convert_proto_depIdxs,
		EnumInfos:         file_buf_alpha_registry_v1alpha1_convert_proto_enumTypes,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_convert_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_convert_proto = out.File
	file_buf_alpha_registry_v1alpha1_convert_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_convert_proto_depIdxs = nil
}
