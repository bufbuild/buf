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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"go.uber.org/multierr"
)

// exitTime is the amount of time we'll wait for git-cat-file(1) to exit.
var exitTime = 5 * time.Second

type objectReader struct {
	rx      *bufio.Reader
	tx      io.WriteCloser
	process command.Process
}

func newObjectReader(gitDirPath string, runner command.Runner) (*objectReader, error) {
	gitDirPath = normalpath.Unnormalize(gitDirPath)
	if err := validateDirPathExists(gitDirPath); err != nil {
		return nil, err
	}
	gitDirPath, err := filepath.Abs(gitDirPath)
	if err != nil {
		return nil, err
	}
	// TODO: do we need to validate `gitDirPath` is not jumping context?
	rx, stdout := io.Pipe()
	stdin, tx := io.Pipe()
	process, err := runner.Start(
		"git",
		command.StartWithArgs("cat-file", "--batch"),
		command.StartWithStdin(stdin),
		command.StartWithStdout(stdout),
		command.StartWithEnv(map[string]string{
			"GIT_DIR": gitDirPath,
		}),
	)
	if err != nil {
		return nil, err
	}
	return &objectReader{
		rx:      bufio.NewReader(rx),
		tx:      tx,
		process: process,
	}, nil
}

func (o *objectReader) Close() error {
	ctx, cancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(exitTime),
	)
	defer cancel()
	return multierr.Combine(
		o.tx.Close(),
		o.process.Wait(ctx),
	)
}

func (o *objectReader) Commit(id Hash) (Commit, error) {
	data, err := o.read("commit", id)
	if err != nil {
		return nil, err
	}
	return parseCommit(id, data)
}

func (o *objectReader) read(objectType string, id Hash) ([]byte, error) {
	// request
	if _, err := fmt.Fprintf(o.tx, "%s\n", id.Hex()); err != nil {
		return nil, err
	}
	// response
	header, err := o.rx.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	headerStr := strings.TrimRight(string(header), "\n")
	parts := strings.Split(headerStr, " ")
	if len(parts) == 2 && parts[1] == "missing" {
		return nil, fmt.Errorf(
			"git-cat-file: object not found: %s", parts[0],
		)
	}
	if len(parts) != 3 {
		return nil, fmt.Errorf("git-cat-file: malformed header: %q", headerStr)
	}
	objID, err := parseHashFromHex(parts[0])
	if err != nil {
		return nil, err
	}
	if id.Hex() != objID.Hex() {
		return nil, fmt.Errorf("git-cat-file: mismatched object ID: %s, %s", id.Hex(), objID.Hex())
	}
	objType := parts[1]
	objLenStr := parts[2]
	objLen, err := strconv.ParseInt(objLenStr, 10, 64)
	if err != nil {
		return nil, err
	}
	objContent := make([]byte, objLen)
	if _, err := io.ReadAtLeast(o.rx, objContent, int(objLen)); err != nil {
		return nil, err
	}
	// TODO: We can verify the object content if we move from opaque object IDs
	// to ones that know about being hardened SHA1 or SHA256.
	trailer, err := o.rx.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(trailer) != 1 {
		return nil, errors.New("git-cat-file: unexpected trailer")
	}
	// Check the response type. It's check here to consume the complete request
	// first.
	if objType != objectType {
		return nil, fmt.Errorf(
			"git-cat-file: object %q is a %s, not a %s", id, objType, objectType,
		)
	}
}
