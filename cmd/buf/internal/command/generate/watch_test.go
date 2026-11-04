// Copyright 2020-2026 Buf Technologies, Inc.
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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/bufbuild/buf/cmd/buf/internal/internaltesting"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/pkg/filewatch"
	"github.com/bufbuild/buf/private/pkg/osext"
	"github.com/bufbuild/buf/private/pkg/slogapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	watchTestTimeout      = 30 * time.Second
	watchTestPollInterval = 50 * time.Millisecond
	// watchTestSettleDuration is how long to wait before asserting that no run happened.
	watchTestSettleDuration = 10 * watchDebounceDuration

	testTemplate = `{"version":"v2","plugins":[{"local":"protoc-gen-files-to-generate-yaml","out":"gen"}]}`
)

func TestIsRelevantEvent(t *testing.T) {
	t.Parallel()
	dirPath := t.TempDir()
	templateFilePath := filepath.Join(dirPath, "template.yaml")
	configFilePath := filepath.Join(dirPath, "config.yaml")
	configFilePaths := []string{templateFilePath, configFilePath}
	isRelevantPath := func(configFilePaths []string, path string) bool {
		return isRelevantEvent(configFilePaths, filewatch.Event{Path: path})
	}

	assert.True(t, isRelevantPath(configFilePaths, templateFilePath))
	assert.True(t, isRelevantPath(configFilePaths, configFilePath))
	assert.True(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "a", "v1", "a.proto")))
	assert.True(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "buf.yaml")))
	assert.True(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "buf.lock")))
	assert.True(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "buf.work.yaml")))

	assert.False(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "other.yaml")))
	assert.False(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "a.go")))
	assert.False(t, isRelevantPath(configFilePaths, filepath.Join(dirPath, "buf.gen.yaml")))
	assert.False(t, isRelevantPath(nil, templateFilePath))

	// A removed directory is always relevant.
	assert.True(t, isRelevantEvent(configFilePaths, filewatch.Event{Path: filepath.Join(dirPath, "a"), IsRemovedDir: true}))
}

func TestLogChanges(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	paths := make([]string, 0, watchMaxLoggedChanges+2)
	for i := range cap(paths) {
		paths = append(paths, filepath.Join(string(filepath.Separator), "x", fmt.Sprintf("%02d.proto", i)))
	}
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))

	logChanges(ctx, logger, paths[:watchMaxLoggedChanges])
	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	assert.Len(t, lines, watchMaxLoggedChanges)
	assert.Contains(t, lines[0], `msg="change detected" path=`+paths[0])

	buffer.Reset()
	logChanges(ctx, logger, paths)
	lines = strings.Split(strings.TrimSpace(buffer.String()), "\n")
	assert.Len(t, lines, watchMaxLoggedChanges+1)
	assert.Contains(t, lines[watchMaxLoggedChanges], `msg="more changes detected" count=2`)
}

func TestGetTemplateFilePath(t *testing.T) {
	t.Parallel()
	pwd, err := osext.Getwd()
	require.NoError(t, err)

	templateFilePath, err := getTemplateFilePath("")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(pwd, "buf.gen.yaml"), templateFilePath)

	templateFilePath, err = getTemplateFilePath(filepath.Join("foo", "bar.yaml"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(pwd, "foo", "bar.yaml"), templateFilePath)

	templateFilePath, err = getTemplateFilePath(`{"version":"v2","plugins":[]}`)
	require.NoError(t, err)
	assert.Empty(t, templateFilePath)
}

func TestGetLocalInputDirPaths(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)
	pwd, err := osext.Getwd()
	require.NoError(t, err)

	bufGenYAMLFile, err := bufconfig.ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-foo
    out: gen
inputs:
  - directory: proto
  - proto_file: other/foo.proto
  - module: buf.build/acme/weather
  - git_repo: https://github.com/acme/weather.git
  - tarball: https://example.com/weather.tar.gz
`),
	)
	require.NoError(t, err)

	dirPaths, err := getLocalInputDirPaths(ctx, logger, "", bufGenYAMLFile)
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{
			filepath.Join(pwd, "proto"),
			filepath.Join(pwd, "other"),
		},
		dirPaths,
	)

	// An input on the command line takes precedence over the template.
	dirPaths, err = getLocalInputDirPaths(ctx, logger, filepath.Join("some", "dir"), bufGenYAMLFile)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(pwd, "some", "dir")}, dirPaths)

	dirPaths, err = getLocalInputDirPaths(ctx, logger, "buf.build/acme/weather", bufGenYAMLFile)
	require.NoError(t, err)
	assert.Empty(t, dirPaths)

	bufGenYAMLFile, err = bufconfig.ReadBufGenYAMLFile(
		strings.NewReader(`version: v2
