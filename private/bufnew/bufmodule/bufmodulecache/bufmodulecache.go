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
	"encoding/hex"
	"io"
	"log"

	"github.com/bufbuild/buf/private/bufnew/bufconfig"
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/normalpath"
	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/multierr"
)

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The given Bucket is used as a cache. This package can choose to use the bucket however it wishes.
func NewModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	moduleCacheBucket storage.ReadWriteBucket,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(delegate, moduleCacheBucket)
}

/// *** PRIVATE ***

type moduleDataProvider struct {
	delegate          bufmodule.ModuleDataProvider
	moduleCacheBucket storage.ReadWriteBucket
}

func newModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	moduleCacheBucket storage.ReadWriteBucket,
) *moduleDataProvider {
	return &moduleDataProvider{
		delegate:          delegate,
		moduleCacheBucket: moduleCacheBucket,
	}
}

func (p *moduleDataProvider) GetModuleDatasForModuleKeys(
	ctx context.Context,
	keys ...bufmodule.ModuleKey,
) ([]bufmodule.ModuleData, error) {
	moduleDatas := make([]bufmodule.ModuleData, len(keys))
	cacheMissKeys := make([]bufmodule.ModuleKey, 0)
	for i, key := range keys {
		cachedData, err := p.getCachedModuleDataForKey(ctx, key)
		if err != nil {
			cacheMissKeys = append(cacheMissKeys, key)
			continue
		}
		moduleDatas[i] = cachedData
	}
	if len(cacheMissKeys) > 0 {
		cacheMissDatas, err := p.delegate.GetModuleDatasForModuleKeys(ctx, cacheMissKeys...)
		if err != nil {
			return nil, err
		}
		j := 0
		for i, key := range cacheMissKeys {
			if keys[j] != key {
				j++
			}
			moduleDatas[j] = cacheMissDatas[i]
			j++
		}
		if err := p.putCachedModuleDatas(ctx, cacheMissDatas); err != nil {
			// TODO: Not clear what to do here.
			log.Printf("error writing cache: %v", err)
		}
	}
	return moduleDatas, nil
}

func (p *moduleDataProvider) getCachedModuleDataForKey(
	ctx context.Context,
	key bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	moduleFullName := key.ModuleFullName()
	digest, err := key.Digest()
	if err != nil {
		return nil, err
	}
	modulePrefix := getModulePrefix(moduleFullName, digest)
	lockFile, err := bufconfig.GetBufLockFileForPrefix(ctx, p.moduleCacheBucket, modulePrefix)
	if err != nil {
		return nil, err
	}
	getBucket := func() (storage.ReadBucket, error) {
		return storage.MapReadBucket(p.moduleCacheBucket, storage.MapOnPrefix(modulePrefix)), nil
	}
	getDeclaredDepModuleKeys := func() ([]bufmodule.ModuleKey, error) {
		return lockFile.DepModuleKeys(), nil
	}
	// TODO: notes from @bufdev says we need to preserve commit ID if present in input module key.
	// However, we always use the input module key verbatim, so that's never a concern.
	// Something is possibly wrong here.
	return bufmodule.NewModuleData(
		key,
		getBucket,
		getDeclaredDepModuleKeys,
		bufmodule.ModuleDataWithActualDigest(digest),
	)
}

func (p *moduleDataProvider) putCachedModuleDatas(
	ctx context.Context,
	moduleDatas []bufmodule.ModuleData,
) error {
	var err error
	for _, moduleData := range moduleDatas {
		err = multierr.Append(err, p.putCachedModuleData(ctx, moduleData))
	}
	return err
}

func (p *moduleDataProvider) putCachedModuleData(
	ctx context.Context,
	moduleData bufmodule.ModuleData,
) error {
	moduleKey := moduleData.ModuleKey()
	moduleFullName := moduleKey.ModuleFullName()
	digest, err := moduleKey.Digest()
	if err != nil {
		return err
	}
	modulePrefix := getModulePrefix(moduleFullName, digest)
	depModuleKeys, err := moduleData.DeclaredDepModuleKeys()
	if err != nil {
		return err
	}
	lockFile, err := bufconfig.NewBufLockFile(bufconfig.FileVersionV2, depModuleKeys)
	if err != nil {
		return err
	}
	bucket, err := moduleData.Bucket()
	if err != nil {
		return err
	}
	err = bucket.Walk(ctx, "", func(info storage.ObjectInfo) error {
		reader, err := bucket.Get(ctx, info.Path())
		if err != nil {
			return err
		}
		// TODO: may need to validate that paths are not e.g. MS-DOS device names.
		// maybe special bucket wrapper for OSes/filesystems that do not have bag-of-bytes filenames
		// TODO: case sensitivity?
		return p.putFileAtomic(ctx, normalpath.Join(modulePrefix, info.Path()), reader)
	})
	if err != nil {
		return err
	}
	// Put the lockfile last, so that we only have a lockfile if the cache is finished writing.
	return bufconfig.PutBufLockFileForPrefix(ctx, p.moduleCacheBucket, modulePrefix, lockFile)
}

