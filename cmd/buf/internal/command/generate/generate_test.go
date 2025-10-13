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

package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xslices"
	"buf.build/go/standard/xtesting"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO FUTURE: this has to change if we split up this repository
var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"..",
	"..",
	"..",
	"private",
	"buf",
	"buftesting",
)

func TestCompareGeneratedStubsGoogleapisGo(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
	)
}

func TestCompareGeneratedStubsGoogleapisGoZip(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
		false,
	)
}

func TestCompareGeneratedStubsGoogleapisGoJar(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
		true,
	)
}

func TestCompareGeneratedStubsGoogleapisObjc(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		googleapisDirPath,
		[]*testPluginInfo{{name: "objc"}},
	)
}

func TestCompareGeneratedStubsGoogleapisPyi(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		googleapisDirPath,
		[]*testPluginInfo{{name: "pyi"}},
	)
}

func TestCompareInsertionPointOutput(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(
		t,
		insertionTestdataDirPath,
		[]*testPluginInfo{
			{name: "insertion-point-receiver"},
			{name: "insertion-point-writer"},
		},
	)
}

func TestGenerateV2LocalPluginBasic(t *testing.T) {
	t.Parallel()

	tempDirPath := t.TempDir()
	input := filepath.Join("testdata", "v2", "local_plugin")
	template := filepath.Join("testdata", "v2", "local_plugin", "buf.basic.gen.yaml")

	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		template,
		input,
	)

	expected, err := storagemem.NewReadBucket(
		map[string][]byte{
			filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Bar
    - a.v1.Foo
`),
			filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
    - b.v1.Foo
`),
		},
	)
	require.NoError(t, err)
	actual, err := storageos.NewProvider().NewReadWriteBucket(tempDirPath)
	require.NoError(t, err)

	diff, err := storage.DiffBytes(context.Background(), expected, actual)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

func TestGenerateV2LocalPluginTypes(t *testing.T) {
	t.Parallel()
	testRunTypeArgs := func(t *testing.T, expect map[string][]byte, args ...string) {
		t.Helper()
		tempDirPath := t.TempDir()
		testRunSuccess(
			t,
			append([]string{
				"--output",
				tempDirPath,
			}, args...)...,
		)
		expected, err := storagemem.NewReadBucket(expect)
		require.NoError(t, err)
		require.NoError(t, err)
		actual, err := storageos.NewProvider().NewReadWriteBucket(tempDirPath)
		require.NoError(t, err)

		diff, err := storage.DiffBytes(context.Background(), expected, actual)
		require.NoError(t, err)
		require.Empty(t, string(diff))
	}

	// buf.basic.gen.yaml
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Bar
    - a.v1.Foo
`),
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
    - b.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.basic.gen.yaml"),
		filepath.Join("testdata", "v2", "local_plugin"),
	)
	// buf.types.gen.yaml
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
	)
	// input specified
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Bar
    - a.v1.Foo
`),
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
    - b.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
		filepath.Join("testdata", "v2", "local_plugin"), // input
	)
	// --template as CLI flag
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Foo
`),
	},
		"--template",
		`version: v2
plugins:
  - local: protoc-gen-top-level-type-names-yaml
    out: gen
