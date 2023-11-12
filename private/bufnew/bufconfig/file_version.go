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
	"fmt"
	"strconv"
)

const (
	// FileVersionV1Beta represents v1beta1 configuration or generation files.
	FileVersionV1Beta1 FileVersion = iota + 1
	// FileVersionV1 represents v1 configuration or generation files.
	FileVersionV1
)

var (
	fileVersionToString = map[FileVersion]string{
		FileVersionV1Beta1: "v1beta1",
		FileVersionV1:      "v1",
	}
	stringToFileVersion = map[string]FileVersion{
		"v1beta1": FileVersionV1Beta1,
		"v1":      FileVersionV1,
	}
)

// FileVersion is the version of a configuration or generation file.
type FileVersion int

// String prints the string representation of the FileVersion.
//
// This is used in configuration and generation files on disk.
func (f FileVersion) String() string {
	s, ok := fileVersionToString[f]
	if !ok {
		return strconv.Itoa(int(f))
	}
	return s
}

func parseFileVersion(s string) (FileVersion, error) {
	// Default to v1beta1 for legacy reasons.
	if s == "" {
		return FileVersionV1Beta1, nil
	}
	c, ok := stringToFileVersion[s]
	if !ok {
		return 0, fmt.Errorf("unknown lock file version: %q", s)
	}
	return c, nil
}

func validateFileVersionExists(fileVersion FileVersion) error {
	if _, ok := fileVersionToString[fileVersion]; !ok {
		return fmt.Errorf("unknown lock file version: %v", fileVersion)
	}
	return nil
}
