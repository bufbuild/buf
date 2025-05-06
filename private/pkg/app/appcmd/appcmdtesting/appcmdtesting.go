// Copyright 2020-2025 Buf Technologies, Inc.
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

// Package appcmdtesting contains test utilities for appcmd.
package appcmdtesting

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

// testingUse is the use string for test commands.
//
// We want to use something different than actual commands to make sure that all code is binary-name-agnostic.
const testingUse = "test"

// Run runs the command created by newCommand with the specified options.
func Run(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	options ...RunOption,
) {
	runOptions := newRunOptions()
	for _, option := range options {
		option(runOptions)
	}

	stdin := runOptions.stdin
	stdout := runOptions.stdout
	stdoutBuffer := bytes.NewBuffer(nil)
	if stdout == nil {
		stdout = stdoutBuffer
	} else {
		stdout = io.MultiWriter(stdout, stdoutBuffer)
	}
	stderr := runOptions.stderr
	stderrBuffer := bytes.NewBuffer(nil)
	if stderr == nil {
		stderr = stderrBuffer
	} else {
		stderr = io.MultiWriter(stderr, stderrBuffer)
	}
	var env map[string]string
	if runOptions.newEnv != nil {
		env = runOptions.newEnv(testingUse)
	}

	exitCode := app.GetExitCode(
		appcmd.Run(
			context.Background(),
			app.NewContainer(
				env,
				stdin,
				stdout,
				stderr,
				append([]string{testingUse}, runOptions.args...)...,
			),
			newCommand(testingUse),
		),
	)

	expectedExitCodes := runOptions.expectedExitCodes
	if len(expectedExitCodes) == 0 {
		// If no expectedExitCodes specified, we expect the 0 exit code.
		expectedExitCodes[0] = struct{}{}
	}
	var foundExpectedExitCode bool
	for expectedExitCode := range runOptions.expectedExitCodes {
		if exitCode == expectedExitCode {
			foundExpectedExitCode = true
			break
		}
	}
	if !foundExpectedExitCode {
		expectedExitCodesSlice := make([]int, 0, len(runOptions.expectedExitCodes))
		for expectedExitCode := range runOptions.expectedExitCodes {
			expectedExitCodesSlice = append(expectedExitCodesSlice, expectedExitCode)
		}
		sort.Ints(expectedExitCodesSlice)
		message := fmt.Sprintf("one of %v", expectedExitCodesSlice)
		if len(expectedExitCodesSlice) == 1 {
			message = strconv.Itoa(expectedExitCodesSlice[0])
		}
		require.True(
			t,
			false,
			"expected exit code %d to be %s\n:%s",
			exitCode,
			message,
			requireErrorMessage(runOptions.args, stdoutBuffer, stderrBuffer),
		)
	}

	if runOptions.expectedStdoutPresent {
		require.Equal(
			t,
			stringutil.TrimLines(runOptions.expectedStdout),
			stringutil.TrimLines(stdoutBuffer.String()),
			requireErrorMessage(runOptions.args, stdoutBuffer, stderrBuffer),
		)
	}
	if runOptions.expectedStderrPresent {
		require.Equal(
			t,
			stringutil.TrimLines(runOptions.expectedStderr),
			stringutil.TrimLines(stderrBuffer.String()),
			requireErrorMessage(runOptions.args, stdoutBuffer, stderrBuffer),
		)
	}
	if runOptions.expectedStderrPartialsPresent {
		if len(runOptions.expectedStderrPartials) == 0 {
			require.Empty(
				t,
				stderrBuffer.String(),
				"stderr was not empty:\n%s",
				requireErrorMessage(runOptions.args, stdoutBuffer, stderrBuffer),
			)
		}
		for _, expectedStderrPartial := range runOptions.expectedStderrPartials {
			require.Contains(
				t,
				stderrBuffer.String(),
				expectedStderrPartial,
				"stderr expected to contain %q:\n:%s",
				expectedStderrPartial,
				requireErrorMessage(runOptions.args, stdoutBuffer, stderrBuffer),
			)
		}
	}
}

// RunOption is a new option for Run.
type RunOption func(*runOptions)

// WithEnv will attach the given environment variable map created by newEnv.
//
// The default is no environment variables.
func WithEnv(newEnv func(use string) map[string]string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.newEnv = newEnv
	}
}

// WithStdin will attach the given stdin to read from.
func WithStdin(stdin io.Reader) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stdin = stdin
	}
}

// WithStdout will attach the given stdout to write to.
func WithStdout(stdout io.Writer) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stdout = stdout
	}
}

// WithStdout will attach the given stderr to write to.
func WithStderr(stderr io.Writer) RunOption {
	return func(runOptions *runOptions) {
		runOptions.stderr = stderr
	}
}

// WithArgs adds the given args.
func WithArgs(args ...string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.args = args
	}
}

// WithExpectedStdout will result in an error if the stdout does not exactly equal the given string.
//
// Note that this can be called with empty, which will result in Run verifying that the stdout is empty.
func WithExpectedStdout(expectedStdout string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.expectedStdout = expectedStdout
		runOptions.expectedStdoutPresent = true
	}
}

// WithExpectedStdout will result in an error if the stderr does not exactly equal the given string.
//
// Note that this can be called with empty, which will result in Run verifying that the stderr is empty.
func WithExpectedStderr(expectedStderr string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.expectedStderr = expectedStderr
		runOptions.expectedStderrPresent = true
	}
}

// WithExpectedStderrPartials will result in Run checking if all the given strings are contained within stderr.
//
// Note that this can be called with empty, which will result in Run verifying that the stderr is empty.
func WithExpectedStderrPartials(expectedStderrPartials ...string) RunOption {
	return func(runOptions *runOptions) {
		runOptions.expectedStderrPartials = expectedStderrPartials
		runOptions.expectedStderrPartialsPresent = true
	}
}

