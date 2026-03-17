// Copyright 2020-2026 Buf Technologies, Inc.
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

package refcount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	t.Parallel()

	table := &Map[string, int]{}

	value, found := table.Insert("foo")
	assert.Equal(t, 0, *value)
	assert.False(t, found)
	*value = 42

	value, found = table.Insert("foo")
	assert.Equal(t, 42, *value)
	assert.True(t, found)

	assert.Equal(t, 42, *table.Get("foo"))
	assert.Nil(t, table.Get("bar"))

	assert.Nil(t, table.Delete("foo"))
	assert.Equal(t, 42, *table.Delete("foo"))
}
