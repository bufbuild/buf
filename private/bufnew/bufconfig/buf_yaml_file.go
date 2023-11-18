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
	// defaultBufYAMLFileName is the default file name you should use for buf.yaml Files.
	defaultBufYAMLFileName = "buf.yaml"
)

var (
	// otherBufYAMLFileNames are all file names we have ever used for configuration files.
	//
	// Originally we thought we were going to move to buf.mod, and had this around for
	// a while, but then reverted back to buf.yaml. We still need to support buf.mod as
	// we released with it, however.
	otherBufYAMLFileNames = []string{
		"buf.mod",
	}
)

// BufYAMLFile represents a buf.yaml file.
type BufYAMLFile interface {
	File

	// ModuleConfigs returns the ModuleConfigs for the File.
	//
	// For v1 buf.yaml, this will only have a single ModuleConfig.
	ModuleConfigs() []ModuleConfig
	// GenerateConfigs returns the GenerateConfigs for the File.
	//
	// For v1 buf.yaml, this will be empty.
	GenerateConfigs() []GenerateConfig

	isBufYAMLFile()
}

// GetBufYAMLFileForPrefix gets the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (BufYAMLFile, error) {
	return getFileForPrefix(ctx, bucket, prefix, defaultBufYAMLFileName, otherBufYAMLFileNames, readBufYAMLFile)
}

// GetBufYAMLFileForPrefix gets the buf.yaml file version at the given bucket prefix.
//
// The buf.yaml file will be attempted to be read at prefix/buf.yaml.
func GetBufYAMLFileVersionForPrefix(
	ctx context.Context,
	bucket storage.ReadBucket,
	prefix string,
) (FileVersion, error) {
	return getFileVersionForPrefix(ctx, bucket, prefix, defaultBufYAMLFileName, otherBufYAMLFileNames)
}

// PutBufYAMLFileForPrefix puts the buf.yaml file at the given bucket prefix.
//
// The buf.yaml file will be attempted to be written to prefix/buf.yaml.
func PutBufYAMLFileForPrefix(
	ctx context.Context,
	bucket storage.WriteBucket,
	prefix string,
	bufYAMLFile BufYAMLFile,
) error {
	return putFileForPrefix(ctx, bucket, prefix, bufYAMLFile, defaultBufYAMLFileName, writeBufYAMLFile)
}

// ReadBufYAMLFile reads the BufYAMLFile from the io.Reader.
func ReadBufYAMLFile(reader io.Reader) (BufYAMLFile, error) {
	return readFile(reader, "config file", readBufYAMLFile)
}

// WriteBufYAMLFile writes the BufYAMLFile to the io.Writer.
func WriteBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	return writeFile(writer, "config file", bufYAMLFile, writeBufYAMLFile)
}

// *** PRIVATE ***

type bufYAMLFile struct{}

func newBufYAMLFile() *bufYAMLFile {
	return &bufYAMLFile{}
}

func (c *bufYAMLFile) FileVersion() FileVersion {
	panic("not implemented") // TODO: Implement
}

func (c *bufYAMLFile) ModuleConfigs() []ModuleConfig {
	panic("not implemented") // TODO: Implement
}

func (c *bufYAMLFile) GenerateConfigs() []GenerateConfig {
	panic("not implemented") // TODO: Implement
}

func (*bufYAMLFile) isBufYAMLFile() {}
func (*bufYAMLFile) isFile()        {}

func readBufYAMLFile(reader io.Reader, allowJSON bool) (BufYAMLFile, error) {
	return nil, errors.New("TODO")
}

func writeBufYAMLFile(writer io.Writer, bufYAMLFile BufYAMLFile) error {
	return errors.New("TODO")
}
