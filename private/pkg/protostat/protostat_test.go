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

package protostat

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatsDeprecatedTypes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                    string
		content                 string
		expectedTypes           int
		expectedDeprecatedTypes int
	}{
		{
			name: "no deprecated types",
			content: `
				syntax = "proto3";
				message Foo {}
				enum Bar { BAR_UNSPECIFIED = 0; }
			`,
			expectedTypes:           2,
			expectedDeprecatedTypes: 0,
		},
		{
			name: "deprecated message",
			content: `
				syntax = "proto3";
				message Foo {
					option deprecated = true;
				}
				message Bar {}
			`,
			expectedTypes:           2,
			expectedDeprecatedTypes: 1,
		},
		{
			name: "deprecated enum",
			content: `
				syntax = "proto3";
				enum Status {
					option deprecated = true;
					STATUS_UNSPECIFIED = 0;
				}
			`,
			expectedTypes:           1,
			expectedDeprecatedTypes: 1,
		},
		{
			name: "deprecated RPC",
			content: `
				syntax = "proto3";
				message Request {}
				message Response {}
				service MyService {
					rpc GetData(Request) returns (Response) {
						option deprecated = true;
					}
				}
			`,
			expectedTypes:           3,
			expectedDeprecatedTypes: 1,
		},
		{
			name: "nested deprecated message",
			content: `
				syntax = "proto3";
				message Outer {
					message Inner {
						option deprecated = true;
					}
				}
			`,
			expectedTypes:           2,
			expectedDeprecatedTypes: 1,
		},
		{
			name: "outer deprecated but nested not",
			content: `
				syntax = "proto3";
				message Outer {
					option deprecated = true;
					message Inner {}
				}
			`,
			expectedTypes:           2,
			expectedDeprecatedTypes: 1,
		},
		{
			name: "deprecated group",
			content: `
				syntax = "proto2";
				message Foo {
					optional group MyGroup = 1 [deprecated = true] {
						optional string name = 2;
					}
				}
			`,
			expectedTypes:           2, // Foo, MyGroup (group is also a message type)
			expectedDeprecatedTypes: 1, // MyGroup
		},
		{
			name: "group without options",
			content: `
				syntax = "proto2";
				message Foo {
					optional group MyGroup = 1 {
						optional string name = 2;
					}
				}
			`,
			expectedTypes:           2,
			expectedDeprecatedTypes: 0,
		},
		{
			name: "all nested types deprecated",
			content: `
				syntax = "proto3";
				message Outer {
					option deprecated = true;
					message Inner {
						option deprecated = true;
					}
					enum Status {
						option deprecated = true;
						STATUS_UNSPECIFIED = 0;
					}
				}
			`,
			expectedTypes:           3,
			expectedDeprecatedTypes: 3,
		},
		{
			name: "all types deprecated",
			content: `
				syntax = "proto3";
				message Foo {
					option deprecated = true;
				}
				enum Bar {
					option deprecated = true;
					BAR_UNSPECIFIED = 0;
				}
				message Request {}
				message Response {}
				service Svc {
					rpc Method(Request) returns (Response) {
						option deprecated = true;
					}
				}
			`,
			expectedTypes:           5, // Foo, Bar, Request, Response, RPC
			expectedDeprecatedTypes: 3, // Foo, Bar, RPC
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			walker := &testFileWalker{contents: []string{tc.content}}
			stats, err := GetStats(context.Background(), walker)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTypes, stats.Types, "Types count mismatch")
			assert.Equal(t, tc.expectedDeprecatedTypes, stats.DeprecatedTypes, "DeprecatedTypes count mismatch")
		})
	}
}

func TestGetStatsMultipleFiles(t *testing.T) {
	t.Parallel()
	file1 := `
		syntax = "proto3";
		message DeprecatedFoo {
			option deprecated = true;
		}
	`
	file2 := `
		syntax = "proto3";
		message DeprecatedBar {
			option deprecated = true;
		}
		enum DeprecatedEnum {
			option deprecated = true;
			DEPRECATED_ENUM_UNSPECIFIED = 0;
		}
	`

	walker := &testFileWalker{contents: []string{file1, file2}}
	stats, err := GetStats(context.Background(), walker)
	require.NoError(t, err)

	assert.Equal(t, 2, stats.Files)
	assert.Equal(t, 3, stats.Types) // DeprecatedFoo, DeprecatedBar, DeprecatedEnum
	assert.Equal(t, 3, stats.DeprecatedTypes)
}

// testFileWalker is a mock FileWalker that provides proto content from strings.
type testFileWalker struct {
	contents []string
}

func (w *testFileWalker) Walk(ctx context.Context, f func(io.Reader) error) error {
	for _, content := range w.contents {
		if err := f(strings.NewReader(content)); err != nil {
			return err
		}
	}
	return nil
}

func TestMergeStats(t *testing.T) {
	t.Parallel()
	stats1 := &Stats{
		Files:           2,
		Types:           10,
		DeprecatedTypes: 5,
		Messages:        5,
		Fields:          10,
		Enums:           3,
		EnumValues:      9,
		Services:        2,
		RPCs:            4,
		Extensions:      3,
	}
	stats2 := &Stats{
		Files:           1,
		Types:           6,
		DeprecatedTypes: 3,
		Messages:        3,
		Fields:          6,
		Enums:           2,
		EnumValues:      6,
		Services:        1,
		RPCs:            1,
		Extensions:      2,
	}

	merged := MergeStats(stats1, stats2)

	assert.Equal(t, 3, merged.Files)
	assert.Equal(t, 16, merged.Types)
	assert.Equal(t, 8, merged.DeprecatedTypes)
	assert.Equal(t, 8, merged.Messages)
	assert.Equal(t, 16, merged.Fields)
	assert.Equal(t, 5, merged.Enums)
	assert.Equal(t, 15, merged.EnumValues)
	assert.Equal(t, 3, merged.Services)
	assert.Equal(t, 5, merged.RPCs)
	assert.Equal(t, 5, merged.Extensions)
}
