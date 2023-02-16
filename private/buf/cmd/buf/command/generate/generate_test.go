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

func TestCompareGeneratedStubsGoogleapisGo(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		googleapisDirPath,
		[]*testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
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
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
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
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
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

func TestCompareInsertionPointOutput(t *testing.T) {
	testingextended.SkipIfShort(t)
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(
		t,
		command.NewRunner(),
		insertionTestdataDirPath,
		[]*testPluginInfo{
			{name: "insertion-point-receiver"},
			{name: "insertion-point-writer"},
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
}

func testCompareGeneratedStubs(
	t *testing.T,
	runner command.Runner,
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
		genFlags...,
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
		nil,
		0,
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
		nil,
		0,
	)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(
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
