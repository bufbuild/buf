// Copyright 2020-2024 Buf Technologies, Inc.
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

package bufcas

import (
	"context"
	"fmt"
	"sort"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

// FileSet is a pair of a Manifest and its associated BlobSet.
//
// This can be read and written from and to a storage.Bucket.
//
// The Manifest is guaranteed to exactly correlate with the Blobs in the BlobSet,
// that is the Digests of the FileNodes in the Manifest will exactly match the
// Digests in the Blobs.
type FileSet interface {
	// Manifest returns the associated Manifest.
	Manifest() Manifest
	// BlobSet returns the associated BlobSet.
	BlobSet() BlobSet

	// Protect against creation of a FileSet outside of this package, as we
	// do very careful validation.
	isFileSet()
}

// NewFileSet returns a new FileSet.
//
// Validation is done to ensure the Manifest exactly matches the BlobSet.
func NewFileSet(manifest Manifest, blobSet BlobSet) (FileSet, error) {
	manifestDigestStringMap := make(map[string]struct{})
	blobDigestStringMap := make(map[string]struct{})
	for _, fileNode := range manifest.FileNodes() {
		manifestDigestStringMap[fileNode.Digest().String()] = struct{}{}
	}
	for _, blob := range blobSet.Blobs() {
		blobDigestStringMap[blob.Digest().String()] = struct{}{}
	}
	var onlyInManifest []string
	var onlyInBlobSet []string
	for manifestDigestString := range manifestDigestStringMap {
		if _, ok := blobDigestStringMap[manifestDigestString]; !ok {
			onlyInManifest = append(onlyInManifest, manifestDigestString)
		}
	}
	for blobDigestString := range blobDigestStringMap {
		if _, ok := manifestDigestStringMap[blobDigestString]; !ok {
			onlyInBlobSet = append(onlyInBlobSet, blobDigestString)
		}
	}
	if len(onlyInManifest) > 0 || len(onlyInBlobSet) > 0 {
		sort.Strings(onlyInManifest)
		sort.Strings(onlyInBlobSet)
		return nil, fmt.Errorf("mismatched Manifest and BlobSet at FileSet construction, digests only in Manifest: [%v], digests only in BlobSet: [%v]", onlyInManifest, onlyInBlobSet)
	}
	return newFileSet(manifest, blobSet), nil
}

// NewFileSetForBucket returns a new FileSet for the given ReadBucket.
func NewFileSetForBucket(ctx context.Context, bucket storage.ReadBucket) (FileSet, error) {
	var fileNodes []FileNode
	var blobs []Blob
	if err := storage.WalkReadObjects(
		ctx,
		bucket,
		"",
		func(readObject storage.ReadObject) error {
			blob, err := NewBlobForContent(readObject)
			if err != nil {
				return fmt.Errorf("error creating Blob for file %q: %w", readObject.Path(), err)
			}
			fileNode, err := NewFileNode(readObject.Path(), blob.Digest())
			if err != nil {
				return fmt.Errorf("error creating FileNode for file %q: %w", readObject.Path(), err)
			}
			fileNodes = append(fileNodes, fileNode)
			blobs = append(blobs, blob)
			return nil
		},
	); err != nil {
		return nil, err
	}
	manifest, err := NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	blobSet, err := NewBlobSet(blobs)
	if err != nil {
		return nil, err
	}
	return newFileSet(
		manifest,
		blobSet,
	), nil
}

// PutFileSetToBucket writes the FileSet to the given WriteBucket.
func PutFileSetToBucket(
	ctx context.Context,
	fileSet FileSet,
	bucket storage.WriteBucket,
) error {
	for _, fileNode := range fileSet.Manifest().FileNodes() {
		writeObjectCloser, err := bucket.Put(ctx, fileNode.Path(), storage.PutWithAtomic())
		if err != nil {
			return err
		}
		blob := fileSet.BlobSet().GetBlob(fileNode.Digest())
		if _, err := writeObjectCloser.Write(blob.Content()); err != nil {
			return multierr.Append(err, writeObjectCloser.Close())
		}
		if err := writeObjectCloser.Close(); err != nil {
			return err
		}
	}
	return nil
}

// *** PRIVATE ****

type fileSet struct {
	manifest Manifest
	blobSet  BlobSet
}

func newFileSet(manifest Manifest, blobSet BlobSet) *fileSet {
	return &fileSet{
		manifest: manifest,
		blobSet:  blobSet,
	}
}

func (f *fileSet) Manifest() Manifest {
	return f.manifest
}

func (f *fileSet) BlobSet() BlobSet {
	return f.blobSet
}

func (*fileSet) isFileSet() {}