inputs:
  - directory: ./testdata/v2/local_plugin
    types:
      - a.v1.Foo`,
	)
	// --type
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
		"--type",
		"b.v1.Bar",
		filepath.Join("testdata", "v2", "local_plugin"),
	)
	// --path
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
    - b.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
		"--path",
		filepath.Join("testdata", "v2", "local_plugin", "b"),
		filepath.Join("testdata", "v2", "local_plugin"),
	)
	// --exclude-path
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Bar
    - a.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "v2", "local_plugin", "b", "v1"),
		filepath.Join("testdata", "v2", "local_plugin"),
	)
	// buf.paths.gen.yaml
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Bar
    - a.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.paths.gen.yaml"),
	)
	// buf.exclude.paths.gen.yaml
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
    - b.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.exclude.paths.gen.yaml"),
	)
	// --type overrides template
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Bar
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "local_plugin", "buf.types.gen.yaml"),
		"--type",
		"b.v1.Bar",
	)
	// buf.gen.yaml has types and exclude_types on inputs and plugins
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Foo
`),
		filepath.Join("gen", "b", "v1", "b.top-level-type-names.yaml"): []byte(`messages:
    - b.v1.Baz
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "types", "buf.gen.yaml"),
	)
	// --exclude-type override
	testRunTypeArgs(t, map[string][]byte{
		filepath.Join("gen", "a", "v1", "a.top-level-type-names.yaml"): []byte(`messages:
    - a.v1.Foo
`),
	},
		"--template",
		filepath.Join("testdata", "v2", "types", "buf.gen.yaml"),
		"--exclude-type",
		"b.v1.Baz",
	)
}

func TestOutputFlag(t *testing.T) {
	t.Parallel()
	for _, paths := range []struct {
		template string
		dir      string
	}{
		// v1 buf.gen.yaml, v1 module
		{filepath.Join("testdata", "simple", "buf.gen.yaml"), filepath.Join("testdata", "simple")},
		// v1 buf.gen.yaml, v2 module
		{filepath.Join("testdata", "simple", "buf.gen.yaml"), filepath.Join("testdata", "v2", "simple")},
		// v2 buf.gen.yaml, v1 module
		{filepath.Join("testdata", "v2", "simple", "buf.gen.yaml"), filepath.Join("testdata", "simple")},
		// v2 buf.gen.yaml, v2 module
		{filepath.Join("testdata", "v2", "simple", "buf.gen.yaml"), filepath.Join("testdata", "v2", "simple")},
	} {
		tempDirPath := t.TempDir()
		testRunSuccess(
			t,
			"--output",
			tempDirPath,
			"--template",
			paths.template,
			paths.dir,
		)
		_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
		require.NoError(t, err)
	}
}

func TestProtoFileRefIncludePackageFiles(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen.yaml"),
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "a", "v1", "a.proto")),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.NoError(t, err)
}

func TestGenerateDuplicatePlugins(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "duplicate_plugins", "buf.gen.yaml"),
		filepath.Join("testdata", "duplicate_plugins"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "foo", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "bar", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestGenerateDuplicatePluginsV2(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "v2", "duplicate_plugins", "buf.gen.yaml"),
		filepath.Join("testdata", "v2", "duplicate_plugins"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "foo", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "bar", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestOutputWithPathEqualToExclude(t *testing.T) {
	t.Parallel()
	tempDirPath := t.TempDir()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: cannot set the same path for both --path and --exclude-path: "testdata/paths/a/v1/a.proto"`),
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.gen.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		filepath.Join("testdata", "paths"),
	)
}

