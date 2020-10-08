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

// Package passwordutil provides utilities to read passwords.
package passwordutil

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/app"
	"golang.org/x/crypto/ssh/terminal"
)

// ReadStdin reads the password from stdin.
func ReadStdin(container app.StdinContainer) (string, error) {
	value, err := ioutil.ReadAll(container.Stdin())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(value)), nil
}

// ReadInteractive reads a password from the container
// by prompting the user on the terminal connected to the containers Stdin.
// It doesn't echo the characters put into the terminal.
func ReadInteractive(container app.StdioContainer, prompt string) (string, error) {
	if _, err := container.Stdout().Write([]byte(prompt)); err != nil {
		return "", err
	}
	value, err := readLine(container.Stdin())
	if err != nil {
		return "", err
	}
	// Print newline after password entry to put next output on new line.
	if _, err := fmt.Fprintln(container.Stdout()); err != nil {
		return "", err
	}
	return strings.TrimSpace(string(value)), nil
}

// readLine provides a way to prompt a user for a password.
// It reads a line from the input without echoing it
// and strips the terminating \n.
//
// If the reader used is not a terminal, this will simply read until the next newline.
func readLine(reader io.Reader) ([]byte, error) {
	if file, ok := reader.(*os.File); ok && terminal.IsTerminal(int(file.Fd())) {
		return terminal.ReadPassword(int(file.Fd()))
	}
	// Fall back to reading until newline for non-terminals
	bufReader := bufio.NewReader(reader)
	read, _, err := bufReader.ReadLine()
	if err != nil {
		return nil, err
	}
	return read, nil
}
