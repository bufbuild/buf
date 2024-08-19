// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufanalysistesting

import (
	"strings"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufanalysis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	fileAnnotations := []bufanalysis.FileAnnotation{
		newFileAnnotation(
			t,
			"path/to/file.proto",
			1,
			0,
			1,
			0,
			"FOO",
			"Hello.",
			"",
		),
		newFileAnnotation(
			t,
			"path/to/file.proto",
			2,
			1,
			2,
			1,
			"FOO",
			"Hello.",
			"buf-plugin-foo",
		),
	}
	sb := &strings.Builder{}
	err := bufanalysis.PrintFileAnnotationSet(sb, bufanalysis.NewFileAnnotationSet(fileAnnotations...), "text")
	require.NoError(t, err)
	assert.Equal(
		t,
		`path/to/file.proto:1:1:Hello.
path/to/file.proto:2:1:Hello. (buf-plugin-foo)
`,
		sb.String(),
	)
	sb.Reset()
	err = bufanalysis.PrintFileAnnotationSet(sb, bufanalysis.NewFileAnnotationSet(fileAnnotations...), "json")
	require.NoError(t, err)
	assert.Equal(
		t,
		`{"path":"path/to/file.proto","start_line":1,"start_column":1,"end_line":1,"end_column":1,"type":"FOO","message":"Hello."}
{"path":"path/to/file.proto","start_line":2,"start_column":1,"end_line":2,"end_column":1,"type":"FOO","message":"Hello.","plugin":"buf-plugin-foo"}
`,
		sb.String(),
	)
	sb.Reset()
	err = bufanalysis.PrintFileAnnotationSet(sb, bufanalysis.NewFileAnnotationSet(fileAnnotations...), "msvs")
	require.NoError(t, err)
	assert.Equal(t,
		`path/to/file.proto(1,1) : error FOO : Hello.
path/to/file.proto(2,1) : error FOO : Hello. (buf-plugin-foo)
`,
		sb.String(),
	)
	sb.Reset()
	err = bufanalysis.PrintFileAnnotationSet(sb, bufanalysis.NewFileAnnotationSet(fileAnnotations...), "junit")
	require.NoError(t, err)
	assert.Equal(t,
		`<testsuites>
  <testsuite name="path/to/file" tests="2" failures="2" errors="0">
    <testcase name="FOO_1">
      <failure message="path/to/file.proto:1:1:Hello." type="FOO"></failure>
    </testcase>
    <testcase name="FOO_2_1">
      <failure message="path/to/file.proto:2:1:Hello. (buf-plugin-foo)" type="FOO"></failure>
    </testcase>
  </testsuite>
</testsuites>
`,
		sb.String(),
	)
	sb.Reset()
	err = bufanalysis.PrintFileAnnotationSet(
		sb,
		bufanalysis.NewFileAnnotationSet(
			append(
				fileAnnotations,
				newFileAnnotation(
					t,
					"path/to/file.proto",
					0,
					0,
					0,
					0,
					"FOO",
					"Hello.",
					"",
				),
				newFileAnnotation(
					t,
					"",
					0,
					0,
					0,
					0,
					"FOO",
					"Hello.",
					"buf-plugin-foo",
				),
			)...,
		),
		"github-actions",
	)
	require.NoError(t, err)
	assert.Equal(t,
		`::error file=<input>::Hello. (buf-plugin-foo)
::error file=path/to/file.proto::Hello.
::error file=path/to/file.proto,line=1,endLine=1::Hello.
::error file=path/to/file.proto,line=2,col=1,endLine=2,endColumn=1::Hello. (buf-plugin-foo)
`,
		sb.String(),
	)
}
