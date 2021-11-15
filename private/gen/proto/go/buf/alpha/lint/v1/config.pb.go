// Copyright 2020-2021 Buf Technologies, Inc.
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
// 	protoc-gen-go v1.27.1
// 	protoc        (unknown)
// source: buf/alpha/lint/v1/config.proto

package lintv1

import (
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

// Config represents the lint configuration for a module. The rule IDs and categories are defined
// by the rules_version and apply across the config. The rules_version is independent of the version of
// the package. The package version refers to the config shape, the rules_version indicates which
// rule IDs and catetgories should be used.
type Config struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// rules_version represents the version of the lint rule IDs and categories that should be used with this config.
	RulesVersion string `protobuf:"bytes,1,opt,name=rules_version,json=rulesVersion,proto3" json:"rules_version,omitempty"`
	// use lists the rule IDs and/or categories that are included in the lint check.
	Use []string `protobuf:"bytes,2,rep,name=use,proto3" json:"use,omitempty"`
	// except lists the rule IDs and/or categories that are excluded from the lint check.
	Except []string `protobuf:"bytes,3,rep,name=except,proto3" json:"except,omitempty"`
	// ignore lists the paths of directories and/or files that should be ignored by the lint check.
	// All paths are relative to the root of the module.
	Ignore []string `protobuf:"bytes,4,rep,name=ignore,proto3" json:"ignore,omitempty"`
	// ignore_only is a map of rule IDs and/or categories to directory and/or file paths to exclude from the
	// lint check.
	IgnoreOnly []*IgnoreOnly `protobuf:"bytes,5,rep,name=ignore_only,json=ignoreOnly,proto3" json:"ignore_only,omitempty"`
	// enum_zero_value_suffix controls the behavior of the ENUM_ZERO_VALUE lint rule ID. By default, this rule
	// verifies that the zero value of all enums ends in _UNSPECIFIED. This config allows the user to override
	// this value with the given string.
	EnumZeroValueSuffix string `protobuf:"bytes,6,opt,name=enum_zero_value_suffix,json=enumZeroValueSuffix,proto3" json:"enum_zero_value_suffix,omitempty"`
	// rpc_allow_same_request_response allows the same message type for both the request and response of an RPC.
	RpcAllowGoogleProtobufEmptyRequests bool `protobuf:"varint,7,opt,name=rpc_allow_google_protobuf_empty_requests,json=rpcAllowGoogleProtobufEmptyRequests,proto3" json:"rpc_allow_google_protobuf_empty_requests,omitempty"`
	// rpc_allow_google_protobuf_empty_responses allows the RPC responses to use the google.protobuf.Empty message.
	RpcAllowGoogleProtobufEmptyResponses bool `protobuf:"varint,8,opt,name=rpc_allow_google_protobuf_empty_responses,json=rpcAllowGoogleProtobufEmptyResponses,proto3" json:"rpc_allow_google_protobuf_empty_responses,omitempty"`
	// service_suffix applies to the SERVICE_SUFFIX rule ID. By default, the rule verifies that all service names
	// end with the suffix Service. This allows users to override the value with the given string.
	ServiceSuffix string `protobuf:"bytes,9,opt,name=service_suffix,json=serviceSuffix,proto3" json:"service_suffix,omitempty"`
	// allow_comment_ignores turns on comment-driven ignores.
	AllowCommentIgnores bool `protobuf:"varint,10,opt,name=allow_comment_ignores,json=allowCommentIgnores,proto3" json:"allow_comment_ignores,omitempty"`
}

func (x *Config) Reset() {
	*x = Config{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_lint_v1_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Config) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Config) ProtoMessage() {}

func (x *Config) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_lint_v1_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Config.ProtoReflect.Descriptor instead.
func (*Config) Descriptor() ([]byte, []int) {
	return file_buf_alpha_lint_v1_config_proto_rawDescGZIP(), []int{0}
}

func (x *Config) GetRulesVersion() string {
	if x != nil {
		return x.RulesVersion
	}
	return ""
}

func (x *Config) GetUse() []string {
	if x != nil {
		return x.Use
	}
	return nil
}

func (x *Config) GetExcept() []string {
	if x != nil {
		return x.Except
	}
	return nil
}