func (p *moduleDataProvider) putFileAtomic(
	ctx context.Context,
	path string,
	contents io.ReadCloser,
) error {
	destination, err := p.moduleCacheBucket.Put(ctx, path, storage.PutWithAtomic())
	if err != nil {
		return err
	}
	return copyAndClose(destination, contents)
}

// Returns the module's path. This is "registry/owner/name/$DIGEST_TYPE/${DIGEST:0:2}/${DIGEST:2}",
// e.g. the module "buf.build/acme/weather" with digest "shake256:12345" will return
// "buf.build/acme/weather/shake256/12/345".
func getModulePrefix(moduleFullName bufmodule.ModuleFullName, digest bufcas.Digest) string {
	digestHexString := hex.EncodeToString(digest.Value())
	return normalpath.Join(
		moduleFullName.String(),
		digest.Type().String(),
		digestHexString[:2],
		digestHexString[2:],
	)
}

func copyAndClose(destination io.WriteCloser, source io.ReadCloser) (err error) {
	defer func() {
		err = multierr.Append(err, destination.Close())
		err = multierr.Append(err, source.Close())
	}()
	_, err = io.Copy(destination, source)
	return err
}

// *** IMPLEMENTATION NOTES ***
//
// - The input bucket will be the path to the module cache. This will likely be ~/.buf/cache/v3/mod
//   in most cases, but this is a detail for the caller of NewModuleDataProvider to figure out (ie
//   likely the bufcli package). You can assume you have complete ownership over the input
//   moduleCacheBucket. We use "$CACHE" to reference the root of the moduleCacheBucket, i.e.
//   "$CACHE/foo/bar" means "~/.buf/cache/v3/mod/foo/bar" in reality.
//
// - A module's files is stored at "$CACHE/registry/owner/name/$DIGEST_TYPE/$DIGEST_HEX". For example,
//   The module "buf.build/acme/weather" with digest "shake256:12345" is stored at
//   "$CACHE/buf.build/acme/weather/shake256/12345".
//
// - A module's file is consist of its .proto files, documentation file (README.md, README.markdown,
//   or buf.md), and license files (LICENSE). All of these should be stored in the cache.
//
// - Additionally, the cache should store the module's dependencies. In lieu of a better way to do
//   this, it might as well re-use buf.lock files. Use
//   bufconfig.NewLockFile(bufconfig.FileVersionV2, depModuleKeys) to create BufLockFiles, and write
//   them using
//   bufconfig.PutBufLockFileForPrefix(
//    ctx,
//    moduleCacheBucket,
//	  normalpath.Join(moduleFullName.String(), digest.Type().String(), hex.EncodeToString(digest.Value())),
//    newBufLockFileYouCreated,
//   )
//
//   This will just write a buf.lock to "$CACHE/registry/owner/name/$DIGEST_TYPE/$DIGEST_HEX".
//
//   Note that even though this buf.lock file lives in the same location as the module's actual files,
//   it will NOT be picked up as part of the module's files, and will NOT be used to compute digests,
//   as newModule -> newModuleReadBucket filters to the specific module files it needs. You'll have to
//   manually read this file via bufconfig.GetBufLockFileForPrefix(...) to get the dependency list,
//   and use these ModuleKeys as your dependencies for NewModuleData (ie return them from getDeclaredDepModuleKeys).
//
// - Use bufmodule.NewModuleData to create the returned ModuleData. The getBucket function can return
//   storage.MapReadBucket(moduleCacheBucket, storage.MapOnPrefix(normalpath.Join(...)), even though this
//   bucket will have the buf.lock in it. See above. You'll need to read the dep ModuleKeys from that
//   buf.lock file manually, see above.
//
// - Make sure to use bufmodule.ModuleDataWithActualDigest(inputModuleKey.Digest()). This will do tamper-proofing.
//
// - You'll need to use storageos.PutWithAtomic when writing to the cache.
//
// - Note that you need to be careful to propagate back the CommitID if it is set on an input ModuleKey. We only
//   are using the digest here, but you may still have a CommitID on the input ModuleKey. The returned ModuleData
//   needs to have this CommitID. See bufmoduleapi's ModuleDataProvider implementation for how we do this (copied below):
//
// // CommitID is optional.
// if commitID == "" {
//   // If we did not have a commitID, make a new ModuleKey with the retrieved commitID.
//   // All remote Modules must have a commitID.
//   moduleKey, err = bufmodule.NewModuleKey(
//     moduleKey.ModuleFullName(),
//     protoCommitNode.Commit.Id,
//     // *** Use the Digest from the moduleKey, NOT from the protoCommitNode. ***
//     // We use this for tamper-proofing, see comment below.
//     moduleKey.Digest,
//   )
// }
