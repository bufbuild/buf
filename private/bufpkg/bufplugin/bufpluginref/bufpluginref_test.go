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

package bufpluginref

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginIdentityForString(t *testing.T) {
	t.Parallel()
	expectedPluginIdentity, err := NewPluginIdentity("foo.com", "bar", "baz")
	require.NoError(t, err)
	assert.Equal(t, "foo.com", expectedPluginIdentity.Remote())
	assert.Equal(t, "bar", expectedPluginIdentity.Owner())
	assert.Equal(t, "baz", expectedPluginIdentity.Plugin())
	pluginIdentity, err := PluginIdentityForString("foo.com/bar/baz")
	require.NoError(t, err)
	assert.Equal(t, expectedPluginIdentity, pluginIdentity)
}

func TestPluginIdentityForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "no remote",
			Input: "/bar",
		},
		{
			Name:  "no owner",
			Input: "foo.com",
		},
		{
			Name:  "empty owner",
			Input: "foo.com//baz",
		},
		{
			Name:  "no plugin",
			Input: "foo.com/bar",
		},
		{
			Name:  "empty plugin",
			Input: "foo.com/bar/",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := PluginIdentityForString(testCase.Input)
			require.Error(t, err)
		})
	}
}

func TestPluginReferenceForString(t *testing.T) {
	t.Parallel()
	expectedPluginReference, err := NewPluginReference("foo.com", "barr", "baz", "v1")
	require.NoError(t, err)
	require.Equal(t, "foo.com/barr/baz:v1", expectedPluginReference.String())
	pluginReference, err := PluginReferenceForString("foo.com/barr/baz:v1")
	require.NoError(t, err)
	require.Equal(t, expectedPluginReference, pluginReference)
}

func TestPluginReferenceForStringError(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name  string
		Input string
	}{
		{
			Name:  "Plugin without a remote",
			Input: "/barr/baz:v1",
		},
		{
			Name:  "Plugin without an owner",
			Input: "foo.com//baz:v1",
		},
		{
			Name:  "Plugin without a plugin name",
			Input: "foo.com/barr/:v1",
		},
		{
			Name:  "Plugin without a reference",
			Input: "foo.com/barr/baz:",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			_, err := PluginReferenceForString(testCase.Input)
			require.Error(t, err)
		})
	}
}
