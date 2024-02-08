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

import "github.com/gofrs/uuid/v5"

// RegistryCommitID is the pair of a Commit ID with the registry the commit belongs to.
//
// We need this to be a public, comparable type, as we use it in the GraphProvider.
type RegistryCommitID struct {
	Registry string
	CommitID uuid.UUID
}

// NewRegistryCommitID returns a new RegistryCommitID.
func NewRegistryCommitID(registry string, commitID uuid.UUID) RegistryCommitID {
	return RegistryCommitID{
		Registry: registry,
		CommitID: commitID,
	}
}

// ModuleKeyToRegistryCommitID converts the ModuleKey to a RegistryCommitID.
func ModuleKeyToRegistryCommitID(moduleKey ModuleKey) RegistryCommitID {
	return NewRegistryCommitID(
		moduleKey.ModuleFullName().Registry(),
		moduleKey.CommitID(),
	)
}
