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

package bufmodule

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
	"github.com/gofrs/uuid/v5"
)

// CommitKey provides identifying information for a Commit when calling the CommitProvider.
type CommitKey interface {
	// String returns "digestType/registry:dashlessCommitID".
	fmt.Stringer

	// Registry returns the registry of the Commit.
	//
	// Always present.
	Registry() string
	// CommitID returns the ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// Always present, that is CommitID().IsNil() will always be false.
	CommitID() uuid.UUID
	// DigestType returns the DigestType of the Commit.
	//
	// Note this is *not* a bufcas.Digest - this is a Digest. bufcas.Digests are a lower-level
	// type that just deal in terms of files and content. A ModuleDigest is a specific algorithm
	// applied to a set of files and dependencies.
	DigestType() DigestType

	isCommitKey()
}

// NewCommitKey returns a new CommitKey.
func NewCommitKey(
	registry string,
	commitID uuid.UUID,
	digestType DigestType,
) (CommitKey, error) {
	return newCommitKey(
		registry,
		commitID,
		digestType,
	)
}

// UniqueDigestTypeForCommitKeys returns the single DigestType for the Digests on the CommitKeys.
//
// If the CommitKeys have different DigestTypes, an error is returned.
// If the CommitKeys slice is empty, an error is returned.
func UniqueDigestTypeForCommitKeys(commitKeys []CommitKey) (DigestType, error) {
	if len(commitKeys) == 0 {
		return 0, syserror.New("empty commitKeys passed to UniqueDigestTypeForCommitKeys")
	}
	digestTypes := slicesext.Map(commitKeys, CommitKey.DigestType)
	digestType := digestTypes[0]
	for _, otherDigestType := range digestTypes[1:] {
		if otherDigestType != digestType {
			return 0, fmt.Errorf(
				"different digest types detected where the same digest type must be used: %v, %v\n%s",
				otherDigestType,
				digestType,
				strings.Join(slicesext.Map(commitKeys, CommitKey.String), "\n"),
			)
		}
	}
	return digestType, nil
}

// *** PRIVATE ***

type commitKey struct {
	registry   string
	commitID   uuid.UUID
	digestType DigestType
}

func newCommitKey(
	registry string,
	commitID uuid.UUID,
	digestType DigestType,
) (*commitKey, error) {
	if registry == "" {
		return nil, errors.New("empty registry when constructing CommitKey")
	}
	if commitID.IsNil() {
		return nil, errors.New("empty commitID when constructing CommitKey")
	}
	if _, ok := digestTypeToString[digestType]; !ok {
		return nil, fmt.Errorf("unknown DigestType when constructing CommitKey: %v", digestType)
	}
	return &commitKey{
		registry:   registry,
		commitID:   commitID,
		digestType: digestType,
	}, nil
}

func (m *commitKey) Registry() string {
	return m.registry
}

func (m *commitKey) CommitID() uuid.UUID {
	return m.commitID
}

func (m *commitKey) DigestType() DigestType {
	return m.digestType
}

func (m *commitKey) String() string {
	return m.digestType.String() + "/" + m.registry + ":" + uuidutil.ToDashless(m.commitID)
}

func (*commitKey) isCommitKey() {}
