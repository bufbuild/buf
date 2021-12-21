// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufreflect

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	// TODO: This should depend on a go.buf.build import path.
	reflectv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/reflect/v1alpha1"
)

// Codec defines an interface used to encode and decode proto.Messages with DescriptorInfo.
type Codec struct {
	name string
}

// CodecOption is an option for a new *Codec.
type CodecOption func(*Codec)

// CodecWithName returns a new CodecOption that sets the Codec's name.
//
// By default, the Codec's name is "proto".
func CodecWithName(name string) CodecOption {
	return func(codec *Codec) {
		codec.name = name
	}
}

// NewCodec returns a new codec that marshals proto.Messages with DescriptorInfo if
// they implement the DescriptorInfoMarshaler interface. Otherwise, the default
// proto.Marshal strategy is used.
func NewCodec(opts ...CodecOption) (*Codec, error) {
	return newCodec(opts...), nil
}

// NewMessage returns a new dynamic proto.Message for the fully qualified typeName
// in the bufimage.Image.
func NewMessage(
	ctx context.Context,
	image bufimage.Image,
	typeName string,
) (proto.Message, error) {
	if err := ValidateTypeName(typeName); err != nil {
		return nil, err
	}
	files, err := protodesc.NewFiles(bufimage.ImageToFileDescriptorSet(image))
	if err != nil {
		return nil, err
	}
	descriptor, err := files.FindDescriptorByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, err
	}
	switch typedDescriptor := descriptor.(type) {
	case protoreflect.MessageDescriptor:
		return dynamicpb.NewMessage(typedDescriptor), nil
	default:
		return nil, fmt.Errorf("%q must be a message but is a %T", typeName, typedDescriptor)
	}
}

// UnmarshalDescriptorInfo acts like proto.Unmarshal, but returns the DescriptorInfo
// associated with the serialized proto.Message, if any.
//
// The bytes MUST be serialized with the generated MarshalWithDescriptorInfo method for
// this to work. If the proto.Message was marshaled with proto.Marshal, the DescriptorInfo
// will not exist and an error will be returned.
func UnmarshalDescriptorInfo(bytes []byte) (*reflectv1alpha1.DescriptorInfo, error) {
	reflector := new(reflectv1alpha1.Reflector)
	if err := proto.Unmarshal(bytes, reflector); err != nil {
		return nil, err
	}
	descriptorInfo := reflector.GetDescriptorInfo()
	if descriptorInfo == nil {
		return nil, errors.New("descriptor does not embed DescriptorInfo")
	}
	return descriptorInfo, nil
}

// ValidateTypeName validates that the typeName is well-formed, such that it has one or more
// '.'-delimited package components and no '/' elements.
func ValidateTypeName(typeName string) error {
	if fullName := protoreflect.FullName(typeName); !fullName.IsValid() {
		return fmt.Errorf("%q is not a valid fully qualified name", fullName)
	}
	return nil
}
