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

// Package tmp provides temporary files and directories.
//
// Usage of this package requires eng approval - ask before using.
package tmp

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"go.uber.org/multierr"
)

// File is a temporary file or directory.
//
// It must be closed when done.
type File interface {
	io.Closer

	Path() string
}

// NewFile returns a new temporary file with the data copied from the Reader.
//
// It must be closed when done. This deletes this file.
// This file will be automatically closed on context cancellation.
//
// Usage of this function requires eng approval - ask before using.
func NewFile(ctx context.Context, reader io.Reader) (File, error) {
	id, err := uuidutil.New()
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp("", id.String())
	if err != nil {
		return nil, err
	}
	path := file.Name()
	// just in case
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	closer := func() error { return os.Remove(absPath) }
	go func() {
		<-ctx.Done()
		_ = closer()
	}()
	_, err = io.Copy(file, reader)
	err = multierr.Append(err, file.Close())
	if err != nil {
		err = multierr.Append(err, closer())
		return nil, err
	}
	return newFile(closerFunc(closer), absPath), nil
}

// NewDir returns a new temporary directory.
//
// It must be closed when done. This deletes this directory and all its contents.
// This directory will be automatically closed on context cancellation.
//
// Usage of this function requires eng approval - ask before using.
func NewDir(ctx context.Context, options ...DirOption) (File, error) {
	dirOptions := newDirOptions()
	for _, option := range options {
		option(dirOptions)
	}
	id, err := uuidutil.New()
	if err != nil {
		return nil, err
	}
	path, err := os.MkdirTemp(dirOptions.basePath, id.String())
	if err != nil {
		return nil, err
	}
	// just in case
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	closer := func() error { return os.RemoveAll(absPath) }
	go func() {
		<-ctx.Done()
		_ = closer()
	}()
	return newFile(closerFunc(closer), absPath), nil
}

// DirOption is an option for NewDir.
type DirOption func(*dirOptions)

// DirWithBasePath returns a new DirOption that sets the base path to create
// the temporary directory in.
//
// The default is to use os.TempDir().
func DirWithBasePath(basePath string) DirOption {
	return func(dirOptions *dirOptions) {
		dirOptions.basePath = basePath
	}
}

type file struct {
	io.Closer

	path string
}

func newFile(closer io.Closer, path string) *file {
	return &file{
		Closer: closer,
		path:   path,
	}
}

func (f *file) Path() string {
	return f.path
}

type closerFunc func() error

func (c closerFunc) Close() error {
	return c()
}

type dirOptions struct {
	basePath string
}

func newDirOptions() *dirOptions {
	return &dirOptions{}
}
