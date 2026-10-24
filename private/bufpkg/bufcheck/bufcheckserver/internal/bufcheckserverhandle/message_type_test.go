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

package bufcheckserverhandle

import (
	"testing"

	"github.com/bufbuild/buf/private/buf/buftesting"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protocompile/experimental/incremental"
	"github.com/bufbuild/protocompile/experimental/incremental/queries"
	"github.com/bufbuild/protocompile/experimental/ir"
	"github.com/bufbuild/protocompile/experimental/report"
	"github.com/bufbuild/protocompile/experimental/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestMessageTypesEqual(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		syntax   string
		previous string
		current  string // Empty means the definition is unchanged.
		equal    bool
	}{
		{
			name:     "empty",
			previous: `message A {}`,
			equal:    true,
		},
		{
			name:     "scalar",
			previous: `message A { uint64 x = 1; }`,
			equal:    true,
		},
		{
			name:     "field order",
			previous: `message A { uint64 x = 1; string y = 2; }`,
			current:  `message A { string y = 2; uint64 x = 1; }`,
			equal:    true,
		},
		{
			name:     "recursive",
			previous: `message A { A next = 1; uint64 x = 2; }`,
			equal:    true,
		},
		{
			name:     "mutually recursive",
			previous: `message A { B b = 1; } message B { A a = 1; }`,
			equal:    true,
		},
		{
			name:     "nested",
			previous: `message A { B b = 1; message B { uint64 x = 1; } }`,
			equal:    true,
		},
		{
			name:     "moved out of enclosing message",
			previous: `message A { B b = 1; message B { uint64 x = 1; } }`,
			current:  `message A { B b = 1; } message B { uint64 x = 1; }`,
			equal:    true,
		},
		{
			name:     "maps and repeated messages",
			previous: `message A { repeated B values = 1; map<string, B> lookup = 2; } message B { A a = 1; }`,
			equal:    true,
		},
		{
			name:     "enum",
			previous: `message A { E e = 1; } enum E { ZERO = 0; ONE = 1; }`,
			equal:    true,
		},
		{
			name:     "oneofs and optional",
			previous: `message A { oneof choice { string x = 1; A a = 2; } optional int32 y = 3; }`,
			equal:    true,
		},
		{
			name:     "groups and defaults",
			syntax:   `syntax = "proto2";`,
			previous: `message A { optional group B = 1 { optional int32 x = 1 [default = 42]; } }`,
			equal:    true,
		},
		{
			name:     "edition",
			syntax:   `edition = "2023";`,
			previous: `message A { A next = 1 [features.message_encoding = DELIMITED]; string x = 2; }`,
			equal:    true,
		},
		{
			name:     "ordinary google.protobuf message",
			previous: `import "google/protobuf/empty.proto"; message A { google.protobuf.Empty empty = 1; }`,
			current:  `message A { Empty empty = 1; } message Empty {}`,
			equal:    true,
		},
		{
			name:     "field type changed",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A { string x = 1; }`,
		},
		{
			name:     "field number changed",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A { uint64 x = 2; }`,
		},
		{
			name:     "field deleted",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A {}`,
		},
		{
			name:     "cardinality changed",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A { repeated uint64 x = 1; }`,
		},
		{
			name:     "presence changed",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A { optional uint64 x = 1; }`,
		},
		{
			name:     "json name changed",
			previous: `message A { uint64 x = 1; }`,
			current:  `message A { uint64 x = 1 [json_name = "y"]; }`,
		},
		{
			name:     "oneof changed",
			previous: `message A { string x = 1; string y = 2; }`,
			current:  `message A { oneof choice { string x = 1; string y = 2; } }`,
		},
		{
			name:     "recursive type changed after cycle",
			previous: `message A { A next = 1; uint64 x = 2; }`,
			current:  `message A { A next = 1; string x = 2; }`,
		},
		{
			name:     "mutually recursive type changed",
			previous: `message A { B b = 1; } message B { A a = 1; uint64 x = 2; }`,
			current:  `message A { B b = 1; } message B { A a = 1; string x = 2; }`,
		},
		{
			name:     "same old type paired with different new types",
			previous: `message A { B b = 1; B other = 2; } message B { uint64 x = 1; }`,
			current:  `message A { B b = 1; Scope.B other = 2; } message B { uint64 x = 1; } message Scope { message B { string x = 1; } }`,
		},
		{
			name:     "map value changed",
			previous: `message A { map<string, B> lookup = 1; } message B { uint64 x = 1; }`,
			current:  `message A { map<string, B> lookup = 1; } message B { string x = 1; }`,
		},
		{
			name:     "enum value changed",
			previous: `message A { E e = 1; } enum E { ZERO = 0; ONE = 1; }`,
			current:  `message A { E e = 1; } enum E { ZERO = 0; TWO = 1; }`,
		},
		{
			name:     "default changed",
			syntax:   `syntax = "proto2";`,
			previous: `message A { optional int32 x = 1 [default = 42]; }`,
			current:  `message A { optional int32 x = 1 [default = 43]; }`,
		},
		{
			name:     "inherited presence changed",
			syntax:   `edition = "2023";`,
			previous: `message A { int32 x = 1; }`,
			current:  `option features.field_presence = IMPLICIT; message A { int32 x = 1; }`,
		},
		{
			name:     "inherited message encoding changed",
			syntax:   `edition = "2023";`,
			previous: `message A { A next = 1; }`,
			current:  `option features.message_encoding = DELIMITED; message A { A next = 1; }`,
		},
		{
			name:     "inherited utf8 validation changed",
			syntax:   `edition = "2023";`,
			previous: `message A { string x = 1; }`,
			current:  `option features.utf8_validation = NONE; message A { string x = 1; }`,
		},
		{
			name:     "inherited enum openness changed",
			syntax:   `edition = "2023";`,
			previous: `message A { E e = 1; } enum E { ZERO = 0; }`,
			current:  `option features.enum_type = CLOSED; message A { E e = 1; } enum E { ZERO = 0; }`,
		},
		{
			name:     "reservations",
			previous: `message A { reserved 2 to 3, 5; reserved "removed"; uint64 x = 1; }`,
			current:  `message A { reserved 5, 2 to 3; reserved "removed"; uint64 x = 1; }`,
			equal:    true,
		},
		{
			name:     "reserved number deleted",
			previous: `message A { reserved 2; uint64 x = 1; }`,
			current:  `message A { uint64 x = 1; }`,
		},
		{
			name:     "reserved name deleted",
			previous: `message A { reserved "removed"; uint64 x = 1; }`,
			current:  `message A { uint64 x = 1; }`,
		},
		{
			name:     "reserved range changed",
			previous: `message A { reserved 2 to 4; uint64 x = 1; }`,
			current:  `message A { reserved 2 to 3; uint64 x = 1; }`,
		},
		{
			name:     "extension ranges",
			syntax:   `syntax = "proto2";`,
			previous: `message A { extensions 100 to max; }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			syntax := test.syntax
			if syntax == "" {
				syntax = `syntax = "proto3";`
			}
			current := test.current
			if current == "" {
				current = test.previous
			}
			previousMessage := compileMessageType(t, syntax+" package foo.v1; "+test.previous, nil)
			currentMessage := compileMessageType(t, syntax+" package bar.v1; "+current, nil)
			for _, wireJSON := range []bool{false, true} {
				equal, err := messageTypesEqual(previousMessage, currentMessage, wireJSON)
				require.NoError(t, err)
				assert.Equal(t, test.equal, equal, "wireJSON: %v", wireJSON)
			}
		})
	}
}

