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

package bufconfig

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteBufYAMLFileRoundTrip(t *testing.T) {
	t.Parallel()

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v1
`,
		// expected output
		`version: v1
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v1
lint:
  use:
    - DEFAULT
  allow_comment_ignores: true
`,
		// expected output
		`version: v1
lint:
  use:
    - DEFAULT
  allow_comment_ignores: true
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		//input
		`version: v1
build:
  excludes:
    - tests
`,
		// expected output
		`version: v1
build:
  excludes:
    - tests
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		//input
		strings.Join(
			[]string{
				fmt.Sprintf(docsLinkComment, "v1"),
				`version: v1
build:
  excludes:
    - tests
`,
			},
			"\n",
		),
		// expected output
		strings.Join(
			[]string{
				fmt.Sprintf(docsLinkComment, "v1"),
				`version: v1
build:
  excludes:
    - tests
`,
			},
			"\n",
		),
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
lint:
  use:
    - DEFAULT
  disallow_comment_ignores: true
modules:
  - path: .
`,
		// expected output
		`version: v2
lint:
  use:
    - DEFAULT
  disallow_comment_ignores: true
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: foo
    lint:
      use:
        - DEFAULT
  - path: bar
    lint:
      use:
        - DEFAULT
`,
		// expected output
		`version: v2
modules:
  - path: bar
  - path: foo
lint:
  use:
    - DEFAULT
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
lint:
  use:
    - DEFAULT
  ignore:
    - foo/one/one.proto
    - bar/two/two.proto
modules:
  - path: foo
  - path: bar
`,
		// expected output
		`version: v2
modules:
  - path: bar
    lint:
      use:
        - DEFAULT
      ignore:
        - bar/two/two.proto
  - path: foo
    lint:
      use:
        - DEFAULT
      ignore:
        - foo/one/one.proto
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
lint:
  use:
    - DEFAULT
  ignore:
    - foo/one/one.proto
    - bar/two/two.proto
modules:
  - path: foo
    lint:
      use:
        - BASIC
  - path: bar
`,
		// expected output
		`version: v2
modules:
  - path: bar
    lint:
      use:
        - DEFAULT
      ignore:
        - bar/two/two.proto
  - path: foo
    lint:
      use:
        - BASIC
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: .
`,
		// expected output
		`version: v2
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
`,
		// expected output
		`version: v2
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: .
`,
		// expected output
		`version: v2
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
name: buf.build/foo/bar
`,
		// expected output
		`version: v2
name: buf.build/foo/bar
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: .
    name: buf.build/foo/bar
`,
		// expected output
		`version: v2
name: buf.build/foo/bar
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: .
    name: buf.build/foo/bar
`,
		// expected output
		`version: v2
name: buf.build/foo/bar
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: .
    excludes:
	  - foo
`,
		// expected output
		`version: v2
modules:
  - path: .
    excludes:
      - foo
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		`version: v2
modules:
  - path: baz
    name: buf.build/foo/baz
  - path: bar
    name: buf.build/foo/bar
`,
		// expected output
		`version: v2
modules:
  - path: bar
    name: buf.build/foo/bar
  - path: baz
    name: buf.build/foo/baz
`,
	)

	testReadWriteBufYAMLFileRoundTrip(
		t,
		// input
		strings.Join(
			[]string{
				fmt.Sprintf(docsLinkComment, "v2"),
				`version: v2
modules:
  - path: baz
    name: buf.build/foo/baz
  - path: bar
    name: buf.build/foo/bar
`,
			},
			"\n",
		),
		// expected output
		strings.Join(
			[]string{
				fmt.Sprintf(docsLinkComment, "v2"),
				`version: v2
modules:
  - path: bar
    name: buf.build/foo/bar
  - path: baz
    name: buf.build/foo/baz
`,
			},
			"\n",
		),
	)
}

func TestBufYAMLFileLintDisabled(t *testing.T) {
	t.Parallel()

	bufYAMLFile := testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
`,
	)
	moduleConfig0 := bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 := bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.LintConfig().Disabled())
	require.False(t, moduleConfig1.LintConfig().Disabled())

	bufYAMLFile = testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
lint:
  ignore:
    - vendor
`,
	)
	moduleConfig0 = bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 = bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.LintConfig().Disabled())
	require.True(t, moduleConfig1.LintConfig().Disabled())

	bufYAMLFile = testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
    lint:
      ignore:
        - vendor
`,
	)
	moduleConfig0 = bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 = bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.LintConfig().Disabled())
	require.True(t, moduleConfig1.LintConfig().Disabled())
}

func TestBufYAMLFileBreakingDisabled(t *testing.T) {
	t.Parallel()

	bufYAMLFile := testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
`,
	)
	moduleConfig0 := bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 := bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.BreakingConfig().Disabled())
	require.False(t, moduleConfig1.BreakingConfig().Disabled())

	bufYAMLFile = testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
breaking:
  ignore:
    - vendor
`,
	)
	moduleConfig0 = bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 = bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.BreakingConfig().Disabled())
	require.True(t, moduleConfig1.BreakingConfig().Disabled())

	bufYAMLFile = testReadBufYAMLFile(
		t,
		`version: v2
modules:
  - path: proto
  - path: vendor
    breaking:
      ignore:
        - vendor
`,
	)
	moduleConfig0 = bufYAMLFile.ModuleConfigs()[0]
	moduleConfig1 = bufYAMLFile.ModuleConfigs()[1]
	require.Equal(t, moduleConfig0.DirPath(), "proto")
	require.Equal(t, moduleConfig1.DirPath(), "vendor")
	require.False(t, moduleConfig0.BreakingConfig().Disabled())
	require.True(t, moduleConfig1.BreakingConfig().Disabled())
}

func testReadWriteBufYAMLFileRoundTrip(
	t *testing.T,
	inputBufYAMLFileData string,
	expectedOutputBufYAMLFileData string,
) {
	bufYAMLFile := testReadBufYAMLFile(t, inputBufYAMLFileData)
	buffer := bytes.NewBuffer(nil)
	err := WriteBufYAMLFile(buffer, bufYAMLFile)
	require.NoError(t, err)
	outputBufYAMLData := testCleanYAMLData(buffer.String())
	assert.Equal(t, testCleanYAMLData(expectedOutputBufYAMLFileData), outputBufYAMLData, "output:\n%s", outputBufYAMLData)
}

func testReadBufYAMLFile(
	t *testing.T,
	inputBufYAMLFileData string,
) BufYAMLFile {
	bufYAMLFile, err := ReadBufYAMLFile(
		strings.NewReader(testCleanYAMLData(inputBufYAMLFileData)),
		DefaultBufYAMLFileName,
	)
	require.NoError(t, err)
	return bufYAMLFile
}

func testCleanYAMLData(data string) string {
	// Just to deal with editor nonsense when writing tests.
	return strings.TrimSpace(strings.ReplaceAll(data, "\t", "  "))
}
