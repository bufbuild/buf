package internal

import (
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// addFunc adds an annotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(protodesc.Descriptor, protodesc.Location, string, ...interface{})

func newFilesCheckFunc(
	f func(addFunc, []protodesc.File, []protodesc.File) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return func(id string, previousFiles []protodesc.File, files []protodesc.File) ([]*analysis.Annotation, error) {
		helper := internal.NewHelper(id)
		if err := f(helper.AddAnnotationf, previousFiles, files); err != nil {
			return nil, err
		}
		return helper.Annotations(), nil
	}
}

func newFilePairCheckFunc(
	f func(addFunc, protodesc.File, protodesc.File) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
			previousFilePathToFile, err := protodesc.FilePathToFile(previousFiles...)
			if err != nil {
				return err
			}
			filePathToFile, err := protodesc.FilePathToFile(files...)
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
	f func(addFunc, protodesc.Enum, protodesc.Enum) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
			previousFullNameToEnum, err := protodesc.FullNameToEnum(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := protodesc.FullNameToEnum(files...)
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
	f func(addFunc, map[string]protodesc.EnumValue, map[string]protodesc.EnumValue) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newEnumPairCheckFunc(
		func(add addFunc, previousEnum protodesc.Enum, enum protodesc.Enum) error {
			previousNumberToNameToEnumValue, err := protodesc.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := protodesc.NumberToNameToEnumValue(enum)
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
	f func(addFunc, protodesc.Message, protodesc.Message) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
			previousFullNameToMessage, err := protodesc.FullNameToMessage(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := protodesc.FullNameToMessage(files...)
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
	f func(addFunc, protodesc.Field, protodesc.Field) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newMessagePairCheckFunc(
		func(add addFunc, previousMessage protodesc.Message, message protodesc.Message) error {
			previousNumberToField, err := protodesc.NumberToMessageField(previousMessage)
			if err != nil {
				return err
			}
			numberToField, err := protodesc.NumberToMessageField(message)
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
	f func(addFunc, protodesc.Service, protodesc.Service) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, previousFiles []protodesc.File, files []protodesc.File) error {
			previousFullNameToService, err := protodesc.FullNameToService(previousFiles...)
			if err != nil {
				return err
			}
			fullNameToService, err := protodesc.FullNameToService(files...)
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
	f func(addFunc, protodesc.Method, protodesc.Method) error,
) func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error) {
	return newServicePairCheckFunc(
		func(add addFunc, previousService protodesc.Service, service protodesc.Service) error {
			previousNameToMethod, err := protodesc.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := protodesc.NameToMethod(service)
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

func getDescriptorAndLocationForDeletedEnum(file protodesc.File, previousNestedName string) (protodesc.Descriptor, protodesc.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := protodesc.NestedNameToMessage(file)
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

func getDescriptorAndLocationForDeletedMessage(file protodesc.File, nestedNameToMessage map[string]protodesc.Message, previousNestedName string) (protodesc.Descriptor, protodesc.Location) {
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

func getSortedEnumValueNames(nameToEnumValue map[string]protodesc.EnumValue) []string {
	names := make([]string, 0, len(nameToEnumValue))
	for name := range nameToEnumValue {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func withBackupLocation(primary protodesc.Location, secondary protodesc.Location) protodesc.Location {
	if primary != nil {
		return primary
	}
	return secondary
}
