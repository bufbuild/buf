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
	"github.com/bufbuild/buf/private/pkg/protodescriptor"
	"github.com/bufbuild/buf/private/pkg/protosource"
	"google.golang.org/protobuf/reflect/protodesc"
)

// Validate validates that all rules on fields are valid, and all CEL expressions compile.
//
// For a set of rules to be valid, it must
//  1. permit _some_ value
//  2. have a type compatible with the field it validates.
func Validate(
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
			if err := validateForMessage(
				add,
				descriptorResolver,
				message,
			); err != nil {
				return err
			}
		}
		for _, extension := range file.Extensions() {
			if err := validateForField(
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

func validateForMessage(
	add func(protosource.Descriptor, protosource.Location, []protosource.Location, string, ...interface{}),
	descriptorResolver protodesc.Resolver,
	message protosource.Message,
) error {
	if err := validateCELForMessage(
		add,
		descriptorResolver,
		message,
	); err != nil {
		return err
	}
	for _, nestedMessage := range message.Messages() {
		if err := validateForMessage(add, descriptorResolver, nestedMessage); err != nil {
			return err
		}
	}
	for _, field := range message.Fields() {
		if err := validateForField(
			add,
			descriptorResolver,
			field,
		); err != nil {
			return err
		}
	}
	for _, extension := range message.Extensions() {
		if err := validateForField(
			add,
			descriptorResolver,
			extension,
		); err != nil {
			return err
		}
	}
	return nil
}