plugins:
  - local: protoc-gen-foo
    out: gen
`),
	)
	require.NoError(t, err)
	dirPaths, err = getLocalInputDirPaths(ctx, logger, "", bufGenYAMLFile)
	require.NoError(t, err)
	assert.Equal(t, []string{pwd}, dirPaths)
}

func TestWatchRequiresLocalInput(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(name, appext.NewBuilder(name))
		},
		appcmdtesting.WithEnv(internaltesting.NewEnvFunc(t)),
		appcmdtesting.WithExpectedExitCode(1),
		appcmdtesting.WithExpectedStderrPartials(
			"--watch requires at least one input that is a local directory or .proto file",
		),
		appcmdtesting.WithArgs(
			"buf.build/acme/weather",
			"--template",
			testTemplate,
			"-o",
			t.TempDir(),
			"--watch",
		),
	)
}

func TestWatch(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	outDirPath := filepath.Join(inputDirPath, "gen")
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	templateFilePath := filepath.Join(inputDirPath, "buf.gen.yaml")
	writeTestFile(
		t,
		templateFilePath,
		`version: v2
plugins:
  - local: protoc-gen-files-to-generate-yaml
    out: gen
`,
	)
	aProtoFilePath := filepath.Join(inputDirPath, "a", "v1", "a.proto")
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--template",
		templateFilePath,
		"-o",
		inputDirPath,
		"--watch",
	)

	aGeneratedFilePath := filepath.Join(outDirPath, "a", "v1", "files-to-generate.yaml")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - a/v1/a.proto\n")
	watchTest.requireStderrContains(t, "INFO\tgeneration succeeded\nINFO\twatching for changes\n")

	// A new file in a new directory.
	bProtoFilePath := filepath.Join(inputDirPath, "b", "v1", "b.proto")
	writeTestFile(t, bProtoFilePath, "syntax = \"proto3\";\npackage b.v1;\n")
	bGeneratedFilePath := filepath.Join(outDirPath, "b", "v1", "files-to-generate.yaml")
	watchTest.requireFileContent(t, bGeneratedFilePath, "files:\n    - b/v1/b.proto\n")
	watchTest.requireStderrContains(t, changeDetectedLine(bProtoFilePath))

	// A build failure is reported and watching continues.
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage {}\n")
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed\n")
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage A {}\n")
	watchTest.requireStderrCount(t, "generation succeeded", 3)

	// Changes within the output directory and to irrelevant files do not trigger a run.
	writeTestFile(t, filepath.Join(outDirPath, "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage gen.a.v1;\n")
	writeTestFile(t, filepath.Join(inputDirPath, "a", "v1", "README.md"), "")
	time.Sleep(watchTestSettleDuration)
	watchTest.requireStderrCount(t, "generation succeeded", 3)

	// A change to the template is picked up.
	writeTestFile(
		t,
		templateFilePath,
		`version: v2
plugins:
  - local: protoc-gen-files-to-generate-yaml
    out: gen2
