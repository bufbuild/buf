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
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

const (
	// DefaultBufWorkYAMLFileName is the default buf.work.yaml file name.
	DefaultBufWorkYAMLFileName = "buf.work.yaml"

	// Originally we thought we were going to move to buf.work, and had this around for
	// a while, but then reverted back to buf.work.yaml. We still need to support buf.work as
	// we released with it, however.
	oldBufWorkYAMLFileName = "buf.work"

	defaultBufWorkYAMLFileVersion = FileVersionV1
)

var (
	// ordered
	bufWorkYAMLFileNames = []string{DefaultBufWorkYAMLFileName, oldBufWorkYAMLFileName}
	// We only supported buf.work.yamls in v1.
	bufWorkYAMLFileNameToSupportedFileVersions = map[string]map[FileVersion]struct{}{
		DefaultBufWorkYAMLFileName: {
			FileVersionV1: struct{}{},
		},
		oldBufWorkYAMLFileName: {
			FileVersionV1: struct{}{},
		},
	}
)

// BufWorkYAMLFile represents a buf.work.yaml file.
//
// For v2, buf.work.yaml files have been eliminated.
// There was never a v1beta1 buf.work.yaml.
type BufWorkYAMLFile interface {
	File

	// DirPaths returns all the directory paths specified in buf.work.yaml,
	// relative to the directory with buf.work.yaml. The following are guaranteed:
	//
	// - There is at least one path, i.e. DirPaths() will never be empty..
	// - There are no duplicate paths - all values of DirPaths() are unique.
	// - No path contains another path, i.e. "foo" and "foo/bar" will not be in DirPaths().
	// - "." is not in DirPaths().
	// - Each path is normalized and validated, because this is guaranteed at the
	//   construction time of a BufWorkYAMLFile.
	//
	// Returned paths are sorted.
	DirPaths() []string

	isBufWorkYAMLFile()
}

// NewBufWorkYAMLFile returns a new validated BufWorkYAMLFile.
func NewBufWorkYAMLFile(fileVersion FileVersion, dirPaths []string) (BufWorkYAMLFile, error) {
	return newBufWorkYAMLFile(fileVersion, nil, dirPaths)
}

// GetBufWorkYAMLFileForPrefix gets the buf.work.yaml file at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be read at prefix/buf.work.yaml.
func GetBufWorkYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufWorkYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufWorkYAMLFileNames, bufWorkYAMLFileNameToSupportedFileVersions, readBufWorkYAMLFile)
}

// GetBufWorkYAMLFileForPrefix gets the buf.work.yaml file version at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be read at prefix/buf.work.yaml.
func GetBufWorkYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, bufWorkYAMLFileNames, bufWorkYAMLFileNameToSupportedFileVersions, true, FileVersionV1, defaultBufWorkYAMLFileVersion)
}

// PutBufWorkYAMLFileForPrefix puts the buf.work.yaml file at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be written to prefix/buf.work.yaml.
// The buf.work.yaml file will be written atomically.
func PutBufWorkYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufWorkYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, DefaultBufWorkYAMLFileName, bufWorkYAMLFileNameToSupportedFileVersions, writeBufWorkYAMLFile)
}

// ReadBufWorkYAMLFile reads the buf.work.yaml file from the io.Reader.
//
// fileName may be empty.
func ReadBufWorkYAMLFile(reader io.Reader, fileName string) (BufWorkYAMLFile, error) {
	return readFile(reader, fileName, readBufWorkYAMLFile)
}

// WriteBufWorkYAMLFile writes the buf.work.yaml to the io.Writer.
func WriteBufWorkYAMLFile(writer io.Writer, bufWorkYAMLFile BufWorkYAMLFile) error {
	return writeFile(writer, bufWorkYAMLFile, writeBufWorkYAMLFile)
}

// *** PRIVATE ***

type bufWorkYAMLFile struct {
	fileVersion FileVersion
	objectData  ObjectData
	dirPaths    []string
}

func newBufWorkYAMLFile(fileVersion FileVersion, objectData ObjectData, dirPaths []string) (*bufWorkYAMLFile, error) {
	if fileVersion != FileVersionV1 {
		return nil, newUnsupportedFileVersionError("", fileVersion)
	}
	sortedNormalizedDirPaths, err := validateBufWorkYAMLDirPaths(dirPaths)
	if err != nil {
		return nil, err
	}
	return &bufWorkYAMLFile{
		fileVersion: fileVersion,
		objectData:  objectData,
		dirPaths:    sortedNormalizedDirPaths,
	}, nil
}

func (w *bufWorkYAMLFile) FileVersion() FileVersion {
	return w.fileVersion
}

