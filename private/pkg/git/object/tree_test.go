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

package object

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileModeUnmarshal(t *testing.T) {
	tests := []struct {
		desc      string
		mode      FileMode
		txt       string
		expectErr bool
	}{
		{
			desc:      "zero value",
			expectErr: true,
		},
		{
			desc: "file",
			mode: ModeFile,
			txt:  "100644",
		},
		{
			desc: "exe",
			mode: ModeExe,
			txt:  "100755",
		},
		{
			desc: "directory",
			mode: ModeDir,
			txt:  "040000",
		},
		{
			desc: "symlink",
			mode: ModeSymlink,
			txt:  "120000",
		},
		{
			desc: "submodule",
			mode: ModeSubmodule,
			txt:  "160000",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			var mode FileMode
			err := mode.UnmarshalText([]byte(test.txt))
			if test.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.mode, mode)
		})
	}
}

func TestTree(t *testing.T) {
	t.Run("empty fail", func(t *testing.T) {
		t.Parallel()
		var tree Tree
		err := tree.UnmarshalBinary([]byte{})
		assert.NoError(t, err)
		assert.Equal(t, 0, len(tree.Entries))
	})
	t.Run("one entry", func(t *testing.T) {
		t.Parallel()
		var tree Tree
		hash, encodedTree := genTree(t, 1)
		err := tree.UnmarshalBinary(encodedTree)
		assert.NoError(t, err)
		assert.Equal(t, []TreeEntry{
			{
				Name: "foo0",
				Mode: ModeFile,
				ID:   hash,
			},
		}, tree.Entries)
	})
	t.Run("two entries", func(t *testing.T) {
		t.Parallel()
		var tree Tree
		hash, encodedTree := genTree(t, 2)
		err := tree.UnmarshalBinary(encodedTree)
		assert.NoError(t, err)
		require.Equal(t, 2, len(tree.Entries))
		assert.Equal(t, []TreeEntry{
			{
				Name: "foo0",
				Mode: ModeFile,
				ID:   hash,
			},
			{
				Name: "foo1",
				Mode: ModeFile,
				ID:   hash,
			},
		}, tree.Entries)
	})
}

func genTree(t *testing.T, n int) (ID, []byte) {
	require.Less(t, n, math.MaxInt8)
	var hash ID
	err := hash.UnmarshalText(
		[]byte("7f8712b58dce376ac1c3ff234163ba59cf28a1f4"),
	)
	require.NoError(t, err)
	var entries []byte
	for i := 0; i < n; i++ {
		entry := []byte{
			// mode
			0x31, 0x30, 0x30, 0x36, 0x34, 0x34, 0x20,
			// name
			0x66, 0x6f, 0x6f, (byte('0') + byte(i)), 0x00,
			// digest
			0x7f, 0x87, 0x12, 0xb5, 0x8d, 0xce, 0x37, 0x6a, 0xc1, 0xc3, 0xff,
			0x23, 0x41, 0x63, 0xba, 0x59, 0xcf, 0x28, 0xa1, 0xf4,
		}
		entries = append(entries, entry...)
	}
	return hash, entries
}
