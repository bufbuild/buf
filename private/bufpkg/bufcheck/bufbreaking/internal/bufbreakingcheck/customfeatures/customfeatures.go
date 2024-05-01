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
	_ "embed"
	"fmt"
	"sync"

	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// These values are defined in cpp_features.proto
// https://github.com/protocolbuffers/protobuf/blob/v27.0-rc1/src/google/protobuf/cpp_features.proto#L42-L44
const (
	CppStringTypeView        CppStringType = 1
	CppStringTypeCord        CppStringType = 2
	CppStringTypeString      CppStringType = 3
	CppStringTypeStringPiece CppStringType = -999999
	// This last one is not really part of the features enum. But it was a
	// value in the old ctype option enum. We have this here so we can
	// compare ctype to string_type values.
)

// These values are defined in java_features.proto
// https://github.com/protocolbuffers/protobuf/blob/v26.1/java/core/src/main/resources/google/protobuf/java_features.proto#L39-L44
const (
	JavaUTF8ValidationDefault JavaUTF8Validation = 1
	JavaUTF8ValidationVerify  JavaUTF8Validation = 2
)

var (
	//go:embed cpp_features.bin
	cppFeaturesBin []byte

	cppFeaturesInit       sync.Once
	cppFeaturesDescriptor protoreflect.ExtensionTypeDescriptor
	cppFeaturesErr        error

	//go:embed java_features.bin
	javaFeaturesBin []byte

	javaFeaturesInit       sync.Once
	javaFeaturesDescriptor protoreflect.ExtensionTypeDescriptor
	javaFeaturesErr        error
)

// CppStringType represents enum values for pb.CppFeatures.StringType.
type CppStringType int32

func (c CppStringType) String() string {
	switch c {
	case CppStringTypeView:
		return "VIEW"
	case CppStringTypeCord:
		return "CORD"
	case CppStringTypeString:
		return "STRING"
	case CppStringTypeStringPiece:
		return "STRING_PIECE"
	default:
		return fmt.Sprintf("%d", c)
	}
}

// JavaUTF8Validation represents enum values for pb.JavaFeatures.Utf8Validation.
type JavaUTF8Validation int32

func (j JavaUTF8Validation) String() string {
	switch j {
	case JavaUTF8ValidationDefault:
		return "DEFAULT"
	case JavaUTF8ValidationVerify:
		return "VERIFY"
	default:
		return fmt.Sprintf("%d", j)
	}
}

// ResolveCppFeature returns a value for the given field name of the (pb.cpp) custom feature
// for the given field.
func ResolveCppFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	extension, err := CppFeatures()
	if err != nil {
		return protoreflect.Value{}, err
	}
	return resolveFeature(field, extension, fieldName, expectedKind)
}

// ResolveJavaFeature returns a value for the given field name of the (pb.java) custom feature
// for the given field.
func ResolveJavaFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	extension, err := JavaFeatures()
	if err != nil {
		return protoreflect.Value{}, err
	}
	return resolveFeature(field, extension, fieldName, expectedKind)
}

// CppFeatures returns the extension of FeatureSet named (pb.cpp).
func CppFeatures() (protoreflect.ExtensionTypeDescriptor, error) {
	cppFeaturesInit.Do(func() {
		cppFeaturesDescriptor, cppFeaturesErr = getExtensionDescriptor(
			cppFeaturesBin,
			"google/protobuf/cpp_features.proto",
			"pb.cpp",
			"pb.CppFeatures",
		)
	})
	return cppFeaturesDescriptor, cppFeaturesErr
}

// JavaFeatures returns the extension of FeatureSet named (pb.java).
func JavaFeatures() (protoreflect.ExtensionTypeDescriptor, error) {
	javaFeaturesInit.Do(func() {
		javaFeaturesDescriptor, javaFeaturesErr = getExtensionDescriptor(
			javaFeaturesBin,
			"google/protobuf/java_features.proto",
			"pb.java",
			"pb.JavaFeatures",
		)
	})
	return javaFeaturesDescriptor, javaFeaturesErr
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

func getExtensionDescriptor(
	fileDescriptorSetData []byte,
	path string,
	extensionName protoreflect.FullName,
	messageTypeName protoreflect.FullName,
) (protoreflect.ExtensionTypeDescriptor, error) {
	var fileDescriptorSet descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(fileDescriptorSetData, &fileDescriptorSet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal descriptor set containing %s: %w", path, err)
	}
	var fileDescriptorProto *descriptorpb.FileDescriptorProto
	for _, file := range fileDescriptorSet.File {
		if file.GetName() == path {
			fileDescriptorProto = file
			break
		}
	}
	if fileDescriptorProto == nil {
		return nil, fmt.Errorf("file %s not found in descriptor set", path)
	}
	fileDescriptor, err := protodesc.NewFile(fileDescriptorProto, protoregistry.GlobalFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to process file descriptor for %s: %w", path, err)
	}
	if fileDescriptor.Package() != extensionName.Parent() {
		return nil, fmt.Errorf("file descriptor for %s does not contain %s", path, extensionName)
	}
	extension := fileDescriptor.Extensions().ByName(extensionName.Name())
	if extension == nil {
		return nil, fmt.Errorf("file descriptor for %s does not contain %s", path, extensionName)
	}
	if extension.Message() == nil || extension.Message().FullName() != messageTypeName {
		var actualType, expectedType string
		if extension.Message() != nil {
			actualType = string(extension.Message().FullName())
			expectedType = string(messageTypeName)
		} else {
			actualType = extension.Kind().String()
			expectedType = "message"
		}
		return nil, fmt.Errorf("file descriptor for %s contains extension %s with unexpected type: %s != %s",
			path, extensionName, actualType, expectedType)
	}
	return dynamicpb.NewExtensionType(extension).TypeDescriptor(), nil
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
