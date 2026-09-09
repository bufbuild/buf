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

// A cache key must contain everything that affects what is fetched, and nothing
// that is applied to the result afterwards, such as SubDirPath and target
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
	// Set only when filter is set, the one case where it changes what is cloned:
	// the clone is then a sparse checkout of subDirPath.
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
// fetched over the network. Local files have no fetch to save, and streams
// cannot be re-read at all, so neither is cached.
func isRemoteFileScheme(fileScheme FileScheme) bool {
	return fileScheme == FileSchemeHTTP || fileScheme == FileSchemeHTTPS
}
