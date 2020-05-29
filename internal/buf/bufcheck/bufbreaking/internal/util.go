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

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/buf/bufsrc"
)

// addFunc adds a FileAnnotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(bufsrc.Descriptor, bufsrc.Location, string, ...interface{})

func newFilesCheckFunc(
	f func(addFunc, []bufsrc.File, []bufsrc.File) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, previousFiles []bufsrc.File, files []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
		helper := internal.NewHelper(id)
		if err := f(helper.AddFileAnnotationf, previousFiles, files); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newFilePairCheckFunc(
	f func(addFunc, bufsrc.File, bufsrc.File) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []bufsrc.File, files []bufsrc.File) error {
			previousFilePathToFile, err := bufsrc.FilePathToFile(previousFiles...)
			if err != nil {
				return err
			}
			filePathToFile, err := bufsrc.FilePathToFile(files...)
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
	f func(addFunc, bufsrc.Enum, bufsrc.Enum) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []bufsrc.File, files []bufsrc.File) error {
			previousFullNameToEnum, err := bufsrc.FullNameToEnum(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := bufsrc.FullNameToEnum(files...)
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
	f func(addFunc, map[string]bufsrc.EnumValue, map[string]bufsrc.EnumValue) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newEnumPairCheckFunc(
		func(add addFunc, previousEnum bufsrc.Enum, enum bufsrc.Enum) error {
			previousNumberToNameToEnumValue, err := bufsrc.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := bufsrc.NumberToNameToEnumValue(enum)
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
	f func(addFunc, bufsrc.Message, bufsrc.Message) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []bufsrc.File, files []bufsrc.File) error {
			previousFullNameToMessage, err := bufsrc.FullNameToMessage(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := bufsrc.FullNameToMessage(files...)
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
	f func(addFunc, bufsrc.Field, bufsrc.Field) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newMessagePairCheckFunc(
		func(add addFunc, previousMessage bufsrc.Message, message bufsrc.Message) error {
			previousNumberToField, err := bufsrc.NumberToMessageField(previousMessage)
			if err != nil {
				return err
			}
			numberToField, err := bufsrc.NumberToMessageField(message)
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
	f func(addFunc, bufsrc.Service, bufsrc.Service) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []bufsrc.File, files []bufsrc.File) error {
			previousFullNameToService, err := bufsrc.FullNameToService(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToService, err := bufsrc.FullNameToService(files...)
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
	f func(addFunc, bufsrc.Method, bufsrc.Method) error,
) func(string, []bufsrc.File, []bufsrc.File) ([]bufanalysis.FileAnnotation, error) {
	return newServicePairCheckFunc(
		func(add addFunc, previousService bufsrc.Service, service bufsrc.Service) error {
			previousNameToMethod, err := bufsrc.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := bufsrc.NameToMethod(service)
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

func getDescriptorAndLocationForDeletedEnum(file bufsrc.File, previousNestedName string) (bufsrc.Descriptor, bufsrc.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := bufsrc.NestedNameToMessage(file)
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

func getDescriptorAndLocationForDeletedMessage(file bufsrc.File, nestedNameToMessage map[string]bufsrc.Message, previousNestedName string) (bufsrc.Descriptor, bufsrc.Location) {
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

func getSortedEnumValueNames(nameToEnumValue map[string]bufsrc.EnumValue) []string {
	names := make([]string, 0, len(nameToEnumValue))
	for name := range nameToEnumValue {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func withBackupLocation(primary bufsrc.Location, secondary bufsrc.Location) bufsrc.Location {
	if primary != nil {
		return primary
	}
	return secondary
}
