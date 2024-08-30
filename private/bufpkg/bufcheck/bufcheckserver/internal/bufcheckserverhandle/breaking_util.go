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

package bufcheckserverhandle

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufcheckserver/internal/bufcheckserverutil/customfeatures/customfeatures"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	featuresFieldName             = "features"
	featureNameUTF8Validation     = "utf8_validation"
	featureNameJSONFormat         = "json_format"
	cppFeatureNameStringType      = "string_type"
	javaFeatureNameUTF8Validation = "utf8_validation"
)

var (
	// https://developers.google.com/protocol-buffers/docs/proto3#updating
	fieldKindToWireCompatibilityGroup = map[protoreflect.Kind]int{
		protoreflect.Int32Kind:  1,
		protoreflect.Int64Kind:  1,
		protoreflect.Uint32Kind: 1,
		protoreflect.Uint64Kind: 1,
		protoreflect.BoolKind:   1,
		protoreflect.Sint32Kind: 2,
		protoreflect.Sint64Kind: 2,
		// While string and bytes are compatible if the bytes are valid UTF-8, we cannot
		// determine if a field will actually be valid UTF-8, as we are concerned with the
		// definitions and not individual messages, so we have these in different
		// compatibility groups. We allow string to evolve to bytes, but not bytes to
		// string, but we need them to be in different compatibility groups so that
		// we have to manually detect this.
		protoreflect.StringKind:   3,
		protoreflect.BytesKind:    4,
		protoreflect.Fixed32Kind:  5,
		protoreflect.Sfixed32Kind: 5,
		protoreflect.Fixed64Kind:  6,
		protoreflect.Sfixed64Kind: 6,
		protoreflect.DoubleKind:   7,
		protoreflect.FloatKind:    8,
		protoreflect.GroupKind:    9,
		// Embedded messages are compatible with bytes if the bytes are serialized versions
		// of the message, but we have no way of verifying this.
		protoreflect.MessageKind: 10,
		// Enum is compatible with int32, uint32, int64, uint64 if the values match
		// an enum value, but we have no way of verifying this.
		protoreflect.EnumKind: 11,
	}

	// httpsKind://developers.google.com/protocol-buffers/docs/proto3#json
	// this is not just JSON-compatible, but also wire-compatible, i.e. the intersection
	fieldKindToWireJSONCompatibilityGroup = map[protoreflect.Kind]int{
		// fixed32 not compatible for wire so not included
		protoreflect.Int32Kind:  1,
		protoreflect.Uint32Kind: 1,
		// fixed64 not compatible for wire so not included
		protoreflect.Int64Kind:    2,
		protoreflect.Uint64Kind:   2,
		protoreflect.Fixed32Kind:  3,
		protoreflect.Sfixed32Kind: 3,
		protoreflect.Fixed64Kind:  4,
		protoreflect.Sfixed64Kind: 4,
		protoreflect.BoolKind:     5,
		protoreflect.Sint32Kind:   6,
		protoreflect.Sint64Kind:   7,
		protoreflect.StringKind:   8,
		protoreflect.BytesKind:    9,
		protoreflect.DoubleKind:   10,
		protoreflect.FloatKind:    11,
		protoreflect.GroupKind:    12,
		protoreflect.MessageKind:  13,
		protoreflect.EnumKind:     14,
	}
)

func fieldDescriptorTypePrettyString(descriptor protoreflect.FieldDescriptor) string {
	if descriptor.Kind() == protoreflect.GroupKind && descriptor.Syntax() != protoreflect.Proto2 {
		// Kind will be set to "group", but it's really a "delimited-encoded message"
		return "message (delimited encoding)"
	}
	return descriptor.Kind().String()
}

func getDescriptorAndLocationForDeletedElement(
	file bufprotosource.File,
	previousNestedName string,
) (bufprotosource.Descriptor, bufprotosource.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
		if err != nil {
			return nil, nil, err
		}
		split := strings.Split(previousNestedName, ".")
		for i := len(split) - 1; i > 0; i-- {
			if message, ok := nestedNameToMessage[strings.Join(split[0:i], ".")]; ok {
				return message, message.Location(), nil
			}
		}
	}
	return file, nil, nil
}

func getDescriptorAndLocationForDeletedMessage(
	file bufprotosource.File,
	nestedNameToMessage map[string]bufprotosource.Message,
	previousNestedName string,
) (bufprotosource.Descriptor, bufprotosource.Location) {
	if strings.Contains(previousNestedName, ".") {
		split := strings.Split(previousNestedName, ".")
		for i := len(split) - 1; i > 0; i-- {
			if message, ok := nestedNameToMessage[strings.Join(split[0:i], ".")]; ok {
				return message, message.Location()
			}
		}
	}
	return file, nil
}

