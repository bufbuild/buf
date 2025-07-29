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
	"sync"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/google/uuid"
)

// Policy presents a BSR policy.
type Policy interface {
	// OpaqueID returns an unstructured ID that can uniquely identify a Policy
	// relative to the Workspace.
	//
	// An OpaqueID's structure should not be relied upon, and is not a
	// globally-unique identifier. It's uniqueness property only applies to
	// the lifetime of the Policy, and only within the Workspace the Policy
	// is defined in.
	//
	// If two Policies have the same Name, they will have the same OpaqueID.
	OpaqueID() string
	// Name returns the name of the Policy.
	//  - For local Policies, this is the path to the policy yaml file.
	//  - For remote Policies, this is the FullName of the Policy in the form
	//    remote/owner/name.
	//
	// This is never empty.
	Name() string
	// FullName returns the full name of the Policy.
	//
	// May be nil. Callers should not rely on this value being present.
	// However, this is always present for remote Policies.
	//
	// Use OpaqueID as an always-present identifier.
	FullName() bufparse.FullName
	// CommitID returns the BSR ID of the Commit.
	//
	// It is up to the caller to convert this to a dashless ID when necessary.
	//
	// May be empty, that is CommitID() == uuid.Nil may be true.
	// Callers should not rely on this value being present.
	//
	// If FullName is nil, this will always be empty.
	CommitID() uuid.UUID
	// Description returns a human-readable description of the Policy.
	//
	// This is used to construct descriptive error messages pointing to configured policies.
	//
	// This will never be empty. If a description was not explicitly set, this falls back to
	// OpaqueID.
	Description() string
	// Digest returns the Policy digest for the given DigestType.
	Digest(DigestType) (Digest, error)
	// Config returns the PolicyConfig for the Policy.
	Config() (PolicyConfig, error)
	// IsLocal return true if the Policy is a local Policy.
	//
	// Policies are either local or remote.
	//
	// A local Policy is one which was contained in the local context.
	//
	// A remote Policy is one which was not contained in the local context,
	// and is a remote reference to a Policy.
	//
	// Remote Policies will always have FullNames.
	IsLocal() bool

	isPolicy()
}

// NewPolicy creates a new Policy.
func NewPolicy(
	description string,
	fullName bufparse.FullName,
	name string,
	commitID uuid.UUID,
	getConfig func() (PolicyConfig, error),
) (Policy, error) {
	return newPolicy(description, fullName, name, commitID, getConfig)
}

// *** PRIVATE ***

type policy struct {
	description           string
	fullName              bufparse.FullName
	name                  string
	commitID              uuid.UUID
	getConfig             func() (PolicyConfig, error)
	digestTypeToGetDigest map[DigestType]func() (Digest, error)
}

func newPolicy(
	description string,
	fullName bufparse.FullName,
	name string,
	commitID uuid.UUID,
	getConfig func() (PolicyConfig, error),
) (*policy, error) {
	if name == "" {
		return nil, syserror.New("name not present when constructing a Policy")
	}
	if fullName == nil && commitID != uuid.Nil {
		return nil, syserror.New("commitID present when constructing a local Policy")
	}
	policy := &policy{
		description: description,
		fullName:    fullName,
		name:        name,
		commitID:    commitID,
		getConfig:   sync.OnceValues(getConfig),
	}
	policy.digestTypeToGetDigest = newSyncOnceValueDigestTypeToGetDigestFuncForPolicy(policy)
	return policy, nil
}

func (p *policy) OpaqueID() string {
	return p.name
}

func (p *policy) Name() string {
	return p.name
}

func (p *policy) FullName() bufparse.FullName {
	return p.fullName
}

func (p *policy) CommitID() uuid.UUID {
	return p.commitID
}

func (p *policy) Description() string {
	if p.description != "" {
		return p.description
	}
	return p.OpaqueID()
}

func (p *policy) Config() (PolicyConfig, error) {
	return p.getConfig()
}

func (p *policy) Digest(digestType DigestType) (Digest, error) {
	getDigest, ok := p.digestTypeToGetDigest[digestType]
	if !ok {
		return nil, syserror.Newf("DigestType %v was not in policy.digestTypeToGetDigest", digestType)
	}
	return getDigest()
}

func (p *policy) IsLocal() bool {
	return p.commitID == uuid.Nil
}

func (p *policy) isPolicy() {}

func newSyncOnceValueDigestTypeToGetDigestFuncForPolicy(policy *policy) map[DigestType]func() (Digest, error) {
	m := make(map[DigestType]func() (Digest, error))
	for digestType := range digestTypeToString {
		m[digestType] = sync.OnceValues(newGetDigestFuncForPolicyAndDigestType(policy, digestType))
	}
	return m
}

func newGetDigestFuncForPolicyAndDigestType(policy *policy, digestType DigestType) func() (Digest, error) {
	return func() (Digest, error) {
		switch digestType {
		case DigestTypeO1:
			policyConfig, err := policy.getConfig()
			if err != nil {
				return nil, err
			}
			return getO1Digest(policyConfig)
		default:
			return nil, syserror.Newf("unknown DigestType: %v", digestType)
		}
	}
}
