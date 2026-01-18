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

package bufgen

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstitutePostCommandVariables(t *testing.T) {
	t.Parallel()

	strategy := bufconfig.GenerateStrategyDirectory
	pluginConfig, err := bufconfig.NewLocalGeneratePluginConfig(
		"protoc-gen-python",
		"gen/python",
		[]string{"paths=source_relative", "module=test"},
		false,
		false,
		nil,
		nil,
		&strategy,
		[]string{"/usr/bin/protoc-gen-python"},
	)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		command  string
		out      string
		expected string
	}{
		{
			name:     "substitute $out",
			command:  "ruff --fix $out",
			out:      "gen/python",
			expected: "ruff --fix gen/python",
		},
		{
			name:     "substitute $out with base dir",
			command:  "black $out",
			out:      "output/gen/python",
			expected: "black output/gen/python",
		},
		{
			name:     "substitute $name",
			command:  "echo $name",
			out:      "gen/python",
			expected: "echo protoc-gen-python",
		},
		{
			name:     "substitute $opt",
			command:  "echo $opt",
			out:      "gen/python",
			expected: "echo paths=source_relative,module=test",
		},
		{
			name:     "substitute $path",
			command:  "echo $path",
			out:      "gen/python",
			expected: "echo /usr/bin/protoc-gen-python",
		},
		{
			name:     "substitute $strategy",
			command:  "echo $strategy",
			out:      "gen/python",
			expected: "echo directory",
		},
		{
			name:     "substitute multiple variables",
			command:  "process --dir $out --plugin $name --strategy $strategy",
			out:      "gen/python",
			expected: "process --dir gen/python --plugin protoc-gen-python --strategy directory",
		},
		{
			name:     "no substitution needed",
			command:  "gofmt -w .",
			out:      "gen/python",
			expected: "gofmt -w .",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := substitutePostCommandVariables(tc.command, pluginConfig, tc.out)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSubstitutePostCommandVariablesStrategyAll(t *testing.T) {
	t.Parallel()

	strategy := bufconfig.GenerateStrategyAll
	pluginConfig, err := bufconfig.NewLocalGeneratePluginConfig(
		"protoc-gen-go",
		"gen/go",
		nil,
		false,
		false,
		nil,
		nil,
		&strategy,
		[]string{"protoc-gen-go"},
	)
	require.NoError(t, err)

	result := substitutePostCommandVariables("echo $strategy", pluginConfig, "gen/go")
	assert.Equal(t, "echo all", result)
}
