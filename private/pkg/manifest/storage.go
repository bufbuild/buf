// Copyright 2020-2022 Buf Technologies, Inc.
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

// Manifests are a list of paths and their hash digests, canonically ordered by
// path in increasing lexographical order. Manifests are encoded as:
//
//	<hash type>:<digest>[SP][SP]<path>[LF]
//
// "shake256" is the only supported hash type. The digest is 64 bytes of hex
// encoded output of SHAKE256. See golang.org/x/crypto/sha3 and FIPS 202 for
// details on the SHAKE hash.
package manifest

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
	"github.com/bufbuild/buf/private/pkg/storage/storagemem"
	"go.uber.org/multierr"
)

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

// ToBucket takes a map of digest to files' contents, and builds a storagemem
// bucket from the manifest.
func (m *Manifest) ToBucket(digestToFiles map[string][]byte) (storage.ReadBucket, error) {
	if len(m.Paths()) == 0 {
		// nothing to build
		return storagemem.NewReadWriteBucketWithOptions()
	}
	if len(digestToFiles) < 1 {
		return nil, errors.New("empty files map")
	}
	bucketFiles := make(map[string][]byte, 0)
	for _, filePath := range m.Paths() {
		fileDigest, ok := m.DigestFor(filePath)
		if !ok {
			return nil, fmt.Errorf("path %q has no digest", filePath)
		}
		fileContent, ok := digestToFiles[fileDigest.String()]
		if !ok {
			return nil, fmt.Errorf(
				"cannot build file %q: digest %q not present in input",
				filePath, fileDigest.String(),
			)
		}
		bucketFiles[filePath] = fileContent
	}
	return storagemem.NewReadWriteBucketWithOptions(
		storagemem.WithFiles(bucketFiles),
	)
}
