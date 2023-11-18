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
)

const (
	// defaultBufGenYAMLFileName is the default file name you should use for buf.gen.yaml Files.
	//
	// This is not included in AllFileNames.
	//
	// For v2, generation configuration is merged into buf.yaml.
	defaultBufGenYAMLFileName = "buf.gen.yaml"
)

// BufGenYAMLFile represents a buf.gen.yaml file.
//
// For v2, generation configuration has been merged into BufYAMLFiles.
type BufGenYAMLFile interface {
	File
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
	return getFileForPrefix(ctx, bucket, prefix, defaultBufGenYAMLFileName, nil, readBufGenYAMLFile)
}

// GetBufGenYAMLFileForPrefix gets the buf.gen.yaml file version at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be read at prefix/buf.gen.yaml.
func GetBufGenYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, defaultBufGenYAMLFileName, nil)
}

// PutBufGenYAMLFileForPrefix puts the buf.gen.yaml file at the given bucket prefix.
//
// The buf.gen.yaml file will be attempted to be written to prefix/buf.gen.yaml.
func PutBufGenYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufGenYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufGenYAMLFileName, writeBufGenYAMLFile)
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
	generateConfig
}

func newBufGenYAMLFile() *bufGenYAMLFile {
	return &bufGenYAMLFile{}
}

func (g *bufGenYAMLFile) FileVersion() FileVersion {
	panic("not implemented") // TODO: Implement
}

func (*bufGenYAMLFile) isBufGenYAMLFile() {}
func (*bufGenYAMLFile) isFile()           {}

func readBufGenYAMLFile(reader io.Reader, allowJSON bool) (BufGenYAMLFile, error) {
	return nil, errors.New("TODO")
}

func writeBufGenYAMLFile(writer io.Writer, bufGenYAMLFile BufGenYAMLFile) error {
	return errors.New("TODO")
}
