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
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/connect-go"
)

type moduleReader struct {
	downloadServiceProvider registryv1alpha1apiclient.DownloadServiceProvider
}

func newModuleReader(
	downloadServiceProvider registryv1alpha1apiclient.DownloadServiceProvider,
) *moduleReader {
	return &moduleReader{
		downloadServiceProvider: downloadServiceProvider,
	}
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
	identityAndCommitOpt := bufmodule.ModuleWithModuleIdentityAndCommit(
		moduleIdentity,
		modulePin.Commit(),
	)
	module, manifest, blobs, err := m.download(ctx, modulePin)
	if err != nil {
		return nil, err
	}
	if manifest != nil {
		// prefer and use manifest and blobs
		return moduleFromBlobs(ctx, manifest, blobs, identityAndCommitOpt)
	} else if module != nil {
		// build proto module instead
		return bufmodule.NewModuleForProto(ctx, module, identityAndCommitOpt)
	}
	// funny, success without a module to build
	return nil, errors.New("no module in response")
}

func (m *moduleReader) download(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (
	module *modulev1alpha1.Module,
	manifest *modulev1alpha1.Blob,
	blobs []*modulev1alpha1.Blob,
	err error,
) {
	downloadService, err := m.downloadServiceProvider.NewDownloadService(
		ctx,
		modulePin.Remote(),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	module, manifest, blobs, err = downloadService.Download(
		ctx,
		modulePin.Owner(),
		modulePin.Repository(),
		modulePin.Commit(),
	)
	if err != nil && connect.CodeOf(err) == connect.CodeNotFound {
		// Required by ModuleReader interface spec
		err = storage.NewErrNotExist(modulePin.String())
	}
	return module, manifest, blobs, err
}

// moduleFromBlobs builds a module from a manifest blob and a set of other
// blobs, provided in protobuf form. It returns a module built from a bucket.
func moduleFromBlobs(
	ctx context.Context,
	manifestBlob *modulev1alpha1.Blob,
	blobs []*modulev1alpha1.Blob,
	options ...bufmodule.ModuleOption,
) (bufmodule.Module, error) {
	if _, err := manifest.FromProtoBlob(manifestBlob); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}
	parsedManifest, err := manifest.NewFromReader(
		bytes.NewReader(manifestBlob.Content),
	)
	if err != nil {
		return nil, fmt.Errorf("parse manifest content: %w", err)
	}
	var memBlobs []manifest.Blob
	for i, modBlob := range blobs {
		memBlob, err := manifest.FromProtoBlob(modBlob)
		if err != nil {
			return nil, fmt.Errorf("invalid blob at index %d: %w", i, err)
		}
		memBlobs = append(memBlobs, memBlob)
	}
	blobSet, err := manifest.NewBlobSet(
		ctx,
		memBlobs,
		manifest.BlobSetWithContentValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid blobs: %w", err)
	}
	manifestBucket, err := manifest.NewBucket(
		*parsedManifest,
		*blobSet,
		manifest.BucketWithAllManifestBlobsValidation(),
		manifest.BucketWithNoExtraBlobsValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("new manifest bucket: %w", err)
	}
	return bufmodule.NewModuleForBucket(
		ctx,
		manifestBucket,
		options...,
	)
}
