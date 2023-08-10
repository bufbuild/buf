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

package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufgen"
	"github.com/bufbuild/buf/private/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/buftesting"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/encoding"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/bufbuild/buf/private/pkg/testingextended"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: this has to change if we split up this repository
var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"..",
	"..",
	"..",
	"private",
	"bufpkg",
	"buftesting",
)

const (
	v1 = "v1"
	v2 = "v2"
)

func TestCompareGeneratedStubsGoogleapisGo(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
	)
}

func TestCompareGeneratedStubsGoogleapisGoZip(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
		false,
	)
}

func TestCompareGeneratedStubsGoogleapisGoJar(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo", isBinary: true},
		},
		true,
	)
}

func TestCompareGeneratedStubsGoogleapisObjc(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{{name: "objc"}},
	)
}

func TestCompareGeneratedStubsGoogleapisPyi(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{{name: "pyi"}},
	)
}

func TestCompareMigrateAndGenerate(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())

	tmpDir := t.TempDir()
	v1Content := `version: v1
managed:
  enabled: true
  cc_enable_arenas: false
  java_multiple_files: false
  java_package_prefix: com
  java_string_check_utf8: false
  optimize_for: CODE_SIZE
  go_package_prefix:
    default: github.com/acme/weather/private/gen/proto/go
    except:
      - buf.build/googleapis/googleapis
    override:
      buf.build/acme/weather: github.com/acme/weather/gen/proto/go
  override:
    JAVA_PACKAGE:
      acme/weather/v1/weather.proto: "org"
plugins:
  - plugin: go
    out: gen/proto/go
    opt: paths=source_relative
  - plugin: java
    out: gen/java
`
	outDir := filepath.Join(tmpDir, "out")
	templatePath := filepath.Join(tmpDir, "buf.gen.yaml")
	err := os.WriteFile(templatePath, []byte(v1Content), 0600)
	require.NoError(t, err)
	protoDir := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)

	// generate with v1 template
	testRunSuccess(
		t,
		protoDir,
		"--template",
		templatePath,
		"-o",
		outDir,
	)
	bucketV1, err := storageosProvider.NewReadWriteBucket(
		outDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	// migrate from v1 to v2
	testRunSuccess(
		t,
		protoDir,
		"--template",
		templatePath,
		"-o",
		outDir,
		"--migrate-v1",
	)

	data, err := os.ReadFile(templatePath)
	require.NoError(t, err)
	versionConfig := bufgen.ExternalConfigVersion{}
	err = encoding.UnmarshalYAMLNonStrict(data, &versionConfig)
	require.NoError(t, err)
	require.Equal(t, "v2", versionConfig.Version)

	// generate with v2 template
	testRunSuccess(
		t,
		protoDir,
		"--template",
		templatePath,
		"-o",
		outDir,
	)
	bucketV2, err := storageosProvider.NewReadWriteBucket(
		outDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	diff, err := storage.DiffBytes(
		context.Background(),
		command.NewRunner(),
		bucketV1,
		bucketV2,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func TestCompareInsertionPointOutput(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		insertionTestdataDirPath,
		[]*testPluginInfo{
			{name: "insertion-point-receiver", isBinary: true},
			{name: "insertion-point-writer", isBinary: true},
		},
	)
}

func TestOutputFlag(t *testing.T) {
	tempDirPath := t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "simple", "buf.gen.yaml"),
		filepath.Join("testdata", "simple"),
	)
	_, err := os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "simple", "buf.genv2.yaml"),
		filepath.Join("testdata", "simple"),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestProtoFileRefIncludePackageFiles(t *testing.T) {
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

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "protofileref", "buf.gen2.yaml"),
		fmt.Sprintf("%s#include_package_files=true", filepath.Join("testdata", "protofileref", "a", "v1", "a.proto")),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "java", "a", "v1", "B.java"))
	require.NoError(t, err)
}

func TestGenerateDuplicatePlugins(t *testing.T) {
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

	tempDirPath = t.TempDir()
	testRunSuccess(
		t,
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "duplicate_plugins", "buf.genv2.yaml"),
		filepath.Join("testdata", "duplicate_plugins"),
	)
	_, err = os.Stat(filepath.Join(tempDirPath, "foo", "a", "v1", "A.java"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(tempDirPath, "bar", "a", "v1", "A.java"))
	require.NoError(t, err)
}

