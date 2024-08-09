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
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/bufplugin-go/check"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

type protosourceFilesContextKey struct{}
type againstProtosourceFilesContextKey struct{}

// RuleSpecBuilder matches check.RuleSpec but without categories.
//
// We have very similar RuleSpecs across our versions of our lint rules, however their categories do change
// across versions. This allows us to share the basic RuleSpec shape across versions.
type RuleSpecBuilder struct {
	// Required.
	ID string
	// Required.
	Purpose string
	// Required.
	Type           check.RuleType
	Deprecated     bool
	ReplacementIDs []string
	// Required.
	Handler check.RuleHandler
}

// Build builds the RuleSpec for the categories.
//
// Not making categories variadic in case we want to add extra parameters later easily.
func (b *RuleSpecBuilder) Build(categories []string) *check.RuleSpec {
	return &check.RuleSpec{
		ID:             b.ID,
		Categories:     categories,
		Purpose:        b.Purpose,
		Type:           b.Type,
		Deprecated:     b.Deprecated,
		ReplacementIDs: b.ReplacementIDs,
		Handler:        b.Handler,
	}
}

// Before should be attached to each check.Spec that uses the functionality in this package.
func Before(
	ctx context.Context,
	request check.Request,
) (context.Context, check.Request, error) {
	protosourceFiles, err := protosourceFilesForFiles(ctx, request.Files())
	if err != nil {
		return nil, nil, err
	}
	againstProtosourceFiles, err := protosourceFilesForFiles(ctx, request.Files())
	if err != nil {
		return nil, nil, err
	}
	if len(protosourceFiles) > 0 {
		ctx = context.WithValue(ctx, protosourceFilesContextKey{}, protosourceFiles)
	}
	if len(againstProtosourceFiles) > 0 {
		ctx = context.WithValue(ctx, againstProtosourceFilesContextKey{}, againstProtosourceFiles)
	}
	return ctx, request, nil
}

// ResponseWriter is a check.ResponseWriter that also includes bufprotosource functionality.
type ResponseWriter interface {
	check.ResponseWriter

	// AddProtosourceAnnotation adds a check.Annotation for bufprotosource.Locations.
	AddProtosourceAnnotation(
		location bufprotosource.Location,
		againstLocation bufprotosource.Location,
		format string,
		args ...any,
	)
}

// Request is a check.Request that also includes bufprotosource functionality.
type Request interface {
	check.Request

	// ProtosourceFiles returns the check.Files as bufprotosource.Files.
	ProtosourceFiles() []bufprotosource.File
	// AgainstProtosourceFiles returns the check.AgainstFiles as bufprotosource.Files.
	AgainstProtosourceFiles() []bufprotosource.File
}

// NewRuleHandler returns a new check.RuleHandler for the given function.
func NewRuleHandler(
	f func(
		ctx context.Context,
		responseWriter ResponseWriter,
		request Request,
	) error,
) check.RuleHandler {
	return check.RuleHandlerFunc(
		func(
			ctx context.Context,
			responseWriter check.ResponseWriter,
			request check.Request,
		) error {
			return f(
				ctx,
				newResponseWriter(responseWriter),
				newRequest(
					request,
					// Is this OK with nil?
					ctx.Value(protosourceFilesContextKey{}).([]bufprotosource.File),
					// Is this OK with nil?
					ctx.Value(againstProtosourceFilesContextKey{}).([]bufprotosource.File),
				),
			)
		},
	)
}

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

func protosourceFilesForFiles(ctx context.Context, files []check.File) ([]bufprotosource.File, error) {
	if len(files) == 0 {
		return nil, nil
	}
	resolver, err := newResolver(files)
	if err != nil {
		return nil, err
	}
	return bufprotosource.NewFiles(ctx, slicesext.Map(files, newInputFile), resolver)
}

func newResolver(files []check.File) (protodesc.Resolver, error) {
	return protodesc.NewFiles(
		&descriptorpb.FileDescriptorSet{
			File: slicesext.Map(files, check.File.FileDescriptorProto),
		},
	)
}
