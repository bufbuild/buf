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

// ReparseUnrecognized uses the given resolver to parse any unrecognized fields in the
// given reflectMessage. It does so recursively, resolving any unrecognized fields in
// nested messages.
//
// Deprecated: Use ReparseExtensions instead.
func ReparseUnrecognized(resolver Resolver, reflectMessage protoreflect.Message) error {
	if resolver == nil {
		return nil
	}
	unknown := reflectMessage.GetUnknown()
	if len(unknown) > 0 {
		reflectMessage.SetUnknown(nil)
		options := proto.UnmarshalOptions{
			Resolver: resolver,
			Merge:    true,
		}
		if err := options.Unmarshal(unknown, reflectMessage.Interface()); err != nil {
			return err
		}
	}
	var err error
	reflectMessage.Range(func(fieldDescriptor protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		err = reparseInField(resolver, fieldDescriptor, value, ReparseUnrecognized)
		return err == nil
	})
	return err
}
