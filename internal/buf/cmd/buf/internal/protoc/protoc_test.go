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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/internal/buf/internal/buftesting"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd"
	"github.com/bufbuild/buf/internal/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/internal/pkg/app/appflag"
	"github.com/bufbuild/buf/internal/pkg/protoencoding"
	"github.com/bufbuild/buf/internal/pkg/prototesting"
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

func TestComparePrintFreeFieldNumbersGoogleapis(t *testing.T) {
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	actualProtocStdout := bytes.NewBuffer(nil)
	buftesting.RunActualProtoc(
		t,
		false,
		false,
		googleapisDirPath,
		actualProtocStdout,
		fmt.Sprintf("--%s", printFreeFieldNumbersFlagName),
	)
	appcmdtesting.RunCommandSuccessStdout(
		t,
		func(use string) *appcmd.Command {
			return NewCommand(
				use,
				appflag.NewBuilder(),
			)
		},
		actualProtocStdout.String(),
		nil,
		append(
			[]string{
				"-I",
				googleapisDirPath,
				fmt.Sprintf("--%s", printFreeFieldNumbersFlagName),
			},
			buftesting.GetProtocFilePaths(t, googleapisDirPath)...,
		)...,
	)
}

func TestCompareOutputJSONGoogleapis(t *testing.T) {
	t.Parallel()
	googleapisDirPath := buftesting.GetGoogleapisDirPath(t, buftestingDirPath)
	actualProtocFileDescriptorSet := buftesting.GetActualProtocFileDescriptorSet(
		t,
		false,
		false,
		googleapisDirPath,
	)
	bufProtocFileDescriptorSet := testGetBufProtocFileDescriptorSet(t, googleapisDirPath)
	diffOne, err := prototesting.DiffFileDescriptorSetsJSON(bufProtocFileDescriptorSet, actualProtocFileDescriptorSet, "buf-protoc")
	assert.NoError(t, err)
	assert.Equal(t, "", diffOne, "JSON diff:\n%s", diffOne)
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
			)
		},
		nil,
		stdout,
		append(
			[]string{
				"-I",
				dirPath,
				"-o",
				"-",
			},
			buftesting.GetProtocFilePaths(t, dirPath)...,
		)...,
	)
	return stdout.Bytes()
}
