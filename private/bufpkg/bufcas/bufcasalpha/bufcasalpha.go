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

// Package bufcasalpha temporarily converts v1alpha1 API types to new API types.
//
// Minimal validation is done as this is assumed to be done by the bufcas package,
// and this allows us to have less error returning.
package bufcasalpha

import (
	modulev1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	storagev1beta1 "github.com/bufbuild/buf/private/gen/proto/go/buf/registry/storage/v1beta1"
)

var (
	digestTypeToAlpha = map[storagev1beta1.Digest_Type]modulev1alpha1.DigestType{
		storagev1beta1.Digest_TYPE_SHAKE256: modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256,
	}
	alphaToDigestType = map[modulev1alpha1.DigestType]storagev1beta1.Digest_Type{
		modulev1alpha1.DigestType_DIGEST_TYPE_SHAKE256: storagev1beta1.Digest_TYPE_SHAKE256,
	}
)

func DigestToAlpha(digest *storagev1beta1.Digest) *modulev1alpha1.Digest {
	if digest == nil {
		return nil
	}
	return &modulev1alpha1.Digest{
		DigestType: digestTypeToAlpha[digest.Type],
		Digest:     digest.Value,
	}
}

func AlphaToDigest(alphaDigest *modulev1alpha1.Digest) *storagev1beta1.Digest {
	if alphaDigest == nil {
		return nil
	}
	return &storagev1beta1.Digest{
		Type:  alphaToDigestType[alphaDigest.DigestType],
		Value: alphaDigest.Digest,
	}
}

func BlobToAlpha(blob *storagev1beta1.Blob) *modulev1alpha1.Blob {
	if blob == nil {
		return nil
	}
	return &modulev1alpha1.Blob{
		Digest:  DigestToAlpha(blob.Digest),
		Content: blob.Content,
	}
}

func AlphaToBlob(alphaBlob *modulev1alpha1.Blob) *storagev1beta1.Blob {
	if alphaBlob == nil {
		return nil
	}
	return &storagev1beta1.Blob{
		Digest:  AlphaToDigest(alphaBlob.Digest),
		Content: alphaBlob.Content,
	}
}

func BlobsToAlpha(blobs []*storagev1beta1.Blob) []*modulev1alpha1.Blob {
	if blobs == nil {
		return nil
	}
	alphaBlobs := make([]*modulev1alpha1.Blob, len(blobs))
	for i, blob := range blobs {
		alphaBlobs[i] = BlobToAlpha(blob)
	}
	return alphaBlobs
}

func AlphaToBlobs(alphaBlobs []*modulev1alpha1.Blob) []*storagev1beta1.Blob {
	if alphaBlobs == nil {
		return nil
	}
	blobs := make([]*storagev1beta1.Blob, len(alphaBlobs))
	for i, alphaBlob := range alphaBlobs {
		blobs[i] = AlphaToBlob(alphaBlob)
	}
	return blobs
}
