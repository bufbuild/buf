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

// subdirectories under ~/.cache/buf/v2/{remote}/{owner}/{repo}
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
	manifestDigestStr := modulePin.Digest()
	if manifestDigestStr == "" {
		// Attempt to look up manifest digest from commit
		commitPath := normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())
		manifestDigestBytes, err := c.loadPath(ctx, commitPath)
		if err != nil {
			return nil, err
		}
		manifestDigestStr = strings.TrimSpace(string(manifestDigestBytes))
	}
	manifestDigest, err := manifest.NewDigestFromString(manifestDigestStr)
	if err != nil {
		return nil, err
	}
	manifestFromCache, err := c.loadManifestFromCache(ctx, moduleBasedir, manifestDigest)
	if err != nil {
		return nil, err
	}
	var blobs []manifest.Blob
	blobBasedir := normalpath.Join(moduleBasedir, blobsDir)
	blobDigests := make(map[string]struct{})
	for _, path := range manifestFromCache.Paths() {
		digest, found := manifestFromCache.DigestFor(path)
		if !found {
			return nil, fmt.Errorf("digest not found for path: %s", path)
		}
		hexDigest := digest.Hex()
		if _, ok := blobDigests[hexDigest]; ok {
			// We've already loaded this blob
			continue
		}
		blobPath := normalpath.Join(blobBasedir, hexDigest[:2], hexDigest[2:])
		contents, err := c.loadPath(ctx, blobPath)
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
	return bufmodule.NewModuleForManifestAndBlobSet(ctx, manifestFromCache, blobSet)
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
		return errors.New("empty manifest digest")
	}
	if modulePinDigestEncoded := modulePin.Digest(); modulePinDigestEncoded != "" {
		modulePinDigest, err := manifest.NewDigestFromString(modulePinDigestEncoded)
		if err != nil {
			return fmt.Errorf("invalid digest %q: %w", modulePinDigestEncoded, err)
		}
		if digest.String() != modulePinDigest.String() {
			return fmt.Errorf("manifest digest mismatch: expected=%q, found=%q", modulePinDigest.String(), digest.String())
		}
	}
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	// Write blobs
	blobsParentDir := normalpath.Join(moduleBasedir, blobsDir)
	writtenDigests := make(map[string]struct{})
	for _, path := range moduleManifest.Paths() {
		blobDigest, found := moduleManifest.DigestFor(path)
		if !found {
			return fmt.Errorf("failed to find digest for path=%q", path)
		}
		blobHexDigest := blobDigest.Hex()
		if _, ok := writtenDigests[blobHexDigest]; ok {
			continue
		}
		blob, found := module.BlobSet().BlobFor(blobDigest.String())
		if !found {
			return fmt.Errorf("blob not found for path=%q, digest=%q", path, blobHexDigest)
		}
		if err := c.writeBlob(ctx, blob, blobsParentDir); err != nil {
			return err
		}
		writtenDigests[blobHexDigest] = struct{}{}
	}
	// Write manifest
	manifestsParentDir := normalpath.Join(moduleBasedir, manifestsDir)
	if err := c.writeBlob(ctx, manifestBlob, manifestsParentDir); err != nil {
		return err
	}
	// Write commit
	commitPath := normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())
	if err := c.atomicWrite(ctx, strings.NewReader(manifestBlob.Digest().String()), commitPath); err != nil {
		return err
	}
	return nil
}

func (c *casModuleCacher) loadManifestFromCache(
	ctx context.Context,
	moduleBasedir string,
	manifestDigest *manifest.Digest,
) (_ *manifest.Manifest, retErr error) {
	manifestHexDigest := manifestDigest.Hex()
	manifestPath := normalpath.Join(moduleBasedir, manifestsDir, manifestHexDigest[:2], manifestHexDigest[2:])
	f, err := c.bucket.Get(ctx, manifestPath)
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
	if cacheManifestBlob.Digest().String() != manifestDigest.String() {
		return nil, fmt.Errorf("digest mismatch - expected: %q, found: %q", manifestDigest.String(), cacheManifestBlob.Digest().String())
	}
	return cacheManifest, nil
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
		retErr = multierr.Append(retErr, contents.Close())
	}()
	hexDigest := blob.Digest().Hex()
	return c.atomicWrite(ctx, contents, normalpath.Join(parentDir, hexDigest[:2], hexDigest[2:]))
}

func (c *casModuleCacher) atomicWrite(ctx context.Context, contents io.Reader, path string) (retErr error) {
	f, err := c.bucket.Put(ctx, path, storage.PutWithAtomic())
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

func (c *casModuleCacher) loadPath(
	ctx context.Context,
	path string,
) (_ []byte, retErr error) {
	f, err := c.bucket.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, f.Close())
	}()
	return io.ReadAll(f)
}
