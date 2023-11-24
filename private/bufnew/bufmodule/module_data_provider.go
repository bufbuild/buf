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
	"errors"
)

var (
	// NopModuleDataProvider is a no-op ModuleDataProvider.
	NopModuleDataProvider ModuleDataProvider = nopModuleDataProvider{}
)

// ModuleDataProvider provides ModulesDatas.
type ModuleDataProvider interface {
	// GetModuleDataForModuleKey gets the ModuleDatas for the ModuleKeys.
	//
	// If there is no error, the length of the ModuleDatas returned will match the length of the ModuleKeys.
	// If there is an error, no ModuleDatas will be returned.
	// An error with fs.ErrNotExist will be returned if a ModuleKey is not found.
	//
	// If the input ModuleKey had a CommitID set, this the returned ModuleData will also have a CommitID
	// set. This is important for i.e. v1beta1 and v1 buf.lock files, where we want to make sure we keep
	// the reference to the CommitID, even if we did not have it stored in our cache.
	GetModuleDatasForModuleKeys(context.Context, ...ModuleKey) ([]ModuleData, error)
}

// nopModuleDataProvider

type nopModuleDataProvider struct{}

func (nopModuleDataProvider) GetModuleDatasForModuleKeys(context.Context, ...ModuleKey) ([]ModuleData, error) {
	return nil, errors.New("nopModuleDataProvider")
}
