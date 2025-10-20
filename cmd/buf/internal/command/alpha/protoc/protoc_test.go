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

package protoc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"buf.build/go/app"
	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"buf.build/go/standard/xtesting"
	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

var buftestingDirPath = filepath.Join(
	"..",
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

type testPluginInfo struct {
	name string
	opt  string
}

func TestOverlap(t *testing.T) {
	t.Parallel()
	// https://github.com/bufbuild/buf/issues/113
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			"-I",
			filepath.Join("testdata", "overlap", "a"),
			"-I",
			filepath.Join("testdata", "overlap", "b"),
			"-o",
			app.DevNullFilePath,
			filepath.Join("testdata", "overlap", "a", "1.proto"),
			filepath.Join("testdata", "overlap", "b", "2.proto"),
		),
	)
}

func TestComparePrintFreeFieldNumbersGoogleapis(t *testing.T) {
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	filePaths := buftesting.GetProtocFilePaths(t, googleapisDirPath, 100)
	actualProtocStdout := bytes.NewBuffer(nil)
	buftesting.RunActualProtoc(
		t,
		false,
		false,
		googleapisDirPath,
		filePaths,
		nil,
		actualProtocStdout,
		fmt.Sprintf("--%s", printFreeFieldNumbersFlagName),
	)
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(actualProtocStdout.String()),
		appcmdtesting.WithArgs(
			append(
				[]string{
					"-I",
					googleapisDirPath,
					fmt.Sprintf("--%s", printFreeFieldNumbersFlagName),
				},
				filePaths...,
			)...,
		),
	)
}

func TestCompareOutputGoogleapis(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	filePaths := buftesting.GetProtocFilePaths(t, googleapisDirPath, 100)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(
		t,
		false,
		false,
		googleapisDirPath,
		filePaths,
	)
	bufProtocFileDescriptorSet := testGetBufProtocFileDescriptorSet(t, googleapisDirPath)
	prototesting.AssertFileDescriptorSetsEqual(t, bufProtocFileDescriptorSet, actualProtocFileDescriptorSet)
}

func TestCompareGeneratedStubsGoogleapisGo(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(
		t,
		googleapisDirPath,
		[]testPluginInfo{
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
		[]testPluginInfo{
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
		[]testPluginInfo{
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
		[]testPluginInfo{{name: "objc"}},
	)
}

func TestCompareInsertionPointOutput(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(
		t,
		insertionTestdataDirPath,
		[]testPluginInfo{
			{name: "insertion-point-receiver"},
			{name: "insertion-point-writer"},
		},
	)
}

func TestInsertionPointMixedPathsSuccess(t *testing.T) {
	xtesting.SkipIfShort(t)
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)
	testInsertionPointMixedPathsSuccess(t, ".", wd)
	testInsertionPointMixedPathsSuccess(t, wd, ".")
}

// testInsertionPointMixedPathsSuccess demonstrates that insertion points are able
// to generate to the same output directory, even if the absolute path points to
// the same place.
func testInsertionPointMixedPathsSuccess(t *testing.T, receiverOut string, writerOut string) {
	dirPath := filepath.Join("testdata", "insertion")
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	protocFlags := []string{
		fmt.Sprintf("--%s_out=%s", "insertion-point-receiver", receiverOut),
		fmt.Sprintf("--%s_out=%s", "insertion-point-writer", writerOut),
	}
	err := prototesting.RunProtoc(
		context.Background(),
		[]string{dirPath},
		filePaths,
		false,
		false,
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		protocFlags...,
	)
	require.Error(t, err)
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithEnv(
			func(string) map[string]string {
				return map[string]string{
					"PATH": os.Getenv("PATH"),
				}
			},
		),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			append(
				append(
					protocFlags,
					"-I",
					dirPath,
					"--by-dir",
				),
				filePaths...,
			)...,
		),
	)
}

