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

package buflintcheck

import (
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

// addFunc adds a FileAnnotation.
//
// descriptor is what the FileAnnotation applies to.
// location is the granular Location of the FileAnnotation.
// extraIgnoreLocations are extra Locations to check for comment ignores. Note that if descriptor is a
// bufprotosource.LocationDescriptor, descriptor.Location() is automatically added to extraIgnoreLocations if
// location != descriptor.Location().
//
// descriptor, location, and extraIgnoreLocations can be nil.
type addFunc func(descriptior bufprotosource.Descriptor, location bufprotosource.Location, extraIgnoreLocations []bufprotosource.Location, message string, args ...interface{})

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

// validLeadingComment returns true if comment has at least one line that isn't empty
// and doesn't start with CommentIgnorePrefix.
func validLeadingComment(comment string) bool {
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, CommentIgnorePrefix) {
			return true
		}
	}
	return false
}

// Returns the usedPackageList if there is an import cycle.
//
// Note this stops on the first import cycle detected, it doesn't attempt to get all of them - not perfect.
func getImportCycleIfExists(
	// Should never be ""
	pkg string,
	packageToDirectlyImportedPackageToFileImports map[string]map[string][]bufprotosource.FileImport,
	usedPackageMap map[string]struct{},
	usedPackageList []string,
) []string {
	// Append before checking so that the returned import cycle is actually a cycle
	usedPackageList = append(usedPackageList, pkg)
	if _, ok := usedPackageMap[pkg]; ok {
		// We have an import cycle, but if the first package in the list does not
		// equal the last, do not return as an import cycle unless the first
		// element equals the last - we do DFS from each package so this will
		// be picked up separately
		if usedPackageList[0] == usedPackageList[len(usedPackageList)-1] {
			return usedPackageList
		}
		return nil
	}
	usedPackageMap[pkg] = struct{}{}
	// Will never equal pkg
	for directlyImportedPackage := range packageToDirectlyImportedPackageToFileImports[pkg] {
		// Can equal "" per the function signature of PackageToDirectlyImportedPackageToFileImports
		if directlyImportedPackage == "" {
			continue
		}
		if importCycle := getImportCycleIfExists(
			directlyImportedPackage,
			packageToDirectlyImportedPackageToFileImports,
			usedPackageMap,
			usedPackageList,
		); len(importCycle) != 0 {
			return importCycle
		}
	}
	delete(usedPackageMap, pkg)
	return nil
}

// Our linters should not consider imports when linting. However, some linters require all files to
// perform their linting - for example, when recursively using the fullNameToMessage
// map. For those linters, use this helper, and make sure to explicitly skip
// linting of any files that are imports via bufprotosource.File.IsImport().
func newFilesWithImportsCheckFunc(
	f func(addFunc, []bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		helper := internal.NewHelper(id, ignoreFunc)
		if err := f(helper.AddFileAnnotationWithExtraIgnoreLocationsf, files); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newFilesCheckFunc(
	f func(addFunc, []bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		filesWithoutImports := make([]bufprotosource.File, 0, len(files))
		for _, file := range files {
			if !file.IsImport() {
				filesWithoutImports = append(filesWithoutImports, file)
			}
		}
		helper := internal.NewHelper(id, ignoreFunc)
		if err := f(helper.AddFileAnnotationWithExtraIgnoreLocationsf, filesWithoutImports); err != nil {
			return nil, err
		}
		return helper.FileAnnotations(), nil
	}
}

func newPackageToFilesCheckFunc(
	f func(add addFunc, pkg string, files []bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []bufprotosource.File) error {
			packageToFiles, err := bufprotosource.PackageToFiles(files...)
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
	f func(add addFunc, dirPath string, files []bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []bufprotosource.File) error {
			dirPathToFiles, err := bufprotosource.DirPathToFiles(files...)
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
	f func(addFunc, bufprotosource.File) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFilesCheckFunc(
		func(add addFunc, files []bufprotosource.File) error {
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
	f func(addFunc, bufprotosource.FileImport) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file bufprotosource.File) error {
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
	f func(addFunc, bufprotosource.Enum) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file bufprotosource.File) error {
			return bufprotosource.ForEachEnum(
				func(enum bufprotosource.Enum) error {
					return f(add, enum)
				},
				file,
			)
		},
	)
}

func newEnumValueCheckFunc(
	f func(addFunc, bufprotosource.EnumValue) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newEnumCheckFunc(
		func(add addFunc, enum bufprotosource.Enum) error {
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
	f func(addFunc, bufprotosource.Message) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file bufprotosource.File) error {
			return bufprotosource.ForEachMessage(
				func(message bufprotosource.Message) error {
					return f(add, message)
				},
				file,
			)
		},
	)
}

func newFieldCheckFunc(
	f func(addFunc, bufprotosource.Field) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return combine(
		newMessageCheckFunc(
			func(add addFunc, message bufprotosource.Message) error {
				for _, field := range message.Fields() {
					if err := f(add, field); err != nil {
						return err
					}
				}
				for _, field := range message.Extensions() {
					if err := f(add, field); err != nil {
						return err
					}
				}
				return nil
			},
		),
		newFileCheckFunc(
			func(add addFunc, file bufprotosource.File) error {
				for _, field := range file.Extensions() {
					if err := f(add, field); err != nil {
						return err
					}
				}
				return nil
			},
		),
	)
}

func newOneofCheckFunc(
	f func(addFunc, bufprotosource.Oneof) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newMessageCheckFunc(
		func(add addFunc, message bufprotosource.Message) error {
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
	f func(addFunc, bufprotosource.Service) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newFileCheckFunc(
		func(add addFunc, file bufprotosource.File) error {
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
	f func(addFunc, bufprotosource.Method) error,
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return newServiceCheckFunc(
		func(add addFunc, service bufprotosource.Service) error {
			for _, method := range service.Methods() {
				if err := f(add, method); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func combine(
	checks ...func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error),
) func(string, internal.IgnoreFunc, []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
	return func(id string, ignoreFunc internal.IgnoreFunc, files []bufprotosource.File) ([]bufanalysis.FileAnnotation, error) {
		var annotations []bufanalysis.FileAnnotation
		for _, check := range checks {
			checkAnnotations, err := check(id, ignoreFunc, files)
			if err != nil {
				return nil, err
			}
			annotations = append(annotations, checkAnnotations...)
		}
		return annotations, nil
	}
}
