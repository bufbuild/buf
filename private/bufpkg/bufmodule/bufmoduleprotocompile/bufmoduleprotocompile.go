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

package bufmoduleprotocompile

import (
	"context"
	"io"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/protocompile/reporter"
)

// ParserAccessorHandler handles source file access operations for protocompile.
type ParserAccessorHandler interface {
	// Open opens the given path, and tracks the external path and import status.
	//
	// This function can be used as the accessor function for a protocompile.SourceResolver.
	Open(path string) (io.ReadCloser, error)
	// ExternalPath returns the external path for the input path.
	//
	// Returns the input path if the external path is not known.
	ExternalPath(path string) string
	// ModuleIdentity returns nil if not available.
	ModuleIdentity(path string) bufmoduleref.ModuleIdentity
	// Commit returns empty if not available.
	Commit(path string) string
}

// NewParserAccessorHandler returns a new ParserAccessorHandler.
//
// TODO: make this dependent on whatever derivative getter type we create to replace ModuleFileSet.
func NewParserAccessorHandler(ctx context.Context, moduleFileSet bufmodule.ModuleFileSet) ParserAccessorHandler {
	return newParserAccessorHandler(ctx, moduleFileSet)
}

// GetFileAnnotations gets the FileAnnotations for the ErrorWithPos errors.
func GetFileAnnotations(
	ctx context.Context,
	parserAccessorHandler ParserAccessorHandler,
	errorsWithPos []reporter.ErrorWithPos,
) ([]bufanalysis.FileAnnotation, error) {
	fileAnnotations := make([]bufanalysis.FileAnnotation, 0, len(errorsWithPos))
	for _, errorWithPos := range errorsWithPos {
		fileAnnotation, err := GetFileAnnotation(
			ctx,
			parserAccessorHandler,
			errorWithPos,
		)
		if err != nil {
			return nil, err
		}
		fileAnnotations = append(fileAnnotations, fileAnnotation)
	}
	return fileAnnotations, nil
}

// GetFileAnnotation gets the FileAnnotation for the ErrorWithPos error.
func GetFileAnnotation(
	ctx context.Context,
	parserAccessorHandler ParserAccessorHandler,
	errorWithPos reporter.ErrorWithPos,
) (bufanalysis.FileAnnotation, error) {
	var fileInfo bufmoduleref.FileInfo
	var startLine int
	var startColumn int
	var endLine int
	var endColumn int
	typeString := "COMPILE"
	message := "Compile error."
	// this should never happen
	// maybe we should error
	if errorWithPos.Unwrap() != nil {
		message = errorWithPos.Unwrap().Error()
	}
	sourcePos := errorWithPos.GetPosition()
	if sourcePos.Filename != "" {
		path, err := normalpath.NormalizeAndValidate(sourcePos.Filename)
		if err != nil {
			return nil, err
		}
		fileInfo, err = bufmoduleref.NewFileInfo(
			path,
			parserAccessorHandler.ExternalPath(path),
			nil,
			"",
		)
		if err != nil {
			return nil, err
		}
	}
	if sourcePos.Line > 0 {
		startLine = sourcePos.Line
		endLine = sourcePos.Line
	}
	if sourcePos.Col > 0 {
		startColumn = sourcePos.Col
		endColumn = sourcePos.Col
	}
	return bufanalysis.NewFileAnnotation(
		fileInfo,
		startLine,
		startColumn,
		endLine,
		endColumn,
		typeString,
		message,
	), nil
}
