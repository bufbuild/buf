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

// FileInfo contains information on a configuration file.
type FileInfo interface {
	// FileVersion returns the version of the file.
	FileVersion() FileVersion
	// FileType returns the type of the file.
	FileType() FileType

	isFileInfo()
}

// *** PRIVATE ***

type fileInfo struct {
	fileVersion FileVersion
	fileType    FileType
}

func newFileInfo(fileVersion FileVersion, fileType FileType) *fileInfo {
	return &fileInfo{
		fileVersion: fileVersion,
		fileType:    fileType,
	}
}

func (f *fileInfo) FileVersion() FileVersion {
	return f.fileVersion
}

func (f *fileInfo) FileType() FileType {
	return f.fileType
}

func (*fileInfo) isFileInfo() {}
