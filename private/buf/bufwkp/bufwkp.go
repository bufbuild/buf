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

// Package bufwkp is buf "well-known plugin", i.e. the prototype to replace
// our lint and breaking change packages with a well-known built-in plugin.
package bufwkp

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/bufbuild/bufplugin-go/check"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	servicePascalCaseRuleID  = "SERVICE_PASCAL_CASE"
	servicePascalCasePurpose = "Checks that services are PascalCase."

	basicCategory   = "BASIC"
	defaultCategory = "DEFAULT"
)

var (
	v2ServicePascalCaseRuleSpec = &check.RuleSpec{
		ID: servicePascalCaseRuleID,
		Categories: []string{
			basicCategory,
			defaultCategory,
		},
		Purpose: servicePascalCasePurpose,
		Type:    check.RuleTypeLint,
		Handler: newLintServiceRuleHandler(checkServicePascalCase),
	}

	v2Spec = &check.Spec{
		Rules: []*check.RuleSpec{
			v2ServicePascalCaseRuleSpec,
		},
		Before: before,
	}
)

type protosourceFilesContextKey struct{}
type againstProtosourceFilesContextKey struct{}

type ResponseWriter interface {
	check.ResponseWriter

	AddProtosourceAnnotation(
		location bufprotosource.Location,
		againstLocation bufprotosource.Location,
		format string,
		args ...any,
	)
}

type Request interface {
	check.Request

	ProtosourceFiles() []bufprotosource.File
	AgainstProtosourceFiles() []bufprotosource.File
}

func checkServicePascalCase(
	responseWriter ResponseWriter,
	request Request,
	service bufprotosource.Service,
) error {
	name := service.Name()
	expectedName := stringutil.ToPascalCase(name)
	if name != expectedName {
		responseWriter.AddProtosourceAnnotation(
			service.NameLocation(),
			nil,
			"Service name %q should be PascalCase, such as %q.",
			name,
			expectedName,
		)
	}
	return nil
}

func newRuleHandler(
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

// Skips imports.
func newLintFilesRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		files []bufprotosource.File,
	) error,
) check.RuleHandler {
	return newRuleHandler(
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

// Skips imports.
func newLintFileRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		file bufprotosource.File,
	) error,
) check.RuleHandler {
	return newLintFilesRuleHandler(
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

// Skips imports.
func newLintServiceRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		service bufprotosource.Service,
	) error,
) check.RuleHandler {
	return newLintFileRuleHandler(
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

// Skips imports.
func newLintMethodRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		method bufprotosource.Method,
	) error,
) check.RuleHandler {
	return newLintServiceRuleHandler(
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

func before(
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

type inputFile struct {
	check.File
}

func newInputFile(file check.File) *inputFile {
	return &inputFile{
		File: file,
	}
}

func (i *inputFile) Path() string {
	return i.File.FileDescriptorProto().GetName()
}

// TODO: We will need to reconcile this on the client-side as right now we rely on ExternalPath
// being passed end-to-end.
func (i *inputFile) ExternalPath() string {
	return i.Path()
}

func (i *inputFile) ModuleFullName() bufmodule.ModuleFullName {
	return nil
}

func (i *inputFile) CommitID() uuid.UUID {
	return uuid.Nil
}

type responseWriter struct {
	check.ResponseWriter
}

func newResponseWriter(checkResponseWriter check.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: checkResponseWriter,
	}
}

func (w *responseWriter) AddProtosourceAnnotation(
	location bufprotosource.Location,
	againstLocation bufprotosource.Location,
	format string,
	args ...any,
) {
	w.ResponseWriter.AddAnnotation(
		check.WithMessagef(format, args...),
		check.WithFileName(location.FilePath()),
		check.WithSourcePath(location.SourcePath()),
	)
}

type request struct {
	check.Request

	protosourceFiles        []bufprotosource.File
	againstProtosourceFiles []bufprotosource.File
}

func newRequest(
	checkRequest check.Request,
	protosourceFiles []bufprotosource.File,
	againstProtosourceFiles []bufprotosource.File,
) *request {
	return &request{
		Request:                 checkRequest,
		protosourceFiles:        protosourceFiles,
		againstProtosourceFiles: againstProtosourceFiles,
	}
}

func (r *request) ProtosourceFiles() []bufprotosource.File {
	return r.protosourceFiles
}

func (r *request) AgainstProtosourceFiles() []bufprotosource.File {
	return r.againstProtosourceFiles
}
