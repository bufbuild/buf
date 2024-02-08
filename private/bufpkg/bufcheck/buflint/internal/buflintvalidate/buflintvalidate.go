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
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"github.com/bufbuild/protovalidate-go/resolver"
	"google.golang.org/protobuf/reflect/protodesc"
)

// https://buf.build/bufbuild/protovalidate/docs/v0.5.1:buf.validate#buf.validate.MessageConstraints
const disabledFieldNumberInMesageConstraints = 1

// Check validates that all rules on fields are valid, and all CEL expressions compile.
//
// For a set of rules to be valid, it must
//  1. permit _some_ value
//  2. have a type compatible with the field it validates.
func Check(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	files []protosource.File,
) error {
	fileDescriptors := make([]protodescriptor.FileDescriptor, 0, len(files))
	for _, file := range files {
		fileDescriptors = append(fileDescriptors, file.FileDescriptor())
	}
	descriptorResolver, err := protodesc.NewFiles(protodescriptor.FileDescriptorSetForFileDescriptors(fileDescriptors...))
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsImport() {
			continue
		}
		for _, message := range file.Messages() {
			if err := checkForMessage(
				add,
				descriptorResolver,
				message,
			); err != nil {
				return err
			}
		}
		for _, extension := range file.Extensions() {
			if err := checkForField(
				add,
				descriptorResolver,
				extension,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkForMessage(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	descriptorResolver protodesc.Resolver,
	message protosource.Message,
) error {
	messageDescriptor, err := getReflectMessageDescriptor(descriptorResolver, message)
	if err != nil {
		return err
	}
	messageConstraints := resolver.DefaultResolver{}.ResolveMessageConstraints(messageDescriptor)
	if messageConstraints.GetDisabled() && len(messageConstraints.GetCel()) > 0 {
		add(
			message,
			message.OptionExtensionLocation(validate.E_Message, disabledFieldNumberInMesageConstraints),
			nil,
			"Message %q has (buf.validate.message).disabled, therefore other rules in (buf.validate.message) are not applied and should be removed.",
			message.Name(),
		)
	}
	if err := checkCELForMessage(
		add,
		messageConstraints,
		messageDescriptor,
		message,
	); err != nil {
		return err
	}
	for _, nestedMessage := range message.Messages() {
		if err := checkForMessage(add, descriptorResolver, nestedMessage); err != nil {
			return err
		}
	}
	for _, field := range message.Fields() {
		if err := checkForField(
			add,
			descriptorResolver,
			field,
		); err != nil {
			return err
		}
	}
	for _, extension := range message.Extensions() {
		if err := checkForField(
			add,
			descriptorResolver,
			extension,
		); err != nil {
			return err
		}
	}
	return nil
}
