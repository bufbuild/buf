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
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/slicesextended"
	"github.com/bufbuild/buf/private/pkg/storage"
)

const (
	// defaultBufWorkYAMLFileName is the default file name you should use for buf.work.yaml Files.
	//
	// For v2, generation configuration is merged into buf.yaml.
	defaultBufWorkYAMLFileName = "buf.work.yaml"
)

var (
	// otherBufWorkYAMLFileNames are all file names we have ever used for workspace files.
	//
	// Originally we thought we were going to move to buf.work, and had this around for
	// a while, but then reverted back to buf.work.yaml. We still need to support buf.work as
	// we released with it, however.
	otherBufWorkYAMLFileNames = []string{
		"buf.work",
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
	//  1. There is at least one path.
	//  2. There are no duplicate paths.
	//  3. No path contains another path.
	//  4. Paths are sorted deterministically.
	DirPaths() []string

	isBufWorkYAMLFile()
}

// NewBufWorkYAMLFile returns a new BufWorkYAMLFile.
func NewBufWorkYAMLFile(fileVersion FileVersion, dirPaths []string) (BufWorkYAMLFile, error) {
	bufWorkYAMLFile, err := newBufWorkYAMLFile(fileVersion, dirPaths)
	if err != nil {
		return nil, err
	}
	if err := checkV2SupportedYet(bufWorkYAMLFile); err != nil {
		return nil, err
	}
	return bufWorkYAMLFile, nil
}

// GetBufWorkYAMLFileForPrefix gets the buf.work.yaml file at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be read at prefix/buf.work.yaml.
func GetBufWorkYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufWorkYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, defaultBufWorkYAMLFileName, otherBufWorkYAMLFileNames, readBufWorkYAMLFile)
}

// GetBufWorkYAMLFileForPrefix gets the buf.work.yaml file version at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be read at prefix/buf.work.yaml.
func GetBufWorkYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, defaultBufWorkYAMLFileName, otherBufWorkYAMLFileNames)
}

// PutBufWorkYAMLFileForPrefix puts the buf.work.yaml file at the given bucket prefix.
//
// The buf.work.yaml file will be attempted to be written to prefix/buf.work.yaml.
func PutBufWorkYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufWorkYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufWorkYAMLFileName, writeBufWorkYAMLFile)
}

// ReadBufWorkYAMLFile reads the buf.work.yaml file from the io.Reader.
func ReadBufWorkYAMLFile(reader io.Reader) (BufWorkYAMLFile, error) {
	return readFile(reader, "workspace file", readBufWorkYAMLFile)
}

// WriteBufWorkYAMLFile writes the buf.work.yaml to the io.Writer.
func WriteBufWorkYAMLFile(writer io.Writer, bufWorkYAMLFile BufWorkYAMLFile) error {
	return writeFile(writer, "workspace file", bufWorkYAMLFile, writeBufWorkYAMLFile)
}

// *** PRIVATE ***

type bufWorkYAMLFile struct {
	fileVersion FileVersion
	dirPaths    []string
}

func newBufWorkYAMLFile(fileVersion FileVersion, dirPaths []string) (*bufWorkYAMLFile, error) {
	if err := validateBufWorkYAMLDirPaths(dirPaths); err != nil {
		return nil, err
	}
	sort.Strings(dirPaths)
	return &bufWorkYAMLFile{
		fileVersion: fileVersion,
		dirPaths:    dirPaths,
	}, nil
}

func (w *bufWorkYAMLFile) FileVersion() FileVersion {
	return w.fileVersion
}

func (w *bufWorkYAMLFile) DirPaths() []string {
	return slicesextended.Copy(w.dirPaths)
}

func (*bufWorkYAMLFile) isBufWorkYAMLFile() {}
func (*bufWorkYAMLFile) isFile()            {}

func readBufWorkYAMLFile(reader io.Reader, allowJSON bool) (BufWorkYAMLFile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	fileVersion, err := getFileVersionForData(data, allowJSON)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1:
		return nil, newUnsupportedFileVersionError(fileVersion)
	case FileVersionV1:
		var externalBufWorkYAMLFile externalBufWorkYAMLFileV1
		if err := getUnmarshalStrict(allowJSON)(data, &externalBufWorkYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		return newBufWorkYAMLFile(fileVersion, externalBufWorkYAMLFile.Directories)
	case FileVersionV2:
		return nil, newUnsupportedFileVersionError(fileVersion)
	default:
		// This is a system error since we've already parsed.
		return nil, fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufWorkYAMLFile(writer io.Writer, bufWorkYAMLFile BufWorkYAMLFile) error {
	switch fileVersion := bufWorkYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1:
		return newUnsupportedFileVersionError(fileVersion)
	case FileVersionV1:
		externalBufWorkYAMLFile := externalBufWorkYAMLFileV1{
			Version: fileVersion.String(),
			// No need to sort - DirPaths() is already sorted per the documentation on BufWorkYAMLFile
			Directories: bufWorkYAMLFile.DirPaths(),
		}
		data, err := encoding.MarshalYAML(&externalBufWorkYAMLFile)
		if err != nil {
			return err
		}
		_, err = writer.Write(append(bufLockFileHeader, data...))
		return err
	case FileVersionV2:
		return newUnsupportedFileVersionError(fileVersion)
	default:
		// This is a system error since we've already parsed.
		return fmt.Errorf("unknown FileVersion: %v", fileVersion)
	}
}

func validateBufWorkYAMLDirPaths(dirPaths []string) error {
	if len(dirPaths) == 0 {
		return fmt.Errorf(`directories is empty`)
	}
	directorySet := make(map[string]struct{}, len(dirPaths))
	for _, directory := range dirPaths {
		normalizedDirectory, err := normalpath.NormalizeAndValidate(directory)
		if err != nil {
			return fmt.Errorf(`directory %q is invalid: %w`, normalpath.Unnormalize(directory), err)
		}
		if _, ok := directorySet[normalizedDirectory]; ok {
			return fmt.Errorf(`directory %q is listed more than once`, normalpath.Unnormalize(normalizedDirectory))
		}
		if normalizedDirectory == "." {
			return fmt.Errorf(`directory "." is listed, it is not valid to have "." as a workspace directory, as this is no different than not having a workspace at all, see https://buf.build/docs/reference/workspaces/#directories for more details`)
		}
		directorySet[normalizedDirectory] = struct{}{}
	}
	directories := slicesextended.MapToSlice(directorySet)
	if err := validateConfigurationOverlap(directories); err != nil {
		return err
	}
	return nil
}

// validateOverlap returns a non-nil error if any of the directories overlap
// with each other.
func validateConfigurationOverlap(directories []string) error {
	for i := 0; i < len(directories); i++ {
		for j := i + 1; j < len(directories); j++ {
			left := directories[i]
			right := directories[j]
			if normalpath.ContainsPath(left, right, normalpath.Relative) {
				return fmt.Errorf(
					`directory %q contains directory %q`,
					normalpath.Unnormalize(left),
					normalpath.Unnormalize(right),
				)
			}
			if normalpath.ContainsPath(right, left, normalpath.Relative) {
				return fmt.Errorf(
					`directory %q contains directory %q`,
					normalpath.Unnormalize(right),
					normalpath.Unnormalize(left),
				)
			}
		}
	}
	return nil
}
