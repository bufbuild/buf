// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufprotoc

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/pkg/slogtestext"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/require"
)

func TestNewModuleSetForProtoc(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	moduleSet, err := NewModuleSetForProtoc(
		ctx,
		slogtestext.NewLogger(t),
		storageos.NewProvider(),
		[]string{
			"testdata/basic/bsr/buf.testing/acme/date",
			"testdata/basic/bsr/buf.testing/acme/extension",
			"testdata/basic/workspacev1/common/geo/proto",
			"testdata/basic/workspacev1/common/money/proto",
			"testdata/basic/workspacev1/finance/bond/proto/root1",
			"testdata/basic/workspacev1/finance/bond/proto/root2",
			"testdata/basic/workspacev1/finance/portfolio/proto",
		},
		[]string{
			"testdata/basic/workspacev1/finance/portfolio/proto/acme/portfolio/v1/portfolio.proto",
		},
	)
	require.NoError(t, err)

	modules := moduleSet.Modules()
	require.Equal(t, 1, len(modules))
	module := modules[0]
	require.Equal(t, ".", module.OpaqueID())
	require.True(t, module.IsTarget())

	fileInfo, err := module.StatFileInfo(ctx, "acme/money/v1/currency_code.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/money/v1/money.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/bond/real/v1/bond.proto")
	require.NoError(t, err)
	require.False(t, fileInfo.IsTargetFile())
	fileInfo, err = module.StatFileInfo(ctx, "acme/portfolio/v1/portfolio.proto")
	require.NoError(t, err)
	require.True(t, fileInfo.IsTargetFile())
}
