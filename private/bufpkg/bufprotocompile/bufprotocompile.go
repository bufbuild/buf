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

package bufprotocompile

import (
	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/protocompile/reporter"
)

// FileAnnotationForErrorWithPos returns a new FileAnnotation for the ErrorWithPos.
func FileAnnotationForErrorWithPos(
	errorWithPos reporter.ErrorWithPos,
	options ...FileAnnotationOption,
) (bufanalysis.FileAnnotation, error) {
	fileAnnotationOptions := newFileAnnotationOptions()
	for _, option := range options {
		option(fileAnnotationOptions)
	}

	var fileInfo bufanalysis.FileInfo
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
		externalPath := path
		if fileAnnotationOptions.externalPathResolver != nil {
			externalPath = fileAnnotationOptions.externalPathResolver(path)
		}
		fileInfo = newFileInfo(path, externalPath)
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

// FileAnnotationSetForErrorWithPos returns new FileAnnotations for the ErrorsWithPos.
func FileAnnotationSetForErrorsWithPos(
	errorsWithPos []reporter.ErrorWithPos,
	options ...FileAnnotationOption,
) (bufanalysis.FileAnnotationSet, error) {
	fileAnnotations, err := slicesext.MapError(
		errorsWithPos,
		func(errorWithPos reporter.ErrorWithPos) (bufanalysis.FileAnnotation, error) {
			return FileAnnotationForErrorWithPos(errorWithPos, options...)
		},
	)
	if err != nil {
		return nil, err
	}
	return bufanalysis.NewFileAnnotationSet(fileAnnotations...), nil
}

// FileAnnotationOption is an option when creating a FileAnnotation.
type FileAnnotationOption func(*fileAnnotationOptions)

// WithExternalPathResolver returns a new FileAnnotationOption that will map the given
// path to an external path.
func WithExternalPathResolver(externalPathResolver func(path string) string) FileAnnotationOption {
	return func(fileAnnotationOptions *fileAnnotationOptions) {
		fileAnnotationOptions.externalPathResolver = externalPathResolver
	}
}

// *** PRIVATE ***

type fileInfo struct {
	path         string
	externalPath string
}

func newFileInfo(path string, externalPath string) *fileInfo {
	return &fileInfo{
		path:         path,
		externalPath: externalPath,
	}
}

func (f *fileInfo) Path() string {
	return f.path
}

func (f *fileInfo) ExternalPath() string {
	return f.externalPath
}

type fileAnnotationOptions struct {
	externalPathResolver func(path string) string
}

func newFileAnnotationOptions() *fileAnnotationOptions {
	return &fileAnnotationOptions{}
}
