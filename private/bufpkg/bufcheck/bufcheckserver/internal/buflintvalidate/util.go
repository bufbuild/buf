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
	"fmt"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/protovalidate-go"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This implements protovalidate.StandardConstraintResolver, see checkExampleValues' comment
// for why this is needed.
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

func (*constraintsResolverForTargetField) ResolveMessageConstraints(protoreflect.MessageDescriptor) *validate.MessageConstraints {
	return nil
}

func (*constraintsResolverForTargetField) ResolveOneofConstraints(protoreflect.OneofDescriptor) *validate.OneofConstraints {
	return nil
}

// This function is copied directly from protovalidate-go, except refactored to use protoencoding
// for marshalling and unmarshalling. We also added error handling for marshal/unmarshal.
//
// This resolves the given extension and returns the constraints for the extension.
func resolveExtension[C proto.Message](
	options proto.Message,
	extType protoreflect.ExtensionType,
	resolver protoencoding.Resolver,
) (constraints C, retErr error) {
	num := extType.TypeDescriptor().Number()
	var message proto.Message

	proto.RangeExtensions(options, func(typ protoreflect.ExtensionType, i interface{}) bool {
		if num != typ.TypeDescriptor().Number() {
			return true
		}
		message, _ = i.(proto.Message)
		return false
	})

	if message == nil {
		return constraints, nil
	} else if m, ok := message.(C); ok {
		return m, nil
	}
	var ok bool
	constraints, ok = constraints.ProtoReflect().New().Interface().(C)
	if !ok {
		return constraints, fmt.Errorf("unexpected type for constraints %T", constraints)
	}
	b, err := protoencoding.NewWireMarshaler().Marshal(message)
	if err != nil {
		return constraints, err
	}
	return constraints, protoencoding.NewWireUnmarshaler(resolver).Unmarshal(b, constraints)
}

func celTypeForStandardRuleMessageDescriptor(
	ruleMessageDescriptor protoreflect.MessageDescriptor,
) *cel.Type {
	switch ruleMessageDescriptor.FullName() {
	case (&validate.AnyRules{}).ProtoReflect().Descriptor().FullName():
		return cel.AnyType
	case (&validate.BoolRules{}).ProtoReflect().Descriptor().FullName():
		return cel.BoolType
	case (&validate.BytesRules{}).ProtoReflect().Descriptor().FullName():
		return cel.BytesType
	case (&validate.StringRules{}).ProtoReflect().Descriptor().FullName():
		return cel.StringType
	case
		(&validate.Int32Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.Int64Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.SInt32Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.SInt64Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.SFixed32Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.SFixed64Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.EnumRules{}).ProtoReflect().Descriptor().FullName():
		return cel.IntType
	case
		(&validate.UInt32Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.UInt64Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.Fixed32Rules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.Fixed64Rules{}).ProtoReflect().Descriptor().FullName():
		return cel.UintType
	case
		(&validate.DoubleRules{}).ProtoReflect().Descriptor().FullName(),
		(&validate.FloatRules{}).ProtoReflect().Descriptor().FullName():
		return cel.DoubleType
	case (&validate.MapRules{}).ProtoReflect().Descriptor().FullName():
		// The key and value constraints are handled separately as field constraints, so we use
		// cel.DynType for key and value here.
		return cel.MapType(cel.DynType, cel.DynType)
	case (&validate.RepeatedRules{}).ProtoReflect().Descriptor().FullName():
		// The repeated type is handled separately as field constraints, so we use cel.DynType
		// for the value type here.
		return cel.ListType(cel.DynType)
	case (&validate.DurationRules{}).ProtoReflect().Descriptor().FullName():
		return cel.DurationType
	case (&validate.TimestampRules{}).ProtoReflect().Descriptor().FullName():
		return cel.TimestampType
	}
	// We default to returning nil if this does not match with one of the *Rule declarations.
	return nil
}
