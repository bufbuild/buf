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
// 	protoc        v3.18.1
// source: buf/alpha/registry/v1alpha1/generate.proto

package registryv1alpha1

import (
	v1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/image/v1"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	pluginpb "google.golang.org/protobuf/types/pluginpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// File defines a file with a path and some content.
type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// path is the relative path of the file.
	// Path can only use '/' as the separator character, and includes no ".." components.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// content is the content of the file.
	Content []byte `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use File.ProtoReflect.Descriptor instead.
func (*File) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{0}
}

func (x *File) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *File) GetContent() []byte {
	if x != nil {
		return x.Content
	}
	return nil
}

// RuntimeLibrary describes a pinned runtime library dependency of the generated code.
type RuntimeLibrary struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the runtime library dependency. The format should match the
	// format used for dependencies in the dependency management tooling of the
	// associated language ecosystem. This is set by the user using Dockerfile Labels.
	// For example, for the plugin "protoc-gen-go", this might be "google.golang.org/protobuf".
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The version of the runtime library dependency associated with the generated
	// code. The format should match the format used for dependency versions in the
	// dependency management tooling of the associated language ecosystem.
	// This is set by the user using Dockerfile Labels.
	// For example, for the plugin "protoc-gen-go", this might be "v1.26.0".
	Version string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *RuntimeLibrary) Reset() {
	*x = RuntimeLibrary{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RuntimeLibrary) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RuntimeLibrary) ProtoMessage() {}

func (x *RuntimeLibrary) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RuntimeLibrary.ProtoReflect.Descriptor instead.
func (*RuntimeLibrary) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{1}
}

func (x *RuntimeLibrary) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *RuntimeLibrary) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type PluginReference struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The owner of the plugin which identifies the
	// plugins to use with this generation.
	Owner string `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// The name of the plugin which identifies the
	// plugins to use with this generation.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// The plugin version to use with this generation.
	Version string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// The parameters to pass to the plugin. These will
	// be merged into a single, comma-separated string.
	Parameters []string `protobuf:"bytes,4,rep,name=parameters,proto3" json:"parameters,omitempty"`
}

func (x *PluginReference) Reset() {
	*x = PluginReference{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PluginReference) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PluginReference) ProtoMessage() {}

func (x *PluginReference) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PluginReference.ProtoReflect.Descriptor instead.
func (*PluginReference) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{2}
}

func (x *PluginReference) GetOwner() string {
	if x != nil {
		return x.Owner
	}
	return ""
}

func (x *PluginReference) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *PluginReference) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *PluginReference) GetParameters() []string {
	if x != nil {
		return x.Parameters
	}
	return nil
}

type GeneratePluginsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The image to run plugins against to generate the desired file outputs.
	//
	// All image files that are not imports and not well-known types will be generated.
	// If you want to filter what files are generated, modify the image.
	// If you want to include imports, set include_imports.
	Image *v1.Image `protobuf:"bytes,1,opt,name=image,proto3" json:"image,omitempty"`
	// The array of plugins to use.
	Plugins []*PluginReference `protobuf:"bytes,2,rep,name=plugins,proto3" json:"plugins,omitempty"`
	// Include imports from the Image in generation.
	IncludeImports bool `protobuf:"varint,3,opt,name=include_imports,json=includeImports,proto3" json:"include_imports,omitempty"`
}

func (x *GeneratePluginsRequest) Reset() {
	*x = GeneratePluginsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GeneratePluginsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GeneratePluginsRequest) ProtoMessage() {}

