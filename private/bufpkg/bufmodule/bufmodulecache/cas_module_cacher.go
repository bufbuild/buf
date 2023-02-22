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

package bufmodulecache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	"github.com/bufbuild/buf/private/pkg/manifest"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

const (
	blobsDir     = "blobs"
	commitsDir   = "commits"
	manifestsDir = "manifests"
)

type casModuleCacher struct {
	logger *zap.Logger
	bucket storage.ReadWriteBucket
}

var _ moduleCache = (*casModuleCacher)(nil)

func (c *casModuleCacher) GetModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (_ bufmodule.Module, retErr error) {
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	var manifestHexDigest string
	if modulePinDigestEncoded := modulePin.Digest(); modulePinDigestEncoded != "" {
		modulePinDigest, err := manifest.NewDigestFromString(modulePinDigestEncoded)
		if err != nil {
			return nil, err
		}
		manifestHexDigest = modulePinDigest.Hex()
	} else {
		// Attempt to look up manifest digest from commit
		commitPath := normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())
		manifestDigest, err := c.bucket.Get(ctx, commitPath)
		if err != nil {
			return nil, err
		}
		manifestDigestBytes, err := io.ReadAll(manifestDigest)
		if err != nil {
			return nil, err
		}
		manifestHexDigest = strings.TrimSpace(string(manifestDigestBytes))
		if manifestHexDigest == "" {
			return nil, fmt.Errorf("invalid manifest digest found in %s", commitPath)
		}
	}
	f, err := c.bucket.Get(ctx, normalpath.Join(moduleBasedir, manifestsDir, manifestHexDigest[:2], manifestHexDigest[2:]))
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, f.Close())
	}()
	cacheManifest, err := manifest.NewFromReader(f)
	if err != nil {
		return nil, err
	}
	cacheManifestBlob, err := cacheManifest.Blob()
	if err != nil {
		return nil, err
	}
	if cacheManifestBlob.Digest().Hex() != manifestHexDigest {
		return nil, storage.NewErrNotExist(fmt.Sprintf("digest mismatch - expected: %q, found: %q", manifestHexDigest, cacheManifestBlob.Digest().Hex()))
	}
	var blobs []manifest.Blob
	blobBasedir := normalpath.Join(moduleBasedir, blobsDir)
	blobDigests := make(map[string]struct{})
	for _, path := range cacheManifest.Paths() {
		digest, found := cacheManifest.DigestFor(path)
		if !found {
			return nil, storage.NewErrNotExist(path)
		}
		hexDigest := digest.Hex()
		if _, ok := blobDigests[hexDigest]; ok {
			// We've already loaded this
			continue
		}
		blobPath := normalpath.Join(blobBasedir, hexDigest[:2], hexDigest[2:])
		f, err := c.bucket.Get(ctx, blobPath)
		if err != nil {
			return nil, err
		}
		contents, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		blob, err := manifest.NewMemoryBlob(*digest, contents, manifest.MemoryBlobWithDigestValidation())
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, blob)
	}
	blobSet, err := manifest.NewBlobSet(ctx, blobs)
	if err != nil {
		return nil, err
	}
	bucket, err := manifest.NewBucket(
		*cacheManifest,
		*blobSet,
		manifest.BucketWithAllManifestBlobsValidation(),
		manifest.BucketWithNoExtraBlobsValidation(),
	)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForBucket(ctx, bucket, bufmodule.ModuleWithManifestAndBlobs(*cacheManifest, *blobSet))
}

func (c *casModuleCacher) PutModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
	module bufmodule.Module,
) (retErr error) {
	moduleManifest := module.Manifest()
	manifestBlob, err := moduleManifest.Blob()
	if err != nil {
		return err
	}
	digest := manifestBlob.Digest()
	if digest == nil {
		return errors.New("invalid manifest digest")
	}
	if modulePinDigestEncoded := modulePin.Digest(); modulePinDigestEncoded != "" {
		modulePinDigest, err := manifest.NewDigestFromString(modulePinDigestEncoded)
		if err != nil {
			return fmt.Errorf("invalid digest %q: %w", modulePinDigestEncoded, err)
		}
		if digest.Hex() != modulePinDigest.Hex() {
			return fmt.Errorf("manifest digest mismatch: expected=%q, found=%q", modulePinDigest.Hex(), digest.Hex())
		}
	}
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	if err := c.atomicWrite(ctx, strings.NewReader(manifestBlob.Digest().Hex()), normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())); err != nil {
		return err
	}
	manifestsParentDir := normalpath.Join(moduleBasedir, manifestsDir)
	if err := c.writeBlob(ctx, manifestBlob, manifestsParentDir); err != nil {
		return err
	}
	blobsParentDir := normalpath.Join(moduleBasedir, blobsDir)
	writtenDigests := make(map[string]struct{})
	for _, path := range moduleManifest.Paths() {
		blobDigest, found := moduleManifest.DigestFor(path)
		if !found {
			return fmt.Errorf("failed to find digest for: %q", path)
		}
		blobHexDigest := blobDigest.Hex()
		blobs := module.Blobs()
		if _, ok := writtenDigests[blobHexDigest]; ok {
			// We've already written this blob
			continue
		}
		blob, found := blobs.BlobFor(blobDigest.String())
		if !found {
			return fmt.Errorf("blob not found for path=%q, digest=%q", path, blobHexDigest)
		}
		if err := c.writeBlob(ctx, blob, blobsParentDir); err != nil {
			return err
		}
		writtenDigests[blobHexDigest] = struct{}{}
	}
	return nil
}

func (c *casModuleCacher) writeBlob(
	ctx context.Context,
	blob manifest.Blob,
	parentDir string,
) (retErr error) {
	contents, err := blob.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := contents.Close(); err != nil {
			retErr = multierr.Append(retErr, err)
		}
	}()
	hexDigest := blob.Digest().Hex()
	return c.atomicWrite(ctx, contents, normalpath.Join(parentDir, hexDigest[:2], hexDigest[2:]))
}

func (c *casModuleCacher) atomicWrite(ctx context.Context, contents io.Reader, path string) (retErr error) {
	f, err := c.bucket.Put(ctx, path, storage.PutAtomic())
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, f.Close())
	}()
	if _, err := io.Copy(f, contents); err != nil {
		return err
	}
	return nil
}
