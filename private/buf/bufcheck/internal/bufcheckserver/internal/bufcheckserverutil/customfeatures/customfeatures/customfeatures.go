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

package customfeatures

import (
	"fmt"

	"github.com/bufbuild/buf/private/gen/proto/go/google/protobuf"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ResolveCppFeature returns a value for the given field name of the (pb.cpp) custom feature
// for the given field.
func ResolveCppFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	return resolveFeature(field, protobuf.E_Cpp.TypeDescriptor(), fieldName, expectedKind)
}

// ResolveJavaFeature returns a value for the given field name of the (pb.java) custom feature
// for the given field.
func ResolveJavaFeature(field protoreflect.FieldDescriptor, fieldName protoreflect.Name, expectedKind protoreflect.Kind) (protoreflect.Value, error) {
	return resolveFeature(field, protobuf.E_Java.TypeDescriptor(), fieldName, expectedKind)
}

func resolveFeature(
	field protoreflect.FieldDescriptor,
	extension protoreflect.ExtensionTypeDescriptor,
	fieldName protoreflect.Name,
	expectedKind protoreflect.Kind,
) (protoreflect.Value, error) {
	featureField := extension.Message().Fields().ByName(fieldName)
	if featureField == nil {
		return protoreflect.Value{}, fmt.Errorf("unable to resolve field descriptor for %s.%s", extension.Message().FullName(), fieldName)
	}
	if featureField.Kind() != expectedKind || featureField.IsList() {
		return protoreflect.Value{}, fmt.Errorf("resolved field descriptor for %s.%s has unexpected type: expected optional %s, got %s %s",
			extension.Message().FullName(), fieldName, expectedKind, featureField.Cardinality(), featureField.Kind())
	}
	return protoutil.ResolveCustomFeature(
		field,
		extension.Type(),
		featureField,
	)
}
