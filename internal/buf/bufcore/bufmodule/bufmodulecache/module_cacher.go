// Copyright 2020 Buf Technologies, Inc.
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

package bufmodulecache

import (
	"context"

	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulestorage"
	"go.uber.org/multierr"
)

type moduleCacher struct {
	moduleStore bufmodulestorage.Store
}

func newModuleCacher(
	moduleStore bufmodulestorage.Store,
) *moduleCacher {
	return &moduleCacher{
		moduleStore: moduleStore,
	}
}

func (m *moduleCacher) GetModule(
	ctx context.Context,
	resolvedModuleName bufmodule.ResolvedModuleName,
) (bufmodule.Module, error) {
	path := getModuleKey(resolvedModuleName)
	module, err := m.moduleStore.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := bufmodule.ValidateModuleDigest(ctx, resolvedModuleName, module); err != nil {
		// Delete module if it's invalid
		deleteErr := m.moduleStore.Delete(ctx, path)
		if deleteErr != nil {
			err = multierr.Append(err, deleteErr)
		}
		return nil, err
	}
	return module, nil
}

func (m *moduleCacher) PutModule(
	ctx context.Context,
	moduleName bufmodule.ModuleName,
	module bufmodule.Module,
) (bufmodule.ModuleName, error) {
	if moduleName.Digest() == "" {
		return nil, bufmodule.NewNoDigestError(moduleName)
	}
	if err := m.moduleStore.Put(ctx, getModuleKey(moduleName), module); err != nil {
		return nil, err
	}
	return moduleName, nil
}

// this assumes the ModuleName is resolved
func getModuleKey(moduleName bufmodule.ModuleName) bufmodulestorage.Key {
	return bufmodulestorage.Key{
		moduleName.Remote(),
		moduleName.Owner(),
		moduleName.Repository(),
		moduleName.Version(),
		moduleName.Digest(),
	}
}
