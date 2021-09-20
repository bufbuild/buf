// Copyright 2020-2021 Buf Technologies, Inc.
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

var (
	_ ParsedSingleFileRef = &singleFileRef{}
)

type singleFileRef struct {
	format string
	path   string
}

func newSingleFileRef(format string, path string) *singleFileRef {
	return &singleFileRef{
		format: format,
		path:   path,
	}
}

func (s *singleFileRef) Format() string {
	return s.format
}

func (s *singleFileRef) Path() string {
	return s.path
}

func (*singleFileRef) ref()           {}
func (*singleFileRef) bucketRef()     {}
func (*singleFileRef) singleFileRef() {}
