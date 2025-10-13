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

package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"buf.build/go/app"
	"buf.build/go/standard/xstrings"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/prototesting"
	"github.com/bufbuild/protoplugin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestRunLint1(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestRunLint8(t *testing.T) {
	t.Parallel()
	testRunLint(
		t,
		filepath.Join("testdata", "fail"),
		[]string{
			filepath.Join("testdata", "fail", "buf", "buf.proto"),
			filepath.Join("testdata", "fail", "buf", "buf_two.proto"),
		},
		`{"input_config":"testdata/fail/v2.yaml","module":"fail"}`,
		[]string{
			normalpath.Join("buf", "buf.proto"),
		},
		0,
		`
		buf/buf.proto:3:1:Files with package "other" must be within a directory "other" relative to root but were in directory "buf".
		`,
	)
}

func TestRunLint_UnusedImports(t *testing.T) {
	unusedImportsFileComponents := [][]string{
		{"buf", "v1", "a.proto"},
		{"buf", "v1", "b.proto"},
		{"buf", "v1", "c.proto"},
		{"buf", "v1", "d.proto"},
		{"buf", "v1", "e.proto"},
		{"buf", "v1", "f.proto"},
		{"buf", "v1", "g.proto"},
		{"buf", "v1", "file_option.proto"},
		{"buf", "v1", "msg_option.proto"},
		{"buf", "v1", "field_option.proto"},
		{"buf", "v1", "oneof_option.proto"},
		{"buf", "v1", "extrange_option.proto"},
		{"buf", "v1", "enum_option.proto"},
		{"buf", "v1", "enumvalue_option.proto"},
		{"buf", "v1", "service_option.proto"},
		{"buf", "v1", "method_option.proto"},
	}
	unusedImportFilesWithFullPath := make([]string, len(unusedImportsFileComponents))
	for i, components := range unusedImportsFileComponents {
		unusedImportFilesWithFullPath[i] = filepath.Join(append([]string{"testdata", "unused-imports"}, components...)...)
	}
	unusedImportFiles := make([]string, len(unusedImportsFileComponents))
	for i, components := range unusedImportsFileComponents {
		unusedImportFiles[i] = normalpath.Join(components...)
	}
	t.Parallel()
	testRunLint(
		t,
		filepath.Join("testdata", "unused-imports"),
		unusedImportFilesWithFullPath,
		``,
		unusedImportFiles,
		0,
		`
		buf/v1/a.proto:13:1:Import "buf/v1/f.proto" is unused.
		buf/v1/a.proto:14:1:Import "buf/v1/extrange_option.proto" is unused.
		buf/v1/b.proto:5:1:Import "buf/v1/c.proto" must not be public.
		buf/v1/c.proto:9:1:Import "buf/v1/e.proto" is unused.
		buf/v1/c.proto:10:1:Import "buf/v1/f.proto" is unused.
		buf/v1/d.proto:8:1:Import "buf/v1/f.proto" is unused.
		buf/v1/e.proto:8:1:Import "buf/v1/g.proto" is unused.
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
	testRunHandlerFunc(
		t,
		protoplugin.HandlerFunc(
			func(
				ctx context.Context,
				pluginEnv protoplugin.PluginEnv,
				responseWriter protoplugin.ResponseWriter,
				request protoplugin.Request,
			) error {
				return handle(
					ctx,
					pluginEnv,
					responseWriter,
					request,
				)
			},
		),
		testBuildRequest(
			t,
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
	handler protoplugin.Handler,
	request protoplugin.Request,
	expectedExitCode int,
	expectedErrorString string,
) {
	requestData, err := protoencoding.NewWireMarshaler().Marshal(request.CodeGeneratorRequest())
	require.NoError(t, err)
	stdin := bytes.NewReader(requestData)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	exitCode := app.GetExitCode(
		protoplugin.Run(
			context.Background(),
			protoplugin.Env{
				Args:    nil,
				Stdin:   stdin,
				Stdout:  stdout,
				Stderr:  stderr,
				Environ: nil,
			},
			handler,
		),
	)

	require.Equal(t, expectedExitCode, exitCode, xstrings.TrimLines(stderr.String()))
	if exitCode == 0 {
		response := &pluginpb.CodeGeneratorResponse{}
		// we do not need fileDescriptorProtos as there are no extensions
		unmarshaler := protoencoding.NewWireUnmarshaler(nil)
		require.NoError(t, unmarshaler.Unmarshal(stdout.Bytes(), response))
		require.Equal(t, xstrings.TrimLines(expectedErrorString), response.GetError(), xstrings.TrimLines(stderr.String()))
	}
}

func testBuildRequest(
	t *testing.T,
	root string,
	realFilePaths []string,
	parameter string,
	fileToGenerate []string,
) protoplugin.Request {
	fileDescriptorSet, err := prototesting.GetProtocFileDescriptorSet(
		context.Background(),
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
			uuid.Nil,
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
	codeGenReq, err := bufimage.ImageToCodeGeneratorRequest(image, parameter, nil, false, false)
	require.NoError(t, err)
	request, err := protoplugin.NewRequest(codeGenReq)
	require.NoError(t, err)
	return request
}
