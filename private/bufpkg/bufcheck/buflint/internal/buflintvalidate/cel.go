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
	"github.com/bufbuild/protovalidate-go/resolver"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.FieldConstraints
	celFieldTagInFieldConstraints = 23
	// https://buf.build/bufbuild/protovalidate/docs/main:buf.validate#buf.validate.MessageConstraints
	celFieldTagInMessageConstraints = 3
)

func checkCelInMessage(
	descriptorResolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	message protosource.Message,
) error {
	for _, field := range message.Fields() {
		if err := checkCelInField(descriptorResolver, add, field); err != nil {
			return err
		}
	}
	for _, nestedMessage := range message.Messages() {
		if err := checkCelInMessage(descriptorResolver, add, nestedMessage); err != nil {
			return err
		}
	}
	messageReflectDescriptor, err := getReflectMessageDescriptor(descriptorResolver, message)
	if err != nil {
		return err
	}
	messageConstraints := resolver.DefaultResolver{}.ResolveMessageConstraints(messageReflectDescriptor)
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	celEnv, err = celEnv.Extend(
		cel.Types(dynamicpb.NewMessage(messageReflectDescriptor)),
		cel.Variable("this", cel.ObjectType(string(messageReflectDescriptor.FullName()))),
	)
	if err != nil {
		return err
	}
	for i, celConstraint := range messageConstraints.GetCel() {
		messageConstraintsOptionLocation := message.OptionExtensionLocation(
			validate.E_Message,
			celFieldTagInMessageConstraints,
			int32(i),
		)
		checkCel(celEnv, celConstraint, func(format string, args ...interface{}) {
			add(message, messageConstraintsOptionLocation, nil, format, args...)
		})
	}
	return nil
}

func checkCelInField(
	descriptorResolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	field protosource.Field,
) error {
	fieldReflectDescrptor, err := getReflectFieldDescriptor(descriptorResolver, field)
	if err != nil {
		return err
	}
	fieldConstraints := resolver.DefaultResolver{}.ResolveFieldConstraints(fieldReflectDescrptor)
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	if fieldReflectDescrptor.Kind() == protoreflect.MessageKind {
		celEnv, err = celEnv.Extend(
			cel.Types(dynamicpb.NewMessage(fieldReflectDescrptor.Message())),
			cel.Variable("this", cel.ObjectType(string(fieldReflectDescrptor.Message().FullName()))),
		)
	} else {
		celEnv, err = celEnv.Extend(
			cel.Variable("this", protoKindToCELType(fieldReflectDescrptor.Kind())),
		)
	}
	if err != nil {
		return err
	}
	for i, celConstraint := range fieldConstraints.GetCel() {
		celLocation := field.OptionExtensionLocation(
			validate.E_Field,
			celFieldTagInFieldConstraints,
			int32(i),
		)
		checkCel(celEnv, celConstraint, func(format string, args ...interface{}) {
			add(field, celLocation, nil, format, args...)
		})
	}
	return nil
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

// This operates on the assumption that id, message and expression share the same
// source code location.
func checkCel(
	celEnv *cel.Env,
	celConstraint *validate.Constraint,
	add func(string, ...interface{}),
) {
	if len(strings.TrimSpace(celConstraint.Expression)) == 0 {
		add("cel expression is empty")
		return
	}
	ast, compileIssues := celEnv.Compile(celConstraint.Expression)
	switch {
	case ast.OutputType().IsAssignableType(cel.BoolType):
		if celConstraint.Message == "" {
			add("validation message isn't specified")
		}
	case ast.OutputType().IsAssignableType(cel.StringType):
		if celConstraint.Message != "" {
			add("validation message is specified but the cel expression's result will be used instead")
		}
	case ast.OutputType().IsExactType(types.ErrorType):
	default:
		add("cel expression evaluates to unsupported type: %v", ast.OutputType())
	}
	if compileIssues.Err() != nil {
		for _, parsedIssue := range parseCelIssuesText(compileIssues.Err().Error()) {
			add(parsedIssue)
		}
	}
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
		parsedIssues = append(parsedIssues, parts[len(parts)-1])
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
