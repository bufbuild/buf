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
	"errors"
	"io"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/syserror"
)

var (
	bufGenYAML          = newFileName("buf.gen.yaml", FileVersionV1Beta1, FileVersionV1, FileVersionV2)
	bufGenYAMLFileNames = []*fileName{bufGenYAML}
)

// BufGenYAMLFile represents a buf.gen.yaml file.
//
// For v2, generation configuration has been merged into BufYAMLFiles.
type BufGenYAMLFile interface {
	File
	// Will always have empty GenerateInputConfigs.
	GenerateConfig

	isBufGenYAMLFile()
}

// GetBufGenYAMLFileForPrefix gets the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufGenYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufGenYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, bufGenYAMLFileNames, readBufGenYAMLFile)
}

// GetBufGenYAMLFileForPrefix gets the buf.gen.yaml file version at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufGenYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, bufGenYAMLFileNames, true, FileVersionV2)
}

// PutBufGenYAMLFileForPrefix puts the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be written to prefix/buf.gen.yaml.
// The buf.gen.yaml file will be written atomically.
func PutBufGenYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufGenYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, bufGenYAML, writeBufGenYAMLFile)
}

// ReadBufGenYAMLFile reads the BufGenYAMLFile from the io.Reader.
func ReadBufGenYAMLFile(reader io.Reader) (BufGenYAMLFile, error) {
	return readFile(reader, "generation file", readBufGenYAMLFile)
}

// WriteBufGenYAMLFile writes the BufGenYAMLFile to the io.Writer.
func WriteBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	return writeFile(writer, "generation file", bufGenYAMLFile, writeBufGenYAMLFile)
}

// *** PRIVATE ***

type bufGenYAMLFile struct {
	GenerateConfig

	fileVersion FileVersion
}

func newBufGenYAMLFile(fileVersion FileVersion, generateConfig GenerateConfig) (*bufGenYAMLFile, error) {
	return &bufGenYAMLFile{
		GenerateConfig: generateConfig,
		fileVersion:    fileVersion,
	}, errors.New("TODO")
}

func (g *bufGenYAMLFile) FileVersion() FileVersion {
	return g.fileVersion
}

func (*bufGenYAMLFile) isBufGenYAMLFile() {}
func (*bufGenYAMLFile) isFile()           {}

func readBufGenYAMLFile(reader io.Reader, allowJSON bool) (BufGenYAMLFile, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	// We have always enforced that buf.gen.yamls have file versions.
	fileVersion, err := getFileVersionForData(data, allowJSON, true, FileVersionV2)
	if err != nil {
		return nil, err
	}
	switch fileVersion {
	case FileVersionV1Beta1:
		return nil, errors.New("TODO")
	case FileVersionV1:
		return nil, errors.New("TODO")
	case FileVersionV2:
		return nil, errors.New("TODO")
	default:
		// This is a system error since we've already parsed.
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	switch fileVersion := bufGenYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1:
		return errors.New("TODO")
	case FileVersionV1:
		return errors.New("TODO")
	case FileVersionV2:
		return errors.New("TODO")
	default:
		// This is a system error since we've already parsed.
		return syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}
