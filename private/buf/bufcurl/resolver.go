// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufcurl

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ResolveMethodDescriptor uses the given resolver to find a descriptor for
// the requested service and method. The service name must be fully-qualified.
func ResolveMethodDescriptor(res Resolver, service, method string) (protoreflect.MethodDescriptor, error) {
	descriptor, err := res.FindDescriptorByName(protoreflect.FullName(service))
	if err == protoregistry.NotFound {
		return nil, fmt.Errorf("failed to find service named %q in schema", service)
	} else if err != nil {
		return nil, err
	}
	serviceDescriptor, ok := descriptor.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("URL indicates service name %q, but that name is a %s", service, DescriptorKind(descriptor))
	}
	methodDescriptor := serviceDescriptor.Methods().ByName(protoreflect.Name(method))
	if methodDescriptor == nil {
		return nil, fmt.Errorf("URL indicates method name %q, but service %q contains no such method", method, service)
	}
	return methodDescriptor, nil
}

// NewImageResolver returns a Resolver that uses the given image to resolve
// symbols and extensions.
func NewImageResolver(image bufimage.Image) (Resolver, error) {
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: bufimage.ImageToFileDescriptorProtos(image),
	})
	if err != nil {
		return nil, err
	}
	return &imageResolver{
		files: files,
	}, nil
}

type imageResolver struct {
	files    *protoregistry.Files
	initExts sync.Once
	exts     *protoregistry.Types
}

func (i *imageResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return i.files.FindDescriptorByName(name)
}

func (i *imageResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	d, err := i.files.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("element %s is a %s, not a message", message, DescriptorKind(d))
	}
	return dynamicpb.NewMessageType(md), nil
}

func (i *imageResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	pos := strings.LastIndexByte(url, '/')
	typeName := url[pos+1:]
	return i.FindMessageByName(protoreflect.FullName(typeName))
}

func (i *imageResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	d, err := i.files.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	fd, ok := d.(protoreflect.FieldDescriptor)
	if !ok || !fd.IsExtension() {
		return nil, fmt.Errorf("element %s is a %s, not an extension", field, DescriptorKind(d))
	}
	return dynamicpb.NewExtensionType(fd), nil
}

func (i *imageResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	// Most usages won't need to resolve extensions. So instead of proactively
	// indexing them, we defer that work until it's actually needed.
	i.initExts.Do(i.doInitExts)
	return i.exts.FindExtensionByNumber(message, field)
}

func (i *imageResolver) doInitExts() {
	var types protoregistry.Types
	i.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		registerExtensions(&types, fd)
		return true
	})
	i.exts = &types
}

type extensionContainer interface {
	Messages() protoreflect.MessageDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func registerExtensions(reg *protoregistry.Types, descriptor extensionContainer) {
	exts := descriptor.Extensions()
	for i := 0; i < exts.Len(); i++ {
		extType := dynamicpb.NewExtensionType(exts.Get(i))
		_ = reg.RegisterExtension(extType)
	}
	msgs := descriptor.Messages()
	for i := 0; i < msgs.Len(); i++ {
		registerExtensions(reg, msgs.Get(i))
	}
}

// DescriptorKind returns a succinct description of the type of the given descriptor.
func DescriptorKind(d protoreflect.Descriptor) string {
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return "file"
	case protoreflect.MessageDescriptor:
		return "message"
	case protoreflect.FieldDescriptor:
		if d.IsExtension() {
			return "extension"
		}
		return "field"
	case protoreflect.OneofDescriptor:
		return "oneof"
	case protoreflect.EnumDescriptor:
		return "enum"
	case protoreflect.EnumValueDescriptor:
		return "enum value"
	case protoreflect.ServiceDescriptor:
		return "service"
	case protoreflect.MethodDescriptor:
		return "method"
	default:
		return fmt.Sprintf("%T", d)
	}
}
