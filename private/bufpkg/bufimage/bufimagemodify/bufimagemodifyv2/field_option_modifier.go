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

package bufimagemodifyv2

import (
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufimage/bufimagemodify/internal"
	"github.com/bufbuild/protocompile/walk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type fieldOptionModifier struct {
	marker                Marker
	imageFile             bufimage.ImageFile
	fieldNameToDescriptor map[string]*descriptorpb.FieldDescriptorProto
	fieldNameToSourcepath map[string][]int32
}

func newFieldOptionModifier(
	imageFile bufimage.ImageFile,
	marker Marker,
) (*fieldOptionModifier, error) {
	fieldNameToDescriptor := make(map[string]*descriptorpb.FieldDescriptorProto)
	fieldNameToSourcePath := make(map[string][]int32)
	err := walk.DescriptorProtosWithPath(
		imageFile.Proto(),
		func(
			fullName protoreflect.FullName,
			path protoreflect.SourcePath,
			message proto.Message,
		) error {
			fieldDescriptor, ok := message.(*descriptorpb.FieldDescriptorProto)
			if !ok {
				return nil
			}
			fieldNameToDescriptor[string(fullName)] = fieldDescriptor
			fieldNameToSourcePath[string(fullName)] = path
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &fieldOptionModifier{
		marker:                marker,
		imageFile:             imageFile,
		fieldNameToDescriptor: fieldNameToDescriptor,
		fieldNameToSourcepath: fieldNameToSourcePath,
	}, nil
}

func (m *fieldOptionModifier) FieldNames() []string {
	fieldNames := make([]string, 0, len(m.fieldNameToDescriptor))
	for fieldName := range m.fieldNameToDescriptor {
		fieldNames = append(fieldNames, fieldName)
	}
	return fieldNames
}

func (m *fieldOptionModifier) ModifyJSType(
	fieldName string,
	override Override,
) error {
	if internal.IsWellKnownType(m.imageFile) {
		return nil
	}
	jsTypeOverride, ok := override.(valueOverride[descriptorpb.FieldOptions_JSType])
	if !ok {
		return fmt.Errorf("unknown Override type: %T", override)
	}
	jsType := jsTypeOverride.get()
	fieldDescriptor, ok := m.fieldNameToDescriptor[fieldName]
	if !ok {
		return fmt.Errorf("could not find field %s in %s", fieldName, m.imageFile.Path())
	}
	if fieldDescriptor.Type == nil || !isJsTypePermittedForType(*fieldDescriptor.Type) {
		return nil
	}
	if fieldDescriptor.Options == nil {
		fieldDescriptor.Options = &descriptorpb.FieldOptions{}
	}
	fieldDescriptor.Options.Jstype = &jsType
	if fieldSourcePath, ok := m.fieldNameToSourcepath[fieldName]; ok {
		if len(fieldSourcePath) > 0 {
			jsTypeOptionPath := append(fieldSourcePath, internal.JSTypePackageSuffix...)
			m.marker.Mark(m.imageFile, jsTypeOptionPath)
		}
	}
	return nil
}

func isJsTypePermittedForType(typ descriptorpb.FieldDescriptorProto_Type) bool {
	return typ == descriptorpb.FieldDescriptorProto_TYPE_INT64 ||
		typ == descriptorpb.FieldDescriptorProto_TYPE_UINT64 ||
		typ == descriptorpb.FieldDescriptorProto_TYPE_SINT64 ||
		typ == descriptorpb.FieldDescriptorProto_TYPE_FIXED64 ||
		typ == descriptorpb.FieldDescriptorProto_TYPE_SFIXED64
}
