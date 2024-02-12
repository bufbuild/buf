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

const (
	// FileTypeBufYAML represents buf.yaml files.
	FileTypeBufYAML FileType = iota + 1
	// FileTypeBufLock represents buf.lock files.
	FileTypeBufLock
	// FileTypeBufGenYAML represents buf.gen.yaml files.
	FileTypeBufGenYAML
	// FileTypeBufWorkYAML represents buf.work.yaml files.
	FileTypeBufWorkYAML
)

var (
	fileNameToFileType = map[string]FileType{
		DefaultBufYAMLFileName:     FileTypeBufYAML,
		oldBufYAMLFileName:         FileTypeBufYAML,
		DefaultBufLockFileName:     FileTypeBufLock,
		defaultBufGenYAMLFileName:  FileTypeBufGenYAML,
		DefaultBufWorkYAMLFileName: FileTypeBufWorkYAML,
		oldBufWorkYAMLFileName:     FileTypeBufWorkYAML,
	}

	fileTypeToDefaultFileVersion = map[FileType]FileVersion{
		FileTypeBufYAML:     defaultBufYAMLFileVersion,
		FileTypeBufLock:     defaultBufLockFileVersion,
		FileTypeBufGenYAML:  defaultBufGenYAMLFileVersion,
		FileTypeBufWorkYAML: defaultBufWorkYAMLFileVersion,
	}
	fileTypeToSupportedFileVersions = map[FileType]map[string]map[FileVersion]struct{}{
		FileTypeBufYAML:     bufYAMLFileNameToSupportedFileVersions,
		FileTypeBufLock:     bufLockFileNameToSupportedFileVersions,
		FileTypeBufGenYAML:  bufGenYAMLFileNameToSupportedFileVersions,
		FileTypeBufWorkYAML: bufWorkYAMLFileNameToSupportedFileVersions,
	}
)

// FileType is the type of a file.
type FileType int
