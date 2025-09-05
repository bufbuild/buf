// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufpolicyconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteBufPolicyYAMLFileRoundTrip(t *testing.T) {
	t.Parallel()

	testReadWriteBufPolicyYAMLFileRoundTrip(
		t,
		// input
		`version: v2
lint:
  disable_builtin: true
breaking:
  disable_builtin: true
`,
		// expected output
		`version: v2
lint:
  disable_builtin: true
breaking:
  disable_builtin: true
`,
	)

	testReadWriteBufPolicyYAMLFileRoundTrip(
		t,
		// input
		`version: v2
breaking:
  use:
    - FILE
  except:
    - FILE_NO_DELETE
  ignore_unstable_packages: true
lint:
  use:
    - DIRECTORY_SAME_PACKAGE
    - ENUM_FIRST_VALUE_ZERO
    - TIMESTAMP_SUFFIX
  enum_zero_value_suffix: _UNSPECIFIED
  rpc_allow_same_request_response: false
  rpc_allow_google_protobuf_empty_requests: false
  rpc_allow_google_protobuf_empty_responses: false
  service_suffix: Service
plugins:
  - plugin: plugin-timestamp-suffix
    options:
      # The TIMESTAMP_SUFFIX rule specified above allows the user to change the suffix by providing a
      # new value here.
      timestamp_suffix: _time
  - plugin: buf.build/bufbuild/buf-lint
`,
		// expected output
		`version: v2
lint:
  use:
    - DIRECTORY_SAME_PACKAGE
    - ENUM_FIRST_VALUE_ZERO
    - TIMESTAMP_SUFFIX
  enum_zero_value_suffix: _UNSPECIFIED
  service_suffix: Service
breaking:
  use:
    - FILE
  except:
    - FILE_NO_DELETE
  ignore_unstable_packages: true
plugins:
  - plugin: plugin-timestamp-suffix
    options:
      timestamp_suffix: _time
  - plugin: buf.build/bufbuild/buf-lint
`,
	)
}

func testReadBufPolicyYAMLFile(
	t *testing.T,
	inputBufPolicyYAMLFileData string,
) BufPolicyYAMLFile {
	bufPolicyYAMLFile, err := ReadBufPolicyYAMLFile(
		strings.NewReader(testCleanYAMLData(inputBufPolicyYAMLFileData)),
		"buf.policy.yaml",
	)
	require.NoError(t, err)
	return bufPolicyYAMLFile
}

func testReadWriteBufPolicyYAMLFileRoundTrip(
	t *testing.T,
	inputBufYAMLFileData string,
	expectedOutputBufYAMLFileData string,
) {
	bufPolicyYAMLFile := testReadBufPolicyYAMLFile(t, inputBufYAMLFileData)
	buffer := bytes.NewBuffer(nil)
	err := WriteBufPolicyYAMLFile(buffer, bufPolicyYAMLFile)
	require.NoError(t, err)
	outputBufPolicyYAMLData := testCleanYAMLData(buffer.String())
	assert.Equal(t, testCleanYAMLData(expectedOutputBufYAMLFileData), outputBufPolicyYAMLData, "output:\n%s", outputBufPolicyYAMLData)
}

func testCleanYAMLData(data string) string {
	// Just to deal with editor nonsense when writing tests.
	return strings.TrimSpace(strings.ReplaceAll(data, "\t", "  "))
}
