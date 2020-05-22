// Copyright 2020 Buf Technologies Inc.
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

package fetch

var (
	_ SingleRef = &singleRef{}
)

type singleRef struct {
	format          string
	path            string
	fileScheme      FileScheme
	compressionType CompressionType
}

func newSingleRef(
	format string,
	path string,
	fileScheme FileScheme,
	compressionType CompressionType,
) *singleRef {
	return &singleRef{
		format:          format,
		path:            path,
		fileScheme:      fileScheme,
		compressionType: compressionType,
	}
}

func (r *singleRef) Format() string {
	return r.format
}

func (r *singleRef) Path() string {
	return r.path
}

func (r *singleRef) FileScheme() FileScheme {
	return r.fileScheme
}

func (r *singleRef) CompressionType() CompressionType {
	return r.compressionType
}

func (*singleRef) ref()       {}
func (*singleRef) fileRef()   {}
func (*singleRef) singleRef() {}
