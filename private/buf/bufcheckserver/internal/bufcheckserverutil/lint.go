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

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/bufplugin-go/check"
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

// NewLintServiceRuleHandler returns a new check.RuleHandler for the given function.
//
// The function will be called for each service within each File in the request.
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
// The function will be called for each method within each File in the request.
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