func TestGenerateInsertionPoint(t *testing.T) {
	t.Parallel()
	testGenerateInsertionPointV1(t, ".", ".", filepath.Join("testdata", "insertion_point"))
	testGenerateInsertionPointV1(t, "gen/proto/insertion", "gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
	testGenerateInsertionPointV1(t, "gen/proto/insertion/", "./gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
	testGenerateInsertionPointV2(t, ".", ".", filepath.Join("testdata", "insertion_point"))
	testGenerateInsertionPointV2(t, "gen/proto/insertion", "gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
	testGenerateInsertionPointV2(t, "gen/proto/insertion/", "./gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
}

func TestGenerateInsertionPointFail(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: gen/proto/insertion
  - name: insertion-point-writer
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: read test.txt: file does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateInsertionPointFailV2(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v2
plugins:
  - protoc_builtin: insertion-point-receiver
    out: gen/proto/insertion
  - protoc_builtin: insertion-point-writer
    out: .
managed:
 enabled: false
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: read test.txt: file does not exist`,
		filepath.Join("testdata", "v2", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateDuplicateFileFail(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: .
  - name: insertion-point-receiver
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "test.txt" was generated multiple times: once by plugin "insertion-point-receiver" and again by plugin "insertion-point-receiver"`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateDuplicateFileFailV2(t *testing.T) {
	t.Parallel()
	successTemplate := `
version: v2
plugins:
  - protoc_builtin: insertion-point-receiver
    out: .
  - protoc_builtin: insertion-point-receiver
    out: .
managed:
  enabled: false
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "test.txt" was generated multiple times: once by plugin "insertion-point-receiver" and again by plugin "insertion-point-receiver"`,
		filepath.Join("testdata", "v2", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)
}

func TestGenerateInsertionPointMixedPathsFail(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)
	testGenerateInsertionPointMixedPathsFailV1(t, ".", wd)
	testGenerateInsertionPointMixedPathsFailV1(t, wd, ".")
	testGenerateInsertionPointMixedPathsFailV2(t, ".", wd)
	testGenerateInsertionPointMixedPathsFailV2(t, wd, ".")
}

func TestGenerateDeleteOutDir(t *testing.T) {
	t.Parallel()
	testGenerateDeleteOuts(t, "", "foo")
	testGenerateDeleteOuts(t, "base", "foo")
	testGenerateDeleteOuts(t, "", "foo", "bar")
	testGenerateDeleteOuts(t, "", "foo", "bar", "foo")
	testGenerateDeleteOuts(t, "base", "foo", "bar")
	testGenerateDeleteOuts(t, "base", "foo", "bar", "foo")
	testGenerateDeleteOuts(t, "", "foo.jar")
	testGenerateDeleteOuts(t, "", "foo.zip")
	testGenerateDeleteOuts(t, "", "foo/bar.jar")
	testGenerateDeleteOuts(t, "", "foo/bar.zip")
	testGenerateDeleteOuts(t, "base", "foo.jar")
	testGenerateDeleteOuts(t, "base", "foo.zip")
	testGenerateDeleteOuts(t, "base", "foo/bar.jar")
	testGenerateDeleteOuts(t, "base", "foo/bar.zip")
}

func TestBoolPointerFlagTrue(t *testing.T) {
	t.Parallel()
	expected := true
	testParseBoolPointer(t, "test-name", &expected, "--test-name")
}

func TestBoolPointerFlagTrueSpecified(t *testing.T) {
	t.Parallel()
	expected := true
	testParseBoolPointer(t, "test-name", &expected, "--test-name=true")
}

func TestBoolPointerFlagFalseSpecified(t *testing.T) {
	t.Parallel()
	expected := false
	testParseBoolPointer(t, "test-name", &expected, "--test-name=false")
}

func TestBoolPointerFlagUnspecified(t *testing.T) {
	t.Parallel()
	testParseBoolPointer(t, "test-name", nil)
}

func testGenerateInsertionPointV1(
	t *testing.T,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	testGenerateInsertionPoint(
		t,
		receiverOut,
		writerOut,
		expectedOutputPath,
		`
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`,
	)
}

func testGenerateInsertionPointV2(
	t *testing.T,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	testGenerateInsertionPoint(
		t,
		receiverOut,
		writerOut,
		expectedOutputPath,
		`
version: v2
plugins:
  - protoc_builtin: insertion-point-receiver
    out: %s
  - protoc_builtin: insertion-point-writer
    out: %s
`,
	)
}

func testGenerateInsertionPoint(
	t *testing.T,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
	successTemplate string,
) {
	storageosProvider := storageos.NewProvider()
	tempDir, readWriteBucket := internaltesting.CopyReadBucketToTempDir(
		context.Background(),
		t,
		storageosProvider,
		storagemem.NewReadWriteBucket(),
	)
	testRunSuccess(
		t,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		tempDir,
	)
	expectedOutput, err := storageosProvider.NewReadWriteBucket(expectedOutputPath)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(context.Background(), expectedOutput, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

func testGenerateInsertionPointMixedPathsFailV1(
	t *testing.T,
	receiverOut string,
	writerOut string,
) {
	testGenerateInsertionPointMixedPathsFail(
		t,
		receiverOut,
		writerOut,
		`
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`,
	)
}

func testGenerateInsertionPointMixedPathsFailV2(
	t *testing.T,
	receiverOut string,
	writerOut string,
) {
	testGenerateInsertionPointMixedPathsFail(
		t,
		receiverOut,
		writerOut,
		`
version: v2
plugins:
  - protoc_builtin: insertion-point-receiver
    out: %s
  - protoc_builtin: insertion-point-writer
    out: %s
`,
	)
}

// testGenerateInsertionPointMixedPathsFail demonstrates that insertion points are only
// able to generate to the same output directory, even if the absolute path points to the
// same place. This is equivalent to protoc's behavior.
func testGenerateInsertionPointMixedPathsFail(
	t *testing.T,
	receiverOut string,
	writerOut string,
	successTemplate string,
) {
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: read test.txt: file does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		t.TempDir(),
	)
}

func testCompareGeneratedStubs(
	t *testing.T,
	dirPath string,
	testPluginInfos []*testPluginInfo,
) {
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	actualProtocDir := t.TempDir()
	bufGenDir := t.TempDir()
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocDir))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		false,
		false,
		dirPath,
		filePaths,
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		actualProtocPluginFlags...,
	)
	genFlags := []string{
		dirPath,
		"--template",
		newExternalConfigV1String(t, testPluginInfos, bufGenDir),
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--path",
			filePath,
		)
	}
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithArgs(genFlags...),
	)
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	actualReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		actualProtocDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	bufReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		bufGenDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(
		context.Background(),
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testCompareGeneratedStubsArchive(
	t *testing.T,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	useJar bool,
) {
	fileExt := ".zip"
	if useJar {
		fileExt = ".jar"
	}
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	tempDir := t.TempDir()
	actualProtocFile := filepath.Join(tempDir, "actual-protoc"+fileExt)
	bufGenFile := filepath.Join(tempDir, "buf-generate"+fileExt)
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocFile))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		false,
		false,
		dirPath,
		filePaths,
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		actualProtocPluginFlags...,
	)
	genFlags := []string{
		dirPath,
		"--template",
		newExternalConfigV1String(t, testPluginInfos, bufGenFile),
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--path",
			filePath,
		)
	}
	testRunSuccess(
		t,
		genFlags...,
	)
	actualData, err := os.ReadFile(actualProtocFile)
	require.NoError(t, err)
	actualReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(actualData),
		int64(len(actualData)),
		actualReadWriteBucket,
	)
	require.NoError(t, err)
	bufData, err := os.ReadFile(bufGenFile)
	require.NoError(t, err)
	bufReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(bufData),
		int64(len(bufData)),
		bufReadWriteBucket,
	)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(
		context.Background(),
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testRunSuccess(t *testing.T, args ...string) {
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithArgs(args...),
	)
}

