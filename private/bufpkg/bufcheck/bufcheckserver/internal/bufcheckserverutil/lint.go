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

package bufcheckserverutil

import (
	"context"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

// NewLintFilesRuleHandler returns a new check.RuleHandler for the given function.
//
// The files slice does not include imports.
func NewLintFilesRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		files []bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			files := request.ProtosourceFiles()
			filesWithoutImports := make([]bufprotosource.File, 0, len(files))
			for _, file := range files {
				if !file.IsImport() {
					filesWithoutImports = append(filesWithoutImports, file)
				}
			}
			return f(responseWriter, request, filesWithoutImports)
		},
	)
}

// NewLintPackageToFilesRuleHandler returns a new check.RuleHandler for the given function.
//
// The pkgFiles slice will only have files for the given package.
// The pkgFiles slice does not include imports.
func NewLintPackageToFilesRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		pkg string,
		pkgFiles []bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewLintFilesRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			files []bufprotosource.File,
		) error {
			pkgToFiles, err := bufprotosource.PackageToFiles(files...)
			if err != nil {
				return err
			}
			for pkg, pkgFiles := range pkgToFiles {
				if err := f(responseWriter, request, pkg, pkgFiles); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintDirPathToFilesRuleHandler returns a new check.RuleHandler for the given function.
//
// The dirFiles slice will only have files for the given directory.
// The dirFiles slice does not include imports.
func NewLintDirPathToFilesRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		dirPath string,
		dirFiles []bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewLintFilesRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			files []bufprotosource.File,
		) error {
			dirPathToFiles, err := bufprotosource.DirPathToFiles(files...)
			if err != nil {
				return err
			}
			for dirPath, dirFiles := range dirPathToFiles {
				if err := f(responseWriter, request, dirPath, dirFiles); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintFilesRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each File in the request.
//
// Files that are imports are skipped.
func NewLintFileRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		file bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewLintFilesRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			files []bufprotosource.File,
		) error {
			for _, file := range files {
				if err := f(responseWriter, request, file); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintFileImportRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each FileImport within each File in the request.
//
// Files that are imports are skipped.
func NewLintFileImportRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		fileImport bufprotosource.FileImport,
	) error,
) check.RuleHandler {
	return NewLintFileRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			file bufprotosource.File,
		) error {
			for _, fileImport := range file.FileImports() {
				if err := f(responseWriter, request, fileImport); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintEnumRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Enum within each File in the request.
//
// Files that are imports are skipped.
func NewLintEnumRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		enum bufprotosource.Enum,
	) error,
) check.RuleHandler {
	return NewLintFileRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			file bufprotosource.File,
		) error {
			return bufprotosource.ForEachEnum(
				func(enum bufprotosource.Enum) error {
					return f(responseWriter, request, enum)
				},
				file,
			)
		},
	)
}

// NewLintEnumValueRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each EnumValue within each File in the request.
//
// Files that are imports are skipped.
func NewLintEnumValueRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		enumValue bufprotosource.EnumValue,
	) error,
) check.RuleHandler {
	return NewLintEnumRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			enum bufprotosource.Enum,
		) error {
			for _, enumValue := range enum.Values() {
				if err := f(responseWriter, request, enumValue); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintMessageRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Message within each File in the request.
//
// Files that are imports are skipped.
func NewLintMessageRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		message bufprotosource.Message,
	) error,
) check.RuleHandler {
	return NewLintFileRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			file bufprotosource.File,
		) error {
			return bufprotosource.ForEachMessage(
				func(message bufprotosource.Message) error {
					return f(responseWriter, request, message)
				},
				file,
			)
		},
	)
}

// NewLintFieldRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Field within each File in the request.
//
// Files that are imports are skipped.
func NewLintFieldRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		field bufprotosource.Field,
	) error,
) check.RuleHandler {
	return NewLintFileRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			file bufprotosource.File,
		) error {
			if err := bufprotosource.ForEachMessage(
				func(message bufprotosource.Message) error {
					for _, field := range message.Fields() {
						if err := f(responseWriter, request, field); err != nil {
							return err
						}
					}
					for _, field := range message.Extensions() {
						if err := f(responseWriter, request, field); err != nil {
							return err
						}
					}
					return nil
				},
				file,
			); err != nil {
				return err
			}
			for _, field := range file.Extensions() {
				if err := f(responseWriter, request, field); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintOneofRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Oneof within each File in the request.
//
// Files that are imports are skipped.
func NewLintOneofRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		oneof bufprotosource.Oneof,
	) error,
) check.RuleHandler {
	return NewLintMessageRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			message bufprotosource.Message,
		) error {
			for _, oneof := range message.Oneofs() {
				if err := f(responseWriter, request, oneof); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintServiceRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Service within each File in the request.
//
// Files that are imports are skipped.
func NewLintServiceRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		service bufprotosource.Service,
	) error,
) check.RuleHandler {
	return NewLintFileRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			file bufprotosource.File,
		) error {
			for _, service := range file.Services() {
				if err := f(responseWriter, request, service); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

// NewLintMethodRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each Method within each File in the request.
//
// Files that are imports are skipped.
func NewLintMethodRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		method bufprotosource.Method,
	) error,
) check.RuleHandler {
	return NewLintServiceRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			service bufprotosource.Service,
		) error {
			for _, method := range service.Methods() {
				if err := f(responseWriter, request, method); err != nil {
					return err
				}
			}
			return nil
		},
	)
}
