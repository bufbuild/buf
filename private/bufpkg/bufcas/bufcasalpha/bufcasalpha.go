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

// Package bufcasalpha temporarily converts v1alpha1 API types to new API types.
package bufcasalpha

import (
	"bytes"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
)

var (
	digestTypeToAlpha = map[bufcas.DigestType]modulev1alpha1.DigestType{
		bufcas.DigestTypeShake256: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
	}
	alphaToDigestType = map[modulev1alpha1.DigestType]bufcas.DigestType{
		modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256: bufcas.DigestTypeShake256,
	}
)

func DigestToAlpha(digest bufcas.Digest) *modulev1alpha1.Digest {
	return &modulev1alpha1.Digest{
		DigestType: digestTypeToAlpha[digest.Type()],
		Digest:     digest.Value(),
	}
}

func AlphaToDigest(alphaDigest *modulev1alpha1.Digest) (bufcas.Digest, error) {
	return bufcas.NewDigest(
		alphaDigest.GetDigest(),
		bufcas.DigestWithDigestType(alphaToDigestType[alphaDigest.GetDigestType()]),
	)
}

func BlobToAlpha(blob bufcas.Blob) *modulev1alpha1.Blob {
	return &modulev1alpha1.Blob{
		Digest:  DigestToAlpha(blob.Digest()),
		Content: blob.Content(),
	}
}

func AlphaToBlob(alphaBlob *modulev1alpha1.Blob) (bufcas.Blob, error) {
	digest, err := AlphaToDigest(alphaBlob.GetDigest())
	if err != nil {
		return nil, err
	}
	return bufcas.NewBlobForContent(bytes.NewReader(alphaBlob.GetContent()), bufcas.BlobWithKnownDigest(digest))
}

func BlobSetToAlpha(blobSet bufcas.BlobSet) []*modulev1alpha1.Blob {
	blobs := blobSet.Blobs()
	alphaBlobs := make([]*modulev1alpha1.Blob, len(blobs))
	for i, blob := range blobs {
		alphaBlobs[i] = BlobToAlpha(blob)
	}
	return alphaBlobs
}

func AlphaToBlobSet(alphaBlobs []*modulev1alpha1.Blob) (bufcas.BlobSet, error) {
	blobs := make([]bufcas.Blob, len(alphaBlobs))
	var err error
	for i, alphaBlob := range alphaBlobs {
		blobs[i], err = AlphaToBlob(alphaBlob)
		if err != nil {
			return nil, err
		}
	}
	return bufcas.NewBlobSet(blobs)
}

func AlphaManifestBlobToManifest(manifestAlphaBlob *modulev1alpha1.Blob) (bufcas.Manifest, error) {
	manifestBlob, err := AlphaToBlob(manifestAlphaBlob)
	if err != nil {
		return nil, err
	}
	return bufcas.BlobToManifest(manifestBlob)
}

func ManifestToAlphaManifestBlob(manifest bufcas.Manifest) (*modulev1alpha1.Blob, error) {
	manifestBlob, err := bufcas.ManifestToBlob(manifest)
	if err != nil {
		return nil, err
	}
	return BlobToAlpha(manifestBlob), nil
}

func AlphaManifestBlobAndBlobsToFileSet(
	manifestAlphaBlob *modulev1alpha1.Blob,
	alphaBlobs []*modulev1alpha1.Blob,
) (bufcas.FileSet, error) {
	manifestBlob, err := AlphaToBlob(manifestAlphaBlob)
	if err != nil {
		return nil, err
	}
	manifest, err := bufcas.BlobToManifest(manifestBlob)
	if err != nil {
		return nil, err
	}
	blobSet, err := AlphaToBlobSet(alphaBlobs)
	if err != nil {
		return nil, err
	}
	return bufcas.NewFileSet(manifest, blobSet)
}

func FileSetToAlphaManifestBlobAndBlobs(fileset bufcas.FileSet) (*modulev1alpha1.Blob, []*modulev1alpha1.Blob, error) {
	manifestBlob, err := bufcas.ManifestToBlob(fileset.Manifest())
	if err != nil {
		return nil, nil, err
	}
	return BlobToAlpha(manifestBlob), BlobSetToAlpha(fileset.BlobSet()), nil
}
