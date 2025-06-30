// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufplugin

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// PluginKey provides identifying information for a Plugin.
//
// PluginKeys are returned from PluginKeyProviders, and represent a Plugin's complete
// identity. They also match to what we store in buf.lock files. PluginKeys can be used
// to get Plugins via a PluginProvider.
type PluginKey interface {
	// String returns "registry/owner/name:dashlessCommitID".
	fmt.Stringer

	// FullName returns the full name of the Plugin.
	//
	// Always present.
	FullName() bufparse.FullName
	// CommitID returns the ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// Always present, that is CommitID() == uuid.Nil will always be false.
	CommitID() uuid.UUID
	// Digest returns the Plugin digest.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest.
	// bufcas.Digests are a lower-level type that just deal in terms of
	// files and content. A PluginDigest is a specific algorithm applied to
	// the Plugin data.
	Digest() (Digest, error)

	isPluginKey()
}

// NewPluginKey returns a new PluginKey.
//
// The Digest will be loaded lazily if needed. Note this means that NewPluginKey does
// *not* validate the digest. If you need to validate the digest, call Digest() and evaluate
// the returned error.
func NewPluginKey(
	pluginFullName bufparse.FullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (PluginKey, error) {
	return newPluginKey(
		pluginFullName,
		commitID,
		getDigest,
	)
}

// UniqueDigestTypeForPluginKeys returns the unique DigestType for the given PluginKeys.
//
// If the PluginKeys have different DigestTypes, an error is returned.
// If the PluginKeys slice is empty, an error is returned.
func UniqueDigestTypeForPluginKeys(pluginKeys []PluginKey) (DigestType, error) {
	if len(pluginKeys) == 0 {
		return 0, syserror.New("empty pluginKeys passed to UniqueDigestTypeForPluginKeys")
	}
	digests, err := xslices.MapError(pluginKeys, PluginKey.Digest)
	if err != nil {
		return 0, err
	}
	digestType := digests[0].Type()
	for _, digest := range digests[1:] {
		if digestType != digest.Type() {
			return 0, fmt.Errorf(
				"different digest types detected where the same digest type must be used: %v, %v\n%s",
				digestType,
				digest.Type(),
				strings.Join(xslices.Map(pluginKeys, PluginKey.String), "\n"),
			)
		}
	}
	return digestType, nil
}

// ** PRIVATE **

type pluginKey struct {
	pluginFullName bufparse.FullName
	commitID       uuid.UUID

	getDigest func() (Digest, error)
}

func newPluginKey(
	pluginFullName bufparse.FullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (*pluginKey, error) {
	if pluginFullName == nil {
		return nil, errors.New("nil FullName when constructing PluginKey")
	}
	if commitID == uuid.Nil {
		return nil, errors.New("empty commitID when constructing PluginKey")
	}
	return &pluginKey{
		pluginFullName: pluginFullName,
		commitID:       commitID,
		getDigest:      sync.OnceValues(getDigest),
	}, nil
}

func (p *pluginKey) FullName() bufparse.FullName {
	return p.pluginFullName
}

func (p *pluginKey) CommitID() uuid.UUID {
	return p.commitID
}

func (p *pluginKey) Digest() (Digest, error) {
	return p.getDigest()
}

func (p *pluginKey) String() string {
	return p.pluginFullName.String() + ":" + uuidutil.ToDashless(p.commitID)
}

func (*pluginKey) isPluginKey() {}
