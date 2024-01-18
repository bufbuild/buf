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
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/protoyaml-go"
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
func NewResolver[F protodescriptor.FileDescriptor](fileDescriptors ...F) (Resolver, error) {
	return newResolver(fileDescriptors...)
}

// NewLazyResolver creates a new Resolver that is constructed from the given
// descriptors only as needed, if invoked.
//
// If there is an error when constructing the resolver, it will be returned by all
// method calls of the returned resolver.
func NewLazyResolver[F protodescriptor.FileDescriptor](fileDescriptors ...F) Resolver {
	return &lazyResolver{fn: func() (Resolver, error) {
		return newResolver(fileDescriptors...)
	}}
}

// CombineResolvers returns a resolver that uses all of the given resolvers. It
// will use the first resolver, and if it returns an error, the second will be
// tried, and so on.
func CombineResolvers(resolvers ...Resolver) Resolver {
	return combinedResolver(resolvers)
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
// resolver can be nil if unknown and is only needed for extensions.
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

// JSONMarshalerWithUseProtoNames says to use proto names.
func JSONMarshalerWithUseProtoNames() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.useProtoNames = true
	}
}

// JSONMarshalerWithUseEnumNumbers says to use enum numbers.
func JSONMarshalerWithUseEnumNumbers() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.useEnumNumbers = true
	}
}

// JSONMarshalerWithEmitUnpopulated says to emit unpopulated values
func JSONMarshalerWithEmitUnpopulated() JSONMarshalerOption {
	return func(jsonMarshaler *jsonMarshaler) {
		jsonMarshaler.emitUnpopulated = true
	}
}

// NewTxtpbMarshaler returns a new Marshaler for txtpb.
//
// resolver can be nil if unknown and is only needed for extensions.
func NewTxtpbMarshaler(resolver Resolver) Marshaler {
	return newTxtpbMarshaler(resolver)
}

// NewYAMLMarshaler returns a new Marshaler for YAML.
//
// resolver can be nil if unknown and is only needed for extensions.
func NewYAMLMarshaler(resolver Resolver, options ...YAMLMarshalerOption) Marshaler {
	return newYAMLMarshaler(resolver, options...)
}

// YAMLMarshalerOption is an option for a new YAMLMarshaler.
type YAMLMarshalerOption func(*yamlMarshaler)

// YAMLMarshalerWithIndent says to use an indent of two spaces.
func YAMLMarshalerWithIndent() YAMLMarshalerOption {
	return func(yamlMarshaler *yamlMarshaler) {
		yamlMarshaler.indent = 2
	}
}

// YAMLMarshalerWithUseProtoNames says to use proto names.
func YAMLMarshalerWithUseProtoNames() YAMLMarshalerOption {
	return func(yamlMarshaler *yamlMarshaler) {
		yamlMarshaler.useProtoNames = true
	}
}

// YAMLMarshalerWithUseEnumNumbers says to use enum numbers.
func YAMLMarshalerWithUseEnumNumbers() YAMLMarshalerOption {
	return func(yamlMarshaler *yamlMarshaler) {
		yamlMarshaler.useEnumNumbers = true
	}
}

// YAMLMarshalerWithEmitUnpopulated says to emit unpopulated values
func YAMLMarshalerWithEmitUnpopulated() YAMLMarshalerOption {
	return func(yamlMarshaler *yamlMarshaler) {
		yamlMarshaler.emitUnpopulated = true
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

// NewJSONUnmarshaler returns a new Unmarshaler for json.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewJSONUnmarshaler(resolver Resolver, options ...JSONUnmarshalerOption) Unmarshaler {
	return newJSONUnmarshaler(resolver, options...)
}

// JSONUnmarshalerOption is an option for a new JSONUnmarshaler.
type JSONUnmarshalerOption func(*jsonUnmarshaler)

// JSONUnmarshalerWithDisallowUnknown says to disallow unrecognized fields.
func JSONUnmarshalerWithDisallowUnknown() JSONUnmarshalerOption {
	return func(jsonUnmarshaler *jsonUnmarshaler) {
		jsonUnmarshaler.disallowUnknown = true
	}
}

// NewTxtpbUnmarshaler returns a new Unmarshaler for txtpb.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewTxtpbUnmarshaler(resolver Resolver) Unmarshaler {
	return newTxtpbUnmarshaler(resolver)
}

// YAMLUnmarshalerOption is an option for a new YAMLUnmarshaler.
type YAMLUnmarshalerOption func(*yamlUnmarshaler)

// YAMLUnmarshalerWithPath says to use the given path.
func YAMLUnmarshalerWithPath(path string) YAMLUnmarshalerOption {
	return func(yamlUnmarshaler *yamlUnmarshaler) {
		yamlUnmarshaler.path = path
	}
}

func YAMLUnmarshalerWithValidator(validator protoyaml.Validator) YAMLUnmarshalerOption {
	return func(yamlUnmarshaler *yamlUnmarshaler) {
		yamlUnmarshaler.validator = validator
	}
}

// NewYAMLUnmarshaler returns a new Unmarshaler for yaml.
//
// resolver can be nil if unknown and are only needed for extensions.
func NewYAMLUnmarshaler(resolver Resolver, options ...YAMLUnmarshalerOption) Unmarshaler {
	return newYAMLUnmarshaler(resolver, options...)
}
