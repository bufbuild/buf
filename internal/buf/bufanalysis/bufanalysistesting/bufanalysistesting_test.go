// Copyright 2020-2021 Buf Technologies, Inc.
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
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufanalysis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	fileAnnotation := newFileAnnotation(
		t,
		"path/to/file.proto",
		1,
		0,
		1,
		0,
		"FOO",
		"Hello.",
	)
	s, err := bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatText)
	require.NoError(t, err)
	assert.Equal(t, `path/to/file.proto:1:1:Hello.`, s)
	s, err = bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatJSON)
	require.NoError(t, err)
	assert.Equal(t, `{"path":"path/to/file.proto","start_line":1,"end_line":1,"type":"FOO","message":"Hello."}`, s)
	s, err = bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatMSVS)
	require.NoError(t, err)
	assert.Equal(t, `path/to/file.proto(1) : error FOO : Hello.`, s)

	fileAnnotation = newFileAnnotation(
		t,
		"path/to/file.proto",
		2,
		1,
		2,
		1,
		"FOO",
		"Hello.",
	)
	s, err = bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatText)
	require.NoError(t, err)
	assert.Equal(t, `path/to/file.proto:2:1:Hello.`, s)
	s, err = bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatJSON)
	require.NoError(t, err)
	assert.Equal(t, `{"path":"path/to/file.proto","start_line":2,"start_column":1,"end_line":2,"end_column":1,"type":"FOO","message":"Hello."}`, s)
	s, err = bufanalysis.FormatFileAnnotation(fileAnnotation, bufanalysis.FormatMSVS)
	require.NoError(t, err)
	assert.Equal(t, `path/to/file.proto(2,1) : error FOO : Hello.`, s)
}
