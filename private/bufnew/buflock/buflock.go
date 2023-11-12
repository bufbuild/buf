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
	"io"

	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"go.uber.org/multierr"
)

const (
	// DefaultFileName is the default file name you should use for buf.lock Files.
	DefaultFileName = "buf.lock"
)

// File represents a buf.lock file.
type File interface {
	// FileVersion returns the file version of the buf.lock file.
	//
	// To migrate a file between versions, use ReadFile -> NewFile(newVersion, file.DepModuleKeys()) -> WriteFile.
	FileVersion() FileVersion
	// DepModuleKeys returns the ModuleKeys representing the dependencies as specified in the buf.lock file.
	//
	// All ModuleKeys will have unique ModuleFullNames.
	// ModuleKeys are sorted by ModuleFullName.
	//
	// TODO: We need to add DigestTypes for all the deprecated digests. We then can handle
	// the fact that they're deprecated outside of this package. Another option is to add a
	// buflock.DeprecatedDigestTypeError to return from Digest(), and then handle that downstream.
	DepModuleKeys() []bufmodule.ModuleKey

	isFile()
}

// NewFile returns a new File.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateFileDigests().
func NewFile(fileVersion FileVersion, depModuleKeys []bufmodule.ModuleKey) (File, error) {
	return newFile(fileVersion, depModuleKeys)
}

// ReadFile reads the File from the io.Reader.
//
// Note that digests are lazily-loaded; if you need to ensure that all digests are valid, run
// ValidateFileDigests().
func ReadFile(reader io.Reader) (File, error) {
	return readFile(reader)
}

// WriteFile writes the File to the io.Writer.
func WriteFile(writer io.Writer, file File) error {
	return writeFile(writer, file)
}

// ValidateFileDigests validates that all Digests on the ModuleKeys are valid, by calling
// each Digest() function.
//
// TODO: should we just ensure this property when returning from NewFile, ReadFile?
func ValidateFileDigests(file File) error {
	var errs []error
	for _, depModuleKey := range file.DepModuleKeys() {
		if _, err := depModuleKey.Digest(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}
