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

package transform

import (
	"net/http"
	"testing"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_errors(t *testing.T) {
	tests := []struct {
		builders []ClientBuilder
		wantErr  string
	}{
		{
			builders: nil,
			wantErr:  "SchemaServiceClient not provided",
		},
		{
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
			},
			wantErr: "buf module owner not provided",
		},
		{
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "", ""),
			},
			wantErr: "buf module repository not provided",
		},
		{
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", ""),
			},
			wantErr: "buf module version not provided",
		},
		{
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", "baz"),
			},
			wantErr: "input_format value FORMAT_UNSPECIFIED is not valid",
		},
		{
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", "baz"),
				FromFormat(registryv1alpha1.Format_FORMAT_BINARY, false),
			},
			wantErr: "output_format not specified",
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.wantErr, func(t *testing.T) {
			got, err := NewClient(test.builders...)
			require.Error(t, err)
			require.Nil(t, got)
			assert.EqualError(t, err, test.wantErr)
		})
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		builders []ClientBuilder
	}{
		{
			name: "client from binary to json",
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", "baz"),
				FromFormat(registryv1alpha1.Format_FORMAT_BINARY, false),
				ToJSONOutput(false, false),
			},
		},
		{
			name: "client from binary to text",
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", "baz"),
				FromFormat(registryv1alpha1.Format_FORMAT_BINARY, false),
				ToTextOutput(),
			},
		},
		{
			name: "client from json to binary",
			builders: []ClientBuilder{
				WithNewSchemaService(http.DefaultClient, "localhost:4567"),
				WithBufModule("foo", "bar", "baz"),
				FromFormat(registryv1alpha1.Format_FORMAT_JSON, false),
				ToBinaryOutput(),
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			got, err := NewClient(test.builders...)
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestWithSchemaService(t *testing.T) {
	c := &Client{}
	WithNewSchemaService(http.DefaultClient, "")(c)
	assert.NotNil(t, c.client)
}

func TestWithCache(t *testing.T) {
	c := &Client{}
	WithCache()(c)
	assert.NotNil(t, c.cache)
}

func TestWithBufModule(t *testing.T) {
	c := &Client{}
	WithBufModule("foo", "bar", "baz")(c)
	assert.Equal(t, "foo", c.owner)
	assert.Equal(t, "bar", c.repository)
	assert.Equal(t, "baz", c.version)
}

func TestIncludeTypes(t *testing.T) {
	c := &Client{}
	IncludeTypes("foo", "bar", "baz")(c)
	assert.Len(t, c.types, 3)
	assert.Contains(t, c.types, "foo")
	assert.Contains(t, c.types, "bar")
	assert.Contains(t, c.types, "baz")
}

func TestFromFormat(t *testing.T) {
	t.Run("binary format without discard unknown", func(t *testing.T) {
		c := &Client{}
		FromFormat(registryv1alpha1.Format_FORMAT_BINARY, false)(c)
		assert.Equal(t, registryv1alpha1.Format_FORMAT_BINARY, c.inputFormat)
		assert.False(t, c.discardUnknown)
	})
	t.Run("binary format with discard unknown", func(t *testing.T) {
		c := &Client{}
		FromFormat(registryv1alpha1.Format_FORMAT_JSON, true)(c)
		assert.Equal(t, registryv1alpha1.Format_FORMAT_JSON, c.inputFormat)
		assert.True(t, c.discardUnknown)
	})
}

func TestExclude(t *testing.T) {
	c := &Client{}
	Exclude(true, true)(c)
	assert.True(t, c.excludeKnownExtensions)
	assert.True(t, c.excludeCustomOptions)
}

func TestIfNotCommit(t *testing.T) {
	c := &Client{}
	IfNotCommit("baz")(c)
	assert.Equal(t, "baz", c.ifNotCommit)
}

func TestToBinaryOutput(t *testing.T) {
	c := &Client{}
	ToBinaryOutput()(c)
	switch c.outputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary:
	case *registryv1alpha1.ConvertMessageRequest_OutputJson:
		assert.FailNow(t, "incorrect format")
	case *registryv1alpha1.ConvertMessageRequest_OutputText:
		assert.FailNow(t, "incorrect format")
	default:
		assert.FailNow(t, "incorrect format")
	}
}

func TestToJSONOutput(t *testing.T) {
	c := &Client{}
	ToJSONOutput(true, true)(c)
	switch format := c.outputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary:
		assert.FailNow(t, "incorrect format")
	case *registryv1alpha1.ConvertMessageRequest_OutputJson:
		assert.True(t, format.OutputJson.IncludeDefaults)
		assert.True(t, format.OutputJson.UseEnumNumbers)
	case *registryv1alpha1.ConvertMessageRequest_OutputText:
		assert.FailNow(t, "incorrect format")
	default:
		assert.FailNow(t, "incorrect format")
	}
}

func TestToTextOutput(t *testing.T) {
	c := &Client{}
	ToTextOutput()(c)
	switch c.outputFormat.(type) {
	case *registryv1alpha1.ConvertMessageRequest_OutputBinary:
		assert.FailNow(t, "incorrect format")
	case *registryv1alpha1.ConvertMessageRequest_OutputJson:
		assert.FailNow(t, "incorrect format")
	case *registryv1alpha1.ConvertMessageRequest_OutputText:
	default:
		assert.FailNow(t, "incorrect format")
	}
}
