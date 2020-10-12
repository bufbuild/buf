// Copyright 2020 Buf Technologies, Inc.
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

package buflintcheck

import (
	"strconv"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/protosource"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

// addFunc adds a FileAnnotation.
//
// Both the Descriptor and Location can be nil.
type addFunc func(protosource.Descriptor, protosource.Location, string, ...interface{})

func fieldToLowerSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return stringutil.ToLowerSnakeCase(s, stringutil.SnakeCaseWithNewWordOnDigits())
	return stringutil.ToLowerSnakeCase(s)
}

func fieldToUpperSnakeCase(s string) string {
	// Try running this on googleapis and watch
	// We allow both effectively by not passing the option
	//return stringutil.ToUpperSnakeCase(s, stringutil.SnakeCaseWithNewWordOnDigits())
	return stringutil.ToUpperSnakeCase(s)
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
	f func(addFunc, []protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, files []protosource.File) ([]bufanalysis.FileAnnotation, error) {
		helper := internal.NewHelper(id, ignoreFunc)
		if err := f(helper.AddFileAnnotationf, files); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newPackageToFilesCheckFunc(
	f func(add addFunc, pkg string, files []protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protosource.File) error {
			packageToFiles, err := protosource.PackageToFiles(files...)
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
	f func(add addFunc, dirPath string, files []protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protosource.File) error {
			dirPathToFiles, err := protosource.DirPathToFiles(files...)
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
	f func(addFunc, protosource.File) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []protosource.File) error {
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
	f func(addFunc, protosource.FileImport) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protosource.File) error {
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
	f func(addFunc, protosource.Enum) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protosource.File) error {
			return protosource.ForEachEnum(
				func(enum protosource.Enum) error {
					return f(add, enum)
				},
				file,
			)
		},
	)
}

func newEnumValueCheckFunc(
	f func(addFunc, protosource.EnumValue) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newEnumCheckFunc(
		func(add addFunc, enum protosource.Enum) error {
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
	f func(addFunc, protosource.Message) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protosource.File) error {
			return protosource.ForEachMessage(
				func(message protosource.Message) error {
					return f(add, message)
				},
				file,
			)
		},
	)
}

func newFieldCheckFunc(
	f func(addFunc, protosource.Field) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newMessageCheckFunc(
		func(add addFunc, message protosource.Message) error {
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
	f func(addFunc, protosource.Oneof) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newMessageCheckFunc(
		func(add addFunc, message protosource.Message) error {
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
	f func(addFunc, protosource.Service) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file protosource.File) error {
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
	f func(addFunc, protosource.Method) error,
) func(string, internal.IgnoreFunc, []protosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newServiceCheckFunc(
		func(add addFunc, service protosource.Service) error {
			for _, method := range service.Methods() {
				if err := f(add, method); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
