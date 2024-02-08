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

package internal

import "github.com/bufbuild/buf/private/pkg/app"

var (
	_ ParsedProtoFileRef = &protoFileRef{}
)

type protoFileRef struct {
	format              string
	path                string
	includePackageFiles bool
}

func newProtoFileRef(format string, path string, includePackageFiles bool) (*protoFileRef, error) {
	if app.IsDevPath(path) || path == "-" {
		return nil, NewProtoFileCannotBeDevPathError(format, path)
	}
	return &protoFileRef{
		format:              format,
		path:                path,
		includePackageFiles: includePackageFiles,
	}, nil
}

func (s *protoFileRef) Format() string {
	return s.format
}

func (s *protoFileRef) Path() string {
	return s.path
}

func (s *protoFileRef) IncludePackageFiles() bool {
	return s.includePackageFiles
}

func (*protoFileRef) ref()          {}
func (*protoFileRef) bucketRef()    {}
func (*protoFileRef) protoFileRef() {}
