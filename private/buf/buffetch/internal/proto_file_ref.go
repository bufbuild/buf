// Copyright 2020-2025 Buf Technologies, Inc.
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

package internal

import "buf.build/go/app"

var (
	_ ParsedProtoFileRef = &protoFileRef{}
)

type protoFileRef struct {
	format              string
	path                string
	fileScheme          FileScheme
	includePackageFiles bool
}

func newProtoFileRef(format string, path string, includePackageFiles bool) (*protoFileRef, error) {
	if app.IsDevStderr(path) {
		return nil, NewInvalidPathError(format, path)
	}
	if path == "-" {
		return newDirectProtoFileRef(
			format,
			"",
			FileSchemeStdio,
			includePackageFiles,
		), nil
	}
	if app.IsDevStdin(path) {
		return newDirectProtoFileRef(
			format,
			"",
			FileSchemeStdin,
			includePackageFiles,
		), nil
	}
	if app.IsDevStdout(path) {
		return newDirectProtoFileRef(
			format,
			"",
			FileSchemeStdout,
			includePackageFiles,
		), nil
	}
	if app.IsDevNull(path) {
		return newDirectProtoFileRef(
			format,
			"",
			FileSchemeNull,
			includePackageFiles,
		), nil
	}
	return &protoFileRef{
		format:              format,
		path:                path,
		fileScheme:          FileSchemeLocal,
		includePackageFiles: includePackageFiles,
	}, nil
}

func newDirectProtoFileRef(format string, path string, fileScheme FileScheme, includePackageFiles bool) *protoFileRef {
	return &protoFileRef{
		format:              format,
		path:                path,
		fileScheme:          fileScheme,
		includePackageFiles: includePackageFiles,
	}
}

func (s *protoFileRef) Format() string {
	return s.format
}

func (s *protoFileRef) Path() string {
	return s.path
}

func (s *protoFileRef) FileScheme() FileScheme {
	return s.fileScheme
}

func (s *protoFileRef) IncludePackageFiles() bool {
	return s.includePackageFiles
}

func (*protoFileRef) ref()          {}
func (*protoFileRef) bucketRef()    {}
func (*protoFileRef) protoFileRef() {}
