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
	"github.com/bufbuild/buf/private/pkg/protoencoding"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/google/cel-go/cel"
)

const (
	celFieldNumberPath = int32(1)
)

func checkPredefinedRuleExtension(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	extension bufprotosource.Field,
	extensionResolver protoencoding.Resolver,
) error {
	extensionDescriptor, err := extension.AsDescriptor()
	if err != nil {
		return err
	}
	// Double check.
	if !extensionDescriptor.IsExtension() {
		return nil
	}
	extendedStandardRuleDescriptor := extensionDescriptor.ContainingMessage()
	extendedRuleFullName := extendedStandardRuleDescriptor.FullName()
	// This function only lints extensions extending buf.validate.*Rules, e.g. buf.validate.StringRules.
	if !(strings.HasPrefix(string(extendedRuleFullName), "buf.validate.") && strings.HasSuffix(string(extendedRuleFullName), "Rules")) {
		return nil
	}
	// Just to be extra sure.
	if validate.File_buf_validate_validate_proto.Messages().ByName(extendedRuleFullName.Name()) == nil {
		return nil
	}
	predefinedConstraints, err := resolveExtension[*validate.PredefinedConstraints](extensionDescriptor.Options(), validate.E_Predefined, extensionResolver)
	if err != nil {
		return err
	}
	if predefinedConstraints == nil {
		return nil
	}
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	// In order to evaluate whether the CEL expression for the rule compiles, we need to check
	// the type declaration for two keywords, "this" and "rule".
	// "this" refers to the type the rule is checking, e.g. StringRules would have type string.
	// "rule" refers to the type of the rule extension, e.g. a rule that checks the length
	// of a string has type int32 to represent the length.
	//
	// In this example, an int32 field is added to extend string rules, and therefore,
	// "rule" has type int32 and "this" has type "string":
	//
	// extend buf.validate.StringRules {
	//	 optional int32 my_max = 47892 [(buf.validate.predefined).cel = {
	//	   id: "mymax"
	//	   message: "at most max"
	//	   expression: "size(this) < rule"
	//	 }];
	// }
	//
	ruleType := celext.ProtoFieldToCELType(extensionDescriptor, false, false)
	// To check for the type of "this", we check the descriptor for the rule type we are extending.
	thisType := celTypeForStandardRuleMessageDescriptor(extendedStandardRuleDescriptor)
	if thisType == nil {
		return syserror.Newf("extension for unexpected rule type %q found", extendedStandardRuleDescriptor.FullName())
	}
	celEnv, err = celEnv.Extend(
		append(
			celext.RequiredCELEnvOptions(extensionDescriptor),
			cel.Variable("rule", ruleType),
			cel.Variable("this", thisType),
		)...,
	)
	if err != nil {
		return err
	}
	checkCEL(
		celEnv,
		predefinedConstraints.GetCel(),
		"extension field",
		"Extension field",
		"(buf.validate.predefined).cel",
		func(index int, format string, args ...interface{}) {
			addAnnotationFunc(
				extension,
				extension.OptionExtensionLocation(validate.E_Predefined, celFieldNumberPath, int32(index)),
				nil,
				format,
				args...,
			)
		},
	)
	return nil
}
