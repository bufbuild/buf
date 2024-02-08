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

var _ TerminateFile = &terminateFile{}

type terminateFile struct {
	name string
	path string
}

func newTerminateFile(name string, path string) TerminateFile {
	return &terminateFile{
		name: name,
		path: path,
	}
}

// Name returns the name of the TerminateFile (i.e. the base of the fully-qualified file paths).
func (t *terminateFile) Name() string {
	return t.name
}

// Path returns the normalized directory path where the TemrinateFile is located.
func (t *terminateFile) Path() string {
	return t.path
}
