// Copyright 2020-2026 Buf Technologies, Inc.
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
	"fmt"
	"io/fs"

	"buf.build/go/standard/xslices"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
)

// ModuleKeyProvider provides ModuleKeys for ModuleRefs.
type ModuleKeyProvider interface {
	// GetModuleKeysForModuleRefs gets the ModuleKeys for the given ModuleRefs.
	//
	// Returned ModuleKeys will be in the same order as the input ModuleRefs.
	//
	// The input ModuleRefs are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the ModuleKeys returned will match the length of the ModuleRefs.
	// If there is an error, no ModuleKeys will be returned.
	// If any ModuleRef is not found, an error with fs.ErrNotExist will be returned.
	GetModuleKeysForModuleRefs(context.Context, []bufparse.Ref, DigestType) ([]ModuleKey, error)
}

// NewStaticModuleKeyProvider returns a new ModuleKeyProvider for a static set of ModuleKeys.
//
// The set of ModuleKeys must be unique by FullName. If there are duplicates,
// an error will be returned.
//
// When resolving Refs, the Ref will be matched to the ModuleKey by FullName.
// If the Ref is not found in the set of provided keys, an fs.ErrNotExist will be returned.
func NewStaticModuleKeyProvider(moduleKeys []ModuleKey) (ModuleKeyProvider, error) {
	return newStaticModuleKeyProvider(moduleKeys)
}

// *** PRIVATE ***

type staticModuleKeyProvider struct {
	moduleKeysByFullName map[string]ModuleKey
}

func newStaticModuleKeyProvider(moduleKeys []ModuleKey) (*staticModuleKeyProvider, error) {
	var moduleKeysByFullName map[string]ModuleKey
	if len(moduleKeys) > 0 {
		var err error
		moduleKeysByFullName, err = xslices.ToUniqueValuesMap(moduleKeys, func(moduleKey ModuleKey) string {
			return moduleKey.FullName().String()
		})
		if err != nil {
			return nil, err
		}
	}
	return &staticModuleKeyProvider{
		moduleKeysByFullName: moduleKeysByFullName,
	}, nil
}

func (s *staticModuleKeyProvider) GetModuleKeysForModuleRefs(
	_ context.Context,
	refs []bufparse.Ref,
	digestType DigestType,
) ([]ModuleKey, error) {
	moduleKeys := make([]ModuleKey, len(refs))
	for i, ref := range refs {
		// Only the FullName is used to match the ModuleKey. The Ref is not
		// validated to match the ModuleKey as there is not enough information
		// to do so.
		moduleKey, ok := s.moduleKeysByFullName[ref.FullName().String()]
		if !ok {
			return nil, fs.ErrNotExist
		}
		digest, err := moduleKey.Digest()
		if err != nil {
			return nil, err
		}
		if digest.Type() != digestType {
			return nil, fmt.Errorf("expected DigestType %v, got %v", digestType, digest.Type())
		}
		moduleKeys[i] = moduleKey
	}
	return moduleKeys, nil
}
