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

func TestOverlap(t *testing.T) {
	t.Parallel()
	// https://github.com/bufbuild/buf/issues/113
	appcmdtesting.RunCommandSuccess(
		t,
		func(use string) *appcmd.Command {
			return NewCommand(
				use,
				appflag.NewBuilder(),
				bufcli.NopModuleReaderProvider,
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
	filePaths := buftesting.GetProtocFilePaths(t, googleapisDirPath, 1000)
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
		func(use string) *appcmd.Command {
			return NewCommand(
				use,
				appflag.NewBuilder(),
				bufcli.NopModuleReaderProvider,
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
	filePaths := buftesting.GetProtocFilePaths(t, googleapisDirPath, 1000)
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
	testCompareGeneratedStubs(t, googleapisDirPath, "go", "Mgoogle/api/auth.proto=foo")
}

func TestCompareGeneratedStubsGoogleapisRuby(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	testCompareGeneratedStubs(t, googleapisDirPath, "ruby", "")
}

func testCompareGeneratedStubs(
	t *testing.T,
	dirPath string,
	pluginName string,
	pluginOpt string,
) {
	filePaths := buftesting.GetProtocFilePaths(t, dirPath, 1000)
	var baseFlags []string
	if pluginOpt != "" {
		baseFlags = append(baseFlags, fmt.Sprintf("--%s_opt=%s", pluginName, pluginOpt))
	}
	actualProtocDir := t.TempDir()
	bufProtocDir := t.TempDir()
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
		append(
			baseFlags,
			fmt.Sprintf("--%s_out=%s", pluginName, actualProtocDir),
		)...,
	)
	appcmdtesting.RunCommandSuccess(
		t,
		func(use string) *appcmd.Command {
			return NewCommand(
				use,
				appflag.NewBuilder(),
				bufcli.NopModuleReaderProvider,
			)
		},
		map[string]string{
			"PATH": os.Getenv("PATH"),
		},
		nil,
		nil,
		append(
			append(
				baseFlags,
				"-I",
				dirPath,
				fmt.Sprintf("--%s_out=%s", pluginName, bufProtocDir),
				"--by-dir",
			),
			filePaths...,
		)...,
	)
	actualReadWriteBucket, err := storageos.NewReadWriteBucket(actualProtocDir)
	require.NoError(t, err)
	bufReadWriteBucket, err := storageos.NewReadWriteBucket(bufProtocDir)
	require.NoError(t, err)
	diff, err := storage.Diff(
		context.Background(),
		actualReadWriteBucket,
		bufReadWriteBucket,
		"protoc",
		"buf-protoc",
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
		func(use string) *appcmd.Command {
			return NewCommand(
				use,
				appflag.NewBuilder(),
				bufcli.NopModuleReaderProvider,
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
			buftesting.GetProtocFilePaths(t, dirPath, 1000)...,
		)...,
	)
	return stdout.Bytes()
}
