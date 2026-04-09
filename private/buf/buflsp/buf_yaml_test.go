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

package buflsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestParseBufYAMLDeps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		content         string
		wantDepsKeyLine uint32
		wantDeps        []bufYAMLDep
	}{
		{
			name:     "empty-no-deps-key",
			content:  "version: v2\n",
			wantDeps: nil,
		},
		{
			name: "v1-single-dep",
			content: `version: v1
deps:
  - buf.build/googleapis/googleapis
`,
			// "deps:" is on line 1 (0-indexed)
			wantDepsKeyLine: 1,
			wantDeps: []bufYAMLDep{
				{
					ref: "buf.build/googleapis/googleapis",
					depRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 4 + uint32(len("buf.build/googleapis/googleapis"))},
					},
				},
			},
		},
		{
			name: "v2-multiple-deps",
			content: `version: v2
deps:
  - buf.build/googleapis/googleapis
  - buf.build/grpc/grpc:v1
`,
			wantDepsKeyLine: 1,
			wantDeps: []bufYAMLDep{
				{
					ref: "buf.build/googleapis/googleapis",
					depRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 4},
						End:   protocol.Position{Line: 2, Character: 4 + uint32(len("buf.build/googleapis/googleapis"))},
					},
				},
				{
					ref: "buf.build/grpc/grpc:v1",
					depRange: protocol.Range{
						Start: protocol.Position{Line: 3, Character: 4},
						End:   protocol.Position{Line: 3, Character: 4 + uint32(len("buf.build/grpc/grpc:v1"))},
					},
				},
			},
		},
		{
			name:     "no-deps-key",
			content:  "version: v2\nmodules:\n  - path: .\n",
			wantDeps: nil,
		},
		{
			name:            "empty-deps",
			content:         "version: v2\ndeps: []\n",
			wantDepsKeyLine: 1,
			wantDeps:        []bufYAMLDep{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotKeyLine, gotDeps, err := parseBufYAMLDeps([]byte(tt.content))
			require.NoError(t, err)
			assert.Equal(t, tt.wantDepsKeyLine, gotKeyLine)
			assert.Equal(t, tt.wantDeps, gotDeps)
		})
	}
}

func TestIsBufYAMLURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		uri      protocol.URI
		expected bool
	}{
		{"file:///home/user/project/buf.yaml", true},
		{"file:///home/user/project/buf.work.yaml", false},
		{"file:///home/user/project/foo.proto", false},
		{"file:///home/user/project/buf.yaml.bak", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.uri), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isBufYAMLURI(tt.uri))
		})
	}
}
