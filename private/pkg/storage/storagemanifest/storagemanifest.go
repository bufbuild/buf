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

package storagemanifest

import (
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// NewReadBucket takes a Manifest and BlobSet and builds a ReadBucket
// that contains the files in the Manifest.
func NewReadBucket(
	m *manifest.Manifest,
	blobSet *manifest.BlobSet,
	options ...ReadBucketOption,
) (storage.ReadBucket, error) {
	return newBucket(m, blobSet, options...)
}

// ReadBucketOption is an option for a  passed when creating a new manifest bucket.
type ReadBucketOption func(*readBucketOptions)

// ReadBucketWithAllManifestBlobs validates that all manifest digests
// have a corresponding blob in the blob set. If this option is not passed, then
// buckets with partial/incomplete blobs are allowed.
func ReadBucketWithAllManifestBlobs() ReadBucketOption {
	return func(readBucketOptions *readBucketOptions) {
		readBucketOptions.allManifestBlobs = true
	}
}

// ReadBucketWithNoExtraBlobs validates that the passed blob set has no
// additional blobs beyond the ones in the manifest.
func ReadBucketWithNoExtraBlobs() ReadBucketOption {
	return func(readBucketOptions *readBucketOptions) {
		readBucketOptions.noExtraBlobs = true
	}
}
