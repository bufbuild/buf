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

package bufmanifest

import (
	"context"

	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemanifest"
)

// NewReadBucketFromManifestBlobs builds a storage bucket from a manifest blob and a
// set of other blobs, provided in protobuf form. It makes sure that all blobs
// (including manifest) content match with their digest, and additionally checks
// that the blob set matches completely with the manifest paths (no missing nor
// extra blobs). This bucket is suitable for building or exporting.
func NewReadBucketFromManifestBlobs(
	ctx context.Context,
	manifestBlob *modulev1alpha1.Blob,
	blobs []*modulev1alpha1.Blob,
) (storage.ReadBucket, error) {
	parsedManifest, err := NewManifestFromProto(ctx, manifestBlob)
	if err != nil {
		return nil, err
	}
	blobSet, err := NewBlobSetFromProto(ctx, blobs)
	if err != nil {
		return nil, err
	}
	return storagemanifest.NewReadBucket(
		parsedManifest,
		blobSet,
		storagemanifest.ReadBucketWithAllManifestBlobs(),
		storagemanifest.ReadBucketWithNoExtraBlobs(),
	)
}
