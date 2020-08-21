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

package appcmdtesting

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

// RunCommandSuccessStdout runs the command and makes sure it was successful, and compares the stdout output.
func RunCommandSuccessStdout(
	t *testing.T,
	newCommand func(string) *appcmd.Command,
	expectedStdout string,
	env map[string]string,
	stdin io.Reader,
	args ...string,
) {
	t.Helper()
	RunCommandExitCodeStdout(t, newCommand, 0, expectedStdout, env, stdin, args...)
}

// RunCommandExitCodeStdout runs the command and compares the exit code and stdout output.
func RunCommandExitCodeStdout(
	t *testing.T,
	newCommand func(string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	env map[string]string,
	stdin io.Reader,
	args ...string,
) {
	t.Helper()
	stdout := bytes.NewBuffer(nil)
	RunCommandExitCode(t, newCommand, expectedExitCode, env, stdin, stdout, args...)
	require.Equal(t, stringutil.TrimLines(expectedStdout), stringutil.TrimLines(stdout.String()))
}

// RunCommandSuccess runs the command and makes sure it was successful.
func RunCommandSuccess(
	t *testing.T,
	newCommand func(string) *appcmd.Command,
	env map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	args ...string,
) {
	t.Helper()
	RunCommandExitCode(t, newCommand, 0, env, stdin, stdout, args...)
}

// RunCommandExitCode runs the command and compares the exit code.
func RunCommandExitCode(
	t *testing.T,
	newCommand func(string) *appcmd.Command,
	expectedExitCode int,
	env map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	args ...string,
) {
	t.Helper()
	stderr := bytes.NewBuffer(nil)
	exitCode := app.GetExitCode(
		appcmd.Run(
			context.Background(),
			app.NewContainer(
				env,
				stdin,
				stdout,
				stderr,
				append([]string{"test"}, args...)...,
			),
			newCommand("test"),
		),
	)
	require.Equal(t, expectedExitCode, exitCode, stringutil.TrimLines(stderr.String()))
}
