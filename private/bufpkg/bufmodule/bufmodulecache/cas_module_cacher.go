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

package bufmodulecache

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
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

func (c *casModuleCacher) GetModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
) (_ bufmodule.Module, retErr error) {
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	digestString := modulePin.Digest()
	if digestString == "" {
		// Attempt to look up manifest digest from commit
		commitPath := normalpath.Join(moduleBasedir, commitsDir, modulePin.Commit())
		digestBytes, err := storage.ReadPath(ctx, c.bucket, commitPath)
		if err != nil {
			return nil, err
		}
		digestString = string(digestBytes)
	}
	digest, err := bufcas.ParseDigest(digestString)
	if err != nil {
		return nil, err
	}
	manifest, err := c.readManifest(ctx, moduleBasedir, digest)
	if err != nil {
		return nil, err
	}
	blobs := make([]bufcas.Blob, 0, len(manifest.FileNodes()))
	for _, fileNode := range manifest.FileNodes() {
		blob, err := c.readBlob(ctx, moduleBasedir, fileNode.Digest())
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, blob)
	}
	blobSet, err := bufcas.NewBlobSet(blobs)
	if err != nil {
		return nil, err
	}
	fileSet, err := bufcas.NewFileSet(manifest, blobSet)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewModuleForFileSet(
		ctx,
		fileSet,
		bufmodule.ModuleWithModuleIdentityAndCommit(
			modulePin,
			modulePin.Commit(),
		),
	)
}

func (c *casModuleCacher) PutModule(
	ctx context.Context,
	modulePin bufmoduleref.ModulePin,
	module bufmodule.Module,
) (retErr error) {
	fileSet := module.FileSet()
	if fileSet == nil {
		return fmt.Errorf("FileSet must be non-nil")
	}
	manifest := fileSet.Manifest()
	// TODO: what about empty modules? Need to handle empty Manifests in bufcas
	manifestBlob, err := bufcas.ManifestToBlob(manifest)
	if err != nil {
		return err
	}
	manifestDigest := manifestBlob.Digest()
	if modulePinDigestEncoded := modulePin.Digest(); modulePinDigestEncoded != "" {
		modulePinDigest, err := bufcas.ParseDigest(modulePinDigestEncoded)
		if err != nil {
			return fmt.Errorf("invalid module pin digest %q: %w", modulePinDigestEncoded, err)
		}
		if !bufcas.DigestEqual(manifestDigest, modulePinDigest) {
			return fmt.Errorf("manifest digest mismatch: pin=%q, module=%q", modulePinDigest.String(), manifestDigest.String())
		}
	}
	moduleBasedir := normalpath.Join(modulePin.Remote(), modulePin.Owner(), modulePin.Repository())
	for _, blob := range fileSet.BlobSet().Blobs() {
		if err := c.writeBlob(ctx, moduleBasedir, blob); err != nil {
			return err
		}
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
	digest bufcas.Digest,
) (_ bufcas.Blob, retErr error) {
	digestHex := hex.EncodeToString(digest.Value())
	blobPath := normalpath.Join(moduleBasedir, blobsDir, digestHex[:2], digestHex[2:])
	readObjectCloser, err := c.bucket.Get(ctx, blobPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	blob, err := bufcas.NewBlobForContent(readObjectCloser, bufcas.BlobWithKnownDigest(digest))
	if err != nil {
		return nil, fmt.Errorf("failed to create blob from path %s: %w", blobPath, err)
	}
	return blob, nil
}

func (c *casModuleCacher) validateBlob(
	ctx context.Context,
	moduleBasedir string,
	digest bufcas.Digest,
) (_ bool, retErr error) {
	digestHex := hex.EncodeToString(digest.Value())
	blobPath := normalpath.Join(moduleBasedir, blobsDir, digestHex[:2], digestHex[2:])
	readObjectCloser, err := c.bucket.Get(ctx, blobPath)
	if err != nil {
		return false, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObjectCloser.Close())
	}()
	cacheDigest, err := bufcas.NewDigestForContent(readObjectCloser, bufcas.DigestWithDigestType(digest.Type()))
	if err != nil {
		return false, err
	}
	return bufcas.DigestEqual(digest, cacheDigest), nil
}

func (c *casModuleCacher) readManifest(
	ctx context.Context,
	moduleBasedir string,
	digest bufcas.Digest,
) (_ bufcas.Manifest, retErr error) {
	blob, err := c.readBlob(ctx, moduleBasedir, digest)
	if err != nil {
		return nil, err
	}
	manifest, err := bufcas.BlobToManifest(blob)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (c *casModuleCacher) writeBlob(
	ctx context.Context,
	moduleBasedir string,
	blob bufcas.Blob,
) (retErr error) {
	// Avoid unnecessary write if the blob is already written to disk
	valid, err := c.validateBlob(ctx, moduleBasedir, blob.Digest())
	if err == nil && valid {
		return nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		c.logger.Debug(
			"repairing cache entry",
			zap.String("basedir", moduleBasedir),
			zap.String("digest", blob.Digest().String()),
		)
	}
	digestHex := hex.EncodeToString(blob.Digest().Value())
	blobPath := normalpath.Join(moduleBasedir, blobsDir, digestHex[:2], digestHex[2:])
	return c.atomicWrite(ctx, bytes.NewReader(blob.Content()), blobPath)
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
