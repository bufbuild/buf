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
	_ DirRef = &dirRef{}
)

type dirRef struct {
	format string
	path   string
}

func newDirRef(
	format string,
	path string,
) *dirRef {
	return &dirRef{
		format: format,
		path:   path,
	}
}

func (r *dirRef) Format() string {
	return r.format
}

func (r *dirRef) Path() string {
	return r.path
}

func (*dirRef) ref()       {}
func (*dirRef) bucketRef() {}
func (*dirRef) dirRef()    {}
