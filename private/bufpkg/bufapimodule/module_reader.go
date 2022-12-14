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

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/connect-go"
)

type moduleReader struct {
	downloadClientFactory DownloadServiceClientFactory
	tamperProofingEnabled bool
}

// WithTamperProofing configures whether the module reader supports tamper proofing.
// If false, when downloading modules the manifest and blobs will be ignored.
func WithTamperProofing(enabled bool) ModuleReaderOption {
	return func(reader *moduleReader) {
		reader.tamperProofingEnabled = enabled
	}
}

func newModuleReader(
	downloadClientFactory DownloadServiceClientFactory,
	opts ...ModuleReaderOption,
) *moduleReader {
	reader := &moduleReader{
		downloadClientFactory: downloadClientFactory,
	}
	for _, opt := range opts {
		opt(reader)
	}
	return reader
}

func (m *moduleReader) GetModule(ctx context.Context, modulePin bufmoduleref.ModulePin) (bufmodule.Module, error) {
	moduleIdentity, err := bufmoduleref.NewModuleIdentity(
		modulePin.Remote(),
		modulePin.Owner(),
		modulePin.Repository(),
	)
	if err != nil {
		// malformed pin
		return nil, err
	}
	resp, err := m.download(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	identityAndCommitOpt := bufmodule.ModuleWithModuleIdentityAndCommit(
		moduleIdentity,
		modulePin.Commit(),
	)
	if m.tamperProofingEnabled {
		if resp.Manifest == nil {
			return nil, errors.New("expected non-nil manifest with tamper proofing enabled")
		}
		// use manifest and blobs
		bucket, err := manifest.NewBucketFromManifestBlobs(
			ctx,
			resp.Manifest,
			resp.Blobs,
		)
		if err != nil {
			return nil, err
		}
		return bufmodule.NewModuleForBucket(ctx, bucket, identityAndCommitOpt)
	}
	if resp.Module != nil {
		// build proto module instead
		return bufmodule.NewModuleForProto(ctx, resp.Module, identityAndCommitOpt)
	}
	// funny, success without a module to build
	return nil, errors.New("no module in response")
}

func (m *moduleReader) download(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (*registryv1alpha1.DownloadResponse, error) {
	downloadService := m.downloadClientFactory(modulePin.Remote())
	resp, err := downloadService.Download(
		ctx,
		connect.NewRequest(&registryv1alpha1.DownloadRequest{
			Owner:      modulePin.Owner(),
			Repository: modulePin.Repository(),
			Reference:  modulePin.Commit(),
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Required by ModuleReader interface spec
			return nil, storage.NewErrNotExist(modulePin.String())
		}
		return nil, err
	}
	return resp.Msg, err
}
