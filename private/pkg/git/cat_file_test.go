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
	"testing"

	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/git/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatFileIntegration(t *testing.T) {
	if testing.Short() {
		// This test spawns a live git-cat-file process.
		t.Skip("skipping git-cat-file integration test")
	}
	runner := command.NewRunner()
	conn, err := NewCatFile(runner)
	require.NoError(t, err)
	// This is the first commit in bufbuild/buf. It most certainly should
	// exist.
	firstCommit := mustID(t, "157c7ae554844ff7ae178536ec10787b5b74b5db")
	commit, err := conn.Commit(firstCommit)
	require.NoError(t, err)
	assert.Equal(t,
		mustID(t, "0760f36a308962f130706202101dfc86349df1df"),
		commit.Tree,
	)
	assert.Nil(t, commit.Parents)
	assert.Equal(t, "bufdev", commit.Author.Name)
	assert.Equal(t, "bufdev-github@buf.build", commit.Author.Email)
	assert.Equal(t, "Copy from internal\n", commit.Message)
	// And this is the second commit in bufbuild/buf.
	secondCommit := mustID(t, "a765578a6b69c391891a79cff85cba9bfa08d792")
	commit, err = conn.Commit(secondCommit)
	require.NoError(t, err)
	assert.Equal(t,
		mustID(t, "67563e7d3436f4a7ca5caff504350ac33dfc4a81"),
		commit.Tree,
	)
	assert.Equal(t, []object.ID{firstCommit}, commit.Parents)
	assert.Equal(t, "Samuel Vaillant", commit.Author.Name)
	assert.Equal(t, "samuel.vllnt@gmail.com", commit.Author.Email)
	assert.Equal(t,
		"docs(README): campatibility -> compatibility",
		commit.Message,
	)
	assert.NoError(t, conn.Close())
}

func mustID(t *testing.T, hexid string) (objID object.ID) {
	err := objID.UnmarshalText([]byte(hexid))
	require.NoError(t, err)
	return objID
}
