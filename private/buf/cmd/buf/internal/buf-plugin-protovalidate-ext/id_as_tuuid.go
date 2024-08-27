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

package protovalidateextplugin

import (
	"context"

	"github.com/bufbuild/bufplugin-go/check"
	"github.com/bufbuild/bufplugin-go/check/checkutil"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	// IDFieldValidationRuleSpec is the RuleSpec for the ID field validation rule.
	IDFieldValidatedAsUUIDRuleSpec = &check.RuleSpec{
		ID:             validateIDDashless,
		CategoryIDs:    nil,
		IsDefault:      true,
		Purpose:        `Checks that all fields named with a certain name (default is "id") are validated as dashless UUIDs in protovalidate.`,
		Type:           check.RuleTypeLint,
		ReplacementIDs: nil,
		Handler:        checkutil.NewFieldRuleHandler(checkValidateIDDashless),
	}
)

const (
	// validateIDDashless is the Rule ID of the valdiating ID fields dashless rule.
	validateIDDashless = "VALIDATE_ID_DASHLESS"

	// idFieldNameOptionKey is the option key to override the default id field name.
	idFieldNameOptionKey = "id_field_name"

	defaultIDFieldName = "id"
)

func checkValidateIDDashless(
	_ context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	idFieldName := defaultIDFieldName
	idFieldNameOptionValue, err := check.GetStringValue(request.Options(), idFieldNameOptionKey)
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
