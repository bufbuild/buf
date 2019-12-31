package internal

import (
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
)

// addFunc adds an annotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(protodesc.Descriptor, protodesc.Location, string, ...interface{})

func fieldToLowerSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return utilstring.ToLowerSnakeCase(s, utilstring.SnakeCaseWithNewWordOnDigits())
	return utilstring.ToLowerSnakeCase(s)
}

func fieldToUpperSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return utilstring.ToUpperSnakeCase(s, utilstring.SnakeCaseWithNewWordOnDigits())
	return utilstring.ToUpperSnakeCase(s)
}

// https://cloud.google.com/apis/design/versioning
//
// All Proto Package values pass.
//
// v1test can be v1test.*
// v1p1alpha1 is also valid in addition to v1p1beta1
func packageHasVersionSuffix(pkg string) bool {
	if pkg == "" {
		return false
	}
	parts := strings.Split(pkg, ".")
	if len(parts) < 2 {
		return false
	}
	lastPart := parts[len(parts)-1]
	if len(lastPart) < 2 {
		return false
	}
	if lastPart[0] != 'v' {
		return false
	}
	version := lastPart[1:]
	if strings.Contains(version, "test") {
		split := strings.SplitN(version, "test", 2)
		if len(split) != 2 {
			return false
		}
		return stringIsPositiveNumber(split[0])
	}
	if strings.Contains(version, "alpha") {
		return packageVersionIsValidAlphaOrBeta(version, "alpha")
	}
	if strings.Contains(version, "beta") {
		return packageVersionIsValidAlphaOrBeta(version, "beta")
	}
	return stringIsPositiveNumber(version)
}

func packageVersionIsValidAlphaOrBeta(version string, name string) bool {
	split := strings.SplitN(version, name, 2)
	if len(split) != 2 {
		return false
	}
	if strings.Contains(split[0], "p") {
		patchSplit := strings.SplitN(split[0], "p", 2)
		if len(patchSplit) != 2 {
			return false
		}
		if !stringIsPositiveNumber(patchSplit[0]) || !stringIsPositiveNumber(patchSplit[1]) {
			return false
		}
	} else {
		if !stringIsPositiveNumber(split[0]) {
			return false
		}
	}
	return stringIsPositiveNumber(split[1])
}

func stringIsPositiveNumber(s string) bool {
	if s == "" {
		return false
	}
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return false
	}
	return value > 0
}

func newFilesCheckFunc(
	f func(addFunc, []protodesc.File) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return func(id string, files []protodesc.File) ([]*analysis.Annotation, error) {
		helper := internal.NewHelper(id)
		if err := f(helper.AddAnnotationf, files); err != nil {
			return nil, err
		}
		return helper.Annotations(), nil
	}
}

func newPackageToFilesCheckFunc(
	f func(add addFunc, pkg string, files []protodesc.File) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protodesc.File) error {
			packageToFiles, err := protodesc.PackageToFiles(files...)
			if err != nil {
				return err
			}
			for pkg, files := range packageToFiles {
				if err := f(add, pkg, files); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newDirToFilesCheckFunc(
	f func(add addFunc, dirPath string, files []protodesc.File) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protodesc.File) error {
			dirPathToFiles, err := protodesc.DirPathToFiles(files...)
			if err != nil {
				return err
			}
			for dirPath, files := range dirPathToFiles {
				if err := f(add, dirPath, files); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newFileCheckFunc(
	f func(addFunc, protodesc.File) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protodesc.File) error {
			for _, file := range files {
				if err := f(add, file); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newFileImportCheckFunc(
	f func(addFunc, protodesc.FileImport) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protodesc.File) error {
			for _, fileImport := range file.FileImports() {
				if err := f(add, fileImport); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newEnumCheckFunc(
	f func(addFunc, protodesc.Enum) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protodesc.File) error {
			return protodesc.ForEachEnum(
				func(enum protodesc.Enum) error {
					return f(add, enum)
				},
				file,
			)
		},
	)
}

func newEnumValueCheckFunc(
	f func(addFunc, protodesc.EnumValue) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newEnumCheckFunc(
		func(add addFunc, enum protodesc.Enum) error {
			for _, enumValue := range enum.Values() {
				if err := f(add, enumValue); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newMessageCheckFunc(
	f func(addFunc, protodesc.Message) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protodesc.File) error {
			return protodesc.ForEachMessage(
				func(message protodesc.Message) error {
					return f(add, message)
				},
				file,
			)
		},
	)
}

func newFieldCheckFunc(
	f func(addFunc, protodesc.Field) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newMessageCheckFunc(
		func(add addFunc, message protodesc.Message) error {
			for _, field := range message.Fields() {
				if err := f(add, field); err != nil {
					return err
				}
			}
			// TODO: is this right?
			for _, field := range message.Extensions() {
				if err := f(add, field); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newOneofCheckFunc(
	f func(addFunc, protodesc.Oneof) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newMessageCheckFunc(
		func(add addFunc, message protodesc.Message) error {
			for _, oneof := range message.Oneofs() {
				if err := f(add, oneof); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newServiceCheckFunc(
	f func(addFunc, protodesc.Service) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protodesc.File) error {
			for _, service := range file.Services() {
				if err := f(add, service); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func newMethodCheckFunc(
	f func(addFunc, protodesc.Method) error,
) func(string, []protodesc.File) ([]*analysis.Annotation, error) {
	return newServiceCheckFunc(
		func(add addFunc, service protodesc.Service) error {
			for _, method := range service.Methods() {
				if err := f(add, method); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