func testGenerateDeleteOuts(
	t *testing.T,
	baseOutDirPath string,
	outputPaths ...string,
) {
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, nil, true, true, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, nil, false, false, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean"}, false, true, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean"}, true, true, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean=true"}, true, true, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean=true"}, false, true, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean=false"}, true, false, outputPaths)
	testGenerateDeleteOutsWithArgAndConfig(t, baseOutDirPath, []string{"--clean=false"}, false, false, outputPaths)
}

func testGenerateDeleteOutsWithArgAndConfig(
	t *testing.T,
	baseOutDirPath string,
	cleanArgs []string,
	cleanOptionInConfig bool,
	expectedClean bool,
	outputPaths []string,
) {
	// Just add more builtins to the plugins slice below if this goes off
	require.True(t, len(outputPaths) < 4, "we want to have unique plugins to work with and this test is only set up for three plugins max right now")
	fullOutputPaths := outputPaths
	if baseOutDirPath != "" && baseOutDirPath != "." {
		fullOutputPaths = xslices.Map(
			outputPaths,
			func(outputPath string) string {
				return normalpath.Join(baseOutDirPath, outputPath)
			},
		)
	}
	ctx := context.Background()
	tmpDirPath := t.TempDir()
	storageBucket, err := storageos.NewProvider().NewReadWriteBucket(tmpDirPath)
	require.NoError(t, err)
	for _, fullOutputPath := range fullOutputPaths {
		switch normalpath.Ext(fullOutputPath) {
		case ".jar", ".zip":
			// Write a one-byte file to the location. We'll compare the size below as a simple test.
			require.NoError(
				t,
				storage.PutPath(
					ctx,
					storageBucket,
					fullOutputPath,
					[]byte(`1`),
				),
			)
		default:
			// Write a file that won't be generated to the location.
			require.NoError(
				t,
				storage.PutPath(
					ctx,
					storageBucket,
					normalpath.Join(fullOutputPath, "foo.txt"),
					[]byte(`1`),
				),
			)
		}
	}
	var templateBuilder strings.Builder
	_, _ = templateBuilder.WriteString(`version: v2
plugins:
`)

	plugins := []string{"java", "cpp", "ruby"}
	for i, outputPath := range outputPaths {
		_, _ = templateBuilder.WriteString(`  - protoc_builtin: `)
		_, _ = templateBuilder.WriteString(plugins[i])
		_, _ = templateBuilder.WriteString("\n")
		_, _ = templateBuilder.WriteString(`    out: `)
		_, _ = templateBuilder.WriteString(outputPath)
		_, _ = templateBuilder.WriteString("\n")
	}
	if cleanOptionInConfig {
		templateBuilder.WriteString("clean: true\n")
	}
	testRunStdoutStderr(
		t,
		nil,
		0,
		``,
		``,
		append(
			[]string{
				filepath.Join("testdata", "simple"),
				"--template",
				templateBuilder.String(),
				"-o",
				filepath.Join(tmpDirPath, normalpath.Unnormalize(baseOutDirPath)),
			},
			cleanArgs...,
		)...,
	)
	for _, fullOutputPath := range fullOutputPaths {
		switch normalpath.Ext(fullOutputPath) {
		case ".jar", ".zip":
			data, err := storage.ReadPath(
				ctx,
				storageBucket,
				fullOutputPath,
			)
			require.NoError(t, err)
			// Always expect non-fake data, because the existing ".jar" or ".zip"
			// file is always replaced by the output. This is the existing and correct
			// behavior.
			require.True(t, len(data) > 1, "expected non-fake data at %q", fullOutputPath)
		default:
			data, err := storage.ReadPath(
				ctx,
				storageBucket,
				normalpath.Join(fullOutputPath, "foo.txt"),
			)
			if expectedClean {
				require.ErrorIs(t, err, fs.ErrNotExist)
			} else {
				require.NoError(t, err, "expected foo.txt at %q", fullOutputPath)
				require.NotNil(t, data, "expected foo.txt at %q", fullOutputPath)
			}
		}
	}
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(
					name,
					appext.BuilderWithInterceptor(
						// TODO FUTURE: use the real interceptor. Currently in buf.go, NewBuilder receives appflag.BuilderWithInterceptor(newErrorInterceptor()).
						// However we cannot depend on newErrorInterceptor because it would create an import cycle, not to mention it needs to be exported first.
						// This can depend on newErrorInterceptor when it's moved to a separate package and made public.
						func(next func(context.Context, appext.Container) error) func(context.Context, appext.Container) error {
							return func(ctx context.Context, container appext.Container) error {
								err := next(ctx, container)
								if err == nil {
									return nil
								}
								return fmt.Errorf("Failure: %w", err)
							}
						},
					),
				),
			)
		},
		appcmdtesting.WithExpectedExitCode(expectedExitCode),
		appcmdtesting.WithExpectedStdout(expectedStdout),
		appcmdtesting.WithExpectedStderr(expectedStderr),
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(args...),
	)
}