func TestMessageTypesEqualJSONEncoding(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		syntax   string
		previous string
		current  string
	}{
		{
			name:     "well-known type",
			syntax:   `syntax = "proto3";`,
			previous: `import "google/protobuf/timestamp.proto"; message A { google.protobuf.Timestamp time = 1; }`,
			current:  `message A { Timestamp time = 1; } message Timestamp { int64 seconds = 1; int32 nanos = 2; }`,
		},
		{
			name:     "well-known enum",
			syntax:   `syntax = "proto3";`,
			previous: `import "google/protobuf/struct.proto"; message A { google.protobuf.NullValue value = 1; }`,
			current:  `message A { NullValue value = 1; } enum NullValue { NULL_VALUE = 0; }`,
		},
		{
			name:     "inherited json format",
			syntax:   `edition = "2023";`,
			previous: `message A { string x = 1; }`,
			current:  `option features.json_format = LEGACY_BEST_EFFORT; message A { string x = 1; }`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			previous := compileMessageType(t, test.syntax+" package foo.v1; "+test.previous, nil)
			current := compileMessageType(t, test.syntax+" package bar.v1; "+test.current, nil)
			for _, wireJSON := range []bool{false, true} {
				equal, err := messageTypesEqual(previous, current, wireJSON)
				require.NoError(t, err)
				assert.Equal(t, !wireJSON, equal)
				equal, err = messageTypesEqual(current, previous, wireJSON)
				require.NoError(t, err)
				assert.Equal(t, !wireJSON, equal)
			}
		})
	}
}

func TestMessageTypesEqualUnmovedDependency(t *testing.T) {
	t.Parallel()
	previous := compileMessageType(t,
		`syntax = "proto3"; package foo.v1; import "shared.proto"; message A { shared.B b = 1; }`,
		map[string]string{"shared.proto": `syntax = "proto3"; package shared; message B { uint64 x = 1; }`},
	)
	current := compileMessageType(t,
		`syntax = "proto3"; package bar.v1; import "shared.proto"; message A { shared.B b = 1; }`,
		map[string]string{"shared.proto": `syntax = "proto3"; package shared; message B { string x = 1; }`},
	)
	for _, wireJSON := range []bool{false, true} {
		equal, err := messageTypesEqual(previous, current, wireJSON)
		require.NoError(t, err)
		assert.False(t, equal)
	}
}

func compileMessageType(t *testing.T, text string, imports map[string]string) protoreflect.MessageDescriptor {
	t.Helper()
	opener := source.NewMap(nil)
	opener.Add("test.proto", text)
	for path, contents := range imports {
		opener.Add(path, contents)
	}
	results, diagnostics, err := incremental.Run(t.Context(), incremental.New(), queries.FDS{
		Opener:    &source.Openers{opener, buftesting.WKTOpener()},
		Session:   new(ir.Session),
		Workspace: source.NewWorkspace("test.proto"),
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NoError(t, results[0].Fatal)
	for _, diagnostic := range diagnostics.Diagnostics {
		require.Greater(t, diagnostic.Level(), report.Error, "compiling %s: %s", text, diagnostic.Message())
	}
	// Materialize options, matching the image builder's wire round-trip.
	data, err := protoencoding.NewWireMarshaler().Marshal(results[0].Value)
	require.NoError(t, err)
	fds := new(descriptorpb.FileDescriptorSet)
	require.NoError(t, protoencoding.NewWireUnmarshaler(nil).Unmarshal(data, fds))
	resolver, err := protoencoding.NewResolver(fds.File...)
	require.NoError(t, err)
	file, err := resolver.FindFileByPath("test.proto")
	require.NoError(t, err, "compiling %s", text)
	message := file.Messages().ByName("A")
	require.NotNil(t, message)
	return message
}
