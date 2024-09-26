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

package protoencoding

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ReparseExtensions uses the given resolver to parse any unrecognized fields in the
// given reflectMessage as well as re-parse any extensions.
func ReparseExtensions(resolver Resolver, reflectMessage protoreflect.Message) error {
	if resolver == nil {
		return nil
	}
	reparseBytes := reflectMessage.GetUnknown()

	if reflectMessage.Descriptor().ExtensionRanges().Len() > 0 {
		// Collect extensions into separate message so we can serialize
		// *just* the extensions and then re-parse them below.
		var msgExts protoreflect.Message
		reflectMessage.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
			if !field.IsExtension() {
				return true
			}
			if msgExts == nil {
				msgExts = reflectMessage.Type().New()
			}
			msgExts.Set(field, value)
			reflectMessage.Clear(field)
			return true
		})
		if msgExts != nil {
			options := proto.MarshalOptions{AllowPartial: true}
			var err error
			reparseBytes, err = options.MarshalAppend(reparseBytes, msgExts.Interface())
			if err != nil {
				return err
			}
		}
	}

	if len(reparseBytes) > 0 {
		reflectMessage.SetUnknown(nil)
		options := proto.UnmarshalOptions{
			Resolver: resolver,
			Merge:    true,
		}
		if err := options.Unmarshal(reparseBytes, reflectMessage.Interface()); err != nil {
			return err
		}
	}
	var err error
	reflectMessage.Range(func(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		err = reparseInField(resolver, fieldDescriptor, value)
		return err == nil
	})
	return err
}

func reparseInField(
	resolver Resolver,
	fieldDescriptor protoreflect.FieldDescriptor,
	value protoreflect.Value,
) error {
	if fieldDescriptor.IsMap() {
		valDesc := fieldDescriptor.MapValue()
		if valDesc.Kind() != protoreflect.MessageKind && valDesc.Kind() != protoreflect.GroupKind {
			// nothing to reparse
			return nil
		}
		var err error
		value.Map().Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			err = ReparseExtensions(resolver, v.Message())
			return err == nil
		})
		return err
	}
	if fieldDescriptor.Kind() != protoreflect.MessageKind && fieldDescriptor.Kind() != protoreflect.GroupKind {
		// nothing to reparse
		return nil
	}
	if fieldDescriptor.IsList() {
		list := value.List()
		for i := 0; i < list.Len(); i++ {
			if err := ReparseExtensions(resolver, list.Get(i).Message()); err != nil {
				return err
			}
		}
		return nil
	}
	return ReparseExtensions(resolver, value.Message())
}
