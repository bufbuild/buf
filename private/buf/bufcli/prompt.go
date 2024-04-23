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

package bufcli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"golang.org/x/term"
)

const userPromptAttempts = 3

// PromptUser reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUser(container app.Container, prompt string) (string, error) {
	return promptUser(container, prompt, false)
}

// PromptUserForPassword reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUserForPassword(container app.Container, prompt string) (string, error) {
	return promptUser(container, prompt, true)
}

// PromptUserForDelete is used to receive user confirmation that a specific
// entity should be deleted. If the user's answer does not match the expected
// answer, an error is returned.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func PromptUserForDelete(container app.Container, entityType string, expectedAnswer string) error {
	confirmation, err := PromptUser(
		container,
		fmt.Sprintf(
			"Please confirm that you want to DELETE this %s by entering its name (%s) again."+
				"\nWARNING: This action is NOT reversible!\n",
			entityType,
			expectedAnswer,
		),
	)
	if err != nil {
		if errors.Is(err, ErrNotATTY) {
			return errors.New("cannot perform an interactive delete from a non-TTY device")
		}
		return err
	}
	if confirmation != expectedAnswer {
		return fmt.Errorf(
			"expected %q, but received %q",
			expectedAnswer,
			confirmation,
		)
	}
	return nil
}

// promptUser reads a line from Stdin, prompting the user with the prompt first.
// The prompt is repeatedly shown until the user provides a non-empty response.
// ErrNotATTY is returned if the input containers Stdin is not a terminal.
func promptUser(container app.Container, prompt string, isPassword bool) (string, error) {
	file, ok := container.Stdin().(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return "", ErrNotATTY
	}
	var attempts int
	for attempts < userPromptAttempts {
		attempts++
		if _, err := fmt.Fprint(
			container.Stdout(),
			prompt,
		); err != nil {
			return "", syserror.Wrap(err)
		}
		var value string
		if isPassword {
			data, err := term.ReadPassword(int(file.Fd()))
			if err != nil {
				// If the user submitted an EOF (e.g. via ^D) then we
				// should not treat it as an internal error; returning
				// the error directly makes it more clear as to
				// why the command failed.
				if errors.Is(err, io.EOF) {
					return "", err
				}
				return "", syserror.Wrap(err)
			}
			value = string(data)
		} else {
			scanner := bufio.NewScanner(container.Stdin())
			if !scanner.Scan() {
				// scanner.Err() returns nil on EOF.
				if err := scanner.Err(); err != nil {
					return "", syserror.Wrap(err)
				}
				return "", io.EOF
			}
			value = scanner.Text()
			if err := scanner.Err(); err != nil {
				return "", syserror.Wrap(err)
			}
		}
		if len(strings.TrimSpace(value)) != 0 {
			// We want to preserve spaces in user input, so we only apply
			// strings.TrimSpace to verify an answer was provided.
			return value, nil
		}
		if attempts < userPromptAttempts {
			// We only want to ask the user to try again if they actually
			// have another attempt.
			if _, err := fmt.Fprintln(
				container.Stdout(),
				"No answer was provided. Please try again.",
			); err != nil {
				return "", syserror.Wrap(err)
			}
		}
	}
	return "", NewTooManyEmptyAnswersError(userPromptAttempts)
}
