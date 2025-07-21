// Copyright 2020-2025 Buf Technologies, Inc.
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
	"buf.build/go/protovalidate"
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
	constraints, err := protovalidate.ResolveFieldRules(fieldDescriptor)
	if err != nil {
		return err
	}
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
	constraints, err := protovalidate.ResolveFieldRules(fieldDescriptor)
	if err != nil {
		return err
	}
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
	againstRules, err := protovalidate.ResolveFieldRules(againstField)
	if err != nil {
		return err
	}
	if againstStringRules := againstRules.GetString(); againstStringRules != nil {
		constraints, _ := protovalidate.ResolveFieldRules(field)
		if stringRules := constraints.GetString(); stringRules != nil {
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
	constraints, err := protovalidate.ResolveFieldRules(fieldDescriptor)
	if err != nil {
		return err
	}
	if stringRules := constraints.GetString(); stringRules == nil || !stringRules.GetTuuid() {
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
