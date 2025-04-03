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

import "github.com/bufbuild/buf/private/bufpkg/bufparse"

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
	// FullName returns the full name of the Policy.
	//
	// May be nil. Callers should not rely on this value being present.
	// However, this is always present for remote Policies.
	//
	// Use OpaqueID as an always-present identifier.
	FullName() bufparse.FullName
	// Digest returns the Policy digest for the given DigestType.
	Digest(DigestType) (Digest, error)
	// Data returns the bytes of the Policy in yaml.
	Data() []byte

	isPolicy()
}
