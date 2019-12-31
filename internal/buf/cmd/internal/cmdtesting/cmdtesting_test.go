package cmdtesting

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	lint "github.com/bufbuild/buf/internal/buf/cmd/protoc-gen-buf-check-lint"
	"github.com/bufbuild/buf/internal/pkg/ext/extdescriptor"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto/utilprototesting"
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
	"github.com/bufbuild/cli/clienv"
	"github.com/bufbuild/cli/cliproto"
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/stretchr/testify/require"
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
			filepath.Join("buf", "buf.proto"),
			filepath.Join("buf", "buf_two.proto"),
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
			filepath.Join("buf", "buf.proto"),
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
			filepath.Join("buf", "buf.proto"),
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
		`{"input_config":{"lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}}}`,
		[]string{
			filepath.Join("buf", "buf.proto"),
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
		`{"input_config":{"lint":{"use":["PACKAGE_DIRECTORY_MATCH"]}},"error_format":"json"}`,
		[]string{
			filepath.Join("buf", "buf.proto"),
		},
		0,
		`
		{"filename":"buf/buf.proto","start_line":3,"start_column":1,"end_line":3,"end_column":15,"type":"PACKAGE_DIRECTORY_MATCH","message":"Files with package \"other\" must be within a directory \"other\" relative to root but were in directory \"buf\"."}
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

	testRunHandlerFunc(
		t,
		cliproto.HandlerFunc(lint.Handle),
		testBuildCodeGeneratorRequest(
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
	handlerFunc cliproto.HandlerFunc,
	request *plugin_go.CodeGeneratorRequest,
	expectedExitCode int,
	expectedErrorString string,
) {
	requestData, err := proto.Marshal(request)
	require.NoError(t, err)
	stdin := bytes.NewReader(requestData)
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	exitCode := cliproto.Run(
		handlerFunc,
		clienv.NewEnv(
			nil,
			stdin,
			stdout,
			stderr,
			nil,
		),
	)

	require.Equal(t, expectedExitCode, exitCode, utilstring.TrimLines(stderr.String()))
	if exitCode == 0 {
		response := &plugin_go.CodeGeneratorResponse{}
		require.NoError(t, proto.Unmarshal(stdout.Bytes(), response))
		require.Equal(t, utilstring.TrimLines(expectedErrorString), response.GetError(), utilstring.TrimLines(stderr.String()))
	}
}

func testBuildCodeGeneratorRequest(
	t *testing.T,
	root string,
	realFilePaths []string,
	parameter string,
	fileToGenerate []string,
) *plugin_go.CodeGeneratorRequest {
	fileDescriptorSet, err := utilprototesting.GetProtocFileDescriptorSet(
		context.Background(),
		[]string{root},
		realFilePaths,
		true,
		true,
	)
	require.NoError(t, err)
	request, err := extdescriptor.FileDescriptorSetToCodeGeneratorRequest(fileDescriptorSet, parameter, fileToGenerate...)
	require.NoError(t, err)
	return request
}