func getSortedEnumValueNames(nameToEnumValue map[string]bufprotosource.EnumValue) []string {
	names := make([]string, 0, len(nameToEnumValue))
	for name := range nameToEnumValue {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getEnumByFullName(files []bufprotosource.File, enumFullName string) (bufprotosource.Enum, error) {
	fullNameToEnum, err := bufprotosource.FullNameToEnum(files...)
	if err != nil {
		return nil, err
	}
	enum, ok := fullNameToEnum[enumFullName]
	if !ok {
		return nil, fmt.Errorf("expected enum %q to exist but was not found", enumFullName)
	}
	return enum, nil
}

func withBackupLocation(locs ...bufprotosource.Location) bufprotosource.Location {
	for _, loc := range locs {
		if loc != nil {
			return loc
		}
	}
	return nil
}

func findFeatureField(name protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.FieldDescriptor, error) {
	featureSetDescriptor := (*descriptorpb.FeatureSet)(nil).ProtoReflect().Descriptor()
	featureField := featureSetDescriptor.Fields().ByName(name)
	if featureField == nil {
		return nil, fmt.Errorf("unable to resolve field descriptor for %s.%s", featureSetDescriptor.FullName(), name)
	}
	if featureField.Kind() != expectedKind || featureField.IsList() {
		return nil, fmt.Errorf("resolved field descriptor for %s.%s has unexpected type: expected optional %s, got %s %s",
			featureSetDescriptor.FullName(), name, expectedKind, featureField.Cardinality(), featureField.Kind())
	}
	return featureField, nil
}

func fieldCppStringType(field bufprotosource.Field, descriptor protoreflect.FieldDescriptor) (protobuf.CppFeatures_StringType, bool, error) {
	// We don't support Edition 2024 yet. But we know of this rule, so we can go ahead and
	// implement it so it's one less thing to do when we DO add support for 2024.
	if field.File().Edition() < descriptorpb.Edition_EDITION_2024 {
		opts, _ := descriptor.Options().(*descriptorpb.FieldOptions)
		// TODO: In Edition 2024, it will be *required* to use the new (pb.cpp).string_type option. So
		//       we shouldn't bother checking the ctype option in editions >= 2024.
		if opts != nil && opts.Ctype != nil {
			switch opts.GetCtype() {
			case descriptorpb.FieldOptions_CORD:
				return protobuf.CppFeatures_CORD, false, nil
			case descriptorpb.FieldOptions_STRING_PIECE:
				return protobuf.CppFeatures_STRING, true, nil
			case descriptorpb.FieldOptions_STRING:
				return protobuf.CppFeatures_STRING, false, nil
			default:
				if descriptor.ParentFile().Syntax() != protoreflect.Editions {
					return protobuf.CppFeatures_STRING, false, nil
				}
				// If the file is edition 2023, we fall through to below since 2023 allows either
				// the ctype field or the (pb.cpp).string_type feature.
			}
		}
	}
	val, err := customfeatures.ResolveCppFeature(descriptor, cppFeatureNameStringType, protoreflect.EnumKind)
	if err != nil {
		return 0, false, err
	}
	return protobuf.CppFeatures_StringType(val.Enum()), false, nil
}

func fieldCppStringTypeLocation(field bufprotosource.Field) bufprotosource.Location {
	ext := protobuf.E_Cpp.TypeDescriptor()
	if ext.Message() == nil {
		return nil
	}
	return getCustomFeatureLocation(field, ext, cppFeatureNameStringType)
}

func fieldJavaUTF8Validation(field protoreflect.FieldDescriptor) (descriptorpb.FeatureSet_Utf8Validation, error) {
	standardFeatureField, err := findFeatureField(featureNameUTF8Validation, protoreflect.EnumKind)
	if err != nil {
		return 0, err
	}
	val, err := protoutil.ResolveFeature(field, standardFeatureField)
	if err != nil {
		return 0, fmt.Errorf("unable to resolve value of %s feature: %w", standardFeatureField.Name(), err)
	}
	defaultValue := descriptorpb.FeatureSet_Utf8Validation(val.Enum())

	opts, _ := field.ParentFile().Options().(*descriptorpb.FileOptions)
	if field.ParentFile().Syntax() != protoreflect.Editions || (opts != nil && opts.JavaStringCheckUtf8 != nil) {
		if opts.GetJavaStringCheckUtf8() {
			return descriptorpb.FeatureSet_VERIFY, nil
		}
		return defaultValue, nil
	}

	val, err = customfeatures.ResolveJavaFeature(field, javaFeatureNameUTF8Validation, protoreflect.EnumKind)
	if err != nil {
		return 0, err
	}
	if protobuf.JavaFeatures_Utf8Validation(val.Enum()) == protobuf.JavaFeatures_VERIFY {
		return descriptorpb.FeatureSet_VERIFY, nil
	}
	return defaultValue, nil
}

func fieldJavaUTF8ValidationLocation(field bufprotosource.Field) bufprotosource.Location {
	ext := protobuf.E_Java.TypeDescriptor()
	if ext.Message() == nil {
		return nil
	}
	return getCustomFeatureLocation(field, ext, javaFeatureNameUTF8Validation)
}

func getCustomFeatureLocation(field bufprotosource.Field, extension protoreflect.ExtensionTypeDescriptor, fieldName protoreflect.Name) bufprotosource.Location {
	if extension.Message() == nil {
		return nil
	}
	feature := extension.Message().Fields().ByName(fieldName)
	if feature == nil {
		return nil
	}
	featureField := (*descriptorpb.FieldOptions)(nil).ProtoReflect().Descriptor().Fields().ByName(featuresFieldName)
	if featureField == nil {
		// should not be possible
		return nil
	}
	return field.OptionLocation(featureField, int32(extension.Number()), int32(feature.Number()))
}

func fieldDescription(field bufprotosource.Field) string {
	var name string
	if field.Extendee() != "" {
		// extensions are known by fully-qualified name
		name = field.FullName()
	} else {
		name = field.Name()
	}
	return fieldDescriptionWithName(field, name)
}

func fieldDescriptionWithName(field bufprotosource.Field, name string) string {
	if name != "" {
		name = fmt.Sprintf(" with name %q", name)
	}
	// otherwise prints as hex
	numberString := strconv.FormatInt(int64(field.Number()), 10)
	var kind, message string
	if field.Extendee() != "" {
		kind = "Extension"
		message = field.Extendee()
	} else {
		kind = "Field"
		message = field.ParentMessage().Name()
	}
	return fmt.Sprintf("%s %q%s on message %q", kind, numberString, name, message)
}

func is64bitInteger(fieldType descriptorpb.FieldDescriptorProto_Type) bool {
	switch fieldType {
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		return true
	default:
		return false
	}
}
