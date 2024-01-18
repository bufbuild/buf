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

package bufmodule

import (
	"fmt"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

const (
	FileTypeProto FileType = iota + 1
	FileTypeDoc
	FileTypeLicense
)

var (
	fileTypeToString = map[FileType]string{
		FileTypeProto:   "proto",
		FileTypeDoc:     "doc",
		FileTypeLicense: "license",
	}
	stringToFileType = map[string]FileType{
		"proto":   FileTypeProto,
		"doc":     FileTypeDoc,
		"license": FileTypeLicense,
	}
)

type FileType int

func (c FileType) String() string {
	s, ok := fileTypeToString[c]
	if !ok {
		return strconv.Itoa(int(c))
	}
	return s
}

// ParseFileType parses the file type from its string representation.
//
// This reverses FileType.String().
//
// Returns an error of type *ParseError if thie string could not be parsed.
func ParseFileType(s string) (FileType, error) {
	c, ok := stringToFileType[s]
	if !ok {
		return 0, &ParseError{
			typeString: "module file type",
			input:      s,
			err:        fmt.Errorf("unknown type: %q", s),
		}
	}
	return c, nil
}

// FileType returns the FileType for the given path.
//
// Returns error if the path cannot be classified as a FileType, that is if it is not a
// .proto file, license file, or documentation file.
//
// Note that license and documentation files must be at the root, and cannot be in subdirectories. That is,
// subdir/LICENSE will not be classified as a FileTypeLicnese, but LICENSE will be.
func FileTypeForPath(path string) (FileType, error) {
	if normalpath.Ext(path) == ".proto" {
		return FileTypeProto, nil
	}
	if path == licenseFilePath {
		return FileTypeLicense, nil
	}
	if _, ok := docFilePathMap[path]; ok {
		return FileTypeDoc, nil
	}
	return 0, fmt.Errorf("could not classify FileType for path %q", path)
}

// IsValidModuleFilePath returns true if the given file path is a valid Module file path.
//
// This will be true if the file path represents a .proto file, license file, or documentation file.
//
// Note that license and documentation files must be at the root, and cannot be in subdirectories. That is,
// subdir/LICENSE is not a valid module file (including on push), but LICENSE is.
func IsValidModuleFilePath(filePath string) bool {
	_, err := FileTypeForPath(filePath)
	return err == nil
}
