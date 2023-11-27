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

// OptionalModuleData is a result from a ModuleDataProvider.
//
// It returns whether or not the ModuleData was found, and a non-nil
// ModuleData if the ModuleData was found.
type OptionalModuleData interface {
	ModuleData() ModuleData
	Found() bool

	isOptionalModuleData()
}

// NewOptionalModuleData returns a new OptionalModuleData.
//
// As opposed to most functions in this codebase, the input ModuleData can be nil.
// If it is nil, then Found() will return false.
func NewOptionalModuleData(moduleData ModuleData) OptionalModuleData {
	return newOptionalModuleData(moduleData)
}

// *** PRIVATE ***

type optionalModuleData struct {
	moduleData ModuleData
}

func newOptionalModuleData(moduleData ModuleData) *optionalModuleData {
	return &optionalModuleData{
		moduleData: moduleData,
	}
}

func (o *optionalModuleData) ModuleData() ModuleData {
	return o.moduleData
}

func (o *optionalModuleData) Found() bool {
	return o.moduleData != nil
}

func (*optionalModuleData) isOptionalModuleData() {}
