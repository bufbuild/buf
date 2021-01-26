// Copyright 2020-2021 Buf Technologies, Inc.
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

// Package main implements a file lister for git that lists unstaged files.
//
// This does not list unstaged deleted files, and does list unignored files that are not added.
// This ignores non-regular files.
//
// This is used for situations like license headers where we want all the potential git files
// during development
package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
)

func main() {
	app.Main(context.Background(), run)
}

func run(ctx context.Context, container app.Container) error {
	lsFilesOutput, err := execCommandStdout(
		ctx,
		container,
		"git",
		append(
			[]string{
				"ls-files",
			},
			app.Args(container)[1:]...,
		)...,
	)
	if err != nil {
		return err
	}
	lsFilesOthersOutput, err := execCommandStdout(
		ctx,
		container,
		"git",
		append(
			[]string{
				"ls-files",
				"--exclude-standard",
				"--others",
			},
			app.Args(container)[1:]...,
		)...,
	)
	if err != nil {
		return err
	}

	var results []string
	for _, filePath := range stringutil.SliceToUniqueSortedSlice(
		append(
			strings.Split(lsFilesOutput, "\n"),
			strings.Split(lsFilesOthersOutput, "\n")...,
		),
	) {
		if filePath := strings.TrimSpace(filePath); filePath != "" {
			if fileInfo, err := os.Stat(filePath); err == nil && fileInfo.Mode().IsRegular() {
				results = append(results, filePath)
			}
		}
	}
	if len(results) > 0 {
		if _, err := container.Stdout().Write([]byte(strings.Join(results, "\n") + "\n")); err != nil {
			return err
		}
	}
	return nil
}

func execCommandStdout(
	ctx context.Context,
	container app.EnvStdioContainer,
	name string,
	args ...string,
) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := execCommand(ctx, container, buffer, name, args...); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func execCommand(
	ctx context.Context,
	container app.EnvStdioContainer,
	stdout io.Writer,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = app.Environ(container)
	cmd.Stdin = container.Stdin()
	cmd.Stdout = stdout
	cmd.Stderr = container.Stderr()
	return cmd.Run()
}
