// Copyright 2020-2023 Buf Technologies, Inc.
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
	// NopModuleDataProvider is a no-op ModuleDataProvider.
	NopModuleDataProvider ModuleDataProvider = nopModuleDataProvider{}
)

// ModuleDataProvider provides ModulesDatas.
type ModuleDataProvider interface {
	// GetModuleDatasForModuleKeys gets the ModuleDatas for the ModuleKeys.
	//
	// The input ModuleKeys are expected to be unique by ModuleFullName. The implementation
	// may error if this is not the case.
	//
	// Returned ModuleDatas will be unique by ModuleFullName and sorted by ModuleFullName.
	// If any ModuleKey is not found, an error with fs.ErrNotExist will be returned.
	GetModuleDatasForModuleKeys(
		context.Context,
		[]ModuleKey,
	) ([]ModuleData, error)
}

// *** PRIVATE ***

type nopModuleDataProvider struct{}

func (nopModuleDataProvider) GetModuleDatasForModuleKeys(
	context.Context,
	[]ModuleKey,
) ([]ModuleData, error) {
	return nil, fs.ErrNotExist
}
