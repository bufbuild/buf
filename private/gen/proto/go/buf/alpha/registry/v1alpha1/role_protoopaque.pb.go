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
// source: buf/alpha/registry/v1alpha1/role.proto

//go:build protoopaque

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// The roles that users can have in a Server.
type ServerRole int32

const (
	ServerRole_SERVER_ROLE_UNSPECIFIED ServerRole = 0
	ServerRole_SERVER_ROLE_ADMIN       ServerRole = 1
	ServerRole_SERVER_ROLE_MEMBER      ServerRole = 2
)

// Enum value maps for ServerRole.
var (
	ServerRole_name = map[int32]string{
		0: "SERVER_ROLE_UNSPECIFIED",
		1: "SERVER_ROLE_ADMIN",
		2: "SERVER_ROLE_MEMBER",
	}
	ServerRole_value = map[string]int32{
		"SERVER_ROLE_UNSPECIFIED": 0,
		"SERVER_ROLE_ADMIN":       1,
		"SERVER_ROLE_MEMBER":      2,
	}
)

func (x ServerRole) Enum() *ServerRole {
	p := new(ServerRole)
	*p = x
	return p
}

func (x ServerRole) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ServerRole) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[0].Descriptor()
}

func (ServerRole) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[0]
}

func (x ServerRole) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// The roles that users can have in a Organization.
type OrganizationRole int32

const (
	OrganizationRole_ORGANIZATION_ROLE_UNSPECIFIED OrganizationRole = 0
	OrganizationRole_ORGANIZATION_ROLE_OWNER       OrganizationRole = 1
	OrganizationRole_ORGANIZATION_ROLE_ADMIN       OrganizationRole = 2
	OrganizationRole_ORGANIZATION_ROLE_MEMBER      OrganizationRole = 3
	OrganizationRole_ORGANIZATION_ROLE_MACHINE     OrganizationRole = 4
)

// Enum value maps for OrganizationRole.
var (
	OrganizationRole_name = map[int32]string{
		0: "ORGANIZATION_ROLE_UNSPECIFIED",
		1: "ORGANIZATION_ROLE_OWNER",
		2: "ORGANIZATION_ROLE_ADMIN",
		3: "ORGANIZATION_ROLE_MEMBER",
		4: "ORGANIZATION_ROLE_MACHINE",
	}
	OrganizationRole_value = map[string]int32{
		"ORGANIZATION_ROLE_UNSPECIFIED": 0,
		"ORGANIZATION_ROLE_OWNER":       1,
		"ORGANIZATION_ROLE_ADMIN":       2,
		"ORGANIZATION_ROLE_MEMBER":      3,
		"ORGANIZATION_ROLE_MACHINE":     4,
	}
)

func (x OrganizationRole) Enum() *OrganizationRole {
	p := new(OrganizationRole)
	*p = x
	return p
}

func (x OrganizationRole) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OrganizationRole) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[1].Descriptor()
}

func (OrganizationRole) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[1]
}

func (x OrganizationRole) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// The source of a user's role in an Organization.
type OrganizationRoleSource int32

const (
	OrganizationRoleSource_ORGANIZATION_ROLE_SOURCE_UNSPECIFIED OrganizationRoleSource = 0
	OrganizationRoleSource_ORGANIZATION_ROLE_SOURCE_DIRECT      OrganizationRoleSource = 1
	OrganizationRoleSource_ORGANIZATION_ROLE_SOURCE_JIT         OrganizationRoleSource = 2
	OrganizationRoleSource_ORGANIZATION_ROLE_SOURCE_IDP_GROUP   OrganizationRoleSource = 3
)

// Enum value maps for OrganizationRoleSource.
var (
	OrganizationRoleSource_name = map[int32]string{
		0: "ORGANIZATION_ROLE_SOURCE_UNSPECIFIED",
		1: "ORGANIZATION_ROLE_SOURCE_DIRECT",
		2: "ORGANIZATION_ROLE_SOURCE_JIT",
		3: "ORGANIZATION_ROLE_SOURCE_IDP_GROUP",
	}
	OrganizationRoleSource_value = map[string]int32{
		"ORGANIZATION_ROLE_SOURCE_UNSPECIFIED": 0,
		"ORGANIZATION_ROLE_SOURCE_DIRECT":      1,
		"ORGANIZATION_ROLE_SOURCE_JIT":         2,
		"ORGANIZATION_ROLE_SOURCE_IDP_GROUP":   3,
	}
)

