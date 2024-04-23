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
	"context"
	"io/fs"
)

var (
	// NopCommitProvider is a no-op CommitProvider.
	NopCommitProvider CommitProvider = nopCommitProvider{}
)

// CommitProvider provides Commits for ModuleKeys or CommitIDs.
//
// Use GetCommitForModuleKeys if you already have the ModuleKey, as it should perform better
// for most implementations, and the ModuleKey Digest will be checked against the retrieved
// Commit Digest.
//
// The CommitProvider relies on the fact that we don't allow renames for Modules at this point
// in the BSR. If this were to change, our underlying cache would have to change. Keep this
// in mind, however so much would have to change in the BSR to allow renames that the local
// CLI cache is the least of our problems.
type CommitProvider interface {
	// GetCommitsForModuleKeys gets the Commits for the given ModuleKeys.
	//
	// Returned Commits will be in the same order as the input ModuleKeys.
	//
	// The input ModuleKeys are expected to have the same DigestType. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the Commits returned will match the length of the ModuleKeys.
	// If there is an error, no Commits will be returned.
	// If any ModuleKey is not found, an error with fs.ErrNotExist will be returned.
	GetCommitsForModuleKeys(context.Context, []ModuleKey) ([]Commit, error)
	// GetCommitsForCommitKeys gets the Commits for the given CommitKeys.
	//
	// Returned Commits will be in the same order as the input CommitKeys.
	//
	// The input CommitKeys are expected to have the same DigestType. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the Commits returned will match the length of the CommitKeys.
	// If there is an error, no Commits will be returned.
	// If any CommitKey is not found, an error with fs.ErrNotExist will be returned.
	GetCommitsForCommitKeys(context.Context, []CommitKey) ([]Commit, error)
}

// *** PRIVATE ***

type nopCommitProvider struct{}

func (nopCommitProvider) GetCommitsForModuleKeys(
	context.Context,
	[]ModuleKey,
) ([]Commit, error) {
	return nil, fs.ErrNotExist
}

func (nopCommitProvider) GetCommitsForCommitKeys(
	context.Context,
	[]CommitKey,
) ([]Commit, error) {
	return nil, fs.ErrNotExist
}
