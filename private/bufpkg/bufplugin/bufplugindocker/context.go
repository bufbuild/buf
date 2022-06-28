// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufplugindocker

import (
	"archive/tar"
	"bytes"
	"io"
	"io/fs"
	"time"

	"go.uber.org/multierr"
)

// createDockerContext builds a light-weight Docker context (tar file) from a given Dockerfile.
// This is used to create images that don't rely on any other filesystem state to copy into the context.
func createDockerContext(dockerfile io.Reader) (reader io.Reader, retErr error) {
	dockerfileBytes, err := io.ReadAll(dockerfile)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	defer func() {
		if err := writer.Close(); err != nil {
			retErr = multierr.Append(retErr, err)
		}
	}()
	dockerfileInfo := &fileInfo{
		name: "Dockerfile",
		size: int64(len(dockerfileBytes)),
		mode: 0644,
	}
	header, err := tar.FileInfoHeader(dockerfileInfo, "")
	if err != nil {
		return nil, err
	}
	if err := writer.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := writer.Write(dockerfileBytes); err != nil {
		return nil, err
	}
	return &buffer, nil
}

// fileInfo allows adding header information to tar files from files that don't necessarily come from a directory on disk.
// In createDockerContext above, we use this to represent a Dockerfile loaded from any of the storage.ReadBucket supported types.
type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	dir     bool
	sys     any
}

var _ fs.FileInfo = (*fileInfo)(nil)

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Size() int64 {
	return f.size
}

func (f *fileInfo) Mode() fs.FileMode {
	return f.mode
}

func (f *fileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *fileInfo) IsDir() bool {
	return f.dir
}

func (f *fileInfo) Sys() any {
	return f.sys
}
