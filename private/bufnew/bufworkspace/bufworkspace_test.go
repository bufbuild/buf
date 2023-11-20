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

package bufworkspace

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufnew/bufmodule/bufmoduletest"
	"github.com/bufbuild/buf/private/pkg/dag/dagtest"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func init() {
	bufconfig.AllowV2ForTesting()
}

func TestBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// This represents some external dependencies from the BSR.
	bsrProvider, err := bufmoduletest.NewOmniProvider(
		bufmoduletest.ModuleData{
			Name:    "buf.build/acme/date",
			DirPath: "testdata/basic/bsr/buf.build/acme/date",
		},
		bufmoduletest.ModuleData{
			Name:    "buf.build/acme/extension",
			DirPath: "testdata/basic/bsr/buf.build/acme/extension",
		},
	)
	require.NoError(t, err)

	storageosProvider := storageos.NewProvider()
	bucket, err := storageosProvider.NewReadWriteBucket(
		"testdata/basic/workspace",
	)
	require.NoError(t, err)

	workspace, err := NewWorkspaceForBucket(
		ctx,
		bucket,
		bsrProvider,
		WorkspaceWithTargetSubDirPath(
			"testdata/basic/workspace/finance/portfolio/proto",
		),
	)
	require.NoError(t, err)

	graph, err := bufmodule.ModuleSetToDAG(workspace)
	require.NoError(t, err)
	dagtest.RequireGraphEqual(
		t,
		[]dagtest.ExpectedNode[string]{
			{
				Key: "buf.build/acme/extension",
			},
			{
				Key: "buf.build/acme/date",
				Outbound: []string{
					"buf.build/acme/extension",
				},
			},
			{
				Key: "buf.build/acme/geo",
			},
			{
				Key: "buf.build/acme/money",
			},
			{
				Key: "buf.build/acme/bond",
				Outbound: []string{
					"buf.build/acme/extension",
					"buf.build/acme/date",
					"buf.build/acme/geo",
					"buf.build/acme/money",
				},
			},
			{
				Key: "testdata/basic/finance/portfolio/proto",
				Outbound: []string{
					"buf.build/acme/extension",
					"buf.build/acme/date",
					"buf.build/acme/geo",
					"buf.build/acme/money",
					"buf.build/acme/bond",
				},
			},
		},
		graph,
	)
}
