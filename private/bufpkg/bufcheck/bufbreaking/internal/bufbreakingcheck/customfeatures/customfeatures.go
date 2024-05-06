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

package customfeatures

import (
	"fmt"

	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// ResolveCppFeature returns a value for the given field name of the (pb.cpp) custom feature
// for the given field.
func ResolveCppFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	return resolveFeature(field, protobuf.E_Cpp.TypeDescriptor(), fieldName, expectedKind)
}

// ResolveJavaFeature returns a value for the given field name of the (pb.java) custom feature
// for the given field.
func ResolveJavaFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	return resolveFeature(field, protobuf.E_Java.TypeDescriptor(), fieldName, expectedKind)
}

type resolverForExtension struct {
	extension protoreflect.ExtensionType
}

func (r resolverForExtension) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if field == r.extension.TypeDescriptor().FullName() {
		return r.extension, nil
	}
	return nil, protoregistry.NotFound
}

func (r resolverForExtension) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	descriptor := r.extension.TypeDescriptor()
	if message == descriptor.ContainingMessage().FullName() && field == descriptor.Number() {
		return r.extension, nil
	}
	return nil, protoregistry.NotFound
}

type fieldDescriptorWithOptions struct {
	protoreflect.FieldDescriptor
	parent  protoreflect.Descriptor
	options proto.Message
}

func (f *fieldDescriptorWithOptions) Options() proto.Message {
	return f.options
}

func (f *fieldDescriptorWithOptions) Parent() protoreflect.Descriptor {
	return f.parent
}

type messageDescriptorWithOptions struct {
	protoreflect.MessageDescriptor
	parent  protoreflect.Descriptor
	options proto.Message
}

func (f *messageDescriptorWithOptions) Options() proto.Message {
	return f.options
}

func (f *messageDescriptorWithOptions) Parent() protoreflect.Descriptor {
	return f.parent
}

type fileDescriptorWithOptions struct {
	protoreflect.FileDescriptor
	options proto.Message
}

func (f *fileDescriptorWithOptions) Options() proto.Message {
	return f.options
}

func resolveFeature(
	field protoreflect.FieldDescriptor,
	extension protoreflect.ExtensionTypeDescriptor,
	fieldName protoreflect.Name,
	expectedKind protoreflect.Kind,
) (protoreflect.Value, error) {
	field, err := reparseFeaturesInField(field, resolverForExtension{extension.Type()})
	if err != nil {
		return protoreflect.Value{}, err
	}
	featureField := extension.Message().Fields().ByName(fieldName)
	if featureField == nil {
		return protoreflect.Value{}, fmt.Errorf("unable to resolve field descriptor for %s.%s", extension.Message().FullName(), fieldName)
	}
	if featureField.Kind() != expectedKind || featureField.IsList() {
		return protoreflect.Value{}, fmt.Errorf("resolved field descriptor for %s.%s has unexpected type: expected optional %s, got %s %s",
			extension.Message().FullName(), fieldName, expectedKind, featureField.Cardinality(), featureField.Kind())
	}
	return protoutil.ResolveCustomFeature(
		field,
		extension.Type(),
		featureField,
	)
}

// reparseFeaturesInField re-parses any features in the given field's options and returns a new descriptor
// that returns options with the re-parsed features. This recursively applies up the hierarchy to the
// field's parent and all ancestors.
//
// We need to reparse the features to make sure that custom feature values are using the right extension
// descriptors. The protobuf-go runtime is quite strict, so we could get panic issues if we queried using
// one extension descriptor, but the value in the message actually refers to a different extension
// descriptor for the same field number (like an analogous descriptor provided by protocompile).
func reparseFeaturesInField(field protoreflect.FieldDescriptor, resolver protoregistry.ExtensionTypeResolver) (protoreflect.FieldDescriptor, error) {
	// TODO: This is inefficient, to reparse the features for each field. We repeatedly reparse
	//       features for the parent/ancestor message(s). Ideally we could cache/re-use the
	//       reparsed features when checking other fields.
	opts, err := reparseFeatures(field.Options(), resolver)
	if err != nil {
		return nil, err
	}
	parent := field.Parent()
	switch descriptor := parent.(type) {
	case protoreflect.MessageDescriptor:
		parent, err = reparseFeaturesInMessage(descriptor, resolver)
		if err != nil {
			return nil, err
		}
	case protoreflect.FileDescriptor:
		parent, err = reparseFeaturesInFile(descriptor, resolver)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("field has unexpected parent type %T", parent)
	}
	return &fieldDescriptorWithOptions{FieldDescriptor: field, parent: parent, options: opts}, nil
}

func reparseFeaturesInMessage(message protoreflect.MessageDescriptor, resolver protoregistry.ExtensionTypeResolver) (protoreflect.MessageDescriptor, error) {
	opts, err := reparseFeatures(message.Options(), resolver)
	if err != nil {
		return nil, err
	}
	parent := message.Parent()
	switch descriptor := parent.(type) {
	case protoreflect.MessageDescriptor:
		parent, err = reparseFeaturesInMessage(descriptor, resolver)
		if err != nil {
			return nil, err
		}
	case protoreflect.FileDescriptor:
		parent, err = reparseFeaturesInFile(descriptor, resolver)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("message has unexpected parent type %T", parent)
	}
	return &messageDescriptorWithOptions{MessageDescriptor: message, parent: parent, options: opts}, nil
}

func reparseFeaturesInFile(file protoreflect.FileDescriptor, resolver protoregistry.ExtensionTypeResolver) (protoreflect.FileDescriptor, error) {
	opts, err := reparseFeatures(file.Options(), resolver)
	if err != nil {
		return nil, err
	}
	return &fileDescriptorWithOptions{FileDescriptor: file, options: opts}, nil
}

func reparseFeatures(options proto.Message, resolver protoregistry.ExtensionTypeResolver) (proto.Message, error) {
	optionsDescriptor := options.ProtoReflect().Descriptor()
	featuresField := optionsDescriptor.Fields().ByName("features")
	if featuresField == nil {
		return nil, fmt.Errorf("options message does not have features field")
	}
	if featuresField.Message() == nil {
		return nil, fmt.Errorf("options message does not have expected features field: expecting message, got %v", featuresField.Kind())
	}
	features := options.ProtoReflect().Get(featuresField).Message()
	if !features.IsValid() {
		// features are absent so nothing to reparse
		return options, nil
	}
	data, err := proto.Marshal(features.Interface())
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		// empty features, nothing to reparse
		return options, nil
	}
	reparsedFeatures := features.Type().New()
	if err := (proto.UnmarshalOptions{Resolver: resolver}).Unmarshal(data, reparsedFeatures.Interface()); err != nil {
		return nil, err
	}
	options = proto.Clone(options) // make a copy so we don't mutate the original protos
	options.ProtoReflect().Set(featuresField, protoreflect.ValueOfMessage(reparsedFeatures))
	return options, nil
}
