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
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/slicesext"
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

	GenerateConfig() GenerateConfig
	InputConfigs() []InputConfig

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
	return getFileVersionForPrefix(ctx, bucket, prefix, bufGenYAMLFileNames)
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
	generateConfig GenerateConfig
	inputConfigs   []InputConfig

	fileVersion FileVersion
}

func newBufGenYAMLFile(
	fileVersion FileVersion,
	generateConfig GenerateConfig,
	inputConfigs []InputConfig,
) *bufGenYAMLFile {
	return &bufGenYAMLFile{
		fileVersion:    fileVersion,
		generateConfig: generateConfig,
		inputConfigs:   inputConfigs,
	}
}

func (g *bufGenYAMLFile) FileVersion() FileVersion {
	return g.fileVersion
}

func (g *bufGenYAMLFile) GenerateConfig() GenerateConfig {
	return g.generateConfig
}

func (g *bufGenYAMLFile) InputConfigs() []InputConfig {
	return g.inputConfigs
}

func (*bufGenYAMLFile) isBufGenYAMLFile() {}
func (*bufGenYAMLFile) isFile()           {}

func readBufGenYAMLFile(reader io.Reader, allowJSON bool) (BufGenYAMLFile, error) {
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
		return nil, errors.New("TODO")
	case FileVersionV1:
		var externalGenYAMLFile externalBufGenYAMLFileV1
		if err := getUnmarshalStrict(allowJSON)(data, &externalGenYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		generateConfig, err := newGenerateConfigFromExternalFileV1(externalGenYAMLFile)
		if err != nil {
			return nil, err
		}
		return newBufGenYAMLFile(
			fileVersion,
			generateConfig,
			nil,
		), nil
	case FileVersionV2:
		var externalGenYAMLFile externalBufGenYAMLFileV2
		if err := getUnmarshalStrict(allowJSON)(data, &externalGenYAMLFile); err != nil {
			return nil, fmt.Errorf("invalid as version %v: %w", fileVersion, err)
		}
		generateConfig, err := newGenerateConfigFromExternalFileV2(externalGenYAMLFile)
		if err != nil {
			return nil, err
		}
		inputConfigs, err := slicesext.MapError(
			externalGenYAMLFile.Inputs,
			newInputConfigFromExternalInputConfigV2,
		)
		if err != nil {
			return nil, err
		}
		return newBufGenYAMLFile(
			fileVersion,
			generateConfig,
			inputConfigs,
		), nil
	default:
		// This is a system error since we've already parsed.
		return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}

func writeBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	// TODO: is this check necessary?
	switch fileVersion := bufGenYAMLFile.FileVersion(); fileVersion {
	case FileVersionV1Beta1, FileVersionV1, FileVersionV2:
		return errors.New("TODO")
	default:
		// This is a system error since we've already parsed.
		return syserror.Newf("unknown FileVersion: %v", fileVersion)
	}
}
