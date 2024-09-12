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
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.MessageConstraints
const disabledFieldNumberInMesageConstraints = 1

// ExtensionTypeResolver is an extension resolver, the same type as the Resolver in proto.UnmarshalOptions.
type ExtensionTypeResolver interface {
	FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error)
	FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error)
}

// CheckAndRegisterSharedRuleExtension checks whether an extension extending a protovalidate rule
// is valid, checking that all of its CEL expressionus compile. If so, the extension type is added to
// the extension types passed in.
func CheckAndRegisterSharedRuleExtension(
	addAnnotationFunc func(bufprotosource.Descriptor, bufprotosource.Location, []bufprotosource.Location, string, ...interface{}),
	field bufprotosource.Field,
	extensionTypesToPopulate *protoregistry.Types,
) error {
	return checkAndRegisterSharedRuleExtension(addAnnotationFunc, field, extensionTypesToPopulate)
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
	extenExtensionTypeResolver ExtensionTypeResolver,
) error {
	return checkField(addAnnotationFunc, field, extenExtensionTypeResolver)
}
