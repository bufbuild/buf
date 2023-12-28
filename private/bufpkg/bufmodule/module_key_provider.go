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

// ModuleKeyProvider provides ModuleKeys for ModuleRefs.
type ModuleKeyProvider interface {
	// GetModuleKeysForModuleRefs gets the ModuleKeys for the given ModuleRefs.
	//
	// Resolution of the ModuleRefs is done per the ModuleRef documentation.
	//
	// If there is no error, the length of the OptionalModuleKeys returned will match the length of the ModuleRefs.
	// If there is an error, no OptionalModuleKeys will be returned.
	// If a ModuleKey is not found, the OptionalModuleKey will have Found() equal to false, otherwise
	// the OptionalModuleKey will have Found() equal to true with non-nil ModuleKey.
	GetOptionalModuleKeysForModuleRefs(context.Context, ...ModuleRef) ([]OptionalModuleKey, error)
}

// GetModuleKeysForModuleRefs calls GetOptionalModuleKeysForModuleRefs, returning an error
// with fs.ErrNotExist if any ModuleRef is not found.
func GetModuleKeysForModuleRefs(
	ctx context.Context,
	moduleKeyProvider ModuleKeyProvider,
	moduleRefs ...ModuleRef,
) ([]ModuleKey, error) {
	optionalModuleKeys, err := moduleKeyProvider.GetOptionalModuleKeysForModuleRefs(ctx, moduleRefs...)
	if err != nil {
		return nil, err
	}
	moduleKeys := make([]ModuleKey, len(optionalModuleKeys))
	for i, optionalModuleKey := range optionalModuleKeys {
		if !optionalModuleKey.Found() {
			return nil, &fs.PathError{Op: "read", Path: moduleRefs[i].String(), Err: fs.ErrNotExist}
		}
		moduleKeys[i] = optionalModuleKey.ModuleKey()
	}
	return moduleKeys, nil
}
