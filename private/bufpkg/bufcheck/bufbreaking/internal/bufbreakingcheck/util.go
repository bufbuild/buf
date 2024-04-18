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
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	cardinalitySingular cardinality = iota + 1
	cardinalityRepeated
	cardinalityMap

	fieldPresenceExplicit fieldPresence = iota + 1
	fieldPresenceImplicit
	fieldPresenceRequired
)

var (
	// https://developers.google.com/protocol-buffers/docs/proto3#updating
	fieldDescriptorProtoTypeToWireCompatiblityGroup = map[protoreflect.Kind]int{
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
	fieldDescriptorProtoTypeToWireJSONCompatiblityGroup = map[protoreflect.Kind]int{
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

type cardinality int

func (c cardinality) String() string {
	switch c {
	case cardinalitySingular:
		return "singular"
	case cardinalityRepeated:
		return "repeated"
	case cardinalityMap:
		return "map"
	default:
		return strconv.Itoa(int(c))
	}
}

func getCardinality(field protoreflect.FieldDescriptor) cardinality {
	switch {
	case field.IsList():
		return cardinalityRepeated
	case field.IsMap():
		return cardinalityMap
	default:
		return cardinalitySingular
	}
}

type fieldPresence int

func (f fieldPresence) String() string {
	switch f {
	case fieldPresenceExplicit:
		return "explicit"
	case fieldPresenceImplicit:
		return "implicit"
	case fieldPresenceRequired:
		return "required"
	default:
		return strconv.Itoa(int(f))
	}
}

func getFieldPresence(field protoreflect.FieldDescriptor) fieldPresence {
	switch {
	case field.Cardinality() == protoreflect.Required:
		return fieldPresenceRequired
	case field.HasPresence():
		return fieldPresenceExplicit
	default:
		return fieldPresenceImplicit
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
	return newMessagePairCheckFunc(
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

func getDescriptorAndLocationForDeletedEnum(file bufprotosource.File, previousNestedName string) (bufprotosource.Descriptor, bufprotosource.Location, error) {
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

func withBackupLocation(primary bufprotosource.Location, secondary bufprotosource.Location) bufprotosource.Location {
	if primary != nil {
		return primary
	}
	return secondary
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
	var kind string
	if field.Extendee() != "" {
		kind = "Extension"
	} else {
		kind = "Field"
	}
	return fmt.Sprintf("%s %q%s on message %q",
		kind, numberString, name, field.ParentMessage().Name())
}
