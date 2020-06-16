// Copyright 2020 Buf Technologies, Inc.
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

package tmp

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gofrs/uuid"
	"go.uber.org/multierr"
)

// File is a temporary file
//
// It must be closed when done.
type File interface {
	io.Closer

	AbsPath() string
}

// NewFileWithData returns a new File.
func NewFileWithData(data []byte) (File, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	file, err := ioutil.TempFile("", id.String())
	if err != nil {
		return nil, err
	}
	path := file.Name()
	_, err = file.Write(data)
	err = multierr.Append(err, file.Close())
	if err != nil {
		return nil, multierr.Append(err, os.Remove(path))
	}
	// just in case
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, multierr.Append(err, os.Remove(path))
	}
	return newFile(absPath), nil
}

// Dir is a temporary directory.
//
// It must be closed when done.
type Dir interface {
	io.Closer

	AbsPath() string
}

// NewDir returns a new Dir.
func NewDir() (Dir, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	path, err := ioutil.TempDir("", id.String())
	if err != nil {
		return nil, err
	}
	// just in case
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, multierr.Append(err, os.RemoveAll(path))
	}
	return newDir(absPath), nil
}

type file struct {
	absPath string
}

func newFile(absPath string) *file {
	return &file{
		absPath: absPath,
	}
}

func (f *file) AbsPath() string {
	return f.absPath
}

func (f *file) Close() error {
	return os.Remove(f.absPath)
}

type dir struct {
	absPath string
}

func newDir(absPath string) *dir {
	return &dir{
		absPath: absPath,
	}
}

func (d *dir) AbsPath() string {
	return d.absPath
}

func (d *dir) Close() error {
	return os.RemoveAll(d.absPath)
}
