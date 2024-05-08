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

package bufbreakingcheck

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/bufbreaking/internal/bufbreakingcheck/customfeatures"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// https://developers.google.com/protocol-buffers/docs/proto3#updating
	fieldKindToWireCompatiblityGroup = map[protoreflect.Kind]int{
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
	fieldKindToWireJSONCompatiblityGroup = map[protoreflect.Kind]int{
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

// addFunc adds a FileAnnotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(bufprotosource.Descriptor, []bufprotosource.Descriptor, bufprotosource.Location, string, ...interface{})

// corpus is a store of the previous files and files given to a check function.
//
// this is passed down so that pair functions have access to the original inputs.
type corpus struct {
	previousFiles []bufprotosource.File
	files         []bufprotosource.File
}

func newCorpus(
	previousFiles []bufprotosource.File,
	files []bufprotosource.File,
) *corpus {
	return &corpus{
		previousFiles: previousFiles,
		files:         files,
	}
}

func fieldDescriptorTypePrettyString(descriptor protoreflect.FieldDescriptor) string {
	if descriptor.Kind() == protoreflect.GroupKind && descriptor.Syntax() != protoreflect.Proto2 {
		// Kind will be set to "group", but it's really a "delimited-encoded message"
		return "message (delimited encoding)"
	}
	return descriptor.Kind().String()
}

func newFilesCheckFunc(
	f func(addFunc, *corpus) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, previousFiles []bufprotosource.File, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		helper := internal.NewHelper(id, ignoreFunc)
		if err := f(helper.AddFileAnnotationWithExtraIgnoreDescriptorsf, newCorpus(previousFiles, files)); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newFilePairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.File, bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, corpus *corpus) error {
			previousFilePathToFile, err := bufprotosource.FilePathToFile(corpus.previousFiles...)
			if err != nil {
				return err
			}
			filePathToFile, err := bufprotosource.FilePathToFile(corpus.files...)
			if err != nil {
				return err
			}
			for previousFilePath, previousFile := range previousFilePathToFile {
				if file, ok := filePathToFile[previousFilePath]; ok {
					if err := f(add, corpus, previousFile, file); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newEnumPairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Enum, bufprotosource.Enum) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, corpus *corpus) error {
			previousFullNameToEnum, err := bufprotosource.FullNameToEnum(corpus.previousFiles...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := bufprotosource.FullNameToEnum(corpus.files...)
			if err != nil {
				return err
			}
			for previousFullName, previousEnum := range previousFullNameToEnum {
				if enum, ok := fullNameToEnum[previousFullName]; ok {
					if err := f(add, corpus, previousEnum, enum); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// compares all the enums that are of the same number
// map is from name to EnumValue for the given number
func newEnumValuePairCheckFunc(
	f func(addFunc, *corpus, map[string]bufprotosource.EnumValue, map[string]bufprotosource.EnumValue) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newEnumPairCheckFunc(
		func(add addFunc, corpus *corpus, previousEnum bufprotosource.Enum, enum bufprotosource.Enum) error {
			previousNumberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(enum)
			if err != nil {
				return err
			}
			for previousNumber, previousNameToEnumValue := range previousNumberToNameToEnumValue {
				if nameToEnumValue, ok := numberToNameToEnumValue[previousNumber]; ok {
					if err := f(add, corpus, previousNameToEnumValue, nameToEnumValue); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newMessagePairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Message, bufprotosource.Message) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, corpus *corpus) error {
			previousFullNameToMessage, err := bufprotosource.FullNameToMessage(corpus.previousFiles...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := bufprotosource.FullNameToMessage(corpus.files...)
			if err != nil {
				return err
			}
			for previousFullName, previousMessage := range previousFullNameToMessage {
				if message, ok := fullNameToMessage[previousFullName]; ok {
					if err := f(add, corpus, previousMessage, message); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newFieldPairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Field, bufprotosource.Field) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return combine(
		// Regular fields
		newMessagePairCheckFunc(
			func(add addFunc, corpus *corpus, previousMessage bufprotosource.Message, message bufprotosource.Message) error {
				previousNumberToField, err := bufprotosource.NumberToMessageField(previousMessage)
				if err != nil {
					return err
				}
				numberToField, err := bufprotosource.NumberToMessageField(message)
				if err != nil {
					return err
				}
				for previousNumber, previousField := range previousNumberToField {
					if field, ok := numberToField[previousNumber]; ok {
						if err := f(add, corpus, previousField, field); err != nil {
							return err
						}
					}
				}
				return nil
			},
		),
		// And extension fields
		newFilesCheckFunc(
			func(add addFunc, corpus *corpus) error {
				previousTypeToNumberToField := make(map[string]map[int]bufprotosource.Field)
				for _, previousFile := range corpus.previousFiles {
					if err := addToTypeToNumberToExtension(previousFile, previousTypeToNumberToField); err != nil {
						return err
					}
				}
				typeToNumberToField := make(map[string]map[int]bufprotosource.Field)
				for _, file := range corpus.files {
					if err := addToTypeToNumberToExtension(file, typeToNumberToField); err != nil {
						return err
					}
				}
				for previousType, previousNumberToField := range previousTypeToNumberToField {
					numberToField := typeToNumberToField[previousType]
					for previousNumber, previousField := range previousNumberToField {
						if field, ok := numberToField[previousNumber]; ok {
							if err := f(add, corpus, previousField, field); err != nil {
								return err
							}
						}
					}
				}
				return nil
			},
		),
	)
}

func newFieldDescriptorPairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Field, protoreflect.FieldDescriptor, bufprotosource.Field, protoreflect.FieldDescriptor) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFieldPairCheckFunc(
		func(add addFunc, corpus *corpus, previousField bufprotosource.Field, field bufprotosource.Field) error {
			previousDescriptor, err := previousField.AsDescriptor()
			if err != nil {
				return err
			}
			descriptor, err := field.AsDescriptor()
			if err != nil {
				return err
			}
			return f(add, corpus, previousField, previousDescriptor, field, descriptor)
		},
	)
}

func newServicePairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Service, bufprotosource.Service) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, corpus *corpus) error {
			previousFullNameToService, err := bufprotosource.FullNameToService(corpus.previousFiles...)
			if err != nil {
				return err
			}
			fullNameToService, err := bufprotosource.FullNameToService(corpus.files...)
			if err != nil {
				return err
			}
			for previousFullName, previousService := range previousFullNameToService {
				if service, ok := fullNameToService[previousFullName]; ok {
					if err := f(add, corpus, previousService, service); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newMethodPairCheckFunc(
	f func(addFunc, *corpus, bufprotosource.Method, bufprotosource.Method) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newServicePairCheckFunc(
		func(add addFunc, corpus *corpus, previousService bufprotosource.Service, service bufprotosource.Service) error {
			previousNameToMethod, err := bufprotosource.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := bufprotosource.NameToMethod(service)
			if err != nil {
				return err
			}
			for previousName, previousMethod := range previousNameToMethod {
				if method, ok := nameToMethod[previousName]; ok {
					if err := f(add, corpus, previousMethod, method); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func combine(
	checks ...func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error),
) func(string, internal.IgnoreFunc, []bufprotosource.File, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, previousFiles, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		var annotations []bufanalysis.FileAnnotation
		for _, check := range checks {
			checkAnnotations, err := check(id, ignoreFunc, previousFiles, files)
			if err != nil {
				return nil, err
			}
			annotations = append(annotations, checkAnnotations...)
		}
		return annotations, nil
	}
}

func getDescriptorAndLocationForDeletedElement(file bufprotosource.File, previousNestedName string) (bufprotosource.Descriptor, bufprotosource.Location, error) {
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

func getDescriptorAndLocationForDeletedMessage(file bufprotosource.File, nestedNameToMessage map[string]bufprotosource.Message, previousNestedName string) (bufprotosource.Descriptor, bufprotosource.Location) {
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

func addToTypeToNumberToExtension(container bufprotosource.ContainerDescriptor, typeToNumberToExt map[string]map[int]bufprotosource.Field) error {
	for _, extension := range container.Extensions() {
		numberToExt := typeToNumberToExt[extension.Extendee()]
		if numberToExt == nil {
			numberToExt = make(map[int]bufprotosource.Field)
			typeToNumberToExt[extension.Extendee()] = numberToExt
		}
		if existing, ok := numberToExt[extension.Number()]; ok {
			return fmt.Errorf("duplicate extension %d of %s: %s in %q and %s in %q",
				extension.Number(), extension.Extendee(),
				existing.FullName(), existing.File().Path(),
				extension.FullName(), extension.File().Path())
		}
		numberToExt[extension.Number()] = extension
	}
	for _, message := range container.Messages() {
		if err := addToTypeToNumberToExtension(message, typeToNumberToExt); err != nil {
			return err
		}
	}
	return nil
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
