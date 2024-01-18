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

package storagegit

import (
	"github.com/bufbuild/buf/private/pkg/git"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// Provider provides storage buckets for a git repository.
type Provider interface {
	// NewReadBucket returns a new ReadBucket that represents
	// the state of the working tree at the particular tree.
	//
	// Typically, callers will want to source the tree from a commit, but
	// they can also use a subtree of another tree.
	NewReadBucket(hash git.Hash, options ...ReadBucketOption) (storage.ReadBucket, error)
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// ProviderWithSymlinks returns a ProviderOption that results in symlink support.
//
// Note that ReadBucketWithSymlinksIfSupported still needs to be passed for a given
// ReadBucket to have symlinks followed.
func ProviderWithSymlinks() ProviderOption {
	return func(provider *provider) {
		provider.symlinks = true
	}
}

// ReadBucketOption is an option for a new ReadBucket.
type ReadBucketOption func(*readBucketOptions)

// ReadBucketWithSymlinksIfSupported returns a ReadBucketOption that results
// in symlink support being enabled for this bucket. If the Provider did not have symlink
// support, this is a no-op.
func ReadBucketWithSymlinksIfSupported() ReadBucketOption {
	return func(b *readBucketOptions) {
		b.symlinksIfSupported = true
	}
}

// NewProvider creates a new Provider for a git repository.
func NewProvider(objectReader git.ObjectReader, options ...ProviderOption) Provider {
	return newProvider(objectReader, options...)
}