`,
	)
	watchTest.requireFileContent(
		t,
		filepath.Join(inputDirPath, "gen2", "b", "v1", "files-to-generate.yaml"),
		"files:\n    - b/v1/b.proto\n",
	)

	// Removing a directory is picked up.
	require.NoError(t, os.RemoveAll(filepath.Join(inputDirPath, "b")))
	watchTest.requireStderrCount(t, "generation succeeded", 5)

	watchTest.requireStop(t)
}

func TestWatchConfigOverrideFile(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	configFilePath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, configFilePath, "version: v2\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--config",
		configFilePath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "watching for changes")

	// A change to the config file given as a flag is picked up.
	writeTestFile(t, configFilePath, "version: v2\nlint:\n  use:\n    - STANDARD\n")
	watchTest.requireStderrContains(t, changeDetectedLine(configFilePath))
	watchTest.requireStderrCount(t, "generation succeeded", 2)

	watchTest.requireStop(t)
}

func TestWatchConfigOverrideFileDirRecreated(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	aProtoFilePath := filepath.Join(inputDirPath, "a", "v1", "a.proto")
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\n")
	configDirPath := filepath.Join(t.TempDir(), "config")
	configFilePath := filepath.Join(configDirPath, "config.yaml")
	writeTestFile(t, configFilePath, "version: v2\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--config",
		configFilePath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "watching for changes")

	// The directory of the config file is removed and recreated, as by
	// switching branches, and the config file is still watched.
	require.NoError(t, os.RemoveAll(configDirPath))
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed")
	writeTestFile(t, configFilePath, "version: v2\n")
	watchTest.requireStderrCount(t, "generation succeeded", 2)
	writeTestFile(t, configFilePath, "version: v2\nlint:\n  use:\n    - STANDARD\n")
	watchTest.requireStderrCount(t, changeDetectedLine(configFilePath), 2)
	watchTest.requireStderrCount(t, "generation succeeded", 3)

	watchTest.requireStop(t)
}

func TestWatchWorkspaceLookupError(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	aDirPath := filepath.Join(rootDirPath, "a")
	writeTestFile(t, filepath.Join(aDirPath, "buf.yaml"), "version: v2\n")
	writeTestFile(t, filepath.Join(aDirPath, "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	// A directory with both a buf.work.yaml and a buf.yaml has no controlling workspace.
	bDirPath := filepath.Join(rootDirPath, "b")
	writeTestFile(t, filepath.Join(bDirPath, "buf.yaml"), "version: v2\n")
	writeTestFile(t, filepath.Join(bDirPath, "buf.work.yaml"), "version: v1\n")
	bProtoFilePath := filepath.Join(bDirPath, "b", "v1", "b.proto")
	writeTestFile(t, bProtoFilePath, "syntax = \"proto3\";\npackage b.v1;\n")
	templateFilePath := filepath.Join(rootDirPath, "buf.gen.yaml")
	writeTestFile(
		t,
		templateFilePath,
		`version: v2
plugins:
  - local: protoc-gen-files-to-generate-yaml
    out: gen
