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

package buflsp

import (
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/syncext"
)

// moduleSetResolver lazily resolves a module set, when possible.
type moduleSetResolver interface {
	ModuleSet() (bufmodule.ModuleSet, error)
	Bucket() (bufmodule.ModuleReadBucket, error)
}

// newModuleSetResolver returns a new module set resolver.
func newModuleSetResolver(getModuleSet func() (bufmodule.ModuleSet, error)) moduleSetResolver {
	getModuleSet = syncext.OnceValues(getModuleSet)
	getBucket := syncext.OnceValues(func() (bufmodule.ModuleReadBucket, error) {
		moduleSet, err := getModuleSet()
		if err != nil {
			return nil, err
		}
		return bufmodule.ModuleSetToModuleReadBucketWithOnlyProtoFiles(moduleSet), nil
	})
	return &onceModuleSetResolver{
		getModuleSet: getModuleSet,
		getBucket:    getBucket,
	}
}

// onceModuleSetResolver is a moduleSetResolver that wraps the getters in sync.OnceValue to memoize
// the results.
type onceModuleSetResolver struct {
	getModuleSet func() (bufmodule.ModuleSet, error)
	getBucket    func() (bufmodule.ModuleReadBucket, error)
}

func (r *onceModuleSetResolver) ModuleSet() (bufmodule.ModuleSet, error) {
	return r.getModuleSet()
}

func (r *onceModuleSetResolver) Bucket() (bufmodule.ModuleReadBucket, error) {
	return r.getBucket()
}
