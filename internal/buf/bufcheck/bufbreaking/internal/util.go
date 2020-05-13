// Copyright 2020 Buf Technologies Inc.
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

package internal

import (
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/proto/protosrc"
)

// addFunc adds a FileAnnotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(protosrc.Descriptor, protosrc.Location, string, ...interface{})

func newFilesCheckFunc(
	f func(addFunc, []protosrc.File, []protosrc.File) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return func(id string, previousFiles []protosrc.File, files []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
		helper := internal.NewHelper(id)
		if err := f(helper.AddFileAnnotationf, previousFiles, files); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newFilePairCheckFunc(
	f func(addFunc, protosrc.File, protosrc.File) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosrc.File, files []protosrc.File) error {
			previousFilePathToFile, err := protosrc.FilePathToFile(previousFiles...)
			if err != nil {
				return err
			}
			filePathToFile, err := protosrc.FilePathToFile(files...)
			if err != nil {
				return err
			}
			for previousFilePath, previousFile := range previousFilePathToFile {
				if file, ok := filePathToFile[previousFilePath]; ok {
					if err := f(add, previousFile, file); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newEnumPairCheckFunc(
	f func(addFunc, protosrc.Enum, protosrc.Enum) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosrc.File, files []protosrc.File) error {
			previousFullNameToEnum, err := protosrc.FullNameToEnum(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := protosrc.FullNameToEnum(files...)
			if err != nil {
				return err
			}
			for previousFullName, previousEnum := range previousFullNameToEnum {
				if enum, ok := fullNameToEnum[previousFullName]; ok {
					if err := f(add, previousEnum, enum); err != nil {
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
	f func(addFunc, map[string]protosrc.EnumValue, map[string]protosrc.EnumValue) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newEnumPairCheckFunc(
		func(add addFunc, previousEnum protosrc.Enum, enum protosrc.Enum) error {
			previousNumberToNameToEnumValue, err := protosrc.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := protosrc.NumberToNameToEnumValue(enum)
			if err != nil {
				return err
			}
			for previousNumber, previousNameToEnumValue := range previousNumberToNameToEnumValue {
				if nameToEnumValue, ok := numberToNameToEnumValue[previousNumber]; ok {
					if err := f(add, previousNameToEnumValue, nameToEnumValue); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newMessagePairCheckFunc(
	f func(addFunc, protosrc.Message, protosrc.Message) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosrc.File, files []protosrc.File) error {
			previousFullNameToMessage, err := protosrc.FullNameToMessage(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := protosrc.FullNameToMessage(files...)
			if err != nil {
				return err
			}
			for previousFullName, previousMessage := range previousFullNameToMessage {
				if message, ok := fullNameToMessage[previousFullName]; ok {
					if err := f(add, previousMessage, message); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newFieldPairCheckFunc(
	f func(addFunc, protosrc.Field, protosrc.Field) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newMessagePairCheckFunc(
		func(add addFunc, previousMessage protosrc.Message, message protosrc.Message) error {
			previousNumberToField, err := protosrc.NumberToMessageField(previousMessage)
			if err != nil {
				return err
			}
			numberToField, err := protosrc.NumberToMessageField(message)
			if err != nil {
				return err
			}
			for previousNumber, previousField := range previousNumberToField {
				if field, ok := numberToField[previousNumber]; ok {
					if err := f(add, previousField, field); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newServicePairCheckFunc(
	f func(addFunc, protosrc.Service, protosrc.Service) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosrc.File, files []protosrc.File) error {
			previousFullNameToService, err := protosrc.FullNameToService(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToService, err := protosrc.FullNameToService(files...)
			if err != nil {
				return err
			}
			for previousFullName, previousService := range previousFullNameToService {
				if service, ok := fullNameToService[previousFullName]; ok {
					if err := f(add, previousService, service); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func newMethodPairCheckFunc(
	f func(addFunc, protosrc.Method, protosrc.Method) error,
) func(string, []protosrc.File, []protosrc.File) ([]*filev1beta1.FileAnnotation, error) {
	return newServicePairCheckFunc(
		func(add addFunc, previousService protosrc.Service, service protosrc.Service) error {
			previousNameToMethod, err := protosrc.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := protosrc.NameToMethod(service)
			if err != nil {
				return err
			}
			for previousName, previousMethod := range previousNameToMethod {
				if method, ok := nameToMethod[previousName]; ok {
					if err := f(add, previousMethod, method); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

func getDescriptorAndLocationForDeletedEnum(file protosrc.File, previousNestedName string) (protosrc.Descriptor, protosrc.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := protosrc.NestedNameToMessage(file)
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

func getDescriptorAndLocationForDeletedMessage(file protosrc.File, nestedNameToMessage map[string]protosrc.Message, previousNestedName string) (protosrc.Descriptor, protosrc.Location) {
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

func getSortedEnumValueNames(nameToEnumValue map[string]protosrc.EnumValue) []string {
	names := make([]string, 0, len(nameToEnumValue))
	for name := range nameToEnumValue {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func withBackupLocation(primary protosrc.Location, secondary protosrc.Location) protosrc.Location {
	if primary != nil {
		return primary
	}
	return secondary
}
