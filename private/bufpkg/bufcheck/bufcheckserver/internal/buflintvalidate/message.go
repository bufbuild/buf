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

package buflintvalidate

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageRules
	oneofFieldNumberInMessageRules = 4
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageOneofRule
	fieldsFieldNumberInMessageOneofRule = 1
)

func checkOneofRulesForMessage(
	add func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	messageRules *validate.MessageRules,
	messageDescriptor protoreflect.MessageDescriptor,
	message bufprotosource.Message,
) {
	oneofRules := messageRules.GetOneof()
	if len(oneofRules) == 0 {
		return
	}
	// Track which oneof entries reference each field name, so we can report
	// fields that appear in more than one entry.
	fieldNameToOneofIndices := make(map[string][]int)
	messageFields := messageDescriptor.Fields()
	for i, oneofRule := range oneofRules {
		oneofLocation := message.OptionExtensionLocation(
			validate.E_Message,
			oneofFieldNumberInMessageRules,
			int32(i),
		)
		fields := oneofRule.GetFields()
		if len(fields) == 0 {
			add(
				message,
				oneofLocation,
				nil,
				"Message %q has a (buf.validate.message).oneof entry with no fields. Each oneof entry must specify at least one field.",
				message.Name(),
			)
			continue
		}
		seenInThisEntry := make(map[string]struct{}, len(fields))
		for j, fieldName := range fields {
			fieldLocation := message.OptionExtensionLocation(
				validate.E_Message,
				oneofFieldNumberInMessageRules,
				int32(i),
				fieldsFieldNumberInMessageOneofRule,
				int32(j),
			)
			if _, ok := seenInThisEntry[fieldName]; ok {
				add(
					message,
					fieldLocation,
					nil,
					"Message %q has a (buf.validate.message).oneof entry that references field %q more than once. Duplicate field names are not permitted.",
					message.Name(),
					fieldName,
				)
				continue
			}
			seenInThisEntry[fieldName] = struct{}{}
			if messageFields.ByName(protoreflect.Name(fieldName)) == nil {
				add(
					message,
					fieldLocation,
					nil,
					"Message %q has a (buf.validate.message).oneof entry that references field %q, which is not defined in the message.",
					message.Name(),
					fieldName,
				)
				continue
			}
			fieldNameToOneofIndices[fieldName] = append(fieldNameToOneofIndices[fieldName], i)
		}
	}
	for fieldName, indices := range fieldNameToOneofIndices {
		if len(indices) < 2 {
			continue
		}
		locations := make([]bufprotosource.Location, 0, len(indices))
		for _, index := range indices {
			locations = append(locations, message.OptionExtensionLocation(
				validate.E_Message,
				oneofFieldNumberInMessageRules,
				int32(index),
			))
		}
		locations = deduplicateLocations(locations)
		for _, location := range locations {
			add(
				message,
				location,
				nil,
				"Message %q references field %q in more than one (buf.validate.message).oneof entry. A field may only appear in one oneof entry.",
				message.Name(),
				fieldName,
			)
		}
	}
}
