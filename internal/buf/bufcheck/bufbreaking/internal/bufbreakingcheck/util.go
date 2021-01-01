// Copyright 2020-2021 Buf Technologies, Inc.
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
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/protosource"
)

// addFunc adds a FileAnnotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(protosource.Descriptor, protosource.Location, string, ...interface{})

func newFilesCheckFunc(
	f func(addFunc, []protosource.File, []protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, previousFiles []protosource.File, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
		helper := internal.NewHelper(id, ignoreFunc)
		if err := f(helper.AddFileAnnotationf, previousFiles, files); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newFilePairCheckFunc(
	f func(addFunc, protosource.File, protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosource.File, files []protosource.File) error {
			previousFilePathToFile, err := protosource.FilePathToFile(previousFiles...)
			if err != nil {
				return err
			}
			filePathToFile, err := protosource.FilePathToFile(files...)
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
	f func(addFunc, protosource.Enum, protosource.Enum) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosource.File, files []protosource.File) error {
			previousFullNameToEnum, err := protosource.FullNameToEnum(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := protosource.FullNameToEnum(files...)
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
	f func(addFunc, map[string]protosource.EnumValue, map[string]protosource.EnumValue) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newEnumPairCheckFunc(
		func(add addFunc, previousEnum protosource.Enum, enum protosource.Enum) error {
			previousNumberToNameToEnumValue, err := protosource.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := protosource.NumberToNameToEnumValue(enum)
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
	f func(addFunc, protosource.Message, protosource.Message) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosource.File, files []protosource.File) error {
			previousFullNameToMessage, err := protosource.FullNameToMessage(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := protosource.FullNameToMessage(files...)
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
	f func(addFunc, protosource.Field, protosource.Field) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newMessagePairCheckFunc(
		func(add addFunc, previousMessage protosource.Message, message protosource.Message) error {
			previousNumberToField, err := protosource.NumberToMessageField(previousMessage)
			if err != nil {
				return err
			}
			numberToField, err := protosource.NumberToMessageField(message)
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
	f func(addFunc, protosource.Service, protosource.Service) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protosource.File, files []protosource.File) error {
			previousFullNameToService, err := protosource.FullNameToService(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToService, err := protosource.FullNameToService(files...)
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
	f func(addFunc, protosource.Method, protosource.Method) error,
) func(string, internal.IgnoreFunc, []protosource.File, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newServicePairCheckFunc(
		func(add addFunc, previousService protosource.Service, service protosource.Service) error {
			previousNameToMethod, err := protosource.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := protosource.NameToMethod(service)
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

func getDescriptorAndLocationForDeletedEnum(file protosource.File, previousNestedName string) (protosource.Descriptor, protosource.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := protosource.NestedNameToMessage(file)
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

func getDescriptorAndLocationForDeletedMessage(file protosource.File, nestedNameToMessage map[string]protosource.Message, previousNestedName string) (protosource.Descriptor, protosource.Location) {
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

func getSortedEnumValueNames(nameToEnumValue map[string]protosource.EnumValue) []string {
	names := make([]string, 0, len(nameToEnumValue))
	for name := range nameToEnumValue {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func withBackupLocation(primary protosource.Location, secondary protosource.Location) protosource.Location {
	if primary != nil {
		return primary
	}
	return secondary
}
