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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
)

type obj struct {
	typ     string
	content []byte
}

type mockCatFile struct {
	ToStdin    io.WriteCloser
	FromStdout io.ReadCloser

	stdin      io.ReadCloser
	stdout     io.WriteCloser
	responses  map[string]obj
	terminated bool
}

func newMockCatFile(
	responses map[string]obj,
) *mockCatFile {
	stdin, toStdin := io.Pipe()
	fromStdout, stdout := io.Pipe()
	m := &mockCatFile{
		ToStdin:    toStdin,
		FromStdout: fromStdout,
		stdin:      stdin,
		stdout:     stdout,
		responses:  responses,
	}
	go m.run()
	return m
}

func (m *mockCatFile) run() {
	reader := bufio.NewReader(m.stdin)
	rawline, err := reader.ReadBytes('\n')
	for err == nil {
		line := strings.TrimRight(string(rawline), "\n")
		if resp, ok := m.responses[line]; ok {
			fmt.Fprintf(m.stdout, "%s %s %d\n",
				line, resp.typ, len(resp.content))
			respReader := bytes.NewReader(resp.content)
			if _, err := io.Copy(m.stdout, respReader); err != nil {
				return
			}
			if _, err := m.stdout.Write([]byte("\n")); err != nil {
				return
			}
		} else {
			fmt.Fprintf(m.stdout, "%s missing\n", line)
		}
		rawline, err = reader.ReadBytes('\n')
	}
}

func (m *mockCatFile) Run(_ context.Context) error {
	return errors.New("unsupported")
}

func (m *mockCatFile) Start() error {
	return errors.New("unsupported")
}

func (m *mockCatFile) Wait(_ context.Context) error {
	m.terminated = true
	return multierr.Append(
		m.ToStdin.Close(),
		m.FromStdout.Close(),
	)
}

func TestCatFileConnection(t *testing.T) {
	t.Parallel()
	catfile := newMockCatFile(map[string]obj{
		"11223344": {typ: "commit", content: []byte("\nhello")},
		"55667788": {typ: "tree"}, // no content is a valid tree
		"99aabbcc": {typ: "blob"}, // no content is a valid blob
	})
	conn := newCatFileConnection(catfile, catfile.ToStdin, catfile.FromStdout)
	defer conn.Close()
	commit, err := conn.Commit(object.ID([]byte{0x11, 0x22, 0x33, 0x44}))
	assert.NoError(t, err)
	assert.NotNil(t, commit)
	assert.Equal(t, object.Commit{Message: "hello"}, *commit)
	tree, err := conn.Tree(object.ID([]byte{0x55, 0x66, 0x77, 0x88}))
	assert.NoError(t, err)
	assert.NotNil(t, tree)
	blob, err := conn.Blob(object.ID([]byte{0x99, 0xaa, 0xbb, 0xcc}))
	assert.NoError(t, err)
	assert.NotNil(t, blob)
}

func TestCatFileConnectionIncorrectObjectType(t *testing.T) {
	t.Parallel()
	catfile := newMockCatFile(map[string]obj{
		"55667788": {typ: "tree", content: []byte("")},
	})
	conn := newCatFileConnection(catfile, catfile.ToStdin, catfile.FromStdout)
	defer conn.Close()
	_, err := conn.Commit(object.ID([]byte{0x55, 0x66, 0x77, 0x88}))
	assert.Error(t, err)
}

func TestCatFileConnectionTermination(t *testing.T) {
	t.Parallel()
	catfile := newMockCatFile(map[string]obj{})
	conn := newCatFileConnection(catfile, catfile.ToStdin, catfile.FromStdout)
	conn.Close()
	assert.True(t, catfile.terminated)
}
