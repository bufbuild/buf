package bin_json

import (
	"github.com/bufbuild/buf/private/buf/cmd/buf"
	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"strings"
	"testing"
)

// Unfortunately os.Chdir is very flaky in tests and cannot be done for only one go routine
// Putting a test file here was the best option I could come up with.
func TestConvertDir(t *testing.T) {
	cmd := func(use string) *appcmd.Command { return buf.NewRootCommand("buf") }
	t.Run("default-input", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			`{"one":"55"}`,
			nil,
			nil,
			"beta",
			"convert",
			"--type",
			"buf.Foo",
			"--from",
			"payload.bin",
		)
	})
	t.Run("from-stdin", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdoutStdinFile(
			t,
			cmd,
			0,
			`{"one":"55"}`,
			nil,
			"payload.bin",
			"beta",
			"convert",
			"--type",
			"buf.Foo",
			"--from",
			"-#format=bin",
		)
	})
	t.Run("discarded-stdin", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			`{"one":"55"}`,
			nil,
			strings.NewReader("this should be discarded"), // stdin is discarded if not needed
			"beta",
			"convert",
			"--type",
			"buf.Foo",
			"--from",
			"payload.bin",
		)
	})
	t.Run("wellknowntype", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			`"3600s"`,
			nil,
			nil,
			"beta",
			"convert",
			"--type",
			"google.protobuf.Duration",
			"--from",
			"duration.bin",
		)
	})
	t.Run("wellknowntype-bin", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdoutFile(
			t,
			cmd,
			0,
			"duration.bin",
			nil,
			nil,
			"beta",
			"convert",
			"--type=google.protobuf.Duration",
			"--from=duration.json",
			"--to",
			"-#format=bin",
		)
	})
	t.Run("wellknowntype-incorrect-input", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			1,
			"",
			nil,
			nil,
			"beta",
			"convert",
			"filedoestexist",
			"--type=google.protobuf.Duration",
			"--from=duration.json",
			"--to",
			"-#format=bin",
		)
	})
	t.Run("wellknowntype-google-file-local", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			1,
			"",
			nil,
			nil,
			"beta",
			"convert",
			"google/protobuf/timestamp.proto", // this file doesn't exist locally
			"--type=google.protobuf.Duration",
			"--from=duration.json",
			"--to",
			"-#format=bin",
		)
	})
	t.Run("wellknowntype-local-wkt-exists", func(t *testing.T) {
		expected := `{"name":"blah"}` // valid google.protobuf.Method message
		stdin := strings.NewReader(expected)
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			expected,
			nil,
			stdin,
			"beta",
			"convert",
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		)
	})
	t.Run("wellknowntype-local-changed", func(t *testing.T) {
		expected := `{"notinoriginal":"blah"}` // notinoriginal exists in the local api.proto
		stdin := strings.NewReader(expected)
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			expected,
			nil,
			stdin,
			"beta",
			"convert",
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		)
	})
	t.Run("wellknowntype-local-changed", func(t *testing.T) {
		stdin := strings.NewReader(`{"notinchanged":"blah"}`) // notinchanged does not exist in the local api.proto
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			"{}", // we expect empty json because the field doesn't exist in api.proto
			nil,
			stdin,
			"beta",
			"convert",
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		)
	})
	t.Run("wellknowntype-import", func(t *testing.T) {
		expected := `{"syntax":"SYNTAX_PROTO3"}` // Syntax is imported into type.proto
		stdin := strings.NewReader(expected)
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			expected,
			nil,
			stdin,
			"beta",
			"convert",
			"--type=google.protobuf.Type",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		)
	})
}
