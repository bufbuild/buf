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
	"strconv"
	"strings"
	"time"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git/object"
	"go.uber.org/multierr"
)

// exitTime is the amount of time we'll wait for git-cat-file(1) to exit.
var exitTime = 5 * time.Second

// catFileConection represents a git-cat-file(1)  process.
type catFileConnection struct {
	process command.Process
	tx      io.WriteCloser
	rx      *bufio.Reader
}

var _ ObjectService = (*catFileConnection)(nil)

func newCatFileConnection(
	process command.Process,
	tx io.WriteCloser,
	rx io.ReadCloser,
) *catFileConnection {
	return &catFileConnection{
		process: process,
		rx:      bufio.NewReader(rx),
		tx:      tx,
	}
}

func (c *catFileConnection) Commit(id object.ID) (*object.Commit, error) {
	objContent, err := c.object("commit", id)
	if err != nil {
		return nil, err
	}
	var commit object.Commit
	if err := commit.UnmarshalText(objContent); err != nil {
		return nil, err
	}
	return &commit, nil
}

func (c *catFileConnection) Tree(id object.ID) (*object.Tree, error) {
	objContent, err := c.object("tree", id)
	if err != nil {
		return nil, err
	}
	var tree object.Tree
	if err := tree.UnmarshalBinary(objContent); err != nil {
		return nil, err
	}
	return &tree, nil
}

func (c *catFileConnection) Blob(id object.ID) ([]byte, error) {
	return c.object("blob", id)
}

// object requests an object at id returning its type and content.
func (c *catFileConnection) object(typ string, id object.ID) ([]byte, error) {
	// request
	if _, err := fmt.Fprintf(c.tx, "%s\n", id); err != nil {
		return nil, err
	}
	// response
	header, err := c.rx.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	headerStr := strings.TrimRight(string(header), "\n")
	parts := strings.Split(headerStr, " ")
	if len(parts) == 2 && parts[1] == "missing" {
		return nil, fmt.Errorf(
			"git-cat-file: %w: %s", ErrObjectNotFound, parts[0],
		)
	}
	if len(parts) != 3 {
		return nil, fmt.Errorf("git-cat-file: malformed header: %q", headerStr)
	}
	var objID object.ID
	if err := objID.UnmarshalText([]byte(parts[0])); err != nil {
		return nil, err
	}
	objType := parts[1]
	objLenStr := parts[2]
	objLen, err := strconv.ParseInt(objLenStr, 10, 64)
	if err != nil {
		return nil, err
	}
	objContent := make([]byte, objLen)
	if _, err := io.ReadAtLeast(c.rx, objContent, int(objLen)); err != nil {
		return nil, err
	}
	// TODO: We can verify the object content if we move from opaque object IDs
	// to ones that know about being hardened SHA1 or SHA256.
	trailer, err := c.rx.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(trailer) != 1 {
		return nil, errors.New("git-cat-file: unexpected trailer")
	}
	// Check the response type. It's check here to consume the complete request
	// first.
	if objType != typ {
		return nil, fmt.Errorf(
			"git-cat-file: object %q is a %s, not a %s", id, objType, typ,
		)
	}
	return objContent, err
}

// Close shuts down cat-file and waits for it to exit.
func (c *catFileConnection) Close() error {
	ctx, cancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(exitTime),
	)
	defer cancel()
	return multierr.Combine(
		c.tx.Close(),
		c.process.Wait(ctx),
	)
}