inputs:
  - directory: `+aDirPath+`
  - directory: `+bDirPath+"\n",
	)

	watchTest := startWatchTest(
		t,
		"--template",
		templateFilePath,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "WARN\tcannot find controlling workspace")
	watchTest.requireStderrContains(t, "watching for changes")

	// The input is watched directly when its workspace cannot be found.
	writeTestFile(t, bProtoFilePath, "syntax = \"proto3\";\npackage b.v1;\nmessage B {}\n")
	watchTest.requireStderrContains(t, changeDetectedLine(bProtoFilePath))

	watchTest.requireStop(t)
}

func TestWatchProtoFileInput(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	aProtoFilePath := filepath.Join(inputDirPath, "a", "v1", "a.proto")
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\n")
	outDirPath := t.TempDir()

	watchTest := startWatchTest(
		t,
		aProtoFilePath,
		"--template",
		testTemplate,
		"-o",
		outDirPath,
		"--watch",
	)
	aGeneratedFilePath := filepath.Join(outDirPath, "gen", "a", "v1", "files-to-generate.yaml")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - a/v1/a.proto\n")
	require.NoError(t, os.Remove(aGeneratedFilePath))
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage A {}\n")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - a/v1/a.proto\n")

	watchTest.requireStop(t)
}

func TestWatchOutDirIsModuleDir(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	writeTestFile(t, filepath.Join(inputDirPath, "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--template",
		`{"version":"v2","plugins":[{"local":"protoc-gen-files-to-generate-yaml","out":"."}]}`,
		"-o",
		inputDirPath,
		"--watch",
	)
	aGeneratedFilePath := filepath.Join(inputDirPath, "a", "v1", "files-to-generate.yaml")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - a/v1/a.proto\n")

	// The output directory contains the module, so it cannot be excluded from watching.
	writeTestFile(t, filepath.Join(inputDirPath, "a", "v1", "b.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - a/v1/a.proto\n    - a/v1/b.proto\n")
	// Generated files within the module do not trigger further runs.
	time.Sleep(watchTestSettleDuration)
	watchTest.requireStderrCount(t, "generation succeeded", 2)

	watchTest.requireStop(t)
}

func TestWatchOutDirContainsInputs(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	writeTestFile(t, filepath.Join(inputDirPath, "api", "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--template",
		`{"version":"v2","plugins":[{"local":"protoc-gen-files-to-generate-yaml","out":"api"}]}`,
		"-o",
		inputDirPath,
		"--watch",
	)
	aGeneratedFilePath := filepath.Join(inputDirPath, "api", "api", "a", "v1", "files-to-generate.yaml")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - api/a/v1/a.proto\n")

	// The output directory contains input files within the module, so it is not excluded from watching.
	writeTestFile(t, filepath.Join(inputDirPath, "api", "a", "v1", "b.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	watchTest.requireFileContent(t, aGeneratedFilePath, "files:\n    - api/a/v1/a.proto\n    - api/a/v1/b.proto\n")
	// Generated files within the output directory do not trigger further runs.
	watchTest.requireStderrCount(t, "generation succeeded", 2)
	watchTest.requireStderrCountUnchanged(t, "generation succeeded", 2)

	watchTest.requireStop(t)
}

func TestGetExcludeDirPaths(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	apiDirPath := filepath.Join(rootDirPath, "api")
	genDirPath := filepath.Join(rootDirPath, "gen")
	apDirPath := filepath.Join(rootDirPath, "ap")
	inputFilePaths := []string{
		filepath.Join(rootDirPath, "api", "a", "v1", "a.proto"),
		filepath.Join(rootDirPath, "apib", "b.proto"),
	}
	assert.Equal(
		t,
		[]string{genDirPath, apDirPath},
		getExcludeDirPaths([]string{apiDirPath, genDirPath, apDirPath}, inputFilePaths),
	)
	assert.Equal(t, []string{apiDirPath}, getExcludeDirPaths([]string{apiDirPath}, nil))
}

func TestWatchWorkspaceConfigAboveModule(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	bufYAMLFilePath := filepath.Join(rootDirPath, "buf.yaml")
	writeTestFile(t, bufYAMLFilePath, "version: v2\nmodules:\n  - path: proto\n")
	writeTestFile(t, filepath.Join(rootDirPath, "proto", "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")

	watchTest := startWatchTest(
		t,
		rootDirPath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "watching for changes")

	// Only module directories are watched, not the whole input directory.
	writeTestFile(t, filepath.Join(rootDirPath, "other", "b.proto"), "syntax = \"proto3\";\npackage b.v1;\n")
	time.Sleep(watchTestSettleDuration)
	watchTest.requireStderrCount(t, "generation succeeded", 1)

	// The workspace configuration is watched even though it is above the module.
	writeTestFile(t, bufYAMLFilePath, "version: v2\nmodules:\n  - path: proto\nlint:\n  use:\n    - STANDARD\n")
	watchTest.requireStderrContains(t, changeDetectedLine(bufYAMLFilePath))
	watchTest.requireStderrCount(t, "generation succeeded", 2)

	// A module directory that is removed and recreated, as by switching
	// branches, is watched again.
	protoDirPath := filepath.Join(rootDirPath, "proto")
	aProtoFilePath := filepath.Join(protoDirPath, "a", "v1", "a.proto")
	require.NoError(t, os.RemoveAll(protoDirPath))
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\n")
	watchTest.requireStderrCount(t, "generation succeeded", 3)
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage A {}\n")
	watchTest.requireStderrContains(t, changeDetectedLine(aProtoFilePath))
	watchTest.requireStderrCount(t, "generation succeeded", 4)

	// A module directory that is recreated only after the build failed
	// without it is watched for its creation, without watching the whole
	// input directory.
	require.NoError(t, os.RemoveAll(protoDirPath))
	watchTest.requireStderrCount(t, "ERROR\tgeneration failed", 1)
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\n")
	watchTest.requireStderrCount(t, "generation succeeded", 5)
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage A {}\n")
	watchTest.requireStderrCount(t, "generation succeeded", 6)
	writeTestFile(t, filepath.Join(rootDirPath, "other", "b.proto"), "syntax = \"proto3\";\npackage b.v1;\nmessage B {}\n")
	time.Sleep(watchTestSettleDuration)
	watchTest.requireStderrCount(t, "generation succeeded", 6)

	// A module added to the workspace is watched while it is still empty.
	proto2DirPath := filepath.Join(rootDirPath, "proto2")
	require.NoError(t, os.Mkdir(proto2DirPath, 0700))
	writeTestFile(t, bufYAMLFilePath, "version: v2\nmodules:\n  - path: proto\n  - path: proto2\n")
	watchTest.requireStderrCount(t, "ERROR\tgeneration failed", 2)
	writeTestFile(t, filepath.Join(proto2DirPath, "c", "v1", "c.proto"), "syntax = \"proto3\";\npackage c.v1;\n")
	watchTest.requireStderrCount(t, "generation succeeded", 7)

	watchTest.requireStop(t)
}

func TestWatchNestedModuleAddedToWorkspace(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	bufYAMLFilePath := filepath.Join(rootDirPath, "buf.yaml")
	writeTestFile(t, bufYAMLFilePath, "version: v2\nmodules:\n  - path: proto\n")
	writeTestFile(t, filepath.Join(rootDirPath, "proto", "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")

	watchTest := startWatchTest(
		t,
		rootDirPath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "watching for changes")

	// A module added to the workspace whose parent directory does not exist
	// either is watched for its creation.
	writeTestFile(t, bufYAMLFilePath, "version: v2\nmodules:\n  - path: proto\n  - path: internal/proto2\n")
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed")
	cProtoFilePath := filepath.Join(rootDirPath, "internal", "proto2", "c", "v1", "c.proto")
	writeTestFile(t, cProtoFilePath, "syntax = \"proto3\";\npackage c.v1;\n")
	watchTest.requireStderrCount(t, "generation succeeded", 2)
	writeTestFile(t, cProtoFilePath, "syntax = \"proto3\";\npackage c.v1;\nmessage C {}\n")
	watchTest.requireStderrContains(t, changeDetectedLine(cProtoFilePath))
	watchTest.requireStderrCount(t, "generation succeeded", 3)

	watchTest.requireStop(t)
}

func TestWatchConfigOverrideModules(t *testing.T) {
	t.Parallel()
	rootDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(rootDirPath, "buf.yaml"), "version: v2\nmodules:\n  - path: proto\n")
	writeTestFile(t, filepath.Join(rootDirPath, "proto", "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	configFilePath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, configFilePath, "version: v2\nmodules:\n  - path: proto\n")

	watchTest := startWatchTest(
		t,
		rootDirPath,
		"--config",
		configFilePath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "watching for changes")

	// A module declared by the config override rather than the buf.yaml of
	// the workspace is watched for its creation.
	writeTestFile(t, configFilePath, "version: v2\nmodules:\n  - path: proto\n  - path: proto2\n")
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed")
	cProtoFilePath := filepath.Join(rootDirPath, "proto2", "c", "v1", "c.proto")
	writeTestFile(t, cProtoFilePath, "syntax = \"proto3\";\npackage c.v1;\n")
	watchTest.requireStderrCount(t, "generation succeeded", 2)

	watchTest.requireStop(t)
}

func TestWatchInitialBuildFailure(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	aProtoFilePath := filepath.Join(inputDirPath, "a", "v1", "a.proto")
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage {}\n")

	watchTest := startWatchTest(
		t,
		inputDirPath,
		"--template",
		testTemplate,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed\nINFO\twatching for changes\n")

	// The module directories are watched even though the build failed.
	writeTestFile(t, aProtoFilePath, "syntax = \"proto3\";\npackage a.v1;\nmessage A {}\n")
	watchTest.requireStderrContains(t, "generation succeeded")

	watchTest.requireStop(t)
}

func TestWatchTemplateInputError(t *testing.T) {
	t.Parallel()
	inputDirPath := t.TempDir()
	writeTestFile(t, filepath.Join(inputDirPath, "buf.yaml"), "version: v2\n")
	writeTestFile(t, filepath.Join(inputDirPath, "a", "v1", "a.proto"), "syntax = \"proto3\";\npackage a.v1;\n")
	templateFilePath := filepath.Join(inputDirPath, "buf.gen.yaml")
	template := `version: v2