func (x *GeneratePluginsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GeneratePluginsRequest.ProtoReflect.Descriptor instead.
func (*GeneratePluginsRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{3}
}

func (x *GeneratePluginsRequest) GetImage() *v1.Image {
	if x != nil {
		return x.Image
	}
	return nil
}

func (x *GeneratePluginsRequest) GetPlugins() []*PluginReference {
	if x != nil {
		return x.Plugins
	}
	return nil
}

func (x *GeneratePluginsRequest) GetIncludeImports() bool {
	if x != nil {
		return x.IncludeImports
	}
	return false
}

type GeneratePluginsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Contains all the responses from the generated plugins. The order
	// is defined by the order of the plugins in the request.
	Responses []*pluginpb.CodeGeneratorResponse `protobuf:"bytes,1,rep,name=responses,proto3" json:"responses,omitempty"`
	// An optional array defining runtime libraries that the generated code
	// requires to run, as specified by the plugin author. This may contain
	// duplicate entries as the generation can be the result of multiple plugins,
	// each of which declares its own runtime library dependencies. The libraries
	// returned are lexicographically ordered by their name, but not deduplicated.
	// How to handle duplicate libraries is left to the user.
	RuntimeLibraries []*RuntimeLibrary `protobuf:"bytes,2,rep,name=runtime_libraries,json=runtimeLibraries,proto3" json:"runtime_libraries,omitempty"`
}

func (x *GeneratePluginsResponse) Reset() {
	*x = GeneratePluginsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GeneratePluginsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GeneratePluginsResponse) ProtoMessage() {}

func (x *GeneratePluginsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GeneratePluginsResponse.ProtoReflect.Descriptor instead.
func (*GeneratePluginsResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{4}
}

func (x *GeneratePluginsResponse) GetResponses() []*pluginpb.CodeGeneratorResponse {
	if x != nil {
		return x.Responses
	}
	return nil
}

func (x *GeneratePluginsResponse) GetRuntimeLibraries() []*RuntimeLibrary {
	if x != nil {
		return x.RuntimeLibraries
	}
	return nil
}

type GenerateTemplateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The image to run plugins against to generate the
	// desired file outputs.
	Image *v1.Image `protobuf:"bytes,1,opt,name=image,proto3" json:"image,omitempty"`
	// The owner of the template which identifies the
	// plugins to use with this generation.
	TemplateOwner string `protobuf:"bytes,2,opt,name=template_owner,json=templateOwner,proto3" json:"template_owner,omitempty"`
	// The name of the template which identifies the
	// plugins to use with this generation.
	TemplateName string `protobuf:"bytes,3,opt,name=template_name,json=templateName,proto3" json:"template_name,omitempty"`
	// The template version to use to determine the
	// plugin versions in the template.
	TemplateVersion string `protobuf:"bytes,4,opt,name=template_version,json=templateVersion,proto3" json:"template_version,omitempty"`
}

func (x *GenerateTemplateRequest) Reset() {
	*x = GenerateTemplateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GenerateTemplateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenerateTemplateRequest) ProtoMessage() {}

