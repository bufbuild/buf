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

package convert

import (
	"os"
	"strings"
	"testing"

	"buf.build/go/app/appcmd"
	"buf.build/go/app/appcmd/appcmdtesting"
	"buf.build/go/app/appext"
	"github.com/stretchr/testify/require"
)

func TestConvertDefaultInputBin(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.bin",
		),
	)
}

func TestConvertDefaultInputBinpb(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.binpb",
		),
	)
}

func TestConvertDefaultInputTxtpb(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.txtpb",
			"--to",
			"-#format=json",
		),
	)
}

func TestConvertDefaultInputYAML(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`one: "55"`),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.txtpb",
			"--to",
			"-#format=yaml",
		),
	)
}
func TestConvertFromStdinBin(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/payload.bin")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"-#format=bin",
		),
	)
}
func TestConvertFromStdinBinpb(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/payload.binpb")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"-#format=binpb",
		),
	)
}
func TestConvertFromStdinTxtpbJSON(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/payload.txtpb")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"-#format=txtpb",
			"--to",
			"-#format=json",
		),
	)
}
func TestConvertFromStdinTxtpbYAML(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/payload.txtpb")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`one: "55"`),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"-#format=txtpb",
			"--to",
			"-#format=yaml",
		),
	)
}
func TestConvertDiscardedStdin(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`{"one":"55"}`),
		appcmdtesting.WithStdin(strings.NewReader("this should be discarded")), // stdin is discarded if not needed
		appcmdtesting.WithArgs(
			"--type",
			"buf.Foo",
			"--from",
			"testdata/convert/bin_json/payload.binpb",
		),
	)
}
func TestConvertWKTBin(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`"3600s"`),
		appcmdtesting.WithArgs(
			"--type",
			"google.protobuf.Duration",
			"--from",
			"testdata/convert/bin_json/duration.bin",
		),
	)
}
func TestConvertWKTBinpb(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`"3600s"`),
		appcmdtesting.WithArgs(
			"--type",
			"google.protobuf.Duration",
			"--from",
			"testdata/convert/bin_json/duration.binpb",
		),
	)
}
func TestConvertWKTTxtpb(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`"3600s"`),
		appcmdtesting.WithArgs(
			"--type",
			"google.protobuf.Duration",
			"--from",
			"testdata/convert/bin_json/duration.txtpb",
			"--to",
			"-#format=json",
		),
	)
}
func TestConvertWKTYAML(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(`3600s`),
		appcmdtesting.WithArgs(
			"--type",
			"google.protobuf.Duration",
			"--from",
			"testdata/convert/bin_json/duration.txtpb",
			"--to",
			"-#format=yaml",
		),
	)
}
func TestConvertWKTFormatBin(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/duration.bin")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.json",
			"--to",
			"-#format=bin",
		),
	)
}
func TestConvertWKTFormatBinYAML(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/duration.bin")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.yaml",
			"--to",
			"-#format=bin",
		),
	)
}
func TestConvertWKTFormatBinpb(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/duration.binpb")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.json",
			"--to",
			"-#format=binpb",
		),
	)
}
func TestConvertWKTFormatBinpbYAML(t *testing.T) {
	t.Parallel()
	stdin, err := os.Open("testdata/convert/bin_json/duration.binpb")
	require.NoError(t, err)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.yaml",
			"--to",
			"-#format=binpb",
		),
	)
}
func TestConvertWKTIncorrectInput(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(1),
		appcmdtesting.WithExpectedStdout(""), // explicitly check for empty stdout
		appcmdtesting.WithArgs(
			"filedoestexist",
			"--type=google.protobuf.Duration",
			"--from=testdata/convert/bin_json/duration.json",
			"--to",
			"-#format=binpb",
		),
	)
}
func TestConvertWKTGoogleFileLocal(t *testing.T) {
	t.Parallel()
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(1),
		appcmdtesting.WithExpectedStdout(""), // explicitly check for empty stdout
		appcmdtesting.WithArgs(
			"google/protobuf/timestamp.proto", // this file doesn't exist locally
			"--type=google.protobuf.Duration",
			"--from=duration.json",
			"--to",
			"-#format=binpb",
		),
	)
}
func TestConvertWKTLocalWKTExists(t *testing.T) {
	t.Parallel()
	expected := `{"name":"blah"}` // valid google.protobuf.Method message
	stdin := strings.NewReader(expected)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(expected),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		),
	)
}
func TestConvertWKTLocalChanged(t *testing.T) {
	t.Parallel()
	expected := `{"notinoriginal":"blah"}` // notinoriginal exists in the local api.proto
	stdin := strings.NewReader(expected)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(expected),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		),
	)
}

// No idea what this does compared to above function - it was the same name in table tests,
// and table tests dont enforce unique test names.
func TestConvertWKTLocalChanged2(t *testing.T) {
	t.Parallel()
	stdin := strings.NewReader(`{"notinchanged":"blah"}`) // notinchanged does not exist in the local api.proto
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout("{}"), // we expect empty json because the field doesn't exist in api.proto
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Method",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		),
	)
}
func TestConvertWKTImport(t *testing.T) {
	t.Parallel()
	expected := `{"syntax":"SYNTAX_PROTO3"}` // Syntax is imported into type.proto
	stdin := strings.NewReader(expected)
	appcmdtesting.Run(
		t,
		testNewCommand,
		appcmdtesting.WithExpectedExitCode(0),
		appcmdtesting.WithExpectedStdout(expected),
		appcmdtesting.WithStdin(stdin),
		appcmdtesting.WithArgs(
			"--type=google.protobuf.Type",
			"--from=-#format=json",
			"--to",
			"-#format=json",
		),
	)
}

func testNewCommand(use string) *appcmd.Command {
	return NewCommand("convert", appext.NewBuilder("convert"))
}
