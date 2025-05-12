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

package bufconfig

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteBufLockFileRoundTrip(t *testing.T) {
	t.Parallel()

	testReadWriteBufLockFileRoundTrip(
		t,
		// input
		`version: v2
`,
		// expected output
		`version: v2
`,
	)

	testReadWriteBufLockFileRoundTrip(
		t,
		// input
		`version: v2
deps:
  - name: buf.testing/acme/date
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: b5:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
  - name: buf.testing/acme/extension
    commit: 6d880cc6cc8d4131bdb5a51399df8faf
    digest: b5:d2c1da8f8331c5c75b50549c79fc360394dedfb6a11f5381c4523592018964119f561088fc8aaddfc9f5773ba02692e6fd9661853450f76a3355dec62c1f57b4
plugins:
  - name: buf.testing/acme/plugin
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
policies:
  - name: buf.testing/acme/policy
    commit: b8488077ea6d4f6d9562a337b98259c8
    digest: o1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
`,
		// expected output
		`version: v2
deps:
  - name: buf.testing/acme/date
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: b5:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
  - name: buf.testing/acme/extension
    commit: 6d880cc6cc8d4131bdb5a51399df8faf
    digest: b5:d2c1da8f8331c5c75b50549c79fc360394dedfb6a11f5381c4523592018964119f561088fc8aaddfc9f5773ba02692e6fd9661853450f76a3355dec62c1f57b4
plugins:
  - name: buf.testing/acme/plugin
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
policies:
  - name: buf.testing/acme/policy
    commit: b8488077ea6d4f6d9562a337b98259c8
    digest: o1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
`,
	)

	// Test a buf.lock file with a local policy file with remote plugins.
	testReadWriteBufLockFileRoundTrip(
		t,
		// input
		`version: v2
policies:
  - name: buf.policy.yaml
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
  - name: buf.testing/acme/policy
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: o1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
`,
		// expected output
		`version: v2
policies:
  - name: buf.policy.yaml
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
  - name: buf.testing/acme/policy
    commit: ffded0b4cf6b47cab74da08d291a3c2f
    digest: o1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
    plugins:
      - name: buf.testing/acme/plugin
        commit: ffded0b4cf6b47cab74da08d291a3c2f
        digest: p1:24ed4f13925cf89ea0ae0127fa28540704c7ae14750af027270221b737a1ce658f8014ca2555f6f7fcd95ea84e071d33f37f86cc36d07fe0d0963329a5ec2462
`,
	)
}

func testReadBufLockFile(
	t *testing.T,
	inputBufLockFileData string,
) BufLockFile {
	bufLockFile, err := ReadBufLockFile(
		context.Background(),
		strings.NewReader(testCleanYAMLData(inputBufLockFileData)),
		DefaultBufLockFileName,
	)
	require.NoError(t, err)
	return bufLockFile
}

func testReadWriteBufLockFileRoundTrip(
	t *testing.T,
	inputBufLockFileData string,
	expectedOutputBufLockFileData string,
) {
	bufLockFile := testReadBufLockFile(t, inputBufLockFileData)
	buffer := bytes.NewBuffer(nil)
	err := WriteBufLockFile(buffer, bufLockFile)
	require.NoError(t, err)
	outputBufLockData := testCleanYAMLData(buffer.String())
	outputBufLockData = strings.TrimPrefix(outputBufLockData, "# Generated by buf. DO NOT EDIT.\n")
	assert.Equal(t, testCleanYAMLData(expectedOutputBufLockFileData), outputBufLockData, "output:\n%s", outputBufLockData)
}