func (x OrganizationRoleSource) Enum() *OrganizationRoleSource {
	p := new(OrganizationRoleSource)
	*p = x
	return p
}

func (x OrganizationRoleSource) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OrganizationRoleSource) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[2].Descriptor()
}

func (OrganizationRoleSource) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[2]
}

func (x OrganizationRoleSource) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// The roles that users can have for a Repository.
type RepositoryRole int32

const (
	RepositoryRole_REPOSITORY_ROLE_UNSPECIFIED   RepositoryRole = 0
	RepositoryRole_REPOSITORY_ROLE_OWNER         RepositoryRole = 1
	RepositoryRole_REPOSITORY_ROLE_ADMIN         RepositoryRole = 2
	RepositoryRole_REPOSITORY_ROLE_WRITE         RepositoryRole = 3
	RepositoryRole_REPOSITORY_ROLE_READ          RepositoryRole = 4
	RepositoryRole_REPOSITORY_ROLE_LIMITED_WRITE RepositoryRole = 5
)

// Enum value maps for RepositoryRole.
var (
	RepositoryRole_name = map[int32]string{
		0: "REPOSITORY_ROLE_UNSPECIFIED",
		1: "REPOSITORY_ROLE_OWNER",
		2: "REPOSITORY_ROLE_ADMIN",
		3: "REPOSITORY_ROLE_WRITE",
		4: "REPOSITORY_ROLE_READ",
		5: "REPOSITORY_ROLE_LIMITED_WRITE",
	}
	RepositoryRole_value = map[string]int32{
		"REPOSITORY_ROLE_UNSPECIFIED":   0,
		"REPOSITORY_ROLE_OWNER":         1,
		"REPOSITORY_ROLE_ADMIN":         2,
		"REPOSITORY_ROLE_WRITE":         3,
		"REPOSITORY_ROLE_READ":          4,
		"REPOSITORY_ROLE_LIMITED_WRITE": 5,
	}
)

func (x RepositoryRole) Enum() *RepositoryRole {
	p := new(RepositoryRole)
	*p = x
	return p
}

func (x RepositoryRole) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RepositoryRole) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[3].Descriptor()
}

func (RepositoryRole) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_role_proto_enumTypes[3]
}

func (x RepositoryRole) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

