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

package bufapimodule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDownloadService struct {
	module   *v1alpha1.Module
	manifest *v1alpha1.Blob
	blobs    []*v1alpha1.Blob
	err      error
}

func (m *mockDownloadService) NewDownloadService(
	ctx context.Context,
	address string,
) (registryv1alpha1api.DownloadService, error) {
	return m, nil
}

func (m *mockDownloadService) Download(
	ctx context.Context,
	owner string,
	repository string,
	reference string,
) (*v1alpha1.Module, *v1alpha1.Blob, []*v1alpha1.Blob, error) {
	if m.err != nil {
		return nil, nil, nil, m.err
	}
	return m.module, m.manifest, m.blobs, nil
}

func TestDownload(t *testing.T) {
	testDownload(
		t,
		"does-not-exist error",
		&mockDownloadService{
			err: connect.NewError(connect.CodeNotFound, nil),
		},
		true,
		"does not exist",
	)
	testDownload(
		t,
		"unexpected download service error",
		&mockDownloadService{
			err: errors.New("internal"),
		},
		true,
		"internal",
	)
	testDownload(
		t,
		"success but response has all empty fields",
		&mockDownloadService{},
		true,
		"module is required",
	)
	testDownload(
		t,
		"success",
		&mockDownloadService{
			module: &v1alpha1.Module{
				Files: []*v1alpha1.ModuleFile{
					{
						Path: "foo.proto",
					},
				},
			},
		},
		false,
		"",
	)
}

func testDownload(
	t *testing.T,
	desc string,
	downloadServiceProvider registryv1alpha1apiclient.DownloadServiceProvider,
	expectError bool,
	errorContains string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		moduleReader := newModuleReader(downloadServiceProvider)
		ctx := context.Background()
		pin, err := bufmoduleref.NewModulePin(
			"remote",
			"owner",
			"repository",
			"branch",
			"commit",
			time.Now(),
		)
		require.NoError(t, err)
		module, err := moduleReader.GetModule(ctx, pin)
		if expectError {
			assert.Error(t, err)
			if errorContains != "" {
				assert.ErrorContains(t, err, errorContains)
			}
		} else {
			assert.NotNil(t, module)
			assert.NoError(t, err)
		}
	})
}
