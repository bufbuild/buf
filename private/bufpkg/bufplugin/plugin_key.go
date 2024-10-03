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

package bufplugin

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/google/uuid"
)

// PluginKey provides identifying information for a Plugin.
//
// TODO(emcfarlane)
type PluginKey interface {
	// String returns "registry/owner/name:dashlessCommitID".
	fmt.Stringer

	// PluginFullName returns the full name of the Plugin.
	//
	// Always present.
	PluginFullName() PluginFullName
	// CommitID returns the ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// Always present, that is CommitID() == uuid.Nil will always be false.
	CommitID() uuid.UUID
	// Digest returns the Plugin digest.
	//
	// TODO(emcfarlane)
	Digest() (Digest, error)

	isPluginKey()
}

func NewPluginKey(
	pluginFullName PluginFullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (PluginKey, error) {
	return newPluginKey(
		pluginFullName,
		commitID,
		getDigest,
	)
}

// ** PRIVATE **

type pluginKey struct {
	pluginFullName PluginFullName
	commitID       uuid.UUID

	getDigest func() (Digest, error)
}

func newPluginKey(
	pluginFullName PluginFullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (*pluginKey, error) {
	if pluginFullName == nil {
		return nil, errors.New("nil PluginFullName when constructing PluginKey")
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

func (p *pluginKey) PluginFullName() PluginFullName {
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
