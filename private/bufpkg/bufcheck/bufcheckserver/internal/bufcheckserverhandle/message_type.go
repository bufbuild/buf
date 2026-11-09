// Copyright 2020-2026 Buf Technologies, Inc.
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

package bufcheckserverhandle

import (
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// messageTypeComparer recognizes identical message types moved to a different
// package or enclosing message. It deliberately requires identical definitions,
// rather than allowing arbitrary wire-compatible changes during a move.
type messageTypeComparer struct {
	wireJSON bool
	seen     map[[2]protoreflect.FullName]struct{}
}

func messageTypesEqual(previous, current protoreflect.MessageDescriptor, wireJSON bool) (bool, error) {
	comparer := messageTypeComparer{
		wireJSON: wireJSON,
		seen:     make(map[[2]protoreflect.FullName]struct{}),
	}
	return comparer.messagesEqual(previous, current)
}

func (c *messageTypeComparer) messagesEqual(previous, current protoreflect.MessageDescriptor) (bool, error) {
	if previous.Name() != current.Name() ||
		previous.Fields().Len() != current.Fields().Len() ||
		previous.ReservedNames().Len() != current.ReservedNames().Len() ||
		previous.ReservedRanges().Len() != current.ReservedRanges().Len() ||
		!proto.Equal(previous.Options(), current.Options()) {
		return false, nil
	}
	if previous.FullName() != current.FullName() {
		// Well-known types can have special JSON encodings that depend on their
		// full names, not just their fields (for example, Timestamp and wrappers).
		if c.wireJSON && (hasWellKnownJSONEncoding(previous.FullName()) ||
			hasWellKnownJSONEncoding(current.FullName())) {
			return false, nil
		}
		// Extensions can be declared outside the message. Comparing its fields
		// alone cannot establish that all of its extensions moved unchanged.
		if previous.ExtensionRanges().Len() != 0 || current.ExtensionRanges().Len() != 0 {
			return false, nil
		}
	}
	if c.wireJSON {
		if equal, err := messageTypeFeatureEqual(previous, current, featureNameJSONFormat); err != nil || !equal {
			return false, err
		}
	}
	pair := [2]protoreflect.FullName{previous.FullName(), current.FullName()}
	if _, ok := c.seen[pair]; ok {
		return true, nil
	}
	// Track pairs, not individual names: one old type may be referenced through
	// multiple new types. A fresh comparer is used for each top-level comparison,
	// and any mismatch immediately fails that comparison, including through cycles.
	c.seen[pair] = struct{}{}
	// A move must not bypass RESERVED_MESSAGE_NO_DELETE. Compare reservations
	// without depending on the order in which they were declared.
	for i := range previous.ReservedNames().Len() {
		if !current.ReservedNames().Has(previous.ReservedNames().Get(i)) {
			return false, nil
		}
	}
	reservedRanges := make(map[[2]protoreflect.FieldNumber]struct{}, current.ReservedRanges().Len())
	for i := range current.ReservedRanges().Len() {
		reservedRanges[current.ReservedRanges().Get(i)] = struct{}{}
	}
	for i := range previous.ReservedRanges().Len() {
		if _, ok := reservedRanges[previous.ReservedRanges().Get(i)]; !ok {
			return false, nil
		}
	}
	for i := range previous.Fields().Len() {
		previousField := previous.Fields().Get(i)
		field := current.Fields().ByNumber(previousField.Number())
		if field == nil {
			return false, nil
		}
		if equal, err := c.fieldsEqual(previousField, field); err != nil || !equal {
			return false, err
		}
	}
	return true, nil
}

func (c *messageTypeComparer) fieldsEqual(previous, current protoreflect.FieldDescriptor) (bool, error) {
	// These properties include resolved Editions features, which may be inherited
	// from a new file or enclosing message even when the field text is unchanged.
	if previous.Kind() != current.Kind() ||
		previous.Cardinality() != current.Cardinality() ||
		previous.HasPresence() != current.HasPresence() ||
		previous.IsPacked() != current.IsPacked() {
		return false, nil
	}
	previousProto := protodesc.ToFieldDescriptorProto(previous)
	currentProto := protodesc.ToFieldDescriptorProto(current)
	// Compare referenced definitions below instead of their qualified names.
	previousProto.TypeName, currentProto.TypeName = nil, nil
	// Oneofs are matched by name, so declaration order does not matter.
	previousProto.OneofIndex, currentProto.OneofIndex = nil, nil
	if !proto.Equal(previousProto, currentProto) {
		return false, nil
	}
	previousOneof, currentOneof := previous.ContainingOneof(), current.ContainingOneof()
	if (previousOneof == nil) != (currentOneof == nil) {
		return false, nil
	}
	if previousOneof != nil && (previousOneof.Name() != currentOneof.Name() ||
		!proto.Equal(previousOneof.Options(), currentOneof.Options())) {
		return false, nil
	}
	switch previous.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		// Even when the qualified names match, compare definitions: a change to
		// a referenced type must not be hidden by moving its enclosing message.
		return c.messagesEqual(previous.Message(), current.Message())
	case protoreflect.EnumKind:
		previousEnum, currentEnum := previous.Enum(), current.Enum()
		if c.wireJSON && previousEnum.FullName() != currentEnum.FullName() &&
			(hasWellKnownJSONEncoding(previousEnum.FullName()) || hasWellKnownJSONEncoding(currentEnum.FullName())) {
			return false, nil
		}
		if previousEnum.IsClosed() != currentEnum.IsClosed() ||
			!proto.Equal(protodesc.ToEnumDescriptorProto(previousEnum), protodesc.ToEnumDescriptorProto(currentEnum)) {
			return false, nil
		}
		if c.wireJSON {
			return messageTypeFeatureEqual(previousEnum, currentEnum, featureNameJSONFormat)
		}
	case protoreflect.StringKind:
		return messageTypeFeatureEqual(previous, current, featureNameUTF8Validation)
	}
	return true, nil
}

func messageTypeFeatureEqual(previous, current protoreflect.Descriptor, featureName protoreflect.Name) (bool, error) {
	feature, err := findFeatureField(featureName, protoreflect.EnumKind)
	if err != nil {
		return false, err
	}
	previousValue, err := protoutil.ResolveFeature(previous, feature)
	if err != nil {
		return false, err
	}
	currentValue, err := protoutil.ResolveFeature(current, feature)
	if err != nil {
		return false, err
	}
	return previousValue.Enum() == currentValue.Enum(), nil
}

// These types have JSON encodings that depend on their fully-qualified names.
// Other types in google.protobuf, such as Empty, use the ordinary JSON mapping.
func hasWellKnownJSONEncoding(name protoreflect.FullName) bool {
	switch name {
	case "google.protobuf.Any",
		"google.protobuf.Timestamp",
		"google.protobuf.Duration",
		"google.protobuf.BoolValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.FloatValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue",
		"google.protobuf.Struct",
		"google.protobuf.ListValue",
		"google.protobuf.Value",
		"google.protobuf.FieldMask",
		"google.protobuf.NullValue":
		return true
	default:
		return false
	}
}