func testParseBoolPointer(t *testing.T, flagName string, expectedResult *bool, args ...string) {
	var boolPointer *bool
	flagSet := pflag.NewFlagSet("test flag set", pflag.ContinueOnError)
	bindBoolPointer(flagSet, flagName, &boolPointer, "test usage")
	err := flagSet.Parse(args)
	require.NoError(t, err)
	require.Equal(t, expectedResult, boolPointer)
}

func newExternalConfigV1String(t *testing.T, plugins []*testPluginInfo, out string) string {
	externalConfig := make(map[string]any)
	externalConfig["version"] = "v1"
	pluginConfigs := []map[string]string{}
	for _, plugin := range plugins {
		pluginConfigs = append(
			pluginConfigs,
			map[string]string{
				"name": plugin.name,
				"opt":  plugin.opt,
				"out":  out,
			},
		)
	}
	externalConfig["plugins"] = pluginConfigs
	data, err := json.Marshal(externalConfig)
	require.NoError(t, err)
	return string(data)
}

type testPluginInfo struct {
	name string
	opt  string
}

func transformGolangProtocVersionToUnknown(t *testing.T) storage.DiffOption {
	return storage.DiffWithTransform(func(_, _ string, content []byte) []byte {
		lines := bytes.Split(content, []byte("\n"))
		filteredLines := make([][]byte, 0, len(lines))
		commentPrefix := []byte("//")
		protocVersionIndicator := []byte("protoc")
		for _, line := range lines {
			if !(bytes.HasPrefix(line, commentPrefix) && bytes.Contains(line, protocVersionIndicator)) {
				filteredLines = append(filteredLines, line)
			}
		}
		return bytes.Join(filteredLines, []byte("\n"))
	})
}
