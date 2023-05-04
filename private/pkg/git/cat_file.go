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

package git

import (
	"fmt"
	"io"
	"os"

	"github.com/bufbuild/buf/private/pkg/command"
)

type catFileOptions struct {
	gitdir string
}

type CatFileOption interface {
	apply(*catFileOptions) error
}

type gitdirOption string

func (gitdir gitdirOption) apply(o *catFileOptions) error {
	dir := string(gitdir)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("git dir: %q does not exist", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("git dir: %q is not a directory", dir)
	}
	o.gitdir = dir
	return nil
}

func CatFileGitDir(dir string) CatFileOption {
	return gitdirOption(dir)
}

// [CatFile] is a handle to create [ObjectService] backed by git-cat-file(1).
type CatFile struct {
	runner command.Runner
	opts   *catFileOptions
}

// NewCatFile returns a [CatFile].
func NewCatFile(
	runner command.Runner,
	options ...CatFileOption,
) (*CatFile, error) {
	opts := &catFileOptions{}
	for _, opt := range options {
		if err := opt.apply(opts); err != nil {
			return nil, err
		}
	}
	return &CatFile{runner: runner, opts: opts}, nil
}

// Connect returns an ObjectService backed by git-cat-file(1).
func (cf *CatFile) Connect() (ObjectService, error) {
	// For now let's spawn a new git-cat-file on every connection request.
	// We can later manage a pool of them and route requests amongst them.
	rx, stdout := io.Pipe()
	stdin, tx := io.Pipe()
	runOpts := []command.StartOption{
		command.StartWithArgs("cat-file", "--batch"),
		command.StartWithStdin(stdin),
		command.StartWithStdout(stdout),
	}
	if cf.opts.gitdir != "" {
		runOpts = append(runOpts,
			command.StartWithEnv(map[string]string{
				"GIT_DIR": cf.opts.gitdir,
			}),
		)
	}
	process, err := cf.runner.Start(
		"git",
		runOpts...,
	)
	if err != nil {
		return nil, err
	}
	return newCatFileConnection(process, tx, rx), nil
}
