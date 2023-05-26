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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCommit(t *testing.T) {
	t.Parallel()
	hash, err := parseHashFromHex("43848150a6f5f6d76eeef6e0f69eb46290eefab6")
	require.NoError(t, err)
	commit, err := parseCommit(
		hash,
		[]byte(`tree 5edab9f970913225f985d9673ac19d61d36f0942
parent aa4f1392d3ee58eacc4c34badd506d83239669ca
author Bob <bob@buf.build> 1680571785 -0700
committer Alice <alice@buf.build> 1680636827 -0700

Hello World
`))
	require.NoError(t, err)
	assert.Equal(t,
		"43848150a6f5f6d76eeef6e0f69eb46290eefab6",
		commit.Hash().String(),
	)
	assert.Equal(t,
		"5edab9f970913225f985d9673ac19d61d36f0942",
		commit.Tree().String(),
	)
	require.Equal(t, 1, len(commit.Parents()))
	assert.Equal(t,
		"aa4f1392d3ee58eacc4c34badd506d83239669ca",
		commit.Parents()[0].String(),
	)
	assert.Equal(t,
		"Bob",
		commit.Author().Name(),
	)
	assert.Equal(t,
		"bob@buf.build",
		commit.Author().Email(),
	)
	assert.Equal(t,
		int64(1680571785),
		commit.Author().Timestamp().Unix(),
		"Bob commit time",
	)
	assert.Equal(t,
		"Alice",
		commit.Committer().Name(),
	)
	assert.Equal(t,
		"alice@buf.build",
		commit.Committer().Email(),
	)
	assert.Equal(t,
		int64(1680636827),
		commit.Committer().Timestamp().Unix(),
		"Alice commit time",
	)
	assert.Equal(t,
		"Hello World",
		commit.Message(),
	)
}