func (x *GenerateTemplateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenerateTemplateRequest.ProtoReflect.Descriptor instead.
func (*GenerateTemplateRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{5}
}

func (x *GenerateTemplateRequest) GetImage() *v1.Image {
	if x != nil {
		return x.Image
	}
	return nil
}

func (x *GenerateTemplateRequest) GetTemplateOwner() string {
	if x != nil {
		return x.TemplateOwner
	}
	return ""
}

func (x *GenerateTemplateRequest) GetTemplateName() string {
	if x != nil {
		return x.TemplateName
	}
	return ""
}

func (x *GenerateTemplateRequest) GetTemplateVersion() string {
	if x != nil {
		return x.TemplateVersion
	}
	return ""
}

type GenerateTemplateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// files contains all the files output by the generation,
	// in lexicographical order.
	Files []*File `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
	// An optional array defining runtime libraries that the generated code
	// requires to run. This may contain duplicate entries as the generation
	// can be the result of multiple plugins, each of which declares its own
	// runtime library dependencies.
	RuntimeLibraries []*RuntimeLibrary `protobuf:"bytes,2,rep,name=runtime_libraries,json=runtimeLibraries,proto3" json:"runtime_libraries,omitempty"`
}

func (x *GenerateTemplateResponse) Reset() {
	*x = GenerateTemplateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GenerateTemplateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenerateTemplateResponse) ProtoMessage() {}

func (x *GenerateTemplateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenerateTemplateResponse.ProtoReflect.Descriptor instead.
func (*GenerateTemplateResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP(), []int{6}
}

func (x *GenerateTemplateResponse) GetFiles() []*File {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *GenerateTemplateResponse) GetRuntimeLibraries() []*RuntimeLibrary {
	if x != nil {
		return x.RuntimeLibraries
	}
	return nil
}

var File_buf_alpha_registry_v1alpha1_generate_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_generate_proto_rawDesc = []byte{
	0x0a, 0x2a, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x67, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75,
	0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x1a, 0x25, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x63, 0x6f, 0x6d, 0x70, 0x69,
	0x6c, 0x65, 0x72, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1e, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x34, 0x0a, 0x04, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x18, 0x0a, 0x07,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x3e, 0x0a, 0x0e, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d,
	0x65, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x75, 0x0a, 0x0f, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e,
	0x52, 0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e,
	0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12,
	0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1e, 0x0a,
	0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x0a, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x65, 0x72, 0x73, 0x22, 0xba, 0x01,
	0x0a, 0x16, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2f, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x6d, 0x61,
	0x67, 0x65, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x46, 0x0a, 0x07, 0x70, 0x6c, 0x75,
	0x67, 0x69, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x52,
	0x65, 0x66, 0x65, 0x72, 0x65, 0x6e, 0x63, 0x65, 0x52, 0x07, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e,
	0x73, 0x12, 0x27, 0x0a, 0x0f, 0x69, 0x6e, 0x63, 0x6c, 0x75, 0x64, 0x65, 0x5f, 0x69, 0x6d, 0x70,
	0x6f, 0x72, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0e, 0x69, 0x6e, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x49, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x22, 0xc2, 0x01, 0x0a, 0x17, 0x47,
	0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4d, 0x0a, 0x09, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x63, 0x6f, 0x6d, 0x70,
	0x69, 0x6c, 0x65, 0x72, 0x2e, 0x43, 0x6f, 0x64, 0x65, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
	0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52, 0x09, 0x72, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x73, 0x12, 0x58, 0x0a, 0x11, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65,
	0x5f, 0x6c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x2b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x52,
	0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x79, 0x52, 0x10, 0x72,
	0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x22,
	0xc1, 0x01, 0x0a, 0x17, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x54, 0x65, 0x6d, 0x70,
	0x6c, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2f, 0x0a, 0x05, 0x69,
	0x6d, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x49, 0x6d, 0x61, 0x67, 0x65, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x25, 0x0a, 0x0e,
	0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x5f, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x4f, 0x77,
	0x6e, 0x65, 0x72, 0x12, 0x23, 0x0a, 0x0d, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x65, 0x6d, 0x70,
	0x6c, 0x61, 0x74, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x74, 0x65, 0x6d, 0x70,
	0x6c, 0x61, 0x74, 0x65, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0f, 0x74, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x22, 0xad, 0x01, 0x0a, 0x18, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x37, 0x0a, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x46, 0x69,
	0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x58, 0x0a, 0x11, 0x72, 0x75, 0x6e,
	0x74, 0x69, 0x6d, 0x65, 0x5f, 0x6c, 0x69, 0x62, 0x72, 0x61, 0x72, 0x69, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x2e, 0x52, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72,
	0x79, 0x52, 0x10, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x4c, 0x69, 0x62, 0x72, 0x61, 0x72,
	0x69, 0x65, 0x73, 0x32, 0x90, 0x02, 0x0a, 0x0f, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x7c, 0x0a, 0x0f, 0x47, 0x65, 0x6e, 0x65, 0x72,
	0x61, 0x74, 0x65, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x12, 0x33, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
	0x65, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x34, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x50, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x7f, 0x0a, 0x10, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
	0x65, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x12, 0x34, 0x2e, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x35, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x47, 0x65,
	0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x54, 0x65, 0x6d, 0x70, 0x6c, 0x61, 0x74, 0x65, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x9a, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x62,
	0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0d, 0x47, 0x65, 0x6e, 0x65,
	0x72, 0x61, 0x74, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64,
	0x2f, 0x62, 0x75, 0x66, 0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42,
	0x75, 0x66, 0x2e, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72,
	0x79, 0x2e, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66,
	0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c,
	0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41,
	0x6c, 0x70, 0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0xea, 0x02, 0x1e, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a,
	0x3a, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_registry_v1alpha1_generate_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_generate_proto_rawDescData = file_buf_alpha_registry_v1alpha1_generate_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_generate_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_generate_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_generate_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_generate_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_generate_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_buf_alpha_registry_v1alpha1_generate_proto_goTypes = []interface{}{
	(*File)(nil),                           // 0: buf.alpha.registry.v1alpha1.File
	(*RuntimeLibrary)(nil),                 // 1: buf.alpha.registry.v1alpha1.RuntimeLibrary
	(*PluginReference)(nil),                // 2: buf.alpha.registry.v1alpha1.PluginReference
	(*GeneratePluginsRequest)(nil),         // 3: buf.alpha.registry.v1alpha1.GeneratePluginsRequest
	(*GeneratePluginsResponse)(nil),        // 4: buf.alpha.registry.v1alpha1.GeneratePluginsResponse
	(*GenerateTemplateRequest)(nil),        // 5: buf.alpha.registry.v1alpha1.GenerateTemplateRequest
	(*GenerateTemplateResponse)(nil),       // 6: buf.alpha.registry.v1alpha1.GenerateTemplateResponse
	(*v1.Image)(nil),                       // 7: buf.alpha.image.v1.Image
	(*pluginpb.CodeGeneratorResponse)(nil), // 8: google.protobuf.compiler.CodeGeneratorResponse
}
var file_buf_alpha_registry_v1alpha1_generate_proto_depIdxs = []int32{
	7, // 0: buf.alpha.registry.v1alpha1.GeneratePluginsRequest.image:type_name -> buf.alpha.image.v1.Image
	2, // 1: buf.alpha.registry.v1alpha1.GeneratePluginsRequest.plugins:type_name -> buf.alpha.registry.v1alpha1.PluginReference
	8, // 2: buf.alpha.registry.v1alpha1.GeneratePluginsResponse.responses:type_name -> google.protobuf.compiler.CodeGeneratorResponse
	1, // 3: buf.alpha.registry.v1alpha1.GeneratePluginsResponse.runtime_libraries:type_name -> buf.alpha.registry.v1alpha1.RuntimeLibrary
	7, // 4: buf.alpha.registry.v1alpha1.GenerateTemplateRequest.image:type_name -> buf.alpha.image.v1.Image
	0, // 5: buf.alpha.registry.v1alpha1.GenerateTemplateResponse.files:type_name -> buf.alpha.registry.v1alpha1.File
	1, // 6: buf.alpha.registry.v1alpha1.GenerateTemplateResponse.runtime_libraries:type_name -> buf.alpha.registry.v1alpha1.RuntimeLibrary
	3, // 7: buf.alpha.registry.v1alpha1.GenerateService.GeneratePlugins:input_type -> buf.alpha.registry.v1alpha1.GeneratePluginsRequest
	5, // 8: buf.alpha.registry.v1alpha1.GenerateService.GenerateTemplate:input_type -> buf.alpha.registry.v1alpha1.GenerateTemplateRequest
	4, // 9: buf.alpha.registry.v1alpha1.GenerateService.GeneratePlugins:output_type -> buf.alpha.registry.v1alpha1.GeneratePluginsResponse
	6, // 10: buf.alpha.registry.v1alpha1.GenerateService.GenerateTemplate:output_type -> buf.alpha.registry.v1alpha1.GenerateTemplateResponse
	9, // [9:11] is the sub-list for method output_type
	7, // [7:9] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_generate_proto_init() }
func file_buf_alpha_registry_v1alpha1_generate_proto_init() {
	if File_buf_alpha_registry_v1alpha1_generate_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*File); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RuntimeLibrary); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PluginReference); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GeneratePluginsRequest); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GeneratePluginsResponse); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GenerateTemplateRequest); i {
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
		file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GenerateTemplateResponse); i {
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
			RawDescriptor: file_buf_alpha_registry_v1alpha1_generate_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_generate_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_generate_proto_depIdxs,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_generate_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_generate_proto = out.File
	file_buf_alpha_registry_v1alpha1_generate_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_generate_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_generate_proto_depIdxs = nil
}
