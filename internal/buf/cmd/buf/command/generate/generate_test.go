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

package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufgen"
	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagearchive"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buftestingDirPath = filepath.Join(
	"..",
	"..",
	"..",
	"..",
	"internal",
	"buftesting",
)

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
	bufGenDir := t.TempDir()
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
	genFlags := []string{
		"--input",
		dirPath,
		"--template",
		newExternalConfigV1Beta1String(t, plugins, bufGenDir),
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--file",
			filePath,
		)
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
		genFlags...,
	)
	actualReadWriteBucket, err := storageos.NewReadWriteBucket(actualProtocDir)
	require.NoError(t, err)
	bufReadWriteBucket, err := storageos.NewReadWriteBucket(bufGenDir)
	require.NoError(t, err)
	diff, err := storage.Diff(
		context.Background(),
		actualReadWriteBucket,
		bufReadWriteBucket,
		"protoc",
		"buf-generate",
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
	bufGenFile := filepath.Join(tempDir, "buf-generate"+fileExt)
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
	genFlags := []string{
		"--input",
		dirPath,
		"--template",
		newExternalConfigV1Beta1String(t, plugins, bufGenFile),
	}
	for _, filePath := range filePaths {
		genFlags = append(
			genFlags,
			"--file",
			filePath,
		)
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
		genFlags...,
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
	bufData, err := ioutil.ReadFile(bufGenFile)
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
	diff, err := storage.Diff(
		context.Background(),
		actualReadBucket,
		bufReadBucket,
		"protoc",
		"buf-generate",
	)
	require.NoError(t, err)
	assert.Empty(t, string(diff))
}

type testPluginInfo struct {
	name string
	opt  string
}

func newExternalConfigV1Beta1String(t *testing.T, plugins []testPluginInfo, out string) string {
	externalConfig := bufgen.ExternalConfigV1Beta1{
		Version: "v1beta1",
	}
	for _, plugin := range plugins {
		externalConfig.Plugins = append(
			externalConfig.Plugins,
			bufgen.ExternalPluginConfigV1Beta1{
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
