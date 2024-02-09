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

package bufconfig

import (
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// FileVersionV1Beta represents v1beta1 files.
	FileVersionV1Beta1 FileVersion = iota + 1
	// FileVersionV1 represents v1 files.
	FileVersionV1
	// FileVersionV2 represents v2 files.
	FileVersionV2
)

var (
	// AllFileVersions are all FileVersions.
	AllFileVersions = []FileVersion{
		FileVersionV1Beta1,
		FileVersionV1,
		FileVersionV2,
	}

	fileVersionToString = map[FileVersion]string{
		FileVersionV1Beta1: "v1beta1",
		FileVersionV1:      "v1",
		FileVersionV2:      "v2",
	}
	stringToFileVersion = map[string]FileVersion{
		"v1beta1": FileVersionV1Beta1,
		"v1":      FileVersionV1,
		"v2":      FileVersionV2,
	}
)

// FileVersion is the version of a file.
type FileVersion int

// String prints the string representation of the FileVersion.
//
// This is used in buf.yaml, buf.gen.yaml, buf.work.yaml, and buf.lock files on disk.
func (f FileVersion) String() string {
	s, ok := fileVersionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

// *** PRIVATE ***

func getFileVersionForData(
	data []byte,
	allowJSON bool,
	fileVersionRequired bool,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
	suggestedFileVersion FileVersion,
	defaultFileVersion FileVersion,
) (FileVersion, error) {
	var externalFileVersion externalFileVersion
	if err := getUnmarshalNonStrict(allowJSON)(data, &externalFileVersion); err != nil {
		return 0, err
	}
	return parseFileVersion(
		externalFileVersion.Version,
		"",
		fileVersionRequired,
		fileNameToSupportedFileVersions,
		suggestedFileVersion,
		defaultFileVersion,
	)
}

func parseFileVersion(
	s string,
	// optional
	fileName string,
	fileVersionRequired bool,
	fileNameToSupportedFileVersions map[string]map[FileVersion]struct{},
	suggestedFileVersion FileVersion,
	defaultFileVersion FileVersion,
) (FileVersion, error) {
	if s == "" {
		if fileVersionRequired {
			return 0, newNoFileVersionError(suggestedFileVersion)
		}
		return defaultFileVersion, nil
	}
	c, ok := stringToFileVersion[s]
	if !ok {
		return 0, fmt.Errorf("unknown file version: %q", s)
	}
	if fileName != "" {
		if err := validateSupportedFileVersion(fileName, c, fileNameToSupportedFileVersions); err != nil {
			return 0, err
		}
	}
	return c, nil
}

func validateSupportedFileVersion(fileName string, fileVersion FileVersion, fileNameToSupportedFileVersions map[string]map[FileVersion]struct{}) error {
	supportedFileVersions, ok := fileNameToSupportedFileVersions[fileName]
	if !ok {
		// This should never happen.
		return syserror.Newf("unknown configuration file name: %q", fileName)
	}
	if _, ok := supportedFileVersions[fileVersion]; !ok {
		return newUnsupportedFileVersionError(fileName, fileVersion)
	}
	return nil
}

// externalFileVersion represents just the version component of any file.
type externalFileVersion struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// newNoFileVersionError returns a new error when a FileVersion is required but was not found.
//
// The suggested FileVersion is printed in the error.
func newNoFileVersionError(suggestedFileVersion FileVersion) error {
	return fmt.Errorf(`"version" is not set. Please add "version: %s"`, suggestedFileVersion.String())
}

func newUnsupportedFileVersionError(name string, fileVersion FileVersion) error {
	if name == "" {
		return fmt.Errorf("%s is not supported", fileVersion.String())
	}
	return fmt.Errorf("%s is not supported for %s files", fileVersion.String(), name)
}
