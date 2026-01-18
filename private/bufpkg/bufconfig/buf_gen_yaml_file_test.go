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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteBufGenYAMLFileRoundTrip(t *testing.T) {
	t.Parallel()

	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v1
plugins:
  - plugin: go
    out: gen/go
    opt: paths=source_relative
    path: custom-gen-go
    strategy: directory
  - plugin: java
    out: gen/java
  - plugin: buf.build/protocolbuffers/python:v21.9
    out: gen/python
`,
		// expected output
		`version: v2
plugins:
  - local: custom-gen-go
    out: gen/go
    opt: paths=source_relative
    strategy: directory
  - protoc_builtin: java
    out: gen/java
  - remote: buf.build/protocolbuffers/python:v21.9
    out: gen/python
`,
	)
	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v2
`,
		// expected output
		`version: v2
`,
	)
	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v2
plugins:
  - local: ["go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go"]
    out: gen/proto
`,
		// expected output
		`version: v2
plugins:
  - local:
      - go
      - run
      - google.golang.org/protobuf/cmd/protoc-gen-go
    out: gen/proto
`,
	)
	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v2
plugins:
  - local:
      - go
      - run
      - google.golang.org/protobuf/cmd/protoc-gen-go
    out: gen/proto
`,
		// expected output
		`version: v2
plugins:
  - local:
      - go
      - run
      - google.golang.org/protobuf/cmd/protoc-gen-go
    out: gen/proto
`,
	)
	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v2
clean: true
plugins:
  - local: custom-gen-go
    out: gen/go
    opt: paths=source_relative
    strategy: directory
`,
		// expected output
		`version: v2
clean: true
plugins:
  - local: custom-gen-go
    out: gen/go
    opt: paths=source_relative
    strategy: directory
`,
	)
	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input
		`version: v2
managed:
  disable:
    - module: buf.build/googleapis/googleapis
    - path: foo/v1
    - file_option: csharp_namespace
    - field_option: jstype
    - module: buf.build/acme/weather
      path: foo/v1
      file_option: java_package
    - module: buf.build/acme/petapis
      field: foo.bar.Baz.field_name
      path: foo/v1
      field_option: jstype
  override:
    - file_option: java_package_prefix
      value: net
    - file_option: java_package_prefix
      module: buf.build/acme/petapis
      value: com
    - file_option: java_package_suffix
      module: buf.build/acme/petapis
      value: com
    - file_option: java_package
      path: foo/bar/baz.proto
      value: com.x.y.z
    - field_option: jstype
      value: JS_NORMAL
      module: buf.build/acme/paymentapis
    - field_option: jstype
      value: JS_STRING
      field: package1.Message2.field3
plugins:
  - remote: buf.build/protocolbuffers/go
    revision: 1
    out: gen/proto
  - protoc_builtin: cpp
    protoc_path: /path/to/protoc
    out: gen/proto
  - local: protoc-gen-validate
    out: gen/proto
  - local: path/to/protoc-gen-validate
    out: gen/proto
  - local: /usr/bin/path/to/protoc-gen-validate
    out: gen/proto2
  - local: ["go", "run", "google.golang.org/protobuf/cmd/protoc-gen-go"]
    out: gen/proto
    opt:
      - paths=source_relative
      - foo=bar
      - baz
    strategy: all
    include_imports: true
    include_wkt: true
    types:
      - "foo.v1.User"
    exclude_types:
      - buf.validate.oneof
      - buf.validate.message
      - buf.validate.field
inputs:
  - git_repo: github.com/acme/weather
    branch: dev
    subdir: proto
    depth: 30
  - module: buf.build/acme/weather
    types:
      - "foo.v1.User"
      - "foo.v1.UserService"
    exclude_types:
      - buf.validate.oneof
    paths:
      - a/b/c
      - a/b/d
    exclude_paths:
      - a/b/c/x.proto
      - a/b/d/y.proto
  - directory: x/y/z
  - tarball: a/b/x.tar.gz
  - tarball: c/d/x.tar.zst
    compression: zstd
    strip_components: 2
    subdir: proto
  - zip_archive: https://github.com/googleapis/googleapis/archive/master.zip
    strip_components: 1
  - proto_file: foo/bar/baz.proto
    include_package_files: true
  - binary_image: image.binpb.gz
    compression: gz