func (x *Config) GetIgnore() []string {
	if x != nil {
		return x.Ignore
	}
	return nil
}

func (x *Config) GetIgnoreOnly() []*IgnoreOnly {
	if x != nil {
		return x.IgnoreOnly
	}
	return nil
}

func (x *Config) GetEnumZeroValueSuffix() string {
	if x != nil {
		return x.EnumZeroValueSuffix
	}
	return ""
}

func (x *Config) GetRpcAllowGoogleProtobufEmptyRequests() bool {
	if x != nil {
		return x.RpcAllowGoogleProtobufEmptyRequests
	}
	return false
}

func (x *Config) GetRpcAllowGoogleProtobufEmptyResponses() bool {
	if x != nil {
		return x.RpcAllowGoogleProtobufEmptyResponses
	}
	return false
}

func (x *Config) GetServiceSuffix() string {
	if x != nil {
		return x.ServiceSuffix
	}
	return ""
}

func (x *Config) GetAllowCommentIgnores() bool {
	if x != nil {
		return x.AllowCommentIgnores
	}
	return false
}

// IgnoreOnly represents a rule ID or category and the file and/or directory paths that are ignored for the rule.
type IgnoreOnly struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id    string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Paths []string `protobuf:"bytes,2,rep,name=paths,proto3" json:"paths,omitempty"`
}

func (x *IgnoreOnly) Reset() {
	*x = IgnoreOnly{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_lint_v1_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IgnoreOnly) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IgnoreOnly) ProtoMessage() {}

func (x *IgnoreOnly) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_lint_v1_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IgnoreOnly.ProtoReflect.Descriptor instead.
func (*IgnoreOnly) Descriptor() ([]byte, []int) {
	return file_buf_alpha_lint_v1_config_proto_rawDescGZIP(), []int{1}
}

func (x *IgnoreOnly) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *IgnoreOnly) GetPaths() []string {
	if x != nil {
		return x.Paths
	}
	return nil
}

var File_buf_alpha_lint_v1_config_proto protoreflect.FileDescriptor

