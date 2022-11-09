// Copyright 2020-2022 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manifest

import (
	"context"
	"fmt"
	"io"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"go.uber.org/multierr"
)

// manifestBucket is a storage.ReadBucket implementation from a manifest and an
// array of blobs.
type manifestBucket struct {
	manifest Manifest
	blobs    BlobSet
}

type manifestBucketObject struct {
	path string
	file io.ReadCloser
}

func (o *manifestBucketObject) Path() string               { return o.path }
func (o *manifestBucketObject) ExternalPath() string       { return o.path }
func (o *manifestBucketObject) Read(p []byte) (int, error) { return o.file.Read(p) }
func (o *manifestBucketObject) Close() error               { return o.file.Close() }

// NewFromBucket creates a manifest from a storage bucket, with all its digests
// in DigestTypeShake256.
func NewFromBucket(
	ctx context.Context,
	bucket storage.ReadBucket,
) (*Manifest, error) {
	m := New()
	digester, err := NewDigester(DigestTypeShake256)
	if err != nil {
		return nil, err
	}
	if walkErr := bucket.Walk(ctx, "", func(info storage.ObjectInfo) (retErr error) {
		path := info.Path()
		obj, err := bucket.Get(ctx, path)
		if err != nil {
			return err
		}
		defer func() { retErr = multierr.Append(retErr, obj.Close()) }()
		digest, err := digester.Digest(obj)
		if err != nil {
			return err
		}
		if err := m.AddEntry(path, *digest); err != nil {
			return err
		}
		return nil
	}); walkErr != nil {
		return nil, walkErr
	}
	return m, nil
}

type bucketOptions struct {
	allManifestBlobs bool
	noExtraBlobs     bool
}

// BucketOption are options passed when creating a new manifest bucket.
type BucketOption func(*bucketOptions)

// BucketWithAllManifestBlobsValidation validates that all the passed digests in
// the manifest have a corresponding blob in the blob set.
func BucketWithAllManifestBlobsValidation() BucketOption {
	return func(opts *bucketOptions) {
		opts.allManifestBlobs = true
	}
}

// BucketWithNoExtraBlobsValidation validates that the passed blob set has no
// additional blobs beyong the ones in the manifest.
func BucketWithNoExtraBlobsValidation() BucketOption {
	return func(opts *bucketOptions) {
		opts.noExtraBlobs = true
	}
}

// NewBucket takes a manifest and a blob set and builds a readable storage
// bucket that contains the files in the manifest.
func NewBucket(m Manifest, blobs BlobSet, opts ...BucketOption) (storage.ReadBucket, error) {
	var config bucketOptions
	for _, option := range opts {
		option(&config)
	}
	if config.allManifestBlobs {
		for _, path := range m.Paths() {
			pathDigest, ok := m.DigestFor(path)
			if !ok {
				// we're iterating manifest paths, this should never happen.
				return nil, fmt.Errorf("path %q not present in manifest", path)
			}
			if _, ok := blobs.digestToBlob[pathDigest.String()]; !ok {
				return nil, fmt.Errorf("manifest path %q with digest %q has no associated blob", path, pathDigest.String())
			}
		}
	}
	if config.noExtraBlobs {
		for digestStr := range blobs.digestToBlob {
			if _, ok := m.digestToPaths[digestStr]; !ok {
				return nil, fmt.Errorf("blob with digest %q is not present in the manifest", digestStr)
			}
		}
	}
	return &manifestBucket{
		manifest: m,
		blobs:    blobs,
	}, nil
}

func (m *manifestBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	digest, ok := m.manifest.pathToDigest[path]
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	blob, ok := m.blobs.digestToBlob[digest.String()]
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	file, err := blob.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &manifestBucketObject{
		path: path,
		file: file,
	}, nil
}

func (m *manifestBucket) Stat(ctx context.Context, path string) (storage.ObjectInfo, error) {
	digest, ok := m.manifest.pathToDigest[path]
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	if _, ok := m.blobs.digestToBlob[digest.String()]; !ok {
		return nil, storage.NewErrNotExist(path)
	}
	// storage.ObjectInfo only requires path
	return &manifestBucketObject{path: path}, nil
}

func (m *manifestBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	// walk order is not guaranteed
	for _, path := range m.manifest.Paths() {
		if !normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			continue
		}
		digest, ok := m.manifest.pathToDigest[path]
		if !ok {
			// we're iterating manifest paths, this should never happen.
			return storage.NewErrNotExist(path)
		}
		if _, ok := m.blobs.digestToBlob[digest.String()]; !ok {
			// this could happen if the bucket was not checked for completeness.
			continue
		}
		// storage.ObjectInfo only requires path
		if err := f(&manifestBucketObject{path: path}); err != nil {
			return err
		}
	}
	return nil
}
