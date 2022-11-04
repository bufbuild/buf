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
	"sort"

	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storageutil"
	"go.uber.org/multierr"
)

// manifestBucket is a storage.ReadBucket implementation from a manifest an an
// array of blobs.
type manifestBucket struct {
	manifest   Manifest
	pathToBlob map[string]Blob
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

// NewBucket takes a manifest and an array of blobs and builds a readable storage bucket that
// contains the files in the manifest, optionally validating its digest-content match.
func NewBucket(m Manifest, blobs []Blob) (storage.ReadBucket, error) {
	digestToBlobs := make(map[string]Blob, len(blobs))
	for _, b := range blobs {
		digest := b.Digest().String()
		_, alreadyPresent := digestToBlobs[digest]
		if alreadyPresent {
			return nil, fmt.Errorf("blob with digest %q duplicated", digest)
		}
		_, presentInManifest := m.PathsFor(digest)
		if !presentInManifest {
			return nil, fmt.Errorf("blob with digest %q is not present in manifest", digest)
		}
		digestToBlobs[digest] = b
	}
	pathToBlob := make(map[string]Blob, len(blobs))
	for _, path := range m.Paths() {
		pathDigest, ok := m.DigestFor(path)
		if !ok {
			return nil, fmt.Errorf("path %q not present in manifest", path)
		}
		blob, ok := digestToBlobs[pathDigest.String()]
		if !ok {
			return nil, fmt.Errorf("manifest path %q with digest %q has no associated blob", path, pathDigest.String())
		}
		pathToBlob[path] = blob
	}
	return &manifestBucket{
		manifest:   m,
		pathToBlob: pathToBlob,
	}, nil
}

func (m *manifestBucket) Get(ctx context.Context, path string) (storage.ReadObjectCloser, error) {
	blob, ok := m.pathToBlob[path]
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
	_, ok := m.pathToBlob[path]
	if !ok {
		return nil, storage.NewErrNotExist(path)
	}
	return &manifestBucketObject{
		path: path,
		// no need for setting file, not gonna be read.
	}, nil
}
func (m *manifestBucket) Walk(ctx context.Context, prefix string, f func(storage.ObjectInfo) error) error {
	prefix, err := storageutil.ValidatePrefix(prefix)
	if err != nil {
		return err
	}
	paths := m.manifest.Paths()
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	for _, path := range paths {
		if !normalpath.EqualsOrContainsPath(prefix, path, normalpath.Relative) {
			continue
		}
		blob, ok := m.pathToBlob[path]
		if !ok {
			return storage.NewErrNotExist(path)
		}
		file, err := blob.Open(ctx)
		if err != nil {
			return err
		}
		if err := f(&manifestBucketObject{
			path: path,
			file: file,
		}); err != nil {
			return err
		}
	}
	return nil
}
