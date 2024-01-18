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

package protoencoding

import (
	"sync"

	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

func newResolver[F protodescriptor.FileDescriptor](fileDescriptors ...F) (Resolver, error) {
	if len(fileDescriptors) == 0 {
		return nil, nil
	}
	// TODO: handle if resolvable
	files, err := protodesc.FileOptions{
		AllowUnresolvable: true,
	}.NewFiles(
		protodescriptor.FileDescriptorSetForFileDescriptors(fileDescriptors...),
	)
	if err != nil {
		return nil, err
	}
	types := &protoregistry.Types{}
	var rangeErr error
	files.RangeFiles(func(fileDescriptor protoreflect.FileDescriptor) bool {
		if err := registerDescriptors(types, fileDescriptor); err != nil {
			rangeErr = err
			return false
		}
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}
	return &resolver{Files: files, Types: types}, nil
}

type resolver struct {
	*protoregistry.Files
	*protoregistry.Types
}

type descriptorContainer interface {
	Messages() protoreflect.MessageDescriptors
	Enums() protoreflect.EnumDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

func registerDescriptors(types *protoregistry.Types, container descriptorContainer) error {
	messageDescriptors := container.Messages()
	for i, messagesLen := 0, messageDescriptors.Len(); i < messagesLen; i++ {
		messageDescriptor := messageDescriptors.Get(i)
		if err := types.RegisterMessage(dynamicpb.NewMessageType(messageDescriptor)); err != nil {
			return err
		}
		// nested types, too
		if err := registerDescriptors(types, messageDescriptor); err != nil {
			return err
		}
	}

	enumDescriptors := container.Enums()
	for i, enumsLen := 0, enumDescriptors.Len(); i < enumsLen; i++ {
		enumDescriptor := enumDescriptors.Get(i)
		if err := types.RegisterEnum(dynamicpb.NewEnumType(enumDescriptor)); err != nil {
			return err
		}
	}

	extensionDescriptors := container.Extensions()
	for i, extensionsLen := 0, extensionDescriptors.Len(); i < extensionsLen; i++ {
		extensionDescriptor := extensionDescriptors.Get(i)
		if err := types.RegisterExtension(dynamicpb.NewExtensionType(extensionDescriptor)); err != nil {
			return err
		}
	}

	return nil
}

type lazyResolver struct {
	fn       func() (Resolver, error)
	init     sync.Once
	resolver Resolver
	err      error
}

func (l *lazyResolver) maybeInit() error {
	l.init.Do(func() {
		l.resolver, l.err = l.fn()
	})
	return l.err
}

func (l *lazyResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindFileByPath(path)
}

func (l *lazyResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindDescriptorByName(name)
}

func (l *lazyResolver) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindEnumByName(enum)
}

func (l *lazyResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindExtensionByName(field)
}

func (l *lazyResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindExtensionByNumber(message, field)
}

func (l *lazyResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindMessageByName(message)
}

func (l *lazyResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	if err := l.maybeInit(); err != nil {
		return nil, err
	}
	return l.resolver.FindMessageByURL(url)
}

type combinedResolver []Resolver

func (c combinedResolver) FindFileByPath(s string) (protoreflect.FileDescriptor, error) {
	var lastErr error
	for _, res := range c {
		file, err := res.FindFileByPath(s)
		if err == nil {
			return file, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	var lastErr error
	for _, res := range c {
		desc, err := res.FindDescriptorByName(name)
		if err == nil {
			return desc, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	var lastErr error
	for _, res := range c {
		extension, err := res.FindExtensionByName(field)
		if err == nil {
			return extension, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	var lastErr error
	for _, res := range c {
		extension, err := res.FindExtensionByNumber(message, field)
		if err == nil {
			return extension, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	var lastErr error
	for _, res := range c {
		msg, err := res.FindMessageByName(message)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	var lastErr error
	for _, res := range c {
		msg, err := res.FindMessageByURL(url)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}

func (c combinedResolver) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	var lastErr error
	for _, res := range c {
		msg, err := res.FindEnumByName(enum)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, protoregistry.NotFound
}
