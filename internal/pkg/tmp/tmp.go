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

	"github.com/bufbuild/buf/internal/pkg/interrupt"
	"github.com/bufbuild/buf/internal/pkg/uuid"
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
//
// This file will be deleted on interrupt signals.
func NewFileWithData(data []byte) (File, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, err
	}
	file, err := ioutil.TempFile("", id.String())
	if err != nil {
		return nil, err
	}
	path := file.Name()
	// just in case
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	signalC, closer := interrupt.NewSignalChannel()
	go func() {
		<-signalC
		_ = os.Remove(absPath)
	}()
	_, err = file.Write(data)
	err = multierr.Append(err, file.Close())
	if err != nil {
		err = multierr.Append(err, os.Remove(absPath))
		closer()
		return nil, err
	}
	return newFile(absPath, closer), nil
}

// Dir is a temporary directory.
//
// It must be closed when done.
type Dir interface {
	io.Closer

	AbsPath() string
}

// NewDir returns a new Dir.
//
// baseDirPath can be empty, in which case os.TempDir() is used.
// This directory will be deleted on interrupt signals.
func NewDir(baseDirPath string) (Dir, error) {
	id, err := uuid.New()
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
		return nil, err
	}
	signalC, closer := interrupt.NewSignalChannel()
	go func() {
		<-signalC
		_ = os.RemoveAll(absPath)
	}()
	return newDir(absPath, closer), nil
}

type file struct {
	absPath string
	closer  func()
}

func newFile(absPath string, closer func()) *file {
	return &file{
		absPath: absPath,
		closer:  closer,
	}
}

func (f *file) AbsPath() string {
	return f.absPath
}

func (f *file) Close() error {
	err := os.Remove(f.absPath)
	f.closer()
	return err
}

type dir struct {
	absPath string
	closer  func()
}

func newDir(absPath string, closer func()) *dir {
	return &dir{
		absPath: absPath,
		closer:  closer,
	}
}

func (d *dir) AbsPath() string {
	return d.absPath
}

func (d *dir) Close() error {
	err := os.RemoveAll(d.absPath)
	d.closer()
	return err
}
