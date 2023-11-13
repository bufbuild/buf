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
	"io"
)

const (
	// DefaultWorkspaceOnlyFileName is the default file name you should use for buf.work.yaml Files.
	//
	// For v2, generation configuration is merged into buf.yaml.
	DefaultWorkspaceOnlyFileName = "buf.work.yaml"
)

var (
	// AllWorkspaceOnlyFileNames are all file names we have ever used for workspace files.
	//
	// Originally we thought we were going to move to buf.work, and had this around for
	// a while, but then reverted back to buf.work.yaml. We still need to support buf.work as
	// we released with it, however.
	AllWorkspaceOnlyFileNames = []string{
		DefaultWorkspaceOnlyFileName,
		"buf.work",
	}
)

// WorkspaceOnlyFile represents a buf.work.yaml file.
//
// For v2, buf.work.yaml files have been eliminated.
// There was never a v1beta1 buf.work.yaml.
type WorkspaceOnlyFile interface {
	// FileVersion returns the version of the buf.gen.yaml file this was read from.
	FileVersion() FileVersion

	DirPaths() string

	isWorkspaceOnlyFile()
}

// ReadWorkspaceOnlyFile reads the WorkspaceOnlyFile from the io.Reader.
func ReadWorkspaceOnlyFile(reader io.Reader) (WorkspaceOnlyFile, error) {
	workspaceOnlyFile, err := readWorkspaceOnlyFile(reader)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(workspaceOnlyFile.FileVersion()); err != nil {
		return nil, err
	}
	return workspaceOnlyFile, nil
}

// WriteWorkspaceOnlyFile writes the WorkspaceOnlyFile to the io.Writer.
func WriteWorkspaceOnlyFile(writer io.Writer, workspaceOnlyFile WorkspaceOnlyFile) error {
	if err := checkV2SupportedYet(workspaceOnlyFile.FileVersion()); err != nil {
		return err
	}
	return writeWorkspaceOnlyFile(writer, workspaceOnlyFile)
}

// *** PRIVATE ***

type workspaceOnlyFile struct{}

func newWorkspaceOnlyFile() *workspaceOnlyFile {
	return &workspaceOnlyFile{}
}

func (w *workspaceOnlyFile) FileVersion() FileVersion {
	panic("not implemented") // TODO: Implement
}

func (w *workspaceOnlyFile) DirPaths() string {
	panic("not implemented") // TODO: Implement
}

func (*workspaceOnlyFile) isWorkspaceOnlyFile() {}

func readWorkspaceOnlyFile(reader io.Reader) (WorkspaceOnlyFile, error) {
	return nil, errors.New("TODO")
}

func writeWorkspaceOnlyFile(writer io.Writer, workspaceOnlyFile WorkspaceOnlyFile) error {
	if workspaceOnlyFile.FileVersion() == FileVersionV1Beta1 {
		return errors.New("v1beta1 is not a valid version for buf.work.yaml files")
	}
	return errors.New("TODO")
}
