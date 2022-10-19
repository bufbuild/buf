// Copyright 2020-2022 Buf Technologies, Inc.
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

package buf

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	t.Run("bin-to-json-file-proto", func(t *testing.T) {
		testRunStdoutFile(t,
			nil,
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"--from=testdata/convert/bin_json/payload.bin",
			"testdata/convert/bin_json/buf.proto",
		)
	})
	t.Run("json-to-bin-file-proto", func(t *testing.T) {
		testRunStdoutFile(t,
			nil,
			0,
			"testdata/convert/bin_json/payload.bin",
			"beta",
			"convert",
			"--type=buf.Foo",
			"--from=testdata/convert/bin_json/payload.json",
			"testdata/convert/bin_json/buf.proto",
		)
	})
	t.Run("stdin-json-to-bin-proto", func(t *testing.T) {
		testRunStdoutFile(t,
			strings.NewReader(`{"one":"55"}`),
			0,
			"testdata/convert/bin_json/payload.bin",
			"beta",
			"convert",
			"--type=buf.Foo",
			"--from",
			"-#format=json",
			"testdata/convert/bin_json/buf.proto",
		)
	})
	t.Run("stdin-bin-to-json-proto", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/payload.bin")
		require.NoError(t, err)
		testRunStdoutFile(t, file,
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"--from",
			"-#format=bin",
			"testdata/convert/bin_json/buf.proto",
		)
	})
	t.Run("stdin-json-to-json-proto", func(t *testing.T) {
		testRunStdoutFile(t,
			strings.NewReader(`{"one":"55"}`),
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"testdata/convert/bin_json/buf.proto",
			"--from",
			"-#format=json",
			"--to",
			"-#format=json")
	})
	t.Run("stdin-input-to-json-image", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/image.bin")
		require.NoError(t, err)
		testRunStdoutFile(t, file,
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"-",
			"--from=testdata/convert/bin_json/payload.bin",
			"--to",
			"-#format=json",
		)
	})
	t.Run("stdin-json-to-json-image", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/payload.bin")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"testdata/convert/bin_json/image.bin",
			"--from",
			"-#format=bin",
			"--to",
			"-#format=json")
	})
	t.Run("stdin-bin-payload-to-json-with-image", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/payload.bin")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			"testdata/convert/bin_json/payload.json",
			"beta",
			"convert",
			"--type=buf.Foo",
			"testdata/convert/bin_json/image.bin",
			"--to",
			"-#format=json",
		)
	})
	t.Run("stdin-json-payload-to-bin-with-image", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/payload.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			"testdata/convert/bin_json/payload.bin",
			"beta",
			"convert",
			"--type=buf.Foo",
			"testdata/convert/bin_json/image.bin",
			"--from",
			"-#format=json",
			"--to",
			"-#format=bin",
		)
	})
	t.Run("stdin-image-json-to-bin", func(t *testing.T) {
		file, err := os.Open("testdata/convert/bin_json/image.json")
		require.NoError(t, err)
		testRunStdoutFile(t,
			file,
			0,
			"testdata/convert/bin_json/payload.bin",
			"beta",
			"convert",
			"--type=buf.Foo",
			"-#format=json",
			"--from=testdata/convert/bin_json/payload.json",
			"--to",
			"-#format=bin",
		)
	})
}
