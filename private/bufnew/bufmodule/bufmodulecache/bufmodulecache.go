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
	"github.com/bufbuild/buf/private/bufnew/bufmodule"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// NewModuleDataProvider returns a new ModuleDataProvider that caches the results of the delegate.
//
// The given Bucket is used as a cache. This package can choose to use the bucket however it wishes.
//
// TODO: Actually implement this. Right now it is just a passthrough.
func NewModuleDataProvider(
	delegate bufmodule.ModuleDataProvider,
	moduleCacheBucket storage.ReadWriteBucket,
) bufmodule.ModuleDataProvider {
	return delegate
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
