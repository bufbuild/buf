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

package buflintvalidate

import (
	"fmt"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	celpv "buf.build/go/protovalidate/cel"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldRules
	celFieldNumberInFieldRules = 23
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageRules
	celFieldNumberInMessageRules = 3
)

func checkCELForMessage(
	add func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...any),
	messageRules *validate.MessageRules,
	messageDescriptor protoreflect.MessageDescriptor,
	message bufprotosource.Message,
) error {
	celEnv, err := cel.NewEnv(
		cel.Lib(celpv.NewLibrary()),
	)
	if err != nil {
		return err
	}
	celEnv, err = celEnv.Extend(
		cel.Types(dynamicpb.NewMessage(messageDescriptor)),
		cel.Variable("this", cel.ObjectType(string(messageDescriptor.FullName()))),
	)
	if err != nil {
		return err
	}
	checkCEL(
		celEnv,
		messageRules.GetCel(),
		fmt.Sprintf("message %q", message.Name()),
		fmt.Sprintf("Message %q", message.Name()),
		"(buf.validate.message).cel",
		func(index int, format string, args ...any) {
			messageRulesOptionLocation := message.OptionExtensionLocation(
				validate.E_Message,
				celFieldNumberInMessageRules,
				int32(index),
			)
			add(message, messageRulesOptionLocation, nil, format, args...)
		},
	)
	return nil
}

func checkCELForField(
	adder *adder,
	fieldRules *validate.FieldRules,
	fieldDescriptor protoreflect.FieldDescriptor,
	// forItems is true if the CEL rule is defined on a non-repeated field or on each item of a repeated field.
	forItems bool,
) error {
	if len(fieldRules.GetCel()) == 0 {
		return nil
	}
	celEnv, err := cel.NewEnv(
		cel.Lib(celpv.NewLibrary()),
	)
	if err != nil {
		return err
	}
	celEnv, err = celEnv.Extend(
		append(
			celpv.RequiredEnvOptions(fieldDescriptor),
			cel.Variable("this", celpv.ProtoFieldToType(fieldDescriptor, false, forItems)),
		)...,
	)
	if err != nil {
		return err
	}
	checkCEL(
		celEnv,
		fieldRules.GetCel(),
		fmt.Sprintf("field %q", adder.fieldName()),
		fmt.Sprintf("Field %q", adder.fieldName()),
		adder.getFieldRuleName(celFieldNumberInFieldRules),
		func(index int, format string, args ...any) {
			adder.addForPathf(
				[]int32{celFieldNumberInFieldRules, int32(index)},
				format,
				args...,
			)
		},
	)
	return nil
}

// Returns true only if all cel expressions compile
func checkCEL(
	celEnv *cel.Env,
	celRules []*validate.Rule,
	parentName string,
	parentNameCapitalized string,
	celName string,
	add func(int, string, ...any),
) bool {
	allCelExpressionsCompile := true
	idToConstraintIndices := make(map[string][]int, len(celRules))
	for i, celConstraint := range celRules {
		if celID := celConstraint.GetId(); celID != "" {
			for _, char := range celID {
				if 'a' <= char && char <= 'z' {
					continue
				} else if 'A' <= char && char <= 'Z' {
					continue
				} else if '0' <= char && char <= '9' {
					continue
				} else if char == '_' || char == '-' || char == '.' {
					continue
				}
				add(
					i,
					"%s has invalid characters for %s.id. IDs must contain only characters a-z, A-Z, 0-9, '.', '_', '-'.",
					parentNameCapitalized,
					celName,
				)
				break
			}
			idToConstraintIndices[celID] = append(idToConstraintIndices[celID], i)
		}
		if len(strings.TrimSpace(celConstraint.GetExpression())) == 0 {
			add(i, "%s has an empty %s.expression. Expressions should always be specified.", parentNameCapitalized, celName)
			continue
		}
		ast, compileIssues := celEnv.Compile(celConstraint.GetExpression())
		switch {
		case ast.OutputType().IsAssignableType(cel.BoolType):
		case ast.OutputType().IsAssignableType(cel.StringType):
			if celConstraint.GetMessage() != "" {
				add(
					i,
					"%s has a %s with an expression that evaluates to a string, and also has a message. The message is redundant - since the expression evaluates to a string, its result will be printed instead of the message, so the message should be removed.",
					parentNameCapitalized,
					celName,
				)
			}
		case ast.OutputType().IsExactType(types.ErrorType):
			// If the output type is error, it means compilation has failed and we
			// only need to add the compilation issues.
		default:
			add(
				i,
				"%s.expression on %s evaluates to a %s, only string and boolean are allowed.",
				celName,
				parentName,
				cel.FormatCELType(ast.OutputType()),
			)
		}
		if compileIssues.Err() != nil {
			allCelExpressionsCompile = false
			for _, parsedIssue := range parseCelIssuesText(compileIssues.Err().Error()) {
				add(
					i,
					"%s.expression on %s fails to compile: %s",
					celName,
					parentName,
					parsedIssue,
				)
			}
		}
	}
	for celID, constraintIndices := range idToConstraintIndices {
		if len(constraintIndices) <= 1 {
			continue
		}
		for _, constraintIndex := range constraintIndices {
			add(
				constraintIndex,
				"%s.id (%q) is not unique within %s.",
				celName,
				celID,
				parentName,
			)
		}
	}
	return allCelExpressionsCompile
}

// this depends on the undocumented behavior of cel-go's error message
//
// maps a string in this form:
// "ERROR: <input>:1:6: found no matching overload for '_+_' applied to '(int, string)'
// | this + 'xyz' > (this * 'xyz')
// | .....^
// ERROR: <input>:1:22: found no matching overload for '_*_' applied to '(int, string)'
// | this + 'xyz' > (this * 'xyz')
// | .....................^"
// to a string slice:
// [ "found no matching overload for '_+_' applied to '(int, string)'
// | this + 'xyz' > (this * 'xyz')
// | .....^",
// "found no matching overload for '_*_' applied to '(int, string)'
// | this + 'xyz' > (this * 'xyz')
// | .....................^"]
func parseCelIssuesText(issuesText string) []string {
	issues := strings.Split(issuesText, "ERROR: <input>:")
	parsedIssues := make([]string, 0, len(issues)-1)
	for _, issue := range issues {
		issue = strings.TrimSpace(issue)
		if len(issue) == 0 {
			continue
		}
		// now issue looks like 1:2:<error message>
		parts := strings.SplitAfterN(issue, ":", 3)
		parsedIssues = append(parsedIssues, strings.TrimSpace(parts[len(parts)-1]))
	}
	return parsedIssues
}
