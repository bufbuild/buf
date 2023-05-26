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

package bufinit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReversePathTrieNode(t *testing.T) {
	t.Parallel()

	node := newReversePathTrieNode()
	node.Insert("a/b/c.proto")
	node.Insert("a/d/c.proto")
	directories, present := node.Get("c.proto")
	assert.True(t, present)
	assert.Equal(t, []string{"a/b", "a/d"}, directories)
	directories, present = node.Get("e/c.proto")
	assert.False(t, present)
	assert.Nil(t, directories)
	directories, present = node.Get("a/b/c.proto")
	assert.True(t, present)
	assert.Equal(t, []string{"."}, directories)
}
