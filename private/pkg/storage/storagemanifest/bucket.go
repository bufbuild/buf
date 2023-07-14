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
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
)

type bucket struct {
	manifest *manifest.Manifest
	blobSet  *manifest.BlobSet
}

func newBucket(
	m *manifest.Manifest,
	blobSet *manifest.BlobSet,
	options ...ReadBucketOption,
) (*bucket, error) {
	readBucketOptions := newReadBucketOptions()
	for _, option := range options {
		option(readBucketOptions)
	}
	// TODO: why is this validation in newBucket, why is this not somewhere else?
	// This should not be in storagemanifest.
	if readBucketOptions.allManifestBlobs {
		if err := m.Range(func(path string, digest manifest.Digest) error {
			if _, ok := blobSet.BlobFor(digest.String()); !ok {
				return fmt.Errorf("manifest path %q with digest %q has no associated blob", path, digest.String())
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if readBucketOptions.noExtraBlobs {
		for _, blob := range blobSet.Blobs() {
			digestString := blob.Digest().String()
			if _, ok := m.PathsFor(digestString); !ok {
				return nil, fmt.Errorf("blob with digest %q is not present in the manifest", digestString)
			}
		}
	}
	return &bucket{
		manifest: m,
		blobSet:  blobSet,
	}, nil
}

func (b *bucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	blob, ok := b.blobFor(path)
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	file, err := blob.Open(ctx)
	if err != nil {
		return nil, err
	}
	return newReadObjectCloser(path, file), nil
}

func (b *bucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	path, err := storageutil.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	if _, ok := b.blobFor(path); !ok {
		return nil, storage.NewErrNotExist(path)
	}
	return storageutil.NewObjectInfo(path, path), nil
}

func (b *bucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	walkChecker := storageutil.NewWalkChecker()
	for _, path := range b.manifest.Paths() {
		if !normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			continue
		}
		if err := walkChecker.Check(ctx); err != nil {
			return err
		}
		if _, ok := b.blobFor(path); !ok {
			// this could happen if the bucket was built with partial blobs
			continue
		}
		if err := f(storageutil.NewObjectInfo(path, path)); err != nil {
			return err
		}
	}
	return nil
}

// blobFor returns a blob for a given path. It returns the blob if found, or nil
// and ok=false if the path has no digest in the manifest, or if the blob for
// that digest is not present.
func (b *bucket) blobFor(path string) (_ manifest.Blob, ok bool) {
	digest, ok := b.manifest.DigestFor(path)
	if !ok {
		return nil, false
	}
	blob, ok := b.blobSet.BlobFor(digest.String())
	if !ok {
		return nil, false
	}
	return blob, true
}

type readBucketOptions struct {
	allManifestBlobs bool
	noExtraBlobs     bool
}

func newReadBucketOptions() *readBucketOptions {
	return &readBucketOptions{}
}
