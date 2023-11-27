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

// OptionalModuleKey is a result from a ModuleKeyProvider.
//
// It returns whether or not the ModuleKey was found, and a non-nil
// ModuleKey if the ModuleKey was found.
type OptionalModuleKey interface {
	ModuleKey() ModuleKey
	Found() bool

	isOptionalModuleKey()
}

// NewOptionalModuleKey returns a new OptionalModuleKey.
//
// As opposed to most functions in this codebase, the input ModuleKey can be nil.
// If it is nil, then Found() will return false.
func NewOptionalModuleKey(moduleKey ModuleKey) OptionalModuleKey {
	return newOptionalModuleKey(moduleKey)
}

// *** PRIVATE ***

type optionalModuleKey struct {
	moduleKey ModuleKey
}

func newOptionalModuleKey(moduleKey ModuleKey) *optionalModuleKey {
	return &optionalModuleKey{
		moduleKey: moduleKey,
	}
}

func (o *optionalModuleKey) ModuleKey() ModuleKey {
	return o.moduleKey
}

func (o *optionalModuleKey) Found() bool {
	return o.moduleKey != nil
}

func (*optionalModuleKey) isOptionalModuleKey() {}
