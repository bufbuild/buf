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
	// GetModuleDatasForModuleKeys gets the ModuleDatas for the ModuleKeys, and optionally
	// the dependencies of the ModuleKeys.
	//
	// Returned  ModuleDatas will be unique by ModuleFullName and sorted by ModuleFullName.
	// If any ModuleKey is not found, an error with fs.ErrNotExist will be returned.
	GetModuleDatasForModuleKeys(context.Context, []ModuleKey) ([]ModuleData, error)
}

// GetModuleDatasForModuleKeysOption is an option for GetModuleDatasForModuleKeys.
type GetModuleDatasForModuleKeysOption func(*getModuleDatasForModuleKeysOptions)

// WithIncludeDepModuleDatas returns a new GetModuleDatasForModuleKeysOption that says
// to also return the dependency ModuleDatas.
func WithIncludeDepModuleDatas() GetModuleDatasForModuleKeysOption {
	return func(getModuleDatasForModuleKeysOptions *getModuleDatasForModuleKeysOptions) {
		getModuleDatasForModuleKeysOptions.includeDepModuleDatas = true
	}
}

// GetModuleDatasForModuleKeysOptions are parsed options for GetModuleDatasForModuleKeys.
//
// This will be used by implementations of ModuleDataProvider. Users of ModuleDataProvider
// do not need to be concerned with this type.
type GetModuleDatasForModuleKeysOptions interface {
	// IncludeDepModuleDatas says to also return the dependency ModuleDatas.
	IncludeDepModuleDatas() bool

	isGetModuleDatasForModuleKeysOptions()
}

// NewGetModuleDatasForModuleKeysOptions returns a new GetModuleDatasForModuleKeysOptions.
//
// This will be used by implementations of ModuleDataProvider. Users of ModuleDataProvider
// do not need to be concerned with this function.
func NewGetModuleDatasForModuleKeysOptions(
	options ...GetModuleDatasForModuleKeysOption,
) GetModuleDatasForModuleKeysOptions {
	getModuleDatasForModuleKeysOptions := newGetModulDatasForModuleKeysOptions()
	for _, option := range options {
		option(getModuleDatasForModuleKeysOptions)
	}
	return getModuleDatasForModuleKeysOptions
}

// *** PRIVATE ***

type nopModuleDataProvider struct{}

func (nopModuleDataProvider) GetModuleDatasForModuleKeys(
	_ context.Context,
	_ []ModuleKey,
) ([]ModuleData, error) {
	return nil, fs.ErrNotExist
}

type getModuleDatasForModuleKeysOptions struct {
	includeDepModuleDatas bool
}

func newGetModulDatasForModuleKeysOptions() *getModuleDatasForModuleKeysOptions {
	return &getModuleDatasForModuleKeysOptions{}
}

func (g *getModuleDatasForModuleKeysOptions) IncludeDepModuleDatas() bool {
	return g.includeDepModuleDatas
}

func (*getModuleDatasForModuleKeysOptions) isGetModuleDatasForModuleKeysOptions() {}
