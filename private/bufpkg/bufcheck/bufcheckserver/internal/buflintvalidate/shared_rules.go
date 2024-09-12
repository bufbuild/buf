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
	"github.com/bufbuild/protovalidate-go/celext"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

func checkAndRegisterSharedRuleExtension(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	extension bufprotosource.Field,
	extensionTypesToPopulate *protoregistry.Types,
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
	sharedConstraints := resolveExt[*validate.SharedFieldConstraints](extensionDescriptor.Options(), validate.E_SharedField)
	if sharedConstraints == nil {
		return nil
	}
	celEnv, err := celext.DefaultEnv(false)
	if err != nil {
		return err
	}
	// Two keywords need type declaration, "this" and "rule", for the expression to compile.
	// In this example, an int32 field is added to extend string rules, and therefore,
	// "rule" has type int32 and "this" has type "string":
	//
	// extend buf.validate.StringRules {
	//	 optional int32 my_max = 47892 [(buf.validate.shared_field).cel = {
	//	   id: "mymax"
	//	   message: "at most max"
	//	   expression: "size(this) < rule"
	//	 }];
	// }
	//
	ruleType := celext.ProtoFieldToCELType(extensionDescriptor, false, false) // TODO: forItems should probably be false?
	// This is a bit of hacky, relying on the fact that each *Rules has a const rule,
	// and we take advantage of each buf.validate.<Foo>Rules.const has type <Foo>, which
	// is the type "this" should have.
	ruleConstFieldDescriptor := extendedStandardRuleDescriptor.Fields().ByName("const")
	if ruleConstFieldDescriptor == nil {
		// This isn't necessarily an error, it could be caused by a future buf.validate.*Rules without a const field.
		return nil
	}
	thisType := celext.ProtoFieldToCELType(ruleConstFieldDescriptor, false, false) // TODO: forItems is probably false?
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
	allCELExpressionsCompile := checkCEL(
		celEnv,
		sharedConstraints.GetCel(),
		"extension field",
		"Extension field",
		"(buf.validate.shared_field).cel",
		func(index int, format string, args ...interface{}) {
			addAnnotationFunc(
				extension,
				// TODO: move 1 to a const
				extension.OptionExtensionLocation(validate.E_SharedField, 1, int32(index)),
				nil,
				format,
				args...,
			)
		},
	)
	if allCELExpressionsCompile {
		if err := extensionTypesToPopulate.RegisterExtension(dynamicpb.NewExtensionType(extensionDescriptor)); err != nil {
			return err
		}
	}
	return nil
}
