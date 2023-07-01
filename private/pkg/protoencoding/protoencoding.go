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

package protoencoding

import (
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Resolver can resolve files, messages, enums, and extensions.
type Resolver interface {
	protodesc.Resolver
	protoregistry.ExtensionTypeResolver
	protoregistry.MessageTypeResolver
	FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error)
}

// NewResolver creates a new Resolver.
//
// If the input slice is empty, this returns nil
// The given FileDescriptors must be self-contained, that is they must contain all imports.
// This can NOT be guaranteed for FileDescriptorSets given over the wire, and can only be guaranteed from builds.
func NewResolver(fileDescriptors ...protodescriptor.FileDescriptor) (Resolver, error) {
	return newResolver(fileDescriptors...)
}

// NewLazyResolver creates a new Resolver that is constructed from the given
// descriptors only as needed, if invoked.
//
// If there is an error when constructing the resolver, it will be returned by all
// method calls of the returned resolver.
func NewLazyResolver(fileDescriptors ...protodescriptor.FileDescriptor) Resolver {
	return &lazyResolver{fn: func() (Resolver, error) {
		return newResolver(fileDescriptors...)
	}}
}

// Marshaler marshals Messages.
type Marshaler interface {
	Marshal(message proto.Message) ([]byte, error)
}

// NewWireMarshaler returns a new Marshaler for wire.
//
// See https://godoc.org/google.golang.org/protobuf/proto#MarshalOptions for a discussion on stability.
// This has the potential to be unstable over time.
func NewWireMarshaler() Marshaler {
	return newWireMarshaler()
}

// NewJSONMarshaler returns a new Marshaler for JSON.
//
// This has the potential to be unstable over time.
// resolver can be nil if unknown and are only needed for extensions.
func NewTextMarshaler(resolver Resolver, options ...TextMarshalerOption) Marshaler {
	return newTextMarshaler(resolver, options...)
}

type TextMarshalerOption func(*textMarshaler)

// TextMarshalerWithIndent says to use an indent of two spaces.
func TextMarshalerWithIndent() TextMarshalerOption {
	return func(textMarshaler *textMarshaler) {
		textMarshaler.indent = "  "
		textMarshaler.multiline = true
	}
}

// NewJSONMarshaler returns a new Marshaler for JSON.
//
// This has the potential to be unstable over time.
// resolver can be nil if unknown and are only needed for extensions.
func NewJSONMarshaler(resolver Resolver, options ...JSONMarshalerOption) Marshaler {
	return newJSONMarshaler(resolver, options...)
}

// JSONMarshalerOption is an option for a new JSONMarshaler.
type JSONMarshalerOption func(*jsonMarshaler)

// JSONMarshalerWithIndent says to use an indent of two spaces.
func JSONMarshalerWithIndent() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.indent = "  "
	}
}

// JSONMarshalerWithUseProtoNames says to use an use proto names.
func JSONMarshalerWithUseProtoNames() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.useProtoNames = true
	}
}

// JSONMarshalerWithEmitUnpopulated says to emit unpopulated values
func JSONMarshalerWithEmitUnpopulated() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.emitUnpopulated = true
	}
}

// Unmarshaler unmarshals Messages.
type Unmarshaler interface {
	Unmarshal(data []byte, message proto.Message) error
}

// NewWireUnmarshaler returns a new Unmarshaler for wire.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewWireUnmarshaler(resolver Resolver) Unmarshaler {
	return newWireUnmarshaler(resolver)
}

// NewTextUnmarshaler returns a new Unmarshaler for prototext.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewTextUnmarshaler(resolver Resolver) Unmarshaler {
	return newTextUnmarshaler(resolver)
}

// NewJSONUnmarshaler returns a new Unmarshaler for json.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewJSONUnmarshaler(resolver Resolver) Unmarshaler {
	return newJSONUnmarshaler(resolver)
}
