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

package buflintvalidate

import (
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/bufbuild/protovalidate-go/resolver"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.MessageConstraints
const disabledFieldNumberInMesageConstraints = 1

// TODO: add doc
// Only registers if cel compiles
func CheckAndRegisterSharedRuleExtension(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	field bufprotosource.Field,
	// TODO: update parameter type and name
	types *protoregistry.Types,
) error {
	fieldDescriptor, err := field.AsDescriptor()
	if err != nil {
		return err
	}
	// TODO: move to a different func instead of having nested ifs
	if fieldDescriptor.IsExtension() {
		extendedStandardRuleDescriptor := fieldDescriptor.ContainingMessage()
		extendedRuleFullName := extendedStandardRuleDescriptor.FullName()
		if strings.HasPrefix(string(extendedRuleFullName), "buf.validate.") && strings.HasSuffix(string(extendedRuleFullName), "Rules") {
			// Just to be extra sure.
			if validate.File_buf_validate_validate_proto.Messages().ByName(extendedRuleFullName.Name()) != nil {
				if extendedRules := resolveExt[*validate.SharedFieldConstraints](fieldDescriptor.Options(), validate.E_SharedField); extendedRules != nil {
					celEnv, err := celext.DefaultEnv(false)
					if err != nil {
						return err
					}
					// TODO: add an example in comment
					// This is a bit of hacky, relying on the fact that each *Rules has a const rule,
					// we take advantage of it to give "this" a type.
					ruleConstFieldDescriptor := extendedStandardRuleDescriptor.Fields().ByName("const")
					if ruleConstFieldDescriptor == nil {
						// This shouldn't happen
						return syserror.Newf("unexpected protovalidate rule without a const rule, which is relied upon by buf lint")
					}
					thisType := celext.ProtoFieldToCELType(ruleConstFieldDescriptor, false, false)
					celEnv, err = celEnv.Extend(
						append(
							celext.RequiredCELEnvOptions(fieldDescriptor),
							cel.Variable("rule", celext.ProtoFieldToCELType(fieldDescriptor, false, false)),
							cel.Variable("this", thisType),
						)...,
					)
					if err != nil {
						return err
					}
					if checkCEL(
						celEnv,
						extendedRules.GetCel(),
						"TODO1",
						"TODO2",
						"TODO3",
						func(index int, format string, args ...interface{}) {
							addAnnotationFunc(
								field,
								// TODO: move 1 to a const
								field.OptionExtensionLocation(validate.E_SharedField, 1, int32(index)),
								nil,
								format,
								args...,
							)
						},
					) {
						if err := types.RegisterExtension(dynamicpb.NewExtensionType(fieldDescriptor)); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

// CheckMessage validates that all rules on the message are valid, and any CEL expressions compile.
//
// addAnnotationFunc adds an annotation with the descriptor and location for check results.
func CheckMessage(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	message bufprotosource.Message,
) error {
	messageDescriptor, err := message.AsDescriptor()
	if err != nil {
		return err
	}
	messageConstraints := resolver.DefaultResolver{}.ResolveMessageConstraints(messageDescriptor)
	if messageConstraints.GetDisabled() && len(messageConstraints.GetCel()) > 0 {
		addAnnotationFunc(
			message,
			message.OptionExtensionLocation(validate.E_Message, disabledFieldNumberInMesageConstraints),
			nil,
			"Message %q has (buf.validate.message).disabled, therefore other rules in (buf.validate.message) are not applied and should be removed.",
			message.Name(),
		)
	}
	return checkCELForMessage(
		addAnnotationFunc,
		messageConstraints,
		messageDescriptor,
		message,
	)
}

// CheckField validates that all rules on the field are valid, and any CEL expressions compile.
//
// For a set of rules to be valid, it must
//  1. permit _some_ value
//  2. have a type compatible with the field it validates.
//
// addAnnotationFunc adds an annotation with the descriptor and location for check results.
func CheckField(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	field bufprotosource.Field,
	// TODO: update parameter type and name
	types *protoregistry.Types,
) error {
	return checkField(addAnnotationFunc, field, types)
}
