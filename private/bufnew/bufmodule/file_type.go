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

package bufmodule

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

const (
	FileTypeProto FileType = iota + 1
	FileTypeDoc
	FileTypeLicense
)

var (
	allFileTypes = []FileType{
		FileTypeProto,
		FileTypeDoc,
		FileTypeLicense,
	}
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

func ParseFileType(s string) (FileType, error) {
	c, ok := stringToFileType[s]
	if !ok {
		return 0, fmt.Errorf("unknown FileType: %q", s)
	}
	return c, nil
}

// *** PRIVATE ***

func classifyPathFileType(path string) (FileType, error) {
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

func fileTypeSliceToMap(fileTypes []FileType) map[FileType]struct{} {
	fileTypeMap := make(map[FileType]struct{})
	for _, fileType := range fileTypes {
		fileTypeMap[fileType] = struct{}{}
	}
	return fileTypeMap
}

func fileTypeMapToSortedSlice(fileTypeMap map[FileType]struct{}) []FileType {
	fileTypes := make([]FileType, 0, len(fileTypeMap))
	for fileType := range fileTypeMap {
		fileTypes = append(fileTypes, fileType)
	}
	sort.Slice(
		fileTypes,
		func(i int, j int) bool {
			return fileTypes[i] < fileTypes[j]
		},
	)
	return fileTypes
}
