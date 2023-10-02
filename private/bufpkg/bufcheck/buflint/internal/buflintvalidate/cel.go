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

	"github.com/bufbuild/buf/private/gen/proto/go/buf/validate"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"google.golang.org/protobuf/proto"
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

var (
	messageConstraintsExtensionType = dynamicpb.NewExtensionType(validate.E_Message.TypeDescriptor())
	fieldConstraintsExtensionType   = dynamicpb.NewExtensionType(validate.E_Field.TypeDescriptor())
)

func checkCelInMessage(
	resolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	message protosource.Message,
) error {
	for _, field := range message.Fields() {
		if err := checkCelInField(resolver, add, field); err != nil {
			return err
		}
	}
	for _, nestedMessage := range message.Messages() {
		if err := checkCelInMessage(resolver, add, nestedMessage); err != nil {
			return err
		}
	}
	messageConstraints, found, err := getMessageConstraints(message)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	reflectMessageDescriptor, err := getReflectMessageDescriptor(resolver, message)
	if err != nil {
		return err
	}
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	celEnv, err = celEnv.Extend(
		cel.Types(dynamicpb.NewMessage(reflectMessageDescriptor)),
		cel.Variable("this", cel.ObjectType(string(reflectMessageDescriptor.FullName()))),
	)
	if err != nil {
		return err
	}
	for i, cel := range messageConstraints.GetCel() {
		messageConstraintsOptionLocation := message.OptionExtensionLocation(
			messageConstraintsExtensionType,
			celFieldTagInMessageConstraints,
			int32(i),
		)
		if len(strings.TrimSpace(cel.GetExpression())) == 0 {
			add(message, messageConstraintsOptionLocation, nil, "cel expression is empty")
			continue
		}
		ast, compileIssues := celEnv.Compile(cel.GetExpression())
		switch ast.OutputType() {
		case types.BoolType, types.StringType, types.ErrorType:
			// If type is types.ErrorType, compilation has failed and we will
			// only add the compilation issues.
		default:
			add(message, messageConstraintsOptionLocation, nil, "cel expression evaluates to unsupported type: %v", ast.OutputType())
		}
		if compileIssues.Err() != nil {
			for _, parsedIssue := range parseCelIssuesText(compileIssues.Err().Error()) {
				add(message, messageConstraintsOptionLocation, nil, parsedIssue)
			}
		}
	}
	return nil
}

func checkCelInField(
	resolver protodesc.Resolver,
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	field protosource.Field,
) error {
	fieldConstraints, found, err := getFieldConstraints(field)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	fieldDesc, err := getReflectFieldDescriptor(resolver, field)
	if err != nil {
		return err
	}
	env, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	if fieldDesc.Kind() == protoreflect.MessageKind {
		env, err = env.Extend(
			cel.Types(dynamicpb.NewMessage(fieldDesc.Message())),
			cel.Variable("this", cel.ObjectType(string(fieldDesc.Message().FullName()))),
		)
	} else {
		env, err = env.Extend(
			cel.Variable("this", protoKindToCELType(fieldDesc.Kind())),
		)
	}
	if err != nil {
		return err
	}
	for i, cel := range fieldConstraints.GetCel() {
		celLocation := field.OptionExtensionLocation(
			fieldConstraintsExtensionType,
			celFieldTagInFieldConstraints,
			int32(i),
		)
		if len(strings.TrimSpace(cel.Expression)) == 0 {
			add(field, celLocation, nil, "cel expression is empty")
			continue
		}
		ast, compileIssues := env.Compile(cel.Expression)
		switch ast.OutputType() {
		case types.BoolType, types.StringType, types.ErrorType:
			// If type is types.ErrorType, compilation has failed and we will
			// only add the compilation issues.
		default:
			add(field, celLocation, nil, "cel expression evaluates to unsupported type: %v", ast.OutputType())
		}
		if compileIssues.Err() != nil {
			for _, parsedIssue := range parseCelIssuesText(compileIssues.Err().Error()) {
				add(field, celLocation, nil, parsedIssue)
			}
		}
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

func getMessageConstraints(message protosource.Message) (*validate.MessageConstraints, bool, error) {
	messageConstraintsMessageUntyped, found := message.OptionExtension(messageConstraintsExtensionType)
	if !found {
		return nil, false, nil
	}
	messageConstraintsMessage, ok := messageConstraintsMessageUntyped.(protoreflect.Message)
	if !ok {
		// this should never happen
		return nil, false, fmt.Errorf("field extension expected to be `protoreflect.Message`, but has type %T", messageConstraintsMessageUntyped)
	}
	data, err := proto.Marshal(messageConstraintsMessage.Interface())
	if err != nil {
		return nil, false, err
	}
	messageConstraints := &validate.MessageConstraints{}
	err = proto.Unmarshal(data, messageConstraints)
	if err != nil {
		return nil, false, err
	}
	return messageConstraints, true, nil
}

func getFieldConstraints(field protosource.Field) (*validate.FieldConstraints, bool, error) {
	fieldConstraintsMessageUntyped, found := field.OptionExtension(fieldConstraintsExtensionType)
	if !found {
		return nil, false, nil
	}
	fieldConstraintsMessage, ok := fieldConstraintsMessageUntyped.(protoreflect.Message)
	if !ok {
		// this should never happen
		return nil, false, fmt.Errorf("field extension expected to be `protoreflect.Message`, but has type %T", fieldConstraintsMessageUntyped)
	}
	data, err := proto.Marshal(fieldConstraintsMessage.Interface())
	if err != nil {
		return nil, false, err
	}
	fieldConstraints := &validate.FieldConstraints{}
	err = proto.Unmarshal(data, fieldConstraints)
	if err != nil {
		return nil, false, err
	}
	return fieldConstraints, true, nil
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
