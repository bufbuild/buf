// Copyright 2020-2025 Buf Technologies, Inc.
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

package cas

import (
	"context"

	"github.com/bufbuild/buf/private/pkg/storage"
)

// BucketToDigest converts the files in the bucket to a digest.
//
// This uses the specified DigestType for the file and manifest digests.
func BucketToDigest(ctx context.Context, bucket storage.ReadBucket, digestType DigestType) (Digest, error) {
	var fileNodes []FileNode
	if err := storage.WalkReadObjects(
		ctx,
		bucket,
		"",
		func(readObject storage.ReadObject) error {
			digest, err := NewDigestForContent(readObject, DigestWithDigestType(digestType))
			if err != nil {
				return err
			}
			fileNode, err := NewFileNode(readObject.Path(), digest)
			if err != nil {
				return err
			}
			fileNodes = append(fileNodes, fileNode)
			return nil
		},
	); err != nil {
		return nil, err
	}
	manifest, err := NewManifest(fileNodes)
	if err != nil {
		return nil, err
	}
	return ManifestToDigest(manifest, digestType)
}
