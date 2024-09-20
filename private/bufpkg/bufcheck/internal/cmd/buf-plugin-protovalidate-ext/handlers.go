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

package main

import (
	"context"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/option"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	idFieldNameOptionKey = "id_field_name"
	defaultIDFieldName   = "id"
)

func checkFieldNotSkippedNoImport(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldDescriptor)
	if constraints.GetIgnore() == validate.Ignore_IGNORE_ALWAYS {
		skippedRuleName := "(buf.validate.field).skipped"
		if fieldDescriptor.Cardinality() == protoreflect.Repeated {
			skippedRuleName = "(buf.validate.field).repeated.items.skipped"
		}
		responseWriter.AddAnnotation(
			check.WithDescriptor(fieldDescriptor),
			check.WithMessagef(
				"field %q has %s set",
				fieldDescriptor.FullName(),
				skippedRuleName,
			),
		)
	}
	return nil
}

func checkFieldNotSkipped(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldDescriptor)
	if constraints.GetIgnore() == validate.Ignore_IGNORE_ALWAYS {
		skippedRuleName := "(buf.validate.field).skipped"
		if fieldDescriptor.Cardinality() == protoreflect.Repeated {
			skippedRuleName = "(buf.validate.field).repeated.items.skipped"
		}
		responseWriter.AddAnnotation(
			check.WithDescriptor(fieldDescriptor),
			check.WithMessagef(
				"field %q has %s set",
				fieldDescriptor.FullName(),
				skippedRuleName,
			),
		)
	}
	return nil
}

func checkStringLenRangeDontShrink(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	field protoreflect.FieldDescriptor,
	againstField protoreflect.FieldDescriptor,
) error {
	againstConstraints := resolver.DefaultResolver{}.ResolveFieldConstraints(againstField)
	if againstStringRules := againstConstraints.GetString_(); againstStringRules != nil {
		constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(field)
		if stringRules := constraints.GetString_(); stringRules != nil {
			if againstStringRules.MinLen != nil && stringRules.MinLen != nil && stringRules.GetMinLen() > againstStringRules.GetMinLen() {
				responseWriter.AddAnnotation(
					check.WithDescriptor(field),
					check.WithAgainstDescriptor(againstField),
					check.WithMessagef("min len requirement raised from %d to %d", againstStringRules.GetMinLen(), stringRules.GetMinLen()),
				)
			}
			if againstStringRules.MaxLen != nil && stringRules.MaxLen != nil && stringRules.GetMaxLen() < againstStringRules.GetMaxLen() {
				responseWriter.AddAnnotation(
					check.WithDescriptor(field),
					check.WithAgainstDescriptor(againstField),
					check.WithMessagef("max len requirement reduced from %d to %d", againstStringRules.GetMaxLen(), stringRules.GetMaxLen()),
				)
			}
		}
	}
	return nil
}

func checkValidateIDDashless(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	idFieldName := defaultIDFieldName
	idFieldNameOptionValue, err := option.GetStringValue(request.Options(), idFieldNameOptionKey)
	if err != nil {
		return err
	}
	if idFieldNameOptionValue != "" {
		idFieldName = idFieldNameOptionValue
	}
	if string(fieldDescriptor.Name()) != idFieldName {
		return nil
	}
	if fieldDescriptor.Kind() != protoreflect.StringKind {
		return nil
	}
	constraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldDescriptor)
	if stringConstraints := constraints.GetString_(); stringConstraints == nil || !stringConstraints.GetTuuid() {
		missingRuleName := "(buf.validate.field).string.tuuid"
		if fieldDescriptor.Cardinality() == protoreflect.Repeated {
			missingRuleName = "(buf.validate.field).repeated.items.string.tuuid"
		}
		responseWriter.AddAnnotation(
			check.WithDescriptor(fieldDescriptor),
			check.WithMessagef(
				"field %q does not have rule %s set",
				fieldDescriptor.FullName(),
				missingRuleName,
			),
		)
	}
	return nil
}

func checkMessageNotDisabled(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	messageDescriptor protoreflect.MessageDescriptor,
) error {
	constraints := resolver.DefaultResolver{}.ResolveMessageConstraints(messageDescriptor)
	if constraints.GetDisabled() {
		responseWriter.AddAnnotation(
			check.WithMessagef("%s has (buf.validate.message).disabled set to true", string(messageDescriptor.Name())),
			check.WithDescriptor(messageDescriptor),
		)
	}
	return nil
}
