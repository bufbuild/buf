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

package appcmdtesting

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RunCommandSuccessStdout runs the command and makes sure it was successful, and compares the stdout output.
func RunCommandSuccessStdout(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	RunCommandExitCodeStdout(t, newCommand, 0, expectedStdout, newEnv, stdin, args...)
}

// RunCommandExitCodeStdout runs the command and compares the exit code and stdout output.
func RunCommandExitCodeStdout(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	RunCommandExitCode(t, newCommand, expectedExitCode, newEnv, stdin, stdout, stderr, args...)
	require.Equal(t, stringutil.TrimLines(expectedStdout), stringutil.TrimLines(stdout.String()))
}

// RunCommandExitCodeStdoutFile runs the command and compares the exit code and stdout output.
func RunCommandExitCodeStdoutFile(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	file, err := os.Open(expectedStdout)
	require.NoError(t, err)
	expectedstdoutConts, err := io.ReadAll(file)
	require.NoError(t, err)
	RunCommandExitCodeStdout(t, newCommand, expectedExitCode, string(expectedstdoutConts), newEnv, stdin, args...)
}

// RunCommandExitCodeStdoutStdinFile runs the command and allows a stdinFile to be opened and piped into the command.
func RunCommandExitCodeStdoutStdinFile(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdinFile string,
	args ...string,
) {
	stdin, err := os.Open(stdinFile)
	require.NoError(t, err)
	RunCommandExitCodeStdout(t, newCommand, expectedExitCode, expectedStdout, newEnv, stdin, args...)
}

// RunCommandExitCodeStderrContains runs the command and compares the exit code and stderr output
// with the passed partial messages.
func RunCommandExitCodeStderrContains(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStderrPartials []string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	RunCommandExitCode(t, newCommand, expectedExitCode, newEnv, stdin, stdout, stderr, args...)
	allStderr := stderr.String()
	for _, expectedPartial := range expectedStderrPartials {
		assert.Contains(t, allStderr, expectedPartial)
	}
}

// RunCommandExitCodeStdoutStderr runs the command and compares the exit code, stdout, and stderr output.
func RunCommandExitCodeStdoutStderr(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	expectedStderr string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	RunCommandExitCode(t, newCommand, expectedExitCode, newEnv, stdin, stdout, stderr, args...)
	require.Equal(t, stringutil.TrimLines(expectedStdout), stringutil.TrimLines(stdout.String()))
	require.Equal(t, stringutil.TrimLines(expectedStderr), stringutil.TrimLines(stderr.String()))
}

// RunCommandSuccess runs the command and makes sure it was successful.
func RunCommandSuccess(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	args ...string,
) {
	stderr := bytes.NewBuffer(nil)
	RunCommandExitCode(t, newCommand, 0, newEnv, stdin, stdout, stderr, args...)
}

// RunCommandExitCode runs the command and compares the exit code.
func RunCommandExitCode(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	args ...string,
) {
	// make the use something different than the actual command
	// to make sure that all code is binary-name-agnostic.
	use := "test"
	stdoutCopy := bytes.NewBuffer(nil)
	if stdout == nil {
		stdout = stdoutCopy
	} else {
		stdout = io.MultiWriter(stdout, stdoutCopy)
	}
	stderrCopy := bytes.NewBuffer(nil)
	if stderr == nil {
		stderr = stderrCopy
	} else {
		stderr = io.MultiWriter(stderr, stderrCopy)
	}
	var env map[string]string
	if newEnv != nil {
		env = newEnv(use)
	}
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
			newCommand(use),
		),
	)
	require.Equal(
		t,
		expectedExitCode,
		exitCode,
		stringutil.TrimLines(stdoutCopy.String())+"\n"+stringutil.TrimLines(stderrCopy.String()),
	)
}
