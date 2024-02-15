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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
`,
		// expected output
		`version: v1
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
modules:
  - path: .
`,
		// expected output
		`version: v2
modules:
  - path: .
lint:
  use:
    - DEFAULT
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
}

func testReadWriteBufYAMLFileRoundTrip(
	t *testing.T,
	inputBufYAMLFileData string,
	expectedOutputBufYAMLFileData string,
) {
	inputBufYAMLFileData = testCleanYAMLData(inputBufYAMLFileData)
	expectedOutputBufYAMLFileData = testCleanYAMLData(expectedOutputBufYAMLFileData)

	bufYAMLFile, err := ReadBufYAMLFile(
		strings.NewReader(inputBufYAMLFileData),
		DefaultBufYAMLFileName,
	)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	buffer := bytes.NewBuffer(nil)
	err = WriteBufYAMLFile(buffer, bufYAMLFile)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	outputBufYAMLData := testCleanYAMLData(string(buffer.Bytes()))
	assert.Equal(t, expectedOutputBufYAMLFileData, outputBufYAMLData, "output:\n%s", outputBufYAMLData)
}

func testCleanYAMLData(data string) string {
	// Just to deal with editor nonsense when writing tests.
	return strings.TrimSpace(strings.ReplaceAll(data, "\t", "  "))
}
