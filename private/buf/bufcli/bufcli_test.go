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

package bufcli_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
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
	bucket := storagemem.NewReadWriteBucket()
	for path, data := range m.files {
		if err := storage.PutPath(context.Background(), bucket, path, data); err != nil {
			return nil, err
		}
	}
	return bucket, nil
}

func TestDiscoverRemote(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                string
		references          []string
		expectedSelectedRef string
	}
	testCases := []testCase{
		{
			name:                "nil_references",
			expectedSelectedRef: "",
		},
		{
			name:                "no_references",
			references:          []string{},
			expectedSelectedRef: "",
		},
		{
			name: "some_references",
			references: []string{
				"buf.build/foo/repo1",
				"buf.build/foo/repo2",
				"buf.build/foo/repo3",
			},
			expectedSelectedRef: "buf.build/foo/repo1",
		},
		{
			name: "some_invalid_references",
			references: []string{
				"buf.build/foo/repo1",
				"",
				"buf.build/foo/repo3",
			},
			expectedSelectedRef: "buf.build/foo/repo1",
		},
		{
			name: "all_single_tenant_references",
			references: []string{
				"buf.acme.com/foo/repo1",
				"buf.acme.com/foo/repo2",
				"buf.acme.com/foo/repo3",
			},
			expectedSelectedRef: "buf.acme.com/foo/repo1",
		},
		{
			name: "some_single_tenant_references",
			references: []string{
				"buf.build/foo/repo1",
				"buf.build/foo/repo2",
				"buf.first.com/foo/repo3",
				"buf.second.com/foo/repo4",
			},
			expectedSelectedRef: "buf.first.com/foo/repo3",
		},
		{
			name: "some_invalid_references_with_single_tenant",
			references: []string{
				"buf.build/foo/repo1",
				"buf.first.com/foo/repo2",
				"",
				"buf.second.com/foo/repo3",
			},
			expectedSelectedRef: "buf.first.com/foo/repo2",
		},
	}
	for _, tc := range testCases {
		func(tc testCase) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var references []bufmoduleref.ModuleReference
				for _, r := range tc.references {
					ref, _ := bufmoduleref.ModuleReferenceForString(r)
					references = append(references, ref)
				}
				selectedRef := bufcli.SelectReferenceForRemote(references)
				if tc.expectedSelectedRef == "" {
					assert.Nil(t, selectedRef)
				} else {
					assert.Equal(t, tc.expectedSelectedRef, selectedRef.IdentityString())
				}
			})
		}(tc)
	}
}

func TestBucketAndConfigForSource(t *testing.T) {
	t.Parallel()
	testBucketAndConfigForSource(
		t,
		"minimal module",
		moduleFiles("remote/owner/repository"),
		".",
		nil,
		"",
	)
	testBucketAndConfigForSource(
		t,
		"bad name",
		moduleFiles("foo"),
		".",
		nil,
		"module identity",
	)
	testBucketAndConfigForSource(
		t,
		"bad path",
		moduleFiles("remote/owner/repository"),
		"astrangescheme://",
		nil,
		"invalid dir path",
	)
	testBucketAndConfigForSource(
		t,
		"no config file",
		nil,
		".",
		bufcli.ErrNoConfigFile,
		"",
	)
	testBucketAndConfigForSource(
		t,
		"no module name",
		moduleFiles(""),
		".",
		bufcli.ErrNoModuleName,
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

func bucketAndConfig(
	ctx context.Context,
	logger *zap.Logger,
	files map[string][]byte,
	source string,
) (storage.ReadBucketCloser, *bufconfig.Config, error) {
	container := app.NewContainer(nil, nil, nil, nil)
	bucketProvider := &mockBucketProvider{
		files: files,
	}
	runner := command.NewRunner()
	return bufcli.BucketAndConfigForSource(
		ctx,
		logger,
		container,
		bucketProvider,
		runner,
		source,
	)
}

func testBucketAndConfigForSource(
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
		sourceBucket, sourceConfig, err := bucketAndConfig(
			ctx,
			logger,
			files,
			source,
		)
		if expectedErr == nil && expectedErrContains == "" {
			assert.NotNil(t, sourceBucket)
			assert.NotNil(t, sourceConfig)
			assert.NoError(t, err)
			return
		}
		if expectedErr != nil {
			assert.ErrorIs(t, err, expectedErr)
		}
		if expectedErrContains != "" {
			assert.ErrorContains(t, err, expectedErrContains)
		}
	})
}
