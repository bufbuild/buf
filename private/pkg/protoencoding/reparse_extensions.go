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
	rep := newReparser(resolver)
	rep.reparseExtensions(reflectMessage)
	return rep.err
}

type reparser struct {
	resolver Resolver

	// These fields hoist state up from reparseExtensions that could be local
	// variables, but then would escape to the heap and result in many per-call
	// allocations. So we allocate them once in this struct, which drastically
	// improves the performance of the recursive calls by eliminating allocations.

	reflectMessage protoreflect.Message
	msgExts        protoreflect.Message
	err            error

	// visitExt refers to msgExts and reflectMessage, so those two fields
	// must be reset prior to a Range call that uses visitExt.
	visitExt      func(protoreflect.FieldDescriptor, protoreflect.Value) bool
	visitMapEntry func(protoreflect.MapKey, protoreflect.Value) bool
}

func newReparser(resolver Resolver) *reparser {
	result := &reparser{resolver: resolver}
	result.visitExt = func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !field.IsExtension() {
			return true
		}
		if result.msgExts == nil {
			result.msgExts = result.reflectMessage.New()
		}
		result.msgExts.Set(field, value)
		result.reflectMessage.Clear(field)
		return true
	}
	result.visitMapEntry = func(_ protoreflect.MapKey, value protoreflect.Value) bool {
		result.reparseExtensions(value.Message())
		return result.err == nil
	}
	return result
}

func (r *reparser) reparseExtensions(reflectMessage protoreflect.Message) {
	reparseBytes := reflectMessage.GetUnknown()

	if reflectMessage.Descriptor().ExtensionRanges().Len() > 0 {
		// Collect extensions into separate message so we can serialize
		// *just* the extensions and then re-parse them below.

		// reset state for range
		r.msgExts = nil
		r.reflectMessage = reflectMessage

		reflectMessage.Range(r.visitExt)
		if r.msgExts != nil {
			options := proto.MarshalOptions{AllowPartial: true}
			var err error
			reparseBytes, err = options.MarshalAppend(reparseBytes, r.msgExts.Interface())
			if err != nil {
				r.err = err
				return
			}
		}
	}

	if len(reparseBytes) > 0 {
		reflectMessage.SetUnknown(nil)
		options := proto.UnmarshalOptions{
			Resolver: r.resolver,
			Merge:    true,
		}
		if err := options.Unmarshal(reparseBytes, reflectMessage.Interface()); err != nil {
			r.err = err
			return
		}
	}
	// Ideally, we'd use reflectMessage.Range here, but that allocates too much since it must
	// allocate a wrapper for every repeated field that implements protoreflect.List. We don't
	// care about visiting extensions (since we just resolved/reparsed those above), so we
	// can iterate over the non-extension fields.
	fields := reflectMessage.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if field.Message() == nil {
			// nothing to reparse if not a message
			continue
		}
		if !reflectMessage.Has(field) {
			continue
		}
		r.reparseInMessageField(reflectMessage, field)
		if r.err != nil {
			return
		}
	}
	return
}

func (r *reparser) reparseInMessageField(msg protoreflect.Message, field protoreflect.FieldDescriptor) {
	if field.IsMap() {
		if field.MapValue().Message() == nil {
			// nothing to reparse
			return
		}
		msg.Get(field).Map().Range(r.visitMapEntry)
		return
	}
	if field.IsList() {
		list := msg.Get(field).List()
		for i := range list.Len() {
			r.reparseExtensions(list.Get(i).Message())
			if r.err != nil {
				return
			}
		}
		return
	}
	r.reparseExtensions(msg.Get(field).Message())
}
