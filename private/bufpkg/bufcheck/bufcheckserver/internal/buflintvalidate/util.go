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

package buflintvalidate

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type constraintsResolverForTargetField struct {
	protovalidate.StandardConstraintResolver
	targetField protoreflect.FieldDescriptor
}

func (r *constraintsResolverForTargetField) ResolveFieldConstraints(desc protoreflect.FieldDescriptor) *validate.FieldConstraints {
	if desc.FullName() != r.targetField.FullName() {
		return nil
	}
	return r.StandardConstraintResolver.ResolveFieldConstraints(desc)
}

// TODO: this function is copied directly from protovalidate-go.
// We have 3 options:
// 1. Go to protovalidate-go and add DefaultResolver.ResolveSharedFieldConstraints.
// 2. Go to protovalidate-go and make this public.
// 3. Leave it as is.
func resolveExt[C proto.Message](
	options proto.Message,
	extType protoreflect.ExtensionType,
) (constraints C) {
	num := extType.TypeDescriptor().Number()
	var msg proto.Message

	proto.RangeExtensions(options, func(typ protoreflect.ExtensionType, i interface{}) bool {
		if num != typ.TypeDescriptor().Number() {
			return true
		}
		msg, _ = i.(proto.Message)
		return false
	})

	if msg == nil {
		return constraints
	} else if m, ok := msg.(C); ok {
		return m
	}

	constraints, _ = constraints.ProtoReflect().New().Interface().(C)
	b, _ := proto.Marshal(msg)
	_ = proto.Unmarshal(b, constraints)
	return constraints
}

// TODO: this is copied from protovalidate-go, with the difference that types is passed as a parameter.
func reparseUnrecognized(
	reflectMessage protoreflect.Message,
	extensionTypeResolver ExtensionTypeResolver,
) error {
	unknown := reflectMessage.GetUnknown()
	if len(unknown) > 0 {
		reflectMessage.SetUnknown(nil)
		options := proto.UnmarshalOptions{
			Resolver: extensionTypeResolver,
			Merge:    true,
		}
		if err := options.Unmarshal(unknown, reflectMessage.Interface()); err != nil {
			return err
		}
	}
	return nil
}
