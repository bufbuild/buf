// Copyright 2020 Buf Technologies, Inc.
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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/app"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"..",
	"internal",
	"buftesting",
)

type testPluginInfo struct {
	name string
	opt  string
}

func TestOverlap(t *testing.T) {
	t.Parallel()
	// https://github.com/bufbuild/buf/issues/113
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
				bufcli.NopModuleResolverReaderProvider{},
			)
		},
		nil,
		nil,
		nil,
		"-I",
		filepath.Join("testdata", "overlap", "a"),
		"-I",
		filepath.Join("testdata", "overlap", "b"),
		"-o",
		app.DevNullFilePath,
		filepath.Join("testdata", "overlap", "a", "1.proto"),
		filepath.Join("testdata", "overlap", "b", "2.proto"),
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
	appcmdtesting.RunCommandSuccessStdout(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
				bufcli.NopModuleResolverReaderProvider{},
			)
		},
		actualProtocStdout.String(),
		nil,
		nil,
		append(
			[]string{
				"-I",
				googleapisDirPath,
				fmt.Sprintf("--%s", printFreeFieldNumbersFlagName),
			},
			filePaths...,
		)...,
	)
}

func TestCompareOutputGoogleapis(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
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
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(t,
		googleapisDirPath,
		[]testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
	)
}

func TestCompareGeneratedStubsGoogleapisGoZip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(t,
		googleapisDirPath,
		[]testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
		false,
	)
}

func TestCompareGeneratedStubsGoogleapisGoJar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubsArchive(t,
		googleapisDirPath,
		[]testPluginInfo{
			{name: "go", opt: "Mgoogle/api/auth.proto=foo"},
		},
		true,
	)
}

func TestCompareGeneratedStubsGoogleapisRuby(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(t,
		googleapisDirPath,
		[]testPluginInfo{{name: "ruby"}},
	)
}

func TestCompareInsertionPointOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	insertionTestdataDirPath := filepath.Join("testdata", "insertion")
	testCompareGeneratedStubs(t,
		insertionTestdataDirPath,
		[]testPluginInfo{
			{name: "insertion-point-receiver"},
			{name: "insertion-point-writer"},
		},
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
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
				bufcli.NopModuleResolverReaderProvider{},
			)
		},
		func(string) map[string]string {
			return map[string]string{
				"PATH": os.Getenv("PATH"),
			}
		},
		nil,
		nil,
		append(
			append(
				bufProtocPluginFlags,
				"-I",
				dirPath,
				"--by-dir",
			),
			filePaths...,
		)...,
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
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
				bufcli.NopModuleResolverReaderProvider{},
			)
		},
		func(string) map[string]string {
			return map[string]string{
				"PATH": os.Getenv("PATH"),
			}
		},
		nil,
		nil,
		append(
			append(
				bufProtocPluginFlags,
				"-I",
				dirPath,
				"--by-dir",
			),
			filePaths...,
		)...,
	)
	actualData, err := ioutil.ReadFile(actualProtocFile)
	require.NoError(t, err)
	actualReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(actualData),
		int64(len(actualData)),
		actualReadBucketBuilder,
		nil,
		0,
	)
	require.NoError(t, err)
	actualReadBucket, err := actualReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	bufData, err := ioutil.ReadFile(bufProtocFile)
	require.NoError(t, err)
	bufReadBucketBuilder := storagemem.NewReadBucketBuilder()
	err = storagearchive.Unzip(
		context.Background(),
		bytes.NewReader(bufData),
		int64(len(bufData)),
		bufReadBucketBuilder,
		nil,
		0,
	)
	require.NoError(t, err)
	bufReadBucket, err := bufReadBucketBuilder.ToReadBucket()
	require.NoError(t, err)
	diff, err := storage.DiffBytes(
		context.Background(),
		actualReadBucket,
		bufReadBucket,
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
	appcmdtesting.RunCommandSuccess(
		t,
		func(name string) *appcmd.Command {
			return NewCommand(
				name,
				appflag.NewBuilder(name),
				bufcli.NopModuleResolverReaderProvider{},
			)
		},
		nil,
		nil,
		stdout,
		append(
			[]string{
				"-I",
				dirPath,
				"-o",
				"-",
			},
			buftesting.GetProtocFilePaths(t, dirPath, 100)...,
		)...,
	)
	return stdout.Bytes()
}
