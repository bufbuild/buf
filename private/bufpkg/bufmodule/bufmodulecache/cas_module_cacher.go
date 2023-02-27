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
	blobsDir   = "blobs"
	commitsDir = "commits"
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
		manifestDigestStr = string(manifestDigestBytes)
	}
	manifestDigest, err := manifest.NewDigestFromString(manifestDigestStr)
	if err != nil {
		return nil, err
	}
	manifestFromCache, err := c.readManifest(ctx, moduleBasedir, manifestDigest)
	if err != nil {
		return nil, err
	}
	var blobs []manifest.Blob
	blobDigests := make(map[string]struct{})
	for _, path := range manifestFromCache.Paths() {
		digest, found := manifestFromCache.DigestFor(path)
		if !found {
			return nil, fmt.Errorf("digest not found for path: %s", path)
		}
		if _, ok := blobDigests[digest.String()]; ok {
			// We've already loaded this blob
			continue
		}
		blob, err := c.readBlob(ctx, moduleBasedir, digest)
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
	if moduleManifest == nil {
		return fmt.Errorf("manifest must be non-nil")
	}
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
		if !digest.Equal(*modulePinDigest) {
			return fmt.Errorf("manifest digest mismatch: expected=%q, found=%q", modulePinDigest.String(), digest.String())
		}
	}
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	// Write blobs
	writtenDigests := make(map[string]struct{})
	for _, path := range moduleManifest.Paths() {
		blobDigest, found := moduleManifest.DigestFor(path)
		if !found {
			return fmt.Errorf("failed to find digest for path=%q", path)
		}
		blobDigestStr := blobDigest.String()
		if _, ok := writtenDigests[blobDigestStr]; ok {
			continue
		}
		blob, found := module.BlobSet().BlobFor(blobDigestStr)
		if !found {
			return fmt.Errorf("blob not found for path=%q, digest=%q", path, blobDigestStr)
		}
		if err := c.writeBlob(ctx, moduleBasedir, blob); err != nil {
			return err
		}
		writtenDigests[blobDigestStr] = struct{}{}
	}
	// Write manifest
	if err := c.writeBlob(ctx, moduleBasedir, manifestBlob); err != nil {
		return err
	}
	// Write commit
	commitPath := normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())
	if err := c.atomicWrite(ctx, strings.NewReader(manifestBlob.Digest().String()), commitPath); err != nil {
		return err
	}
	return nil
}

func (c *casModuleCacher) readBlob(
	ctx context.Context,
	moduleBasedir string,
	digest *manifest.Digest,
) (_ manifest.Blob, retErr error) {
	hexDigest := digest.Hex()
	blobPath := normalpath.Join(moduleBasedir, blobsDir, hexDigest[:2], hexDigest[2:])
	contents, err := c.loadPath(ctx, blobPath)
	if err != nil {
		return nil, err
	}
	blob, err := manifest.NewMemoryBlob(*digest, contents, manifest.MemoryBlobWithDigestValidation())
	if err != nil {
		return nil, fmt.Errorf("failed to create blob from path %s: %w", blobPath, err)
	}
	return blob, nil
}

func (c *casModuleCacher) validateBlob(
	ctx context.Context,
	moduleBasedir string,
	digest *manifest.Digest,
) (bool, error) {
	hexDigest := digest.Hex()
	blobPath := normalpath.Join(moduleBasedir, blobsDir, hexDigest[:2], hexDigest[2:])
	f, err := c.bucket.Get(ctx, blobPath)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			c.logger.Debug("err closing blob", zap.Error(err))
		}
	}()
	digester, err := manifest.NewDigester(digest.Type())
	if err != nil {
		return false, err
	}
	cacheDigest, err := digester.Digest(f)
	if err != nil {
		return false, err
	}
	return digest.Equal(*cacheDigest), nil
}

func (c *casModuleCacher) readManifest(
	ctx context.Context,
	moduleBasedir string,
	manifestDigest *manifest.Digest,
) (_ *manifest.Manifest, retErr error) {
	blob, err := c.readBlob(ctx, moduleBasedir, manifestDigest)
	if err != nil {
		return nil, err
	}
	f, err := blob.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, f.Close())
	}()
	moduleManifest, err := manifest.NewFromReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", manifestDigest.String(), err)
	}
	return moduleManifest, nil
}

func (c *casModuleCacher) writeBlob(
	ctx context.Context,
	moduleBasedir string,
	blob manifest.Blob,
) (retErr error) {
	// Avoid unnecessary write if the blob is already written to disk
	valid, err := c.validateBlob(ctx, moduleBasedir, blob.Digest())
	if err == nil && valid {
		return nil
	}
	if !storage.IsNotExist(err) {
		c.logger.Debug(
			"repairing cache entry",
			zap.String("basedir", moduleBasedir),
			zap.String("digest", blob.Digest().String()),
		)
	}
	contents, err := blob.Open(ctx)
	if err != nil {
		return err
	}
	defer func() {
		retErr = multierr.Append(retErr, contents.Close())
	}()
	hexDigest := blob.Digest().Hex()
	blobPath := normalpath.Join(moduleBasedir, blobsDir, hexDigest[:2], hexDigest[2:])
	return c.atomicWrite(ctx, contents, blobPath)
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
