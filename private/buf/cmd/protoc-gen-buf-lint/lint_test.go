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

package lint

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/buf/private/pkg/stringutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestRunLint1(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		"",
		[]string{
			normalpath.Join("buf", "buf.proto"),
			normalpath.Join("buf", "buf_two.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		buf/buf.proto:3:1:Package name "other" should be suffixed with a correctly formed version, such as "other.v1".
		buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".
		buf/buf_two.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		buf/buf_two.proto:3:1:Package name "other" should be suffixed with a correctly formed version, such as "other.v1".
		buf/buf_two.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".
		`,
	)
}

func TestRunLint2(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		"",
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		buf/buf.proto:3:1:Package name "other" should be suffixed with a correctly formed version, such as "other.v1".
		buf/buf.proto:6:9:Field name "oneTwo" should be lower_snake_case, such as "one_two".
		`,
	)
}

func TestRunLint3(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":"testdata/fail/something.yaml"}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		`,
	)
}

func TestRunLint4(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":{"version":"v1","lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}}}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		`,
	)
}

func TestRunLint5(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":{"version":"v1","lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}}}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		`,
	)
}

func TestRunLint6(t *testing.T) {
	// specifically testing that output is stable
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":{"version":"v1","lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}},"error_format":"json"}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		{"path":"buf/buf.proto","start_line":3,"start_column":1,"end_line":3,"end_column":15,"type":"PACKAGE_DIRECTORY_MATCH","message":"Files with package \"other\" must be within a directory \"other\" relative to root but were in directory \"buf\"."}
		`,
	)
}

func TestRunLint7(t *testing.T) {
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":{"version":"v1","lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}},"error_format":"json"}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		{"path":"buf/buf.proto","start_line":3,"start_column":1,"end_line":3,"end_column":15,"type":"PACKAGE_DIRECTORY_MATCH","message":"Files with package \"other\" must be within a directory \"other\" relative to root but were in directory \"buf\"."}
		`,
	)
}

func testRunLint(
	t *testing.T,
	root string,
	realFilePaths []string,
	parameter string,
	fileToGenerate []string,
	expectedExitCode int,
	expectedErrorString string,
) {
	t.Parallel()
	runner := command.NewRunner()
	testRunHandlerFunc(
		t,
		appproto.HandlerFunc(
			func(
				ctx context.Context,
				container app.EnvStderrContainer,
				responseWriter appproto.ResponseBuilder,
				request *pluginpb.CodeGeneratorRequest,
			) error {
				return handle(
					ctx,
					container,
					responseWriter,
					request,
				)
			},
		),
		testBuildCodeGeneratorRequest(
			t,
			runner,
			root,
			realFilePaths,
			parameter,
			fileToGenerate,
		),
		expectedExitCode,
		expectedErrorString,
	)
}

func testRunHandlerFunc(
	t *testing.T,
	handler appproto.Handler,
	request *pluginpb.CodeGeneratorRequest,
	expectedExitCode int,
	expectedErrorString string,
) {
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request)
	require.NoError(t, err)
	stdin := bytes.NewReader(requestData)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	exitCode := app.GetExitCode(
		appproto.Run(
			context.Background(),
			app.NewContainer(
				nil,
				stdin,
				stdout,
				stderr,
			),
			handler,
		),
	)

	require.Equal(t, expectedExitCode, exitCode, stringutil.TrimLines(stderr.String()))
	if exitCode == 0 {
		response := &pluginpb.CodeGeneratorResponse{}
		// we do not need fileDescriptorProtos as there are no extensions
		unmarshaler := protoencoding.NewWireUnmarshaler(nil)
		require.NoError(t, unmarshaler.Unmarshal(stdout.Bytes(), response))
		require.Equal(t, stringutil.TrimLines(expectedErrorString), response.GetError(), stringutil.TrimLines(stderr.String()))
	}
}

func testBuildCodeGeneratorRequest(
	t *testing.T,
	runner command.Runner,
	root string,
	realFilePaths []string,
	parameter string,
	fileToGenerate []string,
) *pluginpb.CodeGeneratorRequest {
	fileDescriptorSet, err := prototesting.GetProtocFileDescriptorSet(
		context.Background(),
		runner,
		[]string{root},
		realFilePaths,
		true,
		true,
	)
	require.NoError(t, err)
	nonImportRootRelFilePaths := make(map[string]struct{}, len(fileToGenerate))
	for _, fileToGenerateFilePath := range fileToGenerate {
		nonImportRootRelFilePaths[fileToGenerateFilePath] = struct{}{}
	}
	imageFiles := make([]bufimage.ImageFile, len(fileDescriptorSet.File))
	for i, fileDescriptorProto := range fileDescriptorSet.File {
		_, isNotImport := nonImportRootRelFilePaths[fileDescriptorProto.GetName()]
		imageFile, err := bufimage.NewImageFile(
			fileDescriptorProto,
			nil,
			"",
			"",
			!isNotImport,
			false,
			nil,
		)
		require.NoError(t, err)
		imageFiles[i] = imageFile
	}
	image, err := bufimage.NewImage(imageFiles)
	require.NoError(t, err)
	return bufimage.ImageToCodeGeneratorRequest(image, parameter, nil, false, false)
}