plugins:
  - local: protoc-gen-files-to-generate-yaml
    out: gen
inputs:
  - directory: ` + inputDirPath + "\n"
	writeTestFile(t, templateFilePath, template)

	watchTest := startWatchTest(
		t,
		"--template",
		templateFilePath,
		"-o",
		t.TempDir(),
		"--watch",
	)
	watchTest.requireStderrContains(t, "generation succeeded")

	// An invalid input in the template is reported and watching continues.
	writeTestFile(t, templateFilePath, template+"  - module: not a module\n")
	watchTest.requireStderrContains(t, "ERROR\tgeneration failed\t{\"error\":")
	writeTestFile(t, templateFilePath, template)
	watchTest.requireStderrCount(t, "generation succeeded", 2)

	watchTest.requireStop(t)
}

type watchTest struct {
	cancel  context.CancelFunc
	errChan chan error
	stderr  *lockedBuffer
}

// changeDetectedLine returns the log line for a change to the given absolute path,
// which is outside the working directory in tests and so is displayed as is.
func changeDetectedLine(path string) string {
	return "INFO\tchange detected\t{\"path\":\"" + path + "\"}\n"
}

// startWatchTest runs buf generate with the given args until requireStop is called.
//
// Logs are written in text format so that they can be asserted on without color codes.
func startWatchTest(t *testing.T, args ...string) *watchTest {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	use := "test"
	stderr := &lockedBuffer{}
	errChan := make(chan error, 1)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("stderr:\n%s", stderr.String())
		}
	})
	go func() {
		errChan <- appcmd.Run(
			ctx,
			app.NewContainer(
				internaltesting.NewEnvFunc(t)(use),
				nil,
				&bytes.Buffer{},
				stderr,
				append(append([]string{use, "generate"}, args...), "--log-format", "text")...,
			),
			newTestGenerateCommand(use),
		)
	}()
	return &watchTest{
		cancel:  cancel,
		errChan: errChan,
		stderr:  stderr,
	}
}

// newTestGenerateCommand returns a root command with the generate command as its
// only sub-command, wired up the same way as in the buf CLI.
func newTestGenerateCommand(use string) *appcmd.Command {
	builder := appext.NewBuilder(
		use,
		appext.BuilderWithTimeout(0),
		appext.BuilderWithLoggerProvider(slogapp.LoggerProvider),
	)
	return &appcmd.Command{
		Use:                 use,
		BindPersistentFlags: builder.BindRoot,
		SubCommands: []*appcmd.Command{
			NewCommand("generate", builder),
		},
	}
}

// requireStop cancels the command and requires that it exits without error.
func (w *watchTest) requireStop(t *testing.T) {
	w.cancel()
	select {
	case err := <-w.errChan:
		require.NoError(t, err)
	case <-time.After(watchTestTimeout):
		require.Fail(t, "timed out waiting for buf generate --watch to exit")
	}
}

func (w *watchTest) requireFileContent(t *testing.T, filePath string, expectedContent string) {
	require.Eventually(
		t,
		func() bool {
			content, err := os.ReadFile(filePath)
			return err == nil && string(content) == expectedContent
		},
		watchTestTimeout,
		watchTestPollInterval,
		"expected %s to have content %q",
		filePath,
		expectedContent,
	)
}

func (w *watchTest) requireStderrContains(t *testing.T, expected string) {
	require.Eventually(
		t,
		func() bool {
			return strings.Contains(w.stderr.String(), expected)
		},
		watchTestTimeout,
		watchTestPollInterval,
		"expected stderr to contain %q",
		expected,
	)
}

func (w *watchTest) requireStderrCount(t *testing.T, expected string, count int) {
	require.Eventually(
		t,
		func() bool {
			return strings.Count(w.stderr.String(), expected) == count
		},
		watchTestTimeout,
		watchTestPollInterval,
		"expected stderr to contain %q %d times",
		expected,
		count,
	)
}

// requireStderrCountUnchanged requires that stderr keeps containing expected
// count times for watchTestSettleDuration, failing as soon as it does not.
func (w *watchTest) requireStderrCountUnchanged(t *testing.T, expected string, count int) {
	require.Never(
		t,
		func() bool {
			return strings.Count(w.stderr.String(), expected) != count
		},
		watchTestSettleDuration,
		watchTestPollInterval,
		"expected stderr to keep containing %q %d times",
		expected,
		count,
	)
}

// lockedBuffer is a bytes.Buffer that is safe for concurrent use.
type lockedBuffer struct {
	lock   sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buffer.Write(data)
}

func (b *lockedBuffer) String() string {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buffer.String()
}

func writeTestFile(t *testing.T, filePath string, content string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0700))
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0600))
}