var file_buf_alpha_lint_v1_config_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x6c, 0x69, 0x6e, 0x74,
	0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x11, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6c, 0x69, 0x6e, 0x74,
	0x2e, 0x76, 0x31, 0x22, 0xef, 0x03, 0x0a, 0x06, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x23,
	0x0a, 0x0d, 0x72, 0x75, 0x6c, 0x65, 0x73, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x75, 0x6c, 0x65, 0x73, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x73, 0x65, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x03, 0x75, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x65, 0x78, 0x63, 0x65, 0x70, 0x74, 0x18,
	0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x65, 0x78, 0x63, 0x65, 0x70, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x69, 0x67, 0x6e, 0x6f, 0x72, 0x65, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x69,
	0x67, 0x6e, 0x6f, 0x72, 0x65, 0x12, 0x3e, 0x0a, 0x0b, 0x69, 0x67, 0x6e, 0x6f, 0x72, 0x65, 0x5f,
	0x6f, 0x6e, 0x6c, 0x79, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6c, 0x69, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x49,
	0x67, 0x6e, 0x6f, 0x72, 0x65, 0x4f, 0x6e, 0x6c, 0x79, 0x52, 0x0a, 0x69, 0x67, 0x6e, 0x6f, 0x72,
	0x65, 0x4f, 0x6e, 0x6c, 0x79, 0x12, 0x33, 0x0a, 0x16, 0x65, 0x6e, 0x75, 0x6d, 0x5f, 0x7a, 0x65,
	0x72, 0x6f, 0x5f, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x5f, 0x73, 0x75, 0x66, 0x66, 0x69, 0x78, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13, 0x65, 0x6e, 0x75, 0x6d, 0x5a, 0x65, 0x72, 0x6f, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x53, 0x75, 0x66, 0x66, 0x69, 0x78, 0x12, 0x55, 0x0a, 0x28, 0x72, 0x70,
	0x63, 0x5f, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x5f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x5f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x5f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x5f, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x73, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x23, 0x72, 0x70,
	0x63, 0x41, 0x6c, 0x6c, 0x6f, 0x77, 0x47, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x50, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x73, 0x12, 0x57, 0x0a, 0x29, 0x72, 0x70, 0x63, 0x5f, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x5f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x5f, 0x65,
	0x6d, 0x70, 0x74, 0x79, 0x5f, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x73, 0x18, 0x08,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x24, 0x72, 0x70, 0x63, 0x41, 0x6c, 0x6c, 0x6f, 0x77, 0x47, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x73, 0x12, 0x25, 0x0a, 0x0e, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x73, 0x75, 0x66, 0x66, 0x69, 0x78, 0x18, 0x09, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0d, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x53, 0x75, 0x66, 0x66, 0x69,
	0x78, 0x12, 0x32, 0x0a, 0x15, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x65,
	0x6e, 0x74, 0x5f, 0x69, 0x67, 0x6e, 0x6f, 0x72, 0x65, 0x73, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x08,
	0x52, 0x13, 0x61, 0x6c, 0x6c, 0x6f, 0x77, 0x43, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x67,
	0x6e, 0x6f, 0x72, 0x65, 0x73, 0x22, 0x32, 0x0a, 0x0a, 0x49, 0x67, 0x6e, 0x6f, 0x72, 0x65, 0x4f,
	0x6e, 0x6c, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x61, 0x74, 0x68, 0x73, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x05, 0x70, 0x61, 0x74, 0x68, 0x73, 0x42, 0xd2, 0x01, 0x0a, 0x15, 0x63, 0x6f,
	0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6c, 0x69, 0x6e, 0x74,
	0x2e, 0x76, 0x31, 0x42, 0x0b, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x50, 0x72, 0x6f, 0x74, 0x6f,
	0x50, 0x01, 0x5a, 0x45, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62,
	0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76,
	0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f,
	0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x6c, 0x69, 0x6e, 0x74, 0x2f,
	0x76, 0x31, 0x3b, 0x6c, 0x69, 0x6e, 0x74, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x4c, 0xaa,
	0x02, 0x11, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x4c, 0x69, 0x6e, 0x74,
	0x2e, 0x56, 0x31, 0xca, 0x02, 0x11, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c,
	0x4c, 0x69, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x1d, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c,
	0x70, 0x68, 0x61, 0x5c, 0x4c, 0x69, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x14, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41,
	0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x4c, 0x69, 0x6e, 0x74, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_lint_v1_config_proto_rawDescOnce sync.Once
	file_buf_alpha_lint_v1_config_proto_rawDescData = file_buf_alpha_lint_v1_config_proto_rawDesc
)

func file_buf_alpha_lint_v1_config_proto_rawDescGZIP() []byte {
	file_buf_alpha_lint_v1_config_proto_rawDescOnce.Do(func() {
		file_buf_alpha_lint_v1_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_lint_v1_config_proto_rawDescData)
	})
	return file_buf_alpha_lint_v1_config_proto_rawDescData
}

var file_buf_alpha_lint_v1_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_buf_alpha_lint_v1_config_proto_goTypes = []interface{}{
	(*Config)(nil),     // 0: buf.alpha.lint.v1.Config
	(*IgnoreOnly)(nil), // 1: buf.alpha.lint.v1.IgnoreOnly
}
var file_buf_alpha_lint_v1_config_proto_depIdxs = []int32{
	1, // 0: buf.alpha.lint.v1.Config.ignore_only:type_name -> buf.alpha.lint.v1.IgnoreOnly
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_buf_alpha_lint_v1_config_proto_init() }
func file_buf_alpha_lint_v1_config_proto_init() {
	if File_buf_alpha_lint_v1_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_lint_v1_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Config); i {
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
		file_buf_alpha_lint_v1_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IgnoreOnly); i {
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
			RawDescriptor: file_buf_alpha_lint_v1_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_lint_v1_config_proto_goTypes,
		DependencyIndexes: file_buf_alpha_lint_v1_config_proto_depIdxs,
		MessageInfos:      file_buf_alpha_lint_v1_config_proto_msgTypes,
	}.Build()
	File_buf_alpha_lint_v1_config_proto = out.File
	file_buf_alpha_lint_v1_config_proto_rawDesc = nil
	file_buf_alpha_lint_v1_config_proto_goTypes = nil
	file_buf_alpha_lint_v1_config_proto_depIdxs = nil
}
