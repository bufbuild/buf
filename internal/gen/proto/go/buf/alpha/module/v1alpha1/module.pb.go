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
// 	protoc-gen-go v1.26.0-devel
// 	protoc        v3.15.2
// source: buf/alpha/module/v1alpha1/module.proto

package modulev1alpha1

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

// Module is a module.
type Module struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// files are the files that make up the set.
	//
	// Sorted by path.
	// Path must be unique.
	// Only the target files. No imports.
	//
	// Maximum total size of all content: 32MB.
	Files []*ModuleFile `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
	// dependencies are the dependencies.
	Dependencies []*ModulePin `protobuf:"bytes,2,rep,name=dependencies,proto3" json:"dependencies,omitempty"`
	// documentation is the string representation of the contents of the `buf.md` file.
	//
	// string is used to enforce UTF-8 encoding or 7-bit ASCII text.
	Documentation string `protobuf:"bytes,3,opt,name=documentation,proto3" json:"documentation,omitempty"`
}

func (x *Module) Reset() {
	*x = Module{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Module) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Module) ProtoMessage() {}

func (x *Module) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Module.ProtoReflect.Descriptor instead.
func (*Module) Descriptor() ([]byte, []int) {
	return file_buf_alpha_module_v1alpha1_module_proto_rawDescGZIP(), []int{0}
}

func (x *Module) GetFiles() []*ModuleFile {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *Module) GetDependencies() []*ModulePin {
	if x != nil {
		return x.Dependencies
	}
	return nil
}

func (x *Module) GetDocumentation() string {
	if x != nil {
		return x.Documentation
	}
	return ""
}

// ModuleFile is a file within a FileSet.
type ModuleFile struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// path is the relative path of the file.
	// Path can only use '/' as the separator character, and includes no ".." components.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// content is the content of the file.
	Content []byte `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *ModuleFile) Reset() {
	*x = ModuleFile{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleFile) ProtoMessage() {}

func (x *ModuleFile) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleFile.ProtoReflect.Descriptor instead.
func (*ModuleFile) Descriptor() ([]byte, []int) {
	return file_buf_alpha_module_v1alpha1_module_proto_rawDescGZIP(), []int{1}
}

func (x *ModuleFile) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *ModuleFile) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

// ModuleReference is a module reference.
type ModuleReference struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Remote     string `protobuf:"bytes,1,opt,name=remote,proto3" json:"remote,omitempty"`
	Owner      string `protobuf:"bytes,2,opt,name=owner,proto3" json:"owner,omitempty"`
	Repository string `protobuf:"bytes,3,opt,name=repository,proto3" json:"repository,omitempty"`
	// either branch, tag, or commit
	Reference string `protobuf:"bytes,4,opt,name=reference,proto3" json:"reference,omitempty"`
}

func (x *ModuleReference) Reset() {
	*x = ModuleReference{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModuleReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModuleReference) ProtoMessage() {}

func (x *ModuleReference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModuleReference.ProtoReflect.Descriptor instead.
func (*ModuleReference) Descriptor() ([]byte, []int) {
	return file_buf_alpha_module_v1alpha1_module_proto_rawDescGZIP(), []int{2}
}

func (x *ModuleReference) GetRemote() string {
	if x != nil {
		return x.Remote
	}
	return ""
}

func (x *ModuleReference) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *ModuleReference) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *ModuleReference) GetReference() string {
	if x != nil {
		return x.Reference
	}
	return ""
}

// ModulePin is a module pin.
type ModulePin struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Remote     string                 `protobuf:"bytes,1,opt,name=remote,proto3" json:"remote,omitempty"`
	Owner      string                 `protobuf:"bytes,2,opt,name=owner,proto3" json:"owner,omitempty"`
	Repository string                 `protobuf:"bytes,3,opt,name=repository,proto3" json:"repository,omitempty"`
	Branch     string                 `protobuf:"bytes,4,opt,name=branch,proto3" json:"branch,omitempty"`
	Commit     string                 `protobuf:"bytes,5,opt,name=commit,proto3" json:"commit,omitempty"`
	Digest     string                 `protobuf:"bytes,6,opt,name=digest,proto3" json:"digest,omitempty"`
	CreateTime *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
}

func (x *ModulePin) Reset() {
	*x = ModulePin{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ModulePin) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ModulePin) ProtoMessage() {}

func (x *ModulePin) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_module_v1alpha1_module_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ModulePin.ProtoReflect.Descriptor instead.
func (*ModulePin) Descriptor() ([]byte, []int) {
	return file_buf_alpha_module_v1alpha1_module_proto_rawDescGZIP(), []int{3}
}

func (x *ModulePin) GetRemote() string {
	if x != nil {
		return x.Remote
	}
	return ""
}

func (x *ModulePin) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *ModulePin) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *ModulePin) GetBranch() string {
	if x != nil {
		return x.Branch
	}
	return ""
}

func (x *ModulePin) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *ModulePin) GetDigest() string {
	if x != nil {
		return x.Digest
	}
	return ""
}