// WithExpectedExitCode will result in Run checking that the exit code is the expected value.
//
// By default, Run will check that the exit code is 0.
func WithExpectedExitCode(expectedExitCode int) RunOption {
	return func(runOptions *runOptions) {
		runOptions.expectedExitCodes[expectedExitCode] = struct{}{}
	}
}

// WithExpectedExitCodes will result in Run checking that the exit code is one of the expected values.
//
// By default, Run will check that the exit code is 0.
func WithExpectedExitCodes(expectedExitCodes ...int) RunOption {
	return func(runOptions *runOptions) {
		for _, expectedExitCode := range expectedExitCodes {
			runOptions.expectedExitCodes[expectedExitCode] = struct{}{}
		}
	}
}

// RunCommandSuccessStdout runs the command and makes sure it was successful, and compares the stdout output.
//
// Deprecated: Use Run instead.
func RunCommandSuccessStdout(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(0),
		WithStdin(stdin),
		WithExpectedStdout(expectedStdout),
		WithArgs(args...),
	)
}

// RunCommandExitCodeStdout runs the command and compares the exit code and stdout output.
//
// Deprecated: Use Run instead.
func RunCommandExitCodeStdout(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStdout string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(expectedExitCode),
		WithStdin(stdin),
		WithExpectedStdout(expectedStdout),
		WithArgs(args...),
	)
}

// RunCommandExitCodeStdoutFile runs the command and compares the exit code and stdout output.
//
// Deprecated: Use Run and open the file yourself instead.
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
//
// Deprecated: Use Run and open the file yourself instead.
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

// RunCommandExitCodeStderr runs the command and compares the exit code and stderr output.
//
// Deprecated: Use Run instead.
func RunCommandExitCodeStderr(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStderr string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(expectedExitCode),
		WithStdin(stdin),
		WithExpectedStderr(expectedStderr),
		WithArgs(args...),
	)
}

// RunCommandExitCodesStderr runs the command and compares the exit codes and stderr output.
//
// Deprecated: Use Run instead.
func RunCommandExitCodesStderr(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCodes []int,
	expectedStderr string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCodes(expectedExitCodes...),
		WithStdin(stdin),
		WithExpectedStderr(expectedStderr),
		WithArgs(args...),
	)
}

// RunCommandExitCodeStderrContains runs the command and compares the exit code and stderr output
// with the passed partial messages.
//
// Deprecated: Use Run instead.
func RunCommandExitCodeStderrContains(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCode int,
	expectedStderrPartials []string,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(expectedExitCode),
		WithStdin(stdin),
		WithExpectedStderrPartials(expectedStderrPartials...),
		WithArgs(args...),
	)
}

// RunCommandExitCodeStdoutStderr runs the command and compares the exit code, stdout, and stderr output.
//
// Deprecated: Use Run instead.
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
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(expectedExitCode),
		WithStdin(stdin),
		WithExpectedStdout(expectedStdout),
		WithExpectedStderr(expectedStderr),
		WithArgs(args...),
	)
}

// RunCommandSuccess runs the command and makes sure it was successful.
//
// Deprecated: Use Run instead.
func RunCommandSuccess(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(0),
		WithStdin(stdin),
		WithStdout(stdout),
		WithArgs(args...),
	)
}

// RunCommandExitCode runs the command and compares the exit code.
//
// Deprecated: Use Run instead.
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
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithExpectedExitCode(expectedExitCode),
		WithStdin(stdin),
		WithStdout(stdout),
		WithStderr(stderr),
		WithArgs(args...),
	)
}

// RunCommandExitCodes runs the command and compares the exit code to the expected
// exit codes.
//
// It would be nice if we could do:
//
//	type IntOrInts interface {
//	  int | []int
//	}
//
//	func RunCommandExitCode[I IntOrInts](expectedExitCode I)
//
// However we can't: https://github.com/golang/go/issues/49206
//
// Deprecated: Use Run instead.
func RunCommandExitCodes(
	t *testing.T,
	newCommand func(use string) *appcmd.Command,
	expectedExitCodes []int,
	newEnv func(use string) map[string]string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	args ...string,
) {
	Run(
		t,
		newCommand,
		WithEnv(newEnv),
		WithStdin(stdin),
		WithStdout(stdout),
		WithStderr(stderr),
		WithExpectedExitCodes(expectedExitCodes...),
		WithArgs(args...),
	)
}

// *** PRIVATE ***

func requireErrorMessage(args []string, stdout *bytes.Buffer, stderr *bytes.Buffer) string {
	return fmt.Sprintf(
		"args: %s\nstdout: %s\nstderr: %s",
		strings.Join(
			slicesext.Map(
				args,
				// To make the args copy-pastable.
				func(arg string) string {
					return `'` + arg + `'`
				},
			),
			" ",
		),
		stringutil.TrimLines(stdout.String()),
		stringutil.TrimLines(stderr.String()),
	)
}

type runOptions struct {
	newEnv                        func(use string) map[string]string
	stdin                         io.Reader
	stdout                        io.Writer
	stderr                        io.Writer
	args                          []string
	expectedStdout                string
	expectedStdoutPresent         bool
	expectedStderr                string
	expectedStderrPresent         bool
	expectedStderrPartials        []string
	expectedStderrPartialsPresent bool
	expectedExitCodes             map[int]struct{}
}

func newRunOptions() *runOptions {
	return &runOptions{
		expectedExitCodes: make(map[int]struct{}),
	}
}
