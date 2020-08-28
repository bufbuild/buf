// Copyright 2020 Buf Technologies, Inc.
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

package bufmoduletesting_test

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
)

func TestModuleDigestB1(t *testing.T) {
	moduleName, err := bufmodule.ModuleNameForString(bufmoduletesting.TestModuleNameString)
	require.NoError(t, err)
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestData)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(context.Background(), readBucket)
	require.NoError(t, err)
	digest, err := bufmodule.ModuleDigestB1(context.Background(), moduleName.Version(), module)
	require.NoError(t, err)
	require.Equal(t, digest, bufmoduletesting.TestDigest)
}
