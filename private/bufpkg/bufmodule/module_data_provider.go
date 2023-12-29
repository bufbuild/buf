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

	"github.com/bufbuild/buf/private/pkg/slicesext"
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
	// If a ModuleData is not found, the OptionalModuleData will have Found() equal to false, otherwise
	// the OptionalModuleData will have Found() equal to true with non-nil ModuleData.
	GetOptionalModuleDatasForModuleKeys(context.Context, ...ModuleKey) ([]OptionalModuleData, error)
}

// GetModuleDatasForModuleKeys calls GetOptionalModuleDatasForModuleKeys, returning an error
// with fs.ErrNotExist if any ModuleData is not found.
func GetModuleDatasForModuleKeys(
	ctx context.Context,
	moduleDataProvider ModuleDataProvider,
	moduleKeys ...ModuleKey,
) ([]ModuleData, error) {
	optionalModuleDatas, err := moduleDataProvider.GetOptionalModuleDatasForModuleKeys(ctx, moduleKeys...)
	if err != nil {
		return nil, err
	}
	moduleDatas := make([]ModuleData, len(optionalModuleDatas))
	for i, optionalModuleData := range optionalModuleDatas {
		if !optionalModuleData.Found() {
			return nil, &fs.PathError{Op: "read", Path: moduleKeys[i].String(), Err: fs.ErrNotExist}
		}
		moduleDatas[i] = optionalModuleData.ModuleData()
	}
	return moduleDatas, nil
}

// nopModuleDataProvider

type nopModuleDataProvider struct{}

func (nopModuleDataProvider) GetOptionalModuleDatasForModuleKeys(
	_ context.Context,
	moduleKeys ...ModuleKey,
) ([]OptionalModuleData, error) {
	return slicesext.Map(
		moduleKeys,
		func(ModuleKey) OptionalModuleData {
			return NewOptionalModuleData(nil)
		},
	), nil
}
