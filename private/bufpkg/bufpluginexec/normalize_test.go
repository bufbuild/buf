// Copyright 2020-2023 Buf Technologies, Inc.
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

package bufpluginexec

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestNormalizeCodeGeneratorResponse(t *testing.T) {
	t.Parallel()
	for i, testCase := range []struct {
		Input    *pluginpb.CodeGeneratorResponse
		Expected *pluginpb.CodeGeneratorResponse
	}{
		{
			Input:    nil,
			Expected: nil,
		},
		{
			Input:    &pluginpb.CodeGeneratorResponse{},
			Expected: &pluginpb.CodeGeneratorResponse{},
		},
		{
			Input: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
				},
			},
			Expected: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
				},
			},
		},
		{
			Input: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
					{
						Content: proto.String("content2"),
					},
				},
			},
			Expected: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1content2"),
					},
				},
			},
		},
		{
			Input: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
					{
						Content: proto.String("content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3"),
					},
				},
			},
			Expected: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3"),
					},
				},
			},
		},
		{
			Input: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
					{
						Content: proto.String("content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3"),
					},
					{
						Content: proto.String("content4"),
					},
				},
			},
			Expected: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3content4"),
					},
				},
			},
		},
		{
			Input: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1"),
					},
					{
						Content: proto.String("content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3"),
					},
					{
						Content: proto.String("content4"),
					},
					{
						Content: proto.String("content5"),
					},
				},
			},
			Expected: &pluginpb.CodeGeneratorResponse{
				File: []*pluginpb.CodeGeneratorResponse_File{
					{
						Name:    proto.String("file1"),
						Content: proto.String("content1content2"),
					},
					{
						Name:    proto.String("file3"),
						Content: proto.String("content3content4content5"),
					},
				},
			},
		},
	} {
		testCase := testCase
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			actual, err := normalizeCodeGeneratorResponse(testCase.Input)
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(testCase.Expected, actual, protocmp.Transform()))
		})
	}
}