func TestOutputWithPathEqualToExclude(t *testing.T) {
	tempDirPath := t.TempDir()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: cannot set the same path for both --path and --exclude-path flags: a/v1/a.proto`),
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

	tempDirPath = t.TempDir()
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		filepath.FromSlash(`Failure: cannot set the same path for both --path and --exclude-path flags: a/v1/a.proto`),
		"--output",
		tempDirPath,
		"--template",
		filepath.Join("testdata", "paths", "buf.genv2.yaml"),
		"--exclude-path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		"--path",
		filepath.Join("testdata", "paths", "a", "v1", "a.proto"),
		filepath.Join("testdata", "paths"),
	)
}

func TestGenerateInsertionPoint(t *testing.T) {
	t.Parallel()
	runner := command.NewRunner()
	testGenerateInsertionPoint(t, runner, ".", ".", filepath.Join("testdata", "insertion_point"))
	testGenerateInsertionPoint(t, runner, "gen/proto/insertion", "gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
	testGenerateInsertionPoint(t, runner, "gen/proto/insertion/", "./gen/proto/insertion", filepath.Join("testdata", "nested_insertion_point"))
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
		`Failure: plugin insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		successTemplate,
		"-o",
		t.TempDir(),
	)

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: gen/proto/insertion
  - binary: protoc-gen-insertion-point-writer
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin protoc-gen-insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
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

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: .
  - binary: protoc-gen-insertion-point-receiver
    out: .
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: file "test.txt" was generated multiple times: once by plugin "protoc-gen-insertion-point-receiver" and again by plugin "protoc-gen-insertion-point-receiver"`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
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
	testGenerateInsertionPointMixedPathsFail(t, ".", wd)
	testGenerateInsertionPointMixedPathsFail(t, wd, ".")
}

func testGenerateInsertionPoint(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	testGenerateInsertionPointV1(
		t,
		runner,
		receiverOut,
		writerOut,
		expectedOutputPath,
	)
	testGenerateInsertionPointV2(
		t,
		runner,
		receiverOut,
		writerOut,
		expectedOutputPath,
	)
}

func testGenerateInsertionPointV1(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`
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
	diff, err := storage.DiffBytes(context.Background(), runner, expectedOutput, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

func testGenerateInsertionPointV2(
	t *testing.T,
	runner command.Runner,
	receiverOut string,
	writerOut string,
	expectedOutputPath string,
) {
	successTemplate := `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: %s
  - binary: protoc-gen-insertion-point-writer
    out: %s
`
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
	diff, err := storage.DiffBytes(context.Background(), runner, expectedOutput, readWriteBucket)
	require.NoError(t, err)
	require.Empty(t, string(diff))
}

// testGenerateInsertionPointMixedPathsFail demonstrates that insertion points are only
// able to generate to the same output directory, even if the absolute path points to the
// same place. This is equivalent to protoc's behavior.
func testGenerateInsertionPointMixedPathsFail(t *testing.T, receiverOut string, writerOut string) {
	successTemplate := `
version: v1
plugins:
  - name: insertion-point-receiver
    out: %s
  - name: insertion-point-writer
    out: %s
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		t.TempDir(),
	)

	successTemplate = `
version: v2
plugins:
  - binary: protoc-gen-insertion-point-receiver
    out: %s
  - binary: protoc-gen-insertion-point-writer
    out: %s
`
	testRunStdoutStderr(
		t,
		nil,
		1,
		``,
		`Failure: plugin protoc-gen-insertion-point-writer: test.txt: does not exist`,
		filepath.Join("testdata", "simple"), // The input directory is irrelevant for these insertion points.
		"--template",
		fmt.Sprintf(successTemplate, receiverOut, writerOut),
		"-o",
		t.TempDir(),
	)
}

func testCompareGeneratedStubs(
	t *testing.T,
	runner command.Runner,
	dirPath string,
	testPluginInfos []*testPluginInfo,
) {
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	actualProtocDir := t.TempDir()
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocDir))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		runner,
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
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	actualReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		actualProtocDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)

	bufReadWriteBucket := generateIntoBucket(
		t,
		v1,
		storageosProvider,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err := storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))

	bufReadWriteBucket = generateIntoBucket(
		t,
		v2,
		storageosProvider,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err = storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testCompareGeneratedStubsArchive(
	t *testing.T,
	runner command.Runner,
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
	var actualProtocPluginFlags []string
	for _, testPluginInfo := range testPluginInfos {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", testPluginInfo.name, actualProtocFile))
		if testPluginInfo.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", testPluginInfo.name, testPluginInfo.opt))
		}
	}
	buftesting.RunActualProtoc(
		t,
		runner,
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
	actualData, err := os.ReadFile(actualProtocFile)
	require.NoError(t, err)
	actualReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(actualData),
		int64(len(actualData)),
		actualReadWriteBucket,
		nil,
		0,
	)
	require.NoError(t, err)

	bufReadWriteBucket := generateZipIntoBucket(
		t,
		v1,
		fileExt,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err := storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))

	bufReadWriteBucket = generateZipIntoBucket(
		t,
		v2,
		fileExt,
		dirPath,
		testPluginInfos,
		filePaths,
	)
	diff, err = storage.DiffBytes(
		context.Background(),
		runner,
		actualReadWriteBucket,
		bufReadWriteBucket,
		transformGolangProtocVersionToUnknown(t),
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testRunSuccess(t *testing.T, args ...string) {
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
			)
		},
		internaltesting.NewEnvFunc(t),
		nil,
		nil,
		args...,
	)
}

func testRunStdoutStderr(t *testing.T, stdin io.Reader, expectedExitCode int, expectedStdout string, expectedStderr string, args ...string) {
	appcmdtesting.RunCommandExitCodeStdoutStderr(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
			)
		},
		expectedExitCode,
		expectedStdout,
		expectedStderr,
		internaltesting.NewEnvFunc(t),
		stdin,
		args...,
	)
}

func generateIntoBucket(
	t *testing.T,
	version string,
	storageosProvider storageos.Provider,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	filePaths []string,
) storage.ReadWriteBucket {
	out := t.TempDir()
	var genFlags []string
	switch version {
	case v1:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV1String(t, testPluginInfos, out),
		}
	case v2:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV2String(t, testPluginInfos, out),
		}
	default:
		require.Fail(t, "invalid test case")
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
	bufReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		out,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	return bufReadWriteBucket
}

func generateZipIntoBucket(
	t *testing.T,
	version string,
	fileExt string,
	dirPath string,
	testPluginInfos []*testPluginInfo,
	filePaths []string,
) storage.ReadWriteBucket {
	out := filepath.Join(t.TempDir(), "buf-generate"+fileExt)
	var genFlags []string
	switch version {
	case v1:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV1String(t, testPluginInfos, out),
		}
	case v2:
		genFlags = []string{
			dirPath,
			"--template",
			newExternalConfigV2String(t, testPluginInfos, out),
		}
	default:
		require.Fail(t, "invalid test case")
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
	bufData, err := os.ReadFile(out)
	require.NoError(t, err)
	bufReadWriteBucket := storagemem.NewReadWriteBucket()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(bufData),
		int64(len(bufData)),
		bufReadWriteBucket,
		nil,
		0,
	)
	require.NoError(t, err)
	return bufReadWriteBucket
}

func newExternalConfigV1String(t *testing.T, plugins []*testPluginInfo, out string) string {
	externalConfig := bufgen.ExternalConfigV1{
		Version: "v1",
	}
	for _, plugin := range plugins {
		externalConfig.Plugins = append(
			externalConfig.Plugins,
			bufgen.ExternalPluginConfigV1{
				Name: plugin.name,
				Out:  out,
				Opt:  plugin.opt,
			},
		)
	}
	data, err := json.Marshal(externalConfig)
	require.NoError(t, err)
	return string(data)
}

func newExternalConfigV2String(t *testing.T, plugins []*testPluginInfo, out string) string {
	externalConfig := bufgen.ExternalConfigV2{
		Version: "v2",
	}
	for _, plugin := range plugins {
		var pluginConfig bufgen.ExternalPluginConfigV2
		if plugin.isBinary {
			pluginConfig = bufgen.ExternalPluginConfigV2{
				Binary: fmt.Sprintf("protoc-gen-%s", plugin.name),
				Opt:    plugin.opt,
				Out:    out,
			}
		} else {
			pluginConfig = bufgen.ExternalPluginConfigV2{
				ProtocBuiltin: &plugin.name,
				Opt:           plugin.opt,
				Out:           out,
			}
		}
		externalConfig.Plugins = append(
			externalConfig.Plugins,
			pluginConfig,
		)
	}
	data, err := json.Marshal(externalConfig)
	require.NoError(t, err)
	return string(data)
}

type testPluginInfo struct {
	name     string
	opt      string
	isBinary bool
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
