// Copyright 2020-2022 Buf Technologies, Inc.
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

package bufcli

import (
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockBucketProvider struct {
	files map[string][]byte
}

func (m *mockBucketProvider) NewReadWriteBucket(
	_ string,
	_ ...storageos.ReadWriteBucketOption,
) (storage.ReadWriteBucket, error) {
	return storagemem.NewReadWriteBucket2(storagemem.WithFiles(m.files))
}

func TestReadModuleWithWorkspacesDisabled(t *testing.T) {
	testReadModuleWithWorkspacesDisabled(
		t,
		"minimal module",
		moduleFiles("remote/owner/repository"),
		".",
		nil,
		"",
	)
	testReadModuleWithWorkspacesDisabled(
		t,
		"bad name",
		moduleFiles("foo"),
		".",
		nil,
		"module identity",
	)
	testReadModuleWithWorkspacesDisabled(
		t,
		"bad path",
		moduleFiles("remote/owner/repository"),
		"astrangescheme://",
		nil,
		"invalid dir path",
	)
	testReadModuleWithWorkspacesDisabled(
		t,
		"no config file",
		nil,
		".",
		ErrNoConfigFile,
		"",
	)
	testReadModuleWithWorkspacesDisabled(
		t,
		"no module name",
		moduleFiles(""),
		".",
		ErrNoModuleName,
		"",
	)
}

func moduleFiles(name string) map[string][]byte {
	bufConfig := "version: v1\n"
	if name != "" {
		bufConfig += fmt.Sprintf("name: %s\n", name)
	}
	return map[string][]byte{
		"buf.yaml": []byte(bufConfig),
	}
}

func testReadModuleWithWorkspacesDisabled(
	t *testing.T,
	desc string,
	files map[string][]byte,
	source string,
	expectedErr error,
	expectedErrContains string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		logger := zap.NewNop()
		container := app.NewContainer(nil, nil, nil, nil)
		bucketProvider := &mockBucketProvider{
			files: files,
		}
		runner := command.NewRunner()
		module, identity, err := ReadModuleWithWorkspacesDisabled(
			ctx,
			logger,
			container,
			bucketProvider,
			runner,
			source,
		)
		if expectedErr == nil && expectedErrContains == "" {
			assert.NotNil(t, module)
			assert.NotNil(t, identity)
			assert.NoError(t, err)
		} else {
			if expectedErr != nil {
				assert.ErrorIs(t, err, expectedErr)
			}
			if expectedErrContains != "" {
				assert.ErrorContains(t, err, expectedErrContains)
			}
		}
	})
}