func testCompareGeneratedStubs(
	t *testing.T,
	dirPath string,
	plugins []testPluginInfo,
) {
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	actualProtocDir := t.TempDir()
	bufProtocDir := t.TempDir()
	var actualProtocPluginFlags []string
	for _, plugin := range plugins {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", plugin.name, actualProtocDir))
		if plugin.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", plugin.name, plugin.opt))
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
	var bufProtocPluginFlags []string
	for _, plugin := range plugins {
		bufProtocPluginFlags = append(bufProtocPluginFlags, fmt.Sprintf("--%s_out=%s", plugin.name, bufProtocDir))
		if plugin.opt != "" {
			bufProtocPluginFlags = append(bufProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", plugin.name, plugin.opt))
		}
	}
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithEnv(
			func(string) map[string]string {
				return map[string]string{
					"PATH": os.Getenv("PATH"),
				}
			},
		),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			append(
				append(
					bufProtocPluginFlags,
					"-I",
					dirPath,
					"--by-dir",
				),
				filePaths...,
			)...,
		),
	)
	storageosProvider := storageos.NewProvider(storageos.ProviderWithSymlinks())
	actualReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		actualProtocDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	bufReadWriteBucket, err := storageosProvider.NewReadWriteBucket(
		bufProtocDir,
		storageos.ReadWriteBucketWithSymlinksIfSupported(),
	)
	require.NoError(t, err)
	diff, err := storage.DiffBytes(
		context.Background(),
		actualReadWriteBucket,
		bufReadWriteBucket,
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testCompareGeneratedStubsArchive(
	t *testing.T,
	dirPath string,
	plugins []testPluginInfo,
	useJar bool,
) {
	fileExt := ".zip"
	if useJar {
		fileExt = ".jar"
	}
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 100)
	tempDir := t.TempDir()
	actualProtocFile := filepath.Join(tempDir, "actual-protoc"+fileExt)
	bufProtocFile := filepath.Join(tempDir, "buf-protoc"+fileExt)
	var actualProtocPluginFlags []string
	for _, plugin := range plugins {
		actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_out=%s", plugin.name, actualProtocFile))
		if plugin.opt != "" {
			actualProtocPluginFlags = append(actualProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", plugin.name, plugin.opt))
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
	var bufProtocPluginFlags []string
	for _, plugin := range plugins {
		bufProtocPluginFlags = append(bufProtocPluginFlags, fmt.Sprintf("--%s_out=%s", plugin.name, bufProtocFile))
		if plugin.opt != "" {
			bufProtocPluginFlags = append(bufProtocPluginFlags, fmt.Sprintf("--%s_opt=%s", plugin.name, plugin.opt))
		}
	}
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithEnv(
			func(string) map[string]string {
				return map[string]string{
					"PATH": os.Getenv("PATH"),
				}
			},
		),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			append(
				append(
					bufProtocPluginFlags,
					"-I",
					dirPath,
					"--by-dir",
				),
				filePaths...,
			)...,
		),
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
	bufData, err := os.ReadFile(bufProtocFile)
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
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

func testGetBufProtocFileDescriptorSet(t *testing.T, dirPath string) *descriptorpb.FileDescriptorSet {
	data := testGetBufProtocFileDescriptorSetBytes(t, dirPath)
	fileDescriptorSet := &descriptorpb.FileDescriptorSet{}
	// TODO: change to image read logic
	require.NoError(
		t,
		protoencoding.NewWireUnmarshaler(nil).Unmarshal(
			data,
			fileDescriptorSet,
		),
	)
	return fileDescriptorSet
}

func testGetBufProtocFileDescriptorSetBytes(t *testing.T, dirPath string) []byte {
	stdout := bytes.NewBuffer(nil)
	appcmdtesting.Run(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appext.NewBuilder(name),
			)
		},
		appcmdtesting.WithStdout(stdout),
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithArgs(
			append(
				[]string{
					"-I",
					dirPath,
					"-o",
					"-",
				},
				buftesting.GetProtocFilePaths(t, dirPath, 100)...,
			)...,
		),
	)
	return stdout.Bytes()
}
