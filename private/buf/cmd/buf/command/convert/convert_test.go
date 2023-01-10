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

package convert

import (
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
	"github.com/bufbuild/buf/private/pkg/app/appcmd/appcmdtesting"
	"github.com/bufbuild/buf/private/pkg/app/appflag"
)

// This test is in its own file as opposed to buf_test because it needs to test a single module in testdata.
func TestConvertDir(t *testing.T) {
	cmd := func(use string) *appcmd.Command { return NewCommand("convert", appflag.NewBuilder("convert")) }
	t.Run("default-input", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdout(
			t,
			cmd,
			0,
			`{"one":"55"}`,
			nil,
			nil,
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.bin",
		)
	})
	t.Run("from-stdin", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdoutStdinFile(
			t,
			cmd,
			0,
			`{"one":"55"}`,
			nil,
			"testdata/convert/bin_json/payload.bin",

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
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.bin",
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
			"--type",
			"google.protobuf.Duration",
			"--from",
			"testdata/convert/bin_json/duration.bin",
		)
	})
	t.Run("wellknowntype-bin", func(t *testing.T) {
		appcmdtesting.RunCommandExitCodeStdoutFile(
			t,
			cmd,
			0,
			"testdata/convert/bin_json/duration.bin",
			nil,
			nil,
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.json",
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
			"filedoestexist",
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.json",
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
			"--type=google.protobuf.Type",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		)
	})
}
