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

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
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
			gotKeyLine, gotDeps, _, err := parseBufYAMLDeps([]byte(tt.content))
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

func TestIsBufGenYAMLURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		uri      protocol.URI
		expected bool
	}{
		{"file:///home/user/project/buf.gen.yaml", true},
		{"file:///home/user/project/buf.yaml", false},
		{"file:///home/user/project/buf.gen.yaml.bak", false},
		{"file:///home/user/project/foo.proto", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.uri), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isBufGenYAMLURI(tt.uri))
		})
	}
}

func TestIsBufPolicyYAMLURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		uri      protocol.URI
		expected bool
	}{
		{"file:///home/user/project/buf.policy.yaml", true},
		{"file:///home/user/project/buf.yaml", false},
		{"file:///home/user/project/buf.policy.yaml.bak", false},
		{"file:///home/user/project/foo.proto", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.uri), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isBufPolicyYAMLURI(tt.uri))
		})
	}
}

func TestIsBufLockURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		uri      protocol.URI
		expected bool
	}{
		{"file:///home/user/project/buf.lock", true},
		{"file:///home/user/project/buf.yaml", false},
		{"file:///home/user/project/buf.lock.bak", false},
		{"file:///home/user/project/foo.proto", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.uri), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isBufLockURI(tt.uri))
		})
	}
}

func TestParseYAMLDoc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		wantNil bool
	}{
		{
			name: "valid_yaml",
			text: "version: v2\ndeps: []\n",
		},
		{
			name: "empty_string",
			text: "",
			// yaml decoder returns io.EOF for empty input — parseYAMLDoc returns nil.
			wantNil: true,
		},
		{
			name:    "invalid_yaml",
			text:    ": this is not valid yaml: [\n  unclosed bracket\n",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseYAMLDoc(tt.text)
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestParseBufGenYAMLRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		content            string
		wantAllRefs        []bsrRef
		wantVersionedRefs  []bsrRef
		wantPluginsKeyLine uint32
	}{
		{
			name:    "nil_doc",
			content: "", // parseYAMLDoc returns nil for empty input
		},
		{
			name:    "no_plugins_key",
			content: "version: v2\n",
		},
		{
			name: "local_plugin_only",
			content: `version: v2
plugins:
  - local: protoc-gen-go
    out: gen/go
`,
			wantPluginsKeyLine: 1,
		},
		{
			name: "unversioned_remote_only",
			content: `version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
`,
			wantPluginsKeyLine: 1,
			wantAllRefs: []bsrRef{
				{
					ref: "buf.build/protocolbuffers/go",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 12 + uint32(len("buf.build/protocolbuffers/go"))},
					},
				},
			},
			// unversioned remote is not in versionedPluginRefs
		},
		{
			name: "versioned_remote",
			content: `version: v2
plugins:
  - remote: buf.build/bufbuild/es:v2.2.2
    out: gen/es
`,
			wantPluginsKeyLine: 1,
			wantAllRefs: []bsrRef{
				{
					ref: "buf.build/bufbuild/es:v2.2.2",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 12 + uint32(len("buf.build/bufbuild/es:v2.2.2"))},
					},
				},
			},
			// all remote plugins are versioned, so versionedRefs == allRefs
			wantVersionedRefs: []bsrRef{
				{
					ref: "buf.build/bufbuild/es:v2.2.2",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 12 + uint32(len("buf.build/bufbuild/es:v2.2.2"))},
					},
				},
			},
		},
		{
			name: "mixed_plugins_and_inputs",
			content: `version: v2
plugins:
  - remote: buf.build/bufbuild/es:v2.2.2
    out: gen/es
  - remote: buf.build/protocolbuffers/go
    out: gen/go
  - local: protoc-gen-custom
    out: gen/custom
inputs:
  - module: buf.build/acme/petapis
  - directory: proto
`,
			wantPluginsKeyLine: 1,
			wantAllRefs: []bsrRef{
				{
					ref: "buf.build/bufbuild/es:v2.2.2",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 12 + uint32(len("buf.build/bufbuild/es:v2.2.2"))},
					},
				},
				{
					ref: "buf.build/protocolbuffers/go",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 4, Character: 12},
						End:   protocol.Position{Line: 4, Character: 12 + uint32(len("buf.build/protocolbuffers/go"))},
					},
				},
				{
					ref: "buf.build/acme/petapis",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 9, Character: 12},
						End:   protocol.Position{Line: 9, Character: 12 + uint32(len("buf.build/acme/petapis"))},
					},
				},
			},
			wantVersionedRefs: []bsrRef{
				{
					ref: "buf.build/bufbuild/es:v2.2.2",
					refRange: protocol.Range{
						Start: protocol.Position{Line: 2, Character: 12},
						End:   protocol.Position{Line: 2, Character: 12 + uint32(len("buf.build/bufbuild/es:v2.2.2"))},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc := parseYAMLDoc(tt.content)
			allRefs, versionedRefs, pluginsKeyLine := parseBufGenYAMLRefs(doc)
			assert.Equal(t, tt.wantAllRefs, allRefs)
			assert.Equal(t, tt.wantVersionedRefs, versionedRefs)
			assert.Equal(t, tt.wantPluginsKeyLine, pluginsKeyLine)
		})
	}
}

func TestBsrRefDocURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		refStr  string
		wantURL string
	}{
		{
			name:    "default_registry_no_ref",
			refStr:  "buf.build/acme/petapis",
			wantURL: "https://buf.build/acme/petapis",
		},
		{
			name:    "default_registry_with_ref",
			refStr:  "buf.build/bufbuild/es:v2.2.2",
			wantURL: "https://buf.build/bufbuild/es/docs/v2.2.2",
		},
		{
			name: "non_default_registry_with_ref",
			// A private BSR host: /docs/ path is not valid, so no suffix.
			refStr:  "private.example.com/acme/mod:v1.0.0",
			wantURL: "https://private.example.com/acme/mod",
		},
		{
			name:    "non_default_registry_no_ref",
			refStr:  "private.example.com/acme/mod",
			wantURL: "https://private.example.com/acme/mod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ref, err := bufparse.ParseRef(tt.refStr)
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, bsrRefDocURL(ref))
		})
	}
}
