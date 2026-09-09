// Copyright 2020-2026 Buf Technologies, Inc.
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

package internal

import "github.com/bufbuild/buf/private/pkg/git"

// A Reader fetches at most once per cache key below, for the lifetime of the
// Reader, which is the lifetime of a single command invocation. Fetched content
// is held for that whole lifetime, so only remote fetches and clones are
// cached: a local file has no fetch work to save, and a stream cannot be
// re-read at all.
//
// A key must contain every field that affects what is fetched, and must exclude
// everything applied to the result afterwards, such as SubDirPath and target
// paths, so that Refs differing only in those still share a fetch.

// fileDataCacheKey identifies the contents of a fetched file.
type fileDataCacheKey struct {
	path                string
	fileScheme          FileScheme
	compressionType     CompressionType
	keepFileCompression bool
}

// gitBucketCacheKey identifies the contents of a cloned git repository.
type gitBucketCacheKey struct {
	path              string
	gitScheme         GitScheme
	cloneBranch       string
	checkout          string
	depth             uint32
	recurseSubmodules bool
	filter            string
	// subDirPath is part of the key only when filter is set, as that is the only
	// case where it changes what is cloned: the clone is then a sparse checkout
	// of subDirPath. Otherwise the whole repository is cloned and subDirPath is
	// applied to the resulting bucket.
	subDirPath string
}

func newGitBucketCacheKey(gitRef GitRef) gitBucketCacheKey {
	cloneBranch, checkout := git.FetchIdentity(gitRef.GitName())
	filter := gitRef.Filter()
	var subDirPath string
	if filter != "" {
		subDirPath = gitRef.SubDirPath()
	}
	return gitBucketCacheKey{
		path:              gitRef.Path(),
		gitScheme:         gitRef.GitScheme(),
		cloneBranch:       cloneBranch,
		checkout:          checkout,
		depth:             gitRef.Depth(),
		recurseSubmodules: gitRef.RecurseSubmodules(),
		filter:            filter,
		subDirPath:        subDirPath,
	}
}

// isRemoteFileScheme returns whether a FileRef with the given FileScheme is
// fetched from a remote host.
func isRemoteFileScheme(fileScheme FileScheme) bool {
	return fileScheme == FileSchemeHTTP || fileScheme == FileSchemeHTTPS
}