func (*bufWorkYAMLFile) FileType() FileType {
	return FileTypeBufWorkYAML
}

func (w *bufWorkYAMLFile) ObjectData() ObjectData {
	return w.objectData
}

func (w *bufWorkYAMLFile) DirPaths() []string {
	return slicesext.Copy(w.dirPaths)
}

func (*bufWorkYAMLFile) isBufWorkYAMLFile() {}
func (*bufWorkYAMLFile) isFile()            {}
func (*bufWorkYAMLFile) isFileInfo()        {}

func readBufWorkYAMLFile(
	data []byte,
	objectData ObjectData,
	allowJSON bool,
) (BufWorkYAMLFile, error) {
	// We've always required a file version for buf.work.yamls.
	fileVersion, err := getFileVersionForData(data, allowJSON, true, bufWorkYAMLFileNameToSupportedFileVersions, FileVersionV1, defaultBufWorkYAMLFileVersion)
	if err != nil {
		return nil, err
	}
	if fileVersion != FileVersionV1 {
		return nil, newUnsupportedFileVersionError(objectData.Name(), fileVersion)
	}
	var externalBufWorkYAMLFile externalBufWorkYAMLFileV1
	if err := getUnmarshalStrict(allowJSON)(data, &externalBufWorkYAMLFile); err != nil {
		return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
	}
	return newBufWorkYAMLFile(fileVersion, objectData, externalBufWorkYAMLFile.Directories)
}

func writeBufWorkYAMLFile(writer io.Writer, bufWorkYAMLFile BufWorkYAMLFile) error {
	fileVersion := bufWorkYAMLFile.FileVersion()
	if fileVersion != FileVersionV1 {
		// This is effectively a system error.
		return syserror.Wrap(newUnsupportedFileVersionError("", fileVersion))
	}
	externalBufWorkYAMLFile := externalBufWorkYAMLFileV1{
		Version: fileVersion.String(),
		// No need to sort - DirPaths() is already sorted per the documentation on BufWorkYAMLFile
		Directories: bufWorkYAMLFile.DirPaths(),
	}
	data, err := encoding.MarshalYAML(&externalBufWorkYAMLFile)
	if err != nil {
		return err
	}
	_, err = writer.Write(data)
	return err
}

// validateBufWorkYAMLDirPaths validates dirPaths and returns normalized and
// sorted dirPaths.
func validateBufWorkYAMLDirPaths(dirPaths []string) ([]string, error) {
	if len(dirPaths) == 0 {
		return nil, fmt.Errorf(`directories is empty`)
	}
	normalizedDirPathToDirPath := make(map[string]string, len(dirPaths))
	for _, dirPath := range dirPaths {
		normalizedDirPath, err := normalpath.NormalizeAndValidate(dirPath)
		if err != nil {
			return nil, fmt.Errorf(`directory %q is invalid: %w`, dirPath, err)
		}
		if _, ok := normalizedDirPathToDirPath[normalizedDirPath]; ok {
			return nil, fmt.Errorf(`directory %q is listed more than once`, dirPath)
		}
		if normalizedDirPath == "." {
			return nil, fmt.Errorf(`directory "." is listed, it is not valid to have "." as a workspace directory, as this is no different than not having a workspace at all, see https://buf.build/docs/reference/workspaces/#directories for more details`)
		}
		normalizedDirPathToDirPath[normalizedDirPath] = dirPath
	}
	// We already know the paths are unique due to above validation.
	// We sort to print deterministic errors.
	sortedNormalizedDirPaths := slicesext.MapKeysToSortedSlice(normalizedDirPathToDirPath)
	for i := 0; i < len(sortedNormalizedDirPaths); i++ {
		for j := i + 1; j < len(sortedNormalizedDirPaths); j++ {
			left := sortedNormalizedDirPaths[i]
			right := sortedNormalizedDirPaths[j]
			if normalpath.ContainsPath(left, right, normalpath.Relative) {
				return nil, fmt.Errorf(
					`directory %q contains directory %q`,
					normalizedDirPathToDirPath[left],
					normalizedDirPathToDirPath[right],
				)
			}
			if normalpath.ContainsPath(right, left, normalpath.Relative) {
				return nil, fmt.Errorf(
					`directory %q contains directory %q`,
					normalizedDirPathToDirPath[right],
					normalizedDirPathToDirPath[left],
				)
			}
		}
	}
	return sortedNormalizedDirPaths, nil
}

// externalBufWorkYAMLFileV1 represents the v1 buf.work.yaml file.
type externalBufWorkYAMLFileV1 struct {
	Version     string   `json:"version,omitempty" yaml:"version,omitempty"`
	Directories []string `json:"directories,omitempty" yaml:"directories,omitempty"`
}