var File_buf_alpha_registry_v1alpha1_role_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_role_proto_rawDesc = []byte{
	0x0a, 0x26, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x72, 0x6f,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2a, 0x58, 0x0a, 0x0a, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x52,
	0x6f, 0x6c, 0x65, 0x12, 0x1b, 0x0a, 0x17, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x52, 0x4f,
	0x4c, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00,
	0x12, 0x15, 0x0a, 0x11, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f,
	0x41, 0x44, 0x4d, 0x49, 0x4e, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x45, 0x52, 0x56, 0x45,
	0x52, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4d, 0x45, 0x4d, 0x42, 0x45, 0x52, 0x10, 0x02, 0x2a,
	0xac, 0x01, 0x0a, 0x10, 0x4f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x52, 0x6f, 0x6c, 0x65, 0x12, 0x21, 0x0a, 0x1d, 0x4f, 0x52, 0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x1b, 0x0a, 0x17, 0x4f, 0x52, 0x47, 0x41, 0x4e,
	0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4f, 0x57, 0x4e,
	0x45, 0x52, 0x10, 0x01, 0x12, 0x1b, 0x0a, 0x17, 0x4f, 0x52, 0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x41, 0x44, 0x4d, 0x49, 0x4e, 0x10,
	0x02, 0x12, 0x1c, 0x0a, 0x18, 0x4f, 0x52, 0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f,
	0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4d, 0x45, 0x4d, 0x42, 0x45, 0x52, 0x10, 0x03, 0x12,
	0x1d, 0x0a, 0x19, 0x4f, 0x52, 0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4d, 0x41, 0x43, 0x48, 0x49, 0x4e, 0x45, 0x10, 0x04, 0x2a, 0xb1,
	0x01, 0x0a, 0x16, 0x4f, 0x72, 0x67, 0x61, 0x6e, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x6f, 0x6c, 0x65, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x28, 0x0a, 0x24, 0x4f, 0x52, 0x47,
	0x41, 0x4e, 0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x53,
	0x4f, 0x55, 0x52, 0x43, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45,
	0x44, 0x10, 0x00, 0x12, 0x23, 0x0a, 0x1f, 0x4f, 0x52, 0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41, 0x54,
	0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x53, 0x4f, 0x55, 0x52, 0x43, 0x45, 0x5f,
	0x44, 0x49, 0x52, 0x45, 0x43, 0x54, 0x10, 0x01, 0x12, 0x20, 0x0a, 0x1c, 0x4f, 0x52, 0x47, 0x41,
	0x4e, 0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x53, 0x4f,
	0x55, 0x52, 0x43, 0x45, 0x5f, 0x4a, 0x49, 0x54, 0x10, 0x02, 0x12, 0x26, 0x0a, 0x22, 0x4f, 0x52,
	0x47, 0x41, 0x4e, 0x49, 0x5a, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f,
	0x53, 0x4f, 0x55, 0x52, 0x43, 0x45, 0x5f, 0x49, 0x44, 0x50, 0x5f, 0x47, 0x52, 0x4f, 0x55, 0x50,
	0x10, 0x03, 0x2a, 0xbf, 0x01, 0x0a, 0x0e, 0x52, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x52, 0x6f, 0x6c, 0x65, 0x12, 0x1f, 0x0a, 0x1b, 0x52, 0x45, 0x50, 0x4f, 0x53, 0x49, 0x54,
	0x4f, 0x52, 0x59, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49,
	0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x19, 0x0a, 0x15, 0x52, 0x45, 0x50, 0x4f, 0x53, 0x49,
	0x54, 0x4f, 0x52, 0x59, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4f, 0x57, 0x4e, 0x45, 0x52, 0x10,
	0x01, 0x12, 0x19, 0x0a, 0x15, 0x52, 0x45, 0x50, 0x4f, 0x53, 0x49, 0x54, 0x4f, 0x52, 0x59, 0x5f,
	0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x41, 0x44, 0x4d, 0x49, 0x4e, 0x10, 0x02, 0x12, 0x19, 0x0a, 0x15,
	0x52, 0x45, 0x50, 0x4f, 0x53, 0x49, 0x54, 0x4f, 0x52, 0x59, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f,
	0x57, 0x52, 0x49, 0x54, 0x45, 0x10, 0x03, 0x12, 0x18, 0x0a, 0x14, 0x52, 0x45, 0x50, 0x4f, 0x53,
	0x49, 0x54, 0x4f, 0x52, 0x59, 0x5f, 0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x52, 0x45, 0x41, 0x44, 0x10,
	0x04, 0x12, 0x21, 0x0a, 0x1d, 0x52, 0x45, 0x50, 0x4f, 0x53, 0x49, 0x54, 0x4f, 0x52, 0x59, 0x5f,
	0x52, 0x4f, 0x4c, 0x45, 0x5f, 0x4c, 0x49, 0x4d, 0x49, 0x54, 0x45, 0x44, 0x5f, 0x57, 0x52, 0x49,
	0x54, 0x45, 0x10, 0x05, 0x42, 0x96, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x09, 0x52, 0x6f, 0x6c, 0x65, 0x50, 0x72,
	0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70,
	0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61,
	0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x1e, 0x42,
	0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_buf_alpha_registry_v1alpha1_role_proto_enumTypes = make([]protoimpl.EnumInfo, 4)
var file_buf_alpha_registry_v1alpha1_role_proto_goTypes = []any{
	(ServerRole)(0),             // 0: buf.alpha.registry.v1alpha1.ServerRole
	(OrganizationRole)(0),       // 1: buf.alpha.registry.v1alpha1.OrganizationRole
	(OrganizationRoleSource)(0), // 2: buf.alpha.registry.v1alpha1.OrganizationRoleSource
	(RepositoryRole)(0),         // 3: buf.alpha.registry.v1alpha1.RepositoryRole
}
var file_buf_alpha_registry_v1alpha1_role_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_role_proto_init() }
func file_buf_alpha_registry_v1alpha1_role_proto_init() {
	if File_buf_alpha_registry_v1alpha1_role_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_role_proto_rawDesc,
			NumEnums:      4,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_role_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_role_proto_depIdxs,
		EnumInfos:         file_buf_alpha_registry_v1alpha1_role_proto_enumTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_role_proto = out.File
	file_buf_alpha_registry_v1alpha1_role_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_role_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_role_proto_depIdxs = nil
}
