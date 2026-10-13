// Copyright 2020-2026 Buf Technologies, Inc.
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

package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaceSliceOrStringToCommaSepString(t *testing.T) {
	t.Parallel()
	testInterfaceSliceOrStringToCommaSepString(t, "mystring", "mystring")
	testInterfaceSliceOrStringToCommaSepString(
		t,
		[]any{
			any("mystring"),
			any("mystring2"),
		},
		"mystring,mystring2",
	)
	testInterfaceSliceOrStringToCommaSepString(t, nil, "")

	_, err := InterfaceSliceOrStringToCommaSepString(1)
	require.Error(t, err)
}

func testInterfaceSliceOrStringToCommaSepString(t *testing.T, in any, expected string) {
	v, err := InterfaceSliceOrStringToCommaSepString(in)
	require.NoError(t, err)
	require.Equal(t, expected, v)
}

func TestUnmarshalYAMLStrictRejectsLocalTags(t *testing.T) {
	t.Parallel()
	type external struct {
		Ignore []string `yaml:"ignore"`
	}
	// An unquoted value that starts with "!" is a YAML local tag with an
	// empty value, which the decoder would otherwise silently drop.
	var v external
	err := UnmarshalYAMLStrict([]byte("ignore:\n  - foo\n  - !foo/bar.proto\n"), &v)
	require.Error(t, err)
	require.Contains(t, err.Error(), `line 3: unexpected tag "!foo/bar.proto"`)

	v = external{}
	err = UnmarshalYAMLStrict([]byte("ignore:\n  - !local bar\n"), &v)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unexpected tag "!local"`)

	// Standard tags, anchors, aliases, and quoted values that start with "!"
	// are all fine.
	v = external{}
	err = UnmarshalYAMLStrict([]byte("ignore:\n  - !!str 123\n  - &a foo\n  - *a\n  - \"!foo/bar.proto\"\n"), &v)
	require.NoError(t, err)
	require.Equal(t, []string{"123", "foo", "foo", "!foo/bar.proto"}, v.Ignore)

	// Non-strict unmarshalling is unchanged.
	v = external{}
	err = UnmarshalYAMLNonStrict([]byte("ignore:\n  - !foo/bar.proto\n"), &v)
	require.NoError(t, err)
	require.Equal(t, []string{""}, v.Ignore)
}
