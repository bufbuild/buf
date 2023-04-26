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
	"io"

	"github.com/bufbuild/buf/private/pkg/command"
)

type catFileOptions struct {
	gitdir string
}

type CatFileOption interface {
	apply(*catFileOptions)
}

type gitdirOption string

func (gitdir gitdirOption) apply(o *catFileOptions) {
	o.gitdir = string(gitdir)
}

func CatFileGitDir(dir string) CatFileOption {
	return gitdirOption(dir)
}

// NewCatFile starts up a git-cat-file instance and returns a
// ObjectService for it. Call Close to stop the git-cat-file process.
func NewCatFile(
	runner command.Runner,
	options ...CatFileOption,
) (ObjectService, error) {
	var opts catFileOptions
	for _, opt := range options {
		opt.apply(&opts)
	}
	rx, stdout := io.Pipe()
	stdin, tx := io.Pipe()
	runOpts := []command.StartOption{
		command.StartWithArgs("cat-file", "--batch"),
		command.StartWithStdin(stdin),
		command.StartWithStdout(stdout),
	}
	if opts.gitdir != "" {
		runOpts = append(runOpts,
			command.StartWithEnv(map[string]string{
				"GIT_DIR": opts.gitdir,
			}),
		)
	}
	process, err := runner.Start(
		"git",
		runOpts...,
	)
	if err != nil {
		return nil, err
	}
	return newCatFileConnection(process, tx, rx), nil
}
