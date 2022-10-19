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
	t.Run("wkt", func(t *testing.T) {
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
}
