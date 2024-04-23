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

	"github.com/bufbuild/buf/private/pkg/dag"
)

var (
	// NopGraphProvider is a no-op GraphProvider.
	NopGraphProvider GraphProvider = nopGraphProvider{}
)

// GraphProvider provides directed acyclic graphs for ModuleKeys.
type GraphProvider interface {
	// GetGraphForModuleKeys gets the Graph for the given ModuleKeys.
	//
	// The key will be the ModuleKey.CommitID().
	//
	// The input ModuleKeys are expected to be unique by ModuleFullName. The implementation
	// may error if this is not the case.
	//
	// The input ModuleKeys are expected to have the same DigestType. The implementation
	// may error if this is not the case.
	//
	// If any ModuleKey is not found, an error with fs.ErrNotExist will be returned.
	GetGraphForModuleKeys(context.Context, []ModuleKey) (*dag.Graph[RegistryCommitID, ModuleKey], error)
}

// *** PRIVATE ***

type nopGraphProvider struct{}

func (nopGraphProvider) GetGraphForModuleKeys(
	context.Context,
	[]ModuleKey,
) (*dag.Graph[RegistryCommitID, ModuleKey], error) {
	return nil, fs.ErrNotExist
}
