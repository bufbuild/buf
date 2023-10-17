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
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protodesc"
)

// ValidateRules validates that protovalidate rules defined for this field are
// are valid, not including CEL expressions.
func ValidateRules(
	descritporResolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	files []protosource.File,
	field protosource.Field,
) error {
	fieldDescriptor, err := getReflectFieldDescriptor(descritporResolver, field)
	if err != nil {
		return err
	}
	constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldDescriptor)
	newValidateField(
		add,
		files,
		field,
	).CheckConstraintsForField(constraints, field)
	return nil
}

// ValidateCELCompiles validates that all CEL expressions defined for protovalidate
// in the given file compile.
func ValidateCELCompiles(
	resolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	file protosource.File,
) error {
	for _, message := range file.Messages() {
		if err := validateCELCompilesMessage(resolver, add, message); err != nil {
			return err
		}
	}
	for _, extensionField := range file.Extensions() {
		if err := validateCELCompilesField(resolver, add, extensionField); err != nil {
			return err
		}
	}
	return nil
}