`,
		// expected output
		`version: v2
managed:
  disable:
    - module: buf.build/googleapis/googleapis
    - path: foo/v1
    - file_option: csharp_namespace
    - field_option: jstype
    - file_option: java_package
      module: buf.build/acme/weather
      path: foo/v1
    - field_option: jstype
      module: buf.build/acme/petapis
      path: foo/v1
      field: foo.bar.Baz.field_name
  override:
    - file_option: java_package_prefix
      value: net
    - file_option: java_package_prefix
      module: buf.build/acme/petapis
      value: com
    - file_option: java_package_suffix
      module: buf.build/acme/petapis
      value: com
    - file_option: java_package
      path: foo/bar/baz.proto
      value: com.x.y.z
    - field_option: jstype
      module: buf.build/acme/paymentapis
      value: JS_NORMAL
    - field_option: jstype
      field: package1.Message2.field3
      value: JS_STRING
plugins:
  - remote: buf.build/protocolbuffers/go
    revision: 1
    out: gen/proto
  - protoc_builtin: cpp
    protoc_path: /path/to/protoc
    out: gen/proto
  - local: protoc-gen-validate
    out: gen/proto
  - local: path/to/protoc-gen-validate
    out: gen/proto
  - local: /usr/bin/path/to/protoc-gen-validate
    out: gen/proto2
  - local:
      - go
      - run
      - google.golang.org/protobuf/cmd/protoc-gen-go
    out: gen/proto
    opt:
      - paths=source_relative
      - foo=bar
      - baz
    include_imports: true
    include_wkt: true
    strategy: all
inputs:
  - git_repo: github.com/acme/weather
    subdir: proto
    branch: dev
    depth: 30
  - module: buf.build/acme/weather
    types:
      - foo.v1.User
      - foo.v1.UserService
    paths:
      - a/b/c
      - a/b/d
    exclude_paths:
      - a/b/c/x.proto
      - a/b/d/y.proto
  - directory: x/y/z
  - tarball: a/b/x.tar.gz
  - tarball: c/d/x.tar.zst
    compression: zstd
    strip_components: 2
    subdir: proto
  - zip_archive: https://github.com/googleapis/googleapis/archive/master.zip
    strip_components: 1
  - proto_file: foo/bar/baz.proto
    include_package_files: true
  - binary_image: image.binpb.gz
    compression: gz
`,
	)
}

func TestBufGenYAMLFilePostprocessCmd(t *testing.T) {
	t.Parallel()

	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input v2 with postprocess_cmd
		`version: v2
plugins:
  - local: protoc-gen-go
    out: gen/go
    postprocess_cmd:
      - "gofmt -w $out"
      - "goimports -w $out"
`,
		// expected output
		`version: v2
plugins:
  - local: protoc-gen-go
    out: gen/go
    postprocess_cmd:
      - gofmt -w $out
      - goimports -w $out
`,
	)

	testReadWriteBufGenYAMLFileRoundTrip(
		t,
		// input v1 with postprocess_cmd
		`version: v1
plugins:
  - plugin: go
    out: gen/go
    postprocess_cmd:
      - "ruff --fix $out"
`,
		// expected output
		`version: v2
plugins:
  - local: protoc-gen-go
    out: gen/go
    postprocess_cmd:
      - ruff --fix $out
`,
	)

	bufGenYAMLFile := testReadBufGenYAMLFile(t, `version: v2
plugins:
  - local: protoc-gen-python
    out: gen/python
    opt: paths=source_relative
    postprocess_cmd:
      - "ruff --fix $out"
      - "black $out"
