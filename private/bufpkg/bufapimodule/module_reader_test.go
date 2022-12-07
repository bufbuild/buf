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
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"github.com/bufbuild/connect-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	testDownload(
		t,
		"does-not-exist error",
		newMockDownloadService(
			t,
			withError(connect.NewError(connect.CodeNotFound, nil)),
		),
		true,
		"does not exist",
	)
	testDownload(
		t,
		"unexpected download service error",
		newMockDownloadService(
			t,
			withError(errors.New("internal")),
		),
		true,
		"internal",
	)
	testDownload(
		t,
		"success but response has all empty fields",
		newMockDownloadService(t),
		true,
		"no module in response",
	)
	testDownload(
		t,
		"success",
		newMockDownloadService(
			t,
			withModule(&modulev1alpha1.Module{
				Files: []*modulev1alpha1.ModuleFile{
					{
						Path: "foo.proto",
					},
				},
			}),
		),
		false,
		"",
	)
	testDownload(
		t,
		"success with empty manifest module",
		newMockDownloadService(
			t,
			withBlobsFromMap(map[string][]byte{}),
		),
		false,
		"",
	)
	testDownload(
		t,
		"manifest module with invalid lock file",
		newMockDownloadService(
			t,
			withBlobsFromMap(map[string][]byte{
				"buf.lock": []byte("invalid lock file"),
			}),
		),
		true,
		"failed to decode lock file",
	)
}

func testDownload(
	t *testing.T,
	desc string,
	mock *mockDownloadService,
	expectError bool,
	errorContains string,
) {
	t.Helper()
	t.Run(desc, func(t *testing.T) {
		t.Parallel()
		moduleReader := newModuleReader(mock.factory)
		ctx := context.Background()
		pin, err := bufmoduleref.NewModulePin(
			"remote",
			"owner",
			"repository",
			"branch",
			"commit",
			"digest",
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

type mockDownloadService struct {
	module       *modulev1alpha1.Module
	manifestBlob *modulev1alpha1.Blob
	blobs        []*modulev1alpha1.Blob
	err          error
}

type option interface {
	apply(*mockDownloadService) error
}

type filemap map[string][]byte

func (fm filemap) apply(m *mockDownloadService) error {
	bucket, err := storagemem.NewReadBucket(fm)
	if err != nil {
		return err
	}
	ctx := context.Background()
	moduleManifest, blobSet, err := manifest.NewFromBucket(ctx, bucket)
	if err != nil {
		return err
	}
	mBlob, err := moduleManifest.Blob()
	if err != nil {
		return err
	}
	m.manifestBlob, err = manifest.AsProtoBlob(ctx, mBlob)
	if err != nil {
		return err
	}
	blobs := blobSet.Blobs()
	m.blobs = make([]*modulev1alpha1.Blob, 0, len(blobs))
	for _, blob := range blobs {
		protoBlob, err := manifest.AsProtoBlob(ctx, blob)
		if err != nil {
			return err
		}
		m.blobs = append(m.blobs, protoBlob)
	}
	return nil
}

func withBlobsFromMap(files map[string][]byte) option {
	return filemap(files)
}

type retErr struct{ err error }

func (re retErr) apply(m *mockDownloadService) error {
	m.err = re.err
	return nil
}

func withError(err error) option {
	return retErr{err: err}
}

type retModule struct{ module *modulev1alpha1.Module }

func (rm retModule) apply(m *mockDownloadService) error {
	m.module = rm.module
	return nil
}

func withModule(module *modulev1alpha1.Module) option {
	return retModule{module: module}
}

func newMockDownloadService(
	t *testing.T,
	opts ...option,
) *mockDownloadService {
	m := &mockDownloadService{}
	for _, opt := range opts {
		if err := opt.apply(m); err != nil {
			t.Error(err)
		}
	}
	return m
}

func (m *mockDownloadService) factory(_ string) registryv1alpha1connect.DownloadServiceClient {
	return m
}

func (m *mockDownloadService) Download(
	ctx context.Context,
	req *connect.Request[registryv1alpha1.DownloadRequest],
) (*connect.Response[registryv1alpha1.DownloadResponse], error) {
	if m.err != nil {
		return nil, m.err
	}
	return connect.NewResponse(&registryv1alpha1.DownloadResponse{
		Module:   m.module,
		Manifest: m.manifestBlob,
		Blobs:    m.blobs,
	}), nil
}