func (x *ModulePin) GetCreateTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreateTime
	}
	return nil
}

var File_buf_alpha_module_v1alpha1_module_proto protoreflect.FileDescriptor

var file_buf_alpha_module_v1alpha1_module_proto_rawDesc = []byte{
	0x0a, 0x26, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x6d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x6d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb5, 0x01, 0x0a, 0x06, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x12,
	0x3b, 0x0a, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x48, 0x0a, 0x0c,
	0x64, 0x65, 0x70, 0x65, 0x6e, 0x64, 0x65, 0x6e, 0x63, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x24, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6d,
	0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4d,
	0x6f, 0x64, 0x75, 0x6c, 0x65, 0x50, 0x69, 0x6e, 0x52, 0x0c, 0x64, 0x65, 0x70, 0x65, 0x6e, 0x64,
	0x65, 0x6e, 0x63, 0x69, 0x65, 0x73, 0x12, 0x24, 0x0a, 0x0d, 0x64, 0x6f, 0x63, 0x75, 0x6d, 0x65,
	0x6e, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x64,
	0x6f, 0x63, 0x75, 0x6d, 0x65, 0x6e, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x3a, 0x0a, 0x0a,
	0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x18,
	0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x7d, 0x0a, 0x0f, 0x4d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x72,
	0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x6d,
	0x6f, 0x74, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70,
	0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72,
	0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x72, 0x65, 0x66,
	0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x72, 0x65,
	0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x22, 0xde, 0x01, 0x0a, 0x09, 0x4d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x50, 0x69, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x12, 0x14, 0x0a,
	0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77,
	0x6e, 0x65, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72,
	0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74,
	0x6f, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x62, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x12, 0x16, 0x0a, 0x06, 0x63,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d,
	0x6d, 0x69, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74, 0x12, 0x3b, 0x0a, 0x0b, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x63, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x42, 0x84, 0x01, 0x0a, 0x1d, 0x63, 0x6f, 0x6d,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0b, 0x4d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x5a, 0x56, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2f, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x3b, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_module_v1alpha1_module_proto_rawDescOnce sync.Once
	file_buf_alpha_module_v1alpha1_module_proto_rawDescData = file_buf_alpha_module_v1alpha1_module_proto_rawDesc
)

func file_buf_alpha_module_v1alpha1_module_proto_rawDescGZIP() []byte {
	file_buf_alpha_module_v1alpha1_module_proto_rawDescOnce.Do(func() {
		file_buf_alpha_module_v1alpha1_module_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_module_v1alpha1_module_proto_rawDescData)
	})
	return file_buf_alpha_module_v1alpha1_module_proto_rawDescData
}

var file_buf_alpha_module_v1alpha1_module_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_buf_alpha_module_v1alpha1_module_proto_goTypes = []interface{}{
	(*Module)(nil),                // 0: buf.alpha.module.v1alpha1.Module
	(*ModuleFile)(nil),            // 1: buf.alpha.module.v1alpha1.ModuleFile
	(*ModuleReference)(nil),       // 2: buf.alpha.module.v1alpha1.ModuleReference
	(*ModulePin)(nil),             // 3: buf.alpha.module.v1alpha1.ModulePin
	(*timestamppb.Timestamp)(nil), // 4: google.protobuf.Timestamp
}
var file_buf_alpha_module_v1alpha1_module_proto_depIdxs = []int32{
	1, // 0: buf.alpha.module.v1alpha1.Module.files:type_name -> buf.alpha.module.v1alpha1.ModuleFile
	3, // 1: buf.alpha.module.v1alpha1.Module.dependencies:type_name -> buf.alpha.module.v1alpha1.ModulePin
	4, // 2: buf.alpha.module.v1alpha1.ModulePin.create_time:type_name -> google.protobuf.Timestamp
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_module_v1alpha1_module_proto_init() }
func file_buf_alpha_module_v1alpha1_module_proto_init() {
	if File_buf_alpha_module_v1alpha1_module_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_module_v1alpha1_module_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Module); i {
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
		file_buf_alpha_module_v1alpha1_module_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModuleFile); i {
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
		file_buf_alpha_module_v1alpha1_module_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModuleReference); i {
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
		file_buf_alpha_module_v1alpha1_module_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ModulePin); i {
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
			RawDescriptor: file_buf_alpha_module_v1alpha1_module_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_alpha_module_v1alpha1_module_proto_goTypes,
		DependencyIndexes: file_buf_alpha_module_v1alpha1_module_proto_depIdxs,
		MessageInfos:      file_buf_alpha_module_v1alpha1_module_proto_msgTypes,
	}.Build()
	File_buf_alpha_module_v1alpha1_module_proto = out.File
	file_buf_alpha_module_v1alpha1_module_proto_rawDesc = nil
	file_buf_alpha_module_v1alpha1_module_proto_goTypes = nil
	file_buf_alpha_module_v1alpha1_module_proto_depIdxs = nil
}
