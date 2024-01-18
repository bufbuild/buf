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

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFileMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc      string
		mode      ObjectMode
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
			mode, err := parseObjectMode([]byte(test.txt))
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.mode, mode)
			}
		})
	}
}
