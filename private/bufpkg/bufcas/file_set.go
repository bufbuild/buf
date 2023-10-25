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

package bufcas

import (
	"context"
	"fmt"
	"sort"

	storagev1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/storage/v1beta1"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// FileSet is a pair of a Manifest and its associated BlobSet.
//
// This can be read and written from and to a storage.Bucket.
//
// The Manifest is guaranteed to exactly correlate with the Blobs in the BlobSet,
// that is the Digests of the FileNodes in the Manifest will exactly match the
// Digests in the Blobs. Note that some FileNodes may have empty Digests, in which
// case there is no corresponding Blob (as the content is empty).
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
		if digest := fileNode.Digest(); digest != nil {
			manifestDigestStringMap[digest.String()] = struct{}{}
		}
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
			blob, err := NewBlobForContent(DigestTypeShake256, readObject)
			if err != nil {
				return err
			}
			var digest Digest
			// Otherwise, we have an empty file.
			if blob != nil {
				digest = blob.Digest()
			}
			fileNode, err := NewFileNode(readObject.Path(), digest)
			if err != nil {
				return err
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
	return newFileSet(
		manifest,
		newBlobSet(blobs),
	), nil
}

// PutFileSetToBucket writes the FileSet to the given WriteBucket.
func PutFileSetToBucket(
	ctx context.Context,
	fileSet FileSet,
	bucket storage.WriteBucket,
) error {
	for _, fileNode := range fileSet.Manifest().FileNodes() {
		var blob Blob
		if digest := fileNode.Digest(); digest != nil {
			blob = fileSet.BlobSet().GetBlob(digest)
			if blob == nil {
				// This should never happen given our validation.
				return fmt.Errorf("nil Blob with non-empty Digest %v in PutFileSetToBucket", digest)
			}
		}
		writeObjectCloser, err := bucket.Put(ctx, fileNode.Path(), storage.PutWithAtomic())
		if err != nil {
			return err
		}
		if blob != nil {
			if _, err := writeObjectCloser.Write(blob.Content()); err != nil {
				return err
			}
		}
		if err := writeObjectCloser.Close(); err != nil {
			return err
		}
	}
	return nil
}

// FileSetToProtoManifestBlobAndBlobs converts the given FileSet into a proto Blob representing the
// Manifest, and a set of Blobs representing the Files.
//
// TODO: validate the returned proto Blobs.
func FileSetToProtoManifestBlobAndBlobs(fileSet FileSet) (*storagev1beta1.Blob, []*storagev1beta1.Blob, error) {
	protoManifestBlob, err := ManifestToProtoBlob(fileSet.Manifest())
	if err != nil {
		return nil, nil, err
	}
	protoBlobs, err := BlobSetToProtoBlobs(fileSet.BlobSet())
	if err != nil {
		return nil, nil, err
	}
	return protoManifestBlob, protoBlobs, nil
}

// ManifestBlobsAndBlobsToFileSet convers the given manifest Blob and set of Blobs representing
// the Files into a FileSet.
//
// Validation is done to ensure the Manifest exactly matches the BlobSet.
// TODO: validate the input proto Blobs.
func ManifestBlobsAndBlobsToFileSet(
	protoManifestBlob *storagev1beta1.Blob,
	protoBlobs []*storagev1beta1.Blob,
) (FileSet, error) {
	manifest, err := ProtoBlobToManifest(protoManifestBlob)
	if err != nil {
		return nil, err
	}
	blobSet, err := ProtoBlobsToBlobSet(protoBlobs)
	if err != nil {
		return nil, err
	}
	return NewFileSet(manifest, blobSet)
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
