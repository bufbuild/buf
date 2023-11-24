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

package bufconfig

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/slicesext"
)

// fileName is a supported file name for a given file type, along with the FileVersions
// that this file name supports.
//
// We store a slice of fileNames per file type, with the first element of every slice being
// the default file name (and also the file name we write to).
type fileName struct {
	name                  string
	supportedFileVersions map[FileVersion]struct{}
}

func newFileName(name string, supportedFileVersions ...FileVersion) *fileName {
	return &fileName{
		name:                  name,
		supportedFileVersions: slicesext.ToStructMap(supportedFileVersions),
	}
}

func (f *fileName) Name() string {
	return f.name
}

func (f *fileName) CheckSupportedFile(file File) error {
	return f.CheckSupportedFileVersion(file.FileVersion())
}

func (f *fileName) CheckSupportedFileVersion(fileVersion FileVersion) error {
	if _, ok := f.supportedFileVersions[fileVersion]; !ok {
		return newUnsupportedFileVersionError(f.name, fileVersion)
	}
	return checkV2SupportedYet(fileVersion)
}

func newUnsupportedFileVersionError(name string, fileVersion FileVersion) error {
	if name == "" {
		return fmt.Errorf("%s is not supported", fileVersion.String())
	}
	return fmt.Errorf("%s is not supported for %s files", fileVersion.String(), name)
}

// TODO: Remove when V2 is supported.
func checkV2SupportedYet(fileVersion FileVersion) error {
	if !isV2Allowed() && fileVersion == FileVersionV2 {
		return errors.New("v2 is not supported yet")
	}
	return nil
}
