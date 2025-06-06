// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufprotosource

type service struct {
	namedDescriptor
	optionExtensionDescriptor

	methods    []Method
	deprecated bool
}

func newService(
	namedDescriptor namedDescriptor,
	optionExtensionDescriptor optionExtensionDescriptor,
	deprecated bool,
) *service {
	return &service{
		namedDescriptor:           namedDescriptor,
		optionExtensionDescriptor: optionExtensionDescriptor,
		deprecated:                deprecated,
	}
}

func (m *service) Methods() []Method {
	return m.methods
}

func (m *service) addMethod(method Method) {
	m.methods = append(m.methods, method)
}

func (m *service) Deprecated() bool {
	return m.deprecated
}
