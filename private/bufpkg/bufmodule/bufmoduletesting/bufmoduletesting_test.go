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

package bufmoduletesting_test

import (
	"context"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduletesting"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/stretchr/testify/require"
)

func TestModuleDigestB3(t *testing.T) {
	t.Parallel()
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithConfiguration)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(context.Background(), readBucket)
	require.NoError(t, err)
	digest, err := bufmodule.ModuleDigestB3(context.Background(), module)
	require.NoError(t, err)
	require.Equal(t, bufmoduletesting.TestDigestB3WithConfiguration, digest)
}

func TestModuleDigestB3withFallbackDocumentationPath(t *testing.T) {
	t.Parallel()
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithConfigurationAndFallbackDocumentationPath)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(context.Background(), readBucket)
	require.NoError(t, err)
	digest, err := bufmodule.ModuleDigestB3(context.Background(), module)
	require.NoError(t, err)
	require.Equal(t, bufmoduletesting.TestDigestB3WithConfigurationAndFallbackDocumentationPath, digest)
}

func TestModuleDigestB3WithLicense(t *testing.T) {
	t.Parallel()
	readBucket, err := storagemem.NewReadBucket(bufmoduletesting.TestDataWithLicense)
	require.NoError(t, err)
	module, err := bufmodule.NewModuleForBucket(context.Background(), readBucket)
	require.NoError(t, err)
	digest, err := bufmodule.ModuleDigestB3(context.Background(), module)
	require.NoError(t, err)
	require.Equal(t, bufmoduletesting.TestDigestB3WithLicense, digest)
}
