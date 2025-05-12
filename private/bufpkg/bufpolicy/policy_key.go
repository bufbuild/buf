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

package bufpolicy

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

// PolicyKey provides identifying information for a Policy.
//
// PolicyKeys are returned from PolicyKeyProviders, and represent a Policy's complete
// identity. They also match to what we store in buf.lock files. PolicyKeys can be used
// to get Policies via a PolicyProvider.
type PolicyKey interface {
	// String returns "registry/owner/name:dashlessCommitID".
	fmt.Stringer

	// FullName returns the full name of the Policy.
	//
	// Always present.
	FullName() bufparse.FullName
	// CommitID returns the ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// Always present, that is CommitID() == uuid.Nil will always be false.
	CommitID() uuid.UUID
	// Digest returns the Policy digest.
	Digest() (Digest, error)

	isPolicyKey()
}

// NewPolicyKey returns a new PolicyKey.
//
// The Digest will be loaded lazily if needed. Note this means that NewPolicyKey does
// *not* validate the digest. If you need to validate the digest, call Digest() and evaluate
// the returned error.
func NewPolicyKey(
	policyFullName bufparse.FullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (PolicyKey, error) {
	return newPolicyKey(
		policyFullName,
		commitID,
		getDigest,
	)
}

// UniqueDigestTypeForPolicyKeys returns the unique DigestType for the given PolicyKeys.
//
// If the PolicyKeys have different DigestTypes, an error is returned.
// If the PolicyKeys slice is empty, an error is returned.
func UniqueDigestTypeForPolicyKeys(policyKeys []PolicyKey) (DigestType, error) {
	if len(policyKeys) == 0 {
		return 0, syserror.New("empty policyKeys passed to UniqueDigestTypeForPolicyKeys")
	}
	digests, err := xslices.MapError(policyKeys, PolicyKey.Digest)
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
				strings.Join(xslices.Map(policyKeys, PolicyKey.String), "\n"),
			)
		}
	}
	return digestType, nil
}

// ** PRIVATE **

type policyKey struct {
	policyFullName bufparse.FullName
	commitID       uuid.UUID

	getDigest func() (Digest, error)
}

func newPolicyKey(
	policyFullName bufparse.FullName,
	commitID uuid.UUID,
	getDigest func() (Digest, error),
) (*policyKey, error) {
	if policyFullName == nil {
		return nil, errors.New("nil FullName when constructing PolicyKey")
	}
	if commitID == uuid.Nil {
		return nil, errors.New("empty commitID when constructing PolicyKey")
	}
	return &policyKey{
		policyFullName: policyFullName,
		commitID:       commitID,
		getDigest:      sync.OnceValues(getDigest),
	}, nil
}

func (p *policyKey) FullName() bufparse.FullName {
	return p.policyFullName
}

func (p *policyKey) CommitID() uuid.UUID {
	return p.commitID
}

func (p *policyKey) Digest() (Digest, error) {
	return p.getDigest()
}

func (p *policyKey) String() string {
	return p.policyFullName.String() + ":" + uuidutil.ToDashless(p.commitID)
}

func (*policyKey) isPolicyKey() {}
