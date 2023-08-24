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

package buflintvalidate

import (
	"github.com/bufbuild/buf/private/pkg/protosource"
)

// validateModule is a validate module.
type validateModule struct {
	add   func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{})
	files []protosource.File
}

// NewValidateField newValidateField returns a new validate validateField.
func (m *validateModule) NewValidateField(field protosource.Field) *validateField {
	return newValidateField(m, field)
}