`)
	pluginConfigs := bufGenYAMLFile.GenerateConfig().GeneratePluginConfigs()
	require.Len(t, pluginConfigs, 1)
	require.Equal(t, []string{"ruff --fix $out", "black $out"}, pluginConfigs[0].PostCommands())
}

func TestBufGenYAMLFileManagedErrors(t *testing.T) {
	t.Parallel()

	_, err := ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
managed:
  enabled: true
  override:
    - file_option: csharp_namespace
plugins:
  - local: protoc-gen-csharp
    out: gen
`),
	)
	require.ErrorContains(t, err, "must set value for an override")

	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
managed:
  enabled: true
  override:
    - value: "Override"
plugins:
  - local: protoc-gen-csharp
    out: gen
`),
	)
	require.ErrorContains(t, err, "must set file_option or field_option for an override")

	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
managed:
  enabled: true
  override:
    - file_option: csharp_namespace
      field_option: jstype
      value: "Override"
plugins:
  - local: protoc-gen-csharp
    out: gen
`),
	)
	require.ErrorContains(t, err, "exactly one of file_option and field_option must be set for an override")

	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
managed:
  enabled: true
  override:
    - file_option: csharp_namespace
      field: a.v1.field # bogus field.
      value: "fieldCustomNamespace"
plugins:
  - local: protoc-gen-csharp
    out: gen
`),
	)
	require.ErrorContains(t, err, "must not set field for a file_option override")

	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
managed:
  disable:
    - file_option: csharp_namespace
      field_option: jstype
plugins:
  - local: protoc-gen-csharp
`),
	)
	require.ErrorContains(t, err, "at most one of file_option and field_option can be specified")
}

func TestBufGenYAMLFilePluginConfigErrors(t *testing.T) {
	t.Parallel()

	_, err := ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-go
    revision: 1
    out: .
`),
	)
	require.ErrorContains(t, err, "cannot specify revision for local plugin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - protoc_builtin: cpp
    revision: 1
    out: .
`),
	)
	require.ErrorContains(t, err, "cannot specify revision for protoc built-in plugin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    protoc_path: /path/to/protoc
    out: .
`),
	)
	require.ErrorContains(t, err, "cannot specify protoc_path for remote plugin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-go
    protoc_path: /path/to/protoc
    out: .
`),
	)
	require.ErrorContains(t, err, "cannot specify protoc_path for local plugin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - revision: 1
    out: .
`),
	)
	require.ErrorContains(t, err, "must specify one of remote, local or protoc_builtin")
	// Test that out is required.
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-go
`),
	)
	require.ErrorContains(t, err, "must specify out")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    strategy: directory
    out: .
`),
	)
	require.ErrorContains(t, err, "cannot specify strategy for remote plugin")

	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    local: protoc-gen-go
    out: .
`))
	require.ErrorContains(t, err, "only one of remote, local or protoc_builtin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    protoc_builtin: cpp
    out: .
`),
	)
	require.ErrorContains(t, err, "only one of remote, local or protoc_builtin")
	_, err = ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-go
    protoc_builtin: cpp
    out: .
`),
	)
	require.ErrorContains(t, err, "only one of remote, local or protoc_builtin")
}

func testReadBufGenYAMLFile(
	t *testing.T,
	inputBufGenYAMLFileData string,
) BufGenYAMLFile {
	bufGenYAMLFile, err := ReadBufGenYAMLFile(
		strings.NewReader(testCleanYAMLData(inputBufGenYAMLFileData)),
	)
	require.NoError(t, err)
	return bufGenYAMLFile
}

func testReadWriteBufGenYAMLFileRoundTrip(
	t *testing.T,
	inputBufYAMLFileData string,
	expectedOutputBufYAMLFileData string,
) {
	bufGenYAMLFile := testReadBufGenYAMLFile(t, inputBufYAMLFileData)
	buffer := bytes.NewBuffer(nil)
	err := WriteBufGenYAMLFile(buffer, bufGenYAMLFile)
	require.NoError(t, err)
	outputBufGenYAMLData := testCleanYAMLData(buffer.String())
	assert.Equal(t, testCleanYAMLData(expectedOutputBufYAMLFileData), outputBufGenYAMLData, "output:\n%s", outputBufGenYAMLData)
}
