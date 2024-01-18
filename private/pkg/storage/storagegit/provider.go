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

type provider struct {
	objectReader git.ObjectReader
	symlinks     bool
}

func newProvider(objectReader git.ObjectReader, opts ...ProviderOption) *provider {
	p := &provider{
		objectReader: objectReader,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *provider) NewReadBucket(treeHash git.Hash, options ...ReadBucketOption) (storage.ReadBucket, error) {
	var opts readBucketOptions
	for _, opt := range options {
		opt(&opts)
	}
	tree, err := p.objectReader.Tree(treeHash)
	if err != nil {
		return nil, err
	}
	return newBucket(
		p.objectReader,
		p.symlinks && opts.symlinksIfSupported,
		tree,
	)
}

// doing this as a separate struct so that it's clear this is resolved
// as a combination of the provider options and read write bucket options
// so there's no potential issues in newBucket
type readBucketOptions struct {
	symlinksIfSupported bool
}
