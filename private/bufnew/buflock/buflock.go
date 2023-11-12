// Copyright 2020-2023 Buf Technologies, Inc.
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

package buflock

import (
	"errors"
	"io"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
)

// File represents a buf.lock file.
type File interface {
	// FileVersion returns the file version of the buf.lock file.
	//
	// To migrate a file between versions, use ReadFile -> FileWithVersion -> WriteFile.
	FileVersion() FileVersion
	// DepModuleKeys returns the ModuleKeys representing the dependencies as specified in the buf.lock file.
	//
	// No deduplication is performed here, either at read or write.
	// TODO: evaluate this.
	// ModuleKeys may not be sorted.
	// TODO: evaluate this.
	DepModuleKeys() []bufmodule.ModuleKey

	isFile()
}

func FileWithVersion(file File, fileVersion FileVersion) File {
	// TODO
	return nil
}

// ReadFile reads the File from the io.Reader.
func ReadFile(reader io.Reader) (File, error) {
	// TODO
	return nil, errors.New("TODO")
}

// WriteFile writes the File to the io.Writer.
func WriteFile(writer io.Writer, file File) error {
	// TODO
	return errors.New("TODO")
}
