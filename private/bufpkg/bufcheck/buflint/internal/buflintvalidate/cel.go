// Copyright 2020-2023 Buf Technologies, Inc.
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
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldConstraints
	celFieldNumberInFieldConstraints = 23
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageConstraints
	celFieldNumberInMessageConstraints = 3
)

func checkCELForMessage(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	messageConstraints *validate.MessageConstraints,
	messageDescriptor protoreflect.MessageDescriptor,
	message protosource.Message,
) error {
	celEnv, err := celext.DefaultEnv(false)
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
		messageConstraints.GetCel(),
		fmt.Sprintf("message %q", message.Name()),
		fmt.Sprintf("Message %q", message.Name()),
		"(buf.validate.message).cel",
		func(index int, format string, args ...interface{}) {
			messageConstraintsOptionLocation := message.OptionExtensionLocation(
				validate.E_Message,
				celFieldNumberInMessageConstraints,
				int32(index),
			)
			add(message, messageConstraintsOptionLocation, nil, format, args...)
		},
	)
	return nil
}

func checkCELForField(
	adder *adder,
	fieldConstraints *validate.FieldConstraints,
	fieldDescriptor protoreflect.FieldDescriptor,
) error {
	if len(fieldConstraints.GetCel()) == 0 {
		return nil
	}
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	if fieldDescriptor.Kind() == protoreflect.MessageKind {
		celEnv, err = celEnv.Extend(
			cel.Types(
				dynamicpb.NewMessage(fieldDescriptor.ContainingMessage()),
				dynamicpb.NewMessage(fieldDescriptor.Message()),
			),
			cel.Variable("this", cel.ObjectType(string(fieldDescriptor.Message().FullName()))),
		)
	} else {
		celEnv, err = celEnv.Extend(
			cel.Variable("this", protoKindToCELType(fieldDescriptor.Kind())),
		)
	}
	if err != nil {
		return err
	}
	checkCEL(
		celEnv,
		fieldConstraints.GetCel(),
		fmt.Sprintf("field %q", adder.fieldName()),
		fmt.Sprintf("Field %q", adder.fieldName()),
		adder.getFieldRuleName(celFieldNumberInFieldConstraints),
		func(index int, format string, args ...interface{}) {
			adder.addForPathf(
				[]int32{celFieldNumberInFieldConstraints, int32(index)},
				format,
				args...,
			)
		},
	)
	return nil
}

func checkCEL(
	celEnv *cel.Env,
	celConstraints []*validate.Constraint,
	parentName string,
	parentNameCapitalized string,
	celName string,
	add func(int, string, ...interface{}),
) {
	idToConstraintIndices := make(map[string][]int, len(celConstraints))
	for i, celConstraint := range celConstraints {
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
		} else {
			add(i, "%s has an empty %s.id. IDs should always be specified.", parentNameCapitalized, celName)
		}
		if len(strings.TrimSpace(celConstraint.Expression)) == 0 {
			add(i, "%s has an empty %s.expression. Expressions should always be specified.", parentNameCapitalized, celName)
			continue
		}
		ast, compileIssues := celEnv.Compile(celConstraint.Expression)
		switch {
		case ast.OutputType().IsAssignableType(cel.BoolType):
			if celConstraint.Message == "" {
				add(
					i,
					"%s has an empty %s.message for an expression that evaluates to a boolean. If an expression evaluates to a boolean, a message should always be specified.",
					parentNameCapitalized,
					celName,
				)
			}
		case ast.OutputType().IsAssignableType(cel.StringType):
			if celConstraint.Message != "" {
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
}

func getReflectMessageDescriptor(resolver protodesc.Resolver, message protosource.Message) (protoreflect.MessageDescriptor, error) {
	descriptor, err := resolver.FindDescriptorByName(protoreflect.FullName(message.FullName()))
	if err == protoregistry.NotFound {
		return nil, fmt.Errorf("unable to resolve MessageDescriptor: %s", message.FullName())
	}
	if err != nil {
		return nil, err
	}
	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		// this should not happen
		return nil, fmt.Errorf("%s is not a message", descriptor.FullName())
	}
	return messageDescriptor, nil
}

func getReflectFieldDescriptor(resolver protodesc.Resolver, field protosource.Field) (protoreflect.FieldDescriptor, error) {
	descriptor, err := resolver.FindDescriptorByName(protoreflect.FullName(field.FullName()))
	if err == protoregistry.NotFound {
		return nil, fmt.Errorf("unable to resolve FieldDescriptor: %s", field.FullName())
	}
	if err != nil {
		return nil, err
	}
	fieldDescriptor, ok := descriptor.(protoreflect.FieldDescriptor)
	if !ok {
		// this should never happen
		return nil, fmt.Errorf("%s is not a field", descriptor.FullName())
	}
	return fieldDescriptor, nil
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

// copied directly from protovalidate-go
func protoKindToCELType(kind protoreflect.Kind) *cel.Type {
	switch kind {
	case
		protoreflect.FloatKind,
		protoreflect.DoubleKind:
		return cel.DoubleType
	case
		protoreflect.Int32Kind,
		protoreflect.Int64Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.EnumKind:
		return cel.IntType
	case
		protoreflect.Uint32Kind,
		protoreflect.Uint64Kind,
		protoreflect.Fixed32Kind,
		protoreflect.Fixed64Kind:
		return cel.UintType
	case protoreflect.BoolKind:
		return cel.BoolType
	case protoreflect.StringKind:
		return cel.StringType
	case protoreflect.BytesKind:
		return cel.BytesType
	case
		protoreflect.MessageKind,
		protoreflect.GroupKind:
		return cel.DynType
	default:
		return cel.DynType
	}
}
