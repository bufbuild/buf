// Copyright 2020-2021 Buf Technologies, Inc.
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

var (
	_ ReadBucketCloserWithTerminateFiles = &readBucketCloserWithTerminateFiles{}
	_ TerminateFilePriority              = &terminateFilePriority{}
	_ TerminateFile                      = &terminateFile{}
)

type readBucketCloserWithTerminateFiles struct {
	ReadBucketCloser

	terminateFilePriority TerminateFilePriority
}

func newReadBucketCloserWithTerminateFiles(
	readBucketCloser ReadBucketCloser,
	terminateFilePriority TerminateFilePriority,
) *readBucketCloserWithTerminateFiles {
	return &readBucketCloserWithTerminateFiles{
		ReadBucketCloser:      readBucketCloser,
		terminateFilePriority: terminateFilePriority,
	}
}

func (r *readBucketCloserWithTerminateFiles) TerminateFilePriority() TerminateFilePriority {
	return r.terminateFilePriority
}

type terminateFilePriority struct {
	terminateFiles []TerminateFile
}

func newTerminateFilePriority(terminateFiles []TerminateFile) TerminateFilePriority {
	return &terminateFilePriority{
		terminateFiles: terminateFiles,
	}
}

// TerminateFiles returns the terminate files found.
func (t *terminateFilePriority) TerminateFiles() []TerminateFile {
	return t.terminateFiles
}

type terminateFile struct {
	configFile string
	path       string
}

func newTerminateFile(configFile string, path string) TerminateFile {
	return &terminateFile{
		configFile: configFile,
		path:       path,
	}
}

// ConfigFile returns the config file type of the terminate file.
func (t *terminateFile) ConfigFile() string {
	return t.configFile
}

// Path returns the path where the terminate file is located.
func (t *terminateFile) Path() string {
	return t.path
}
