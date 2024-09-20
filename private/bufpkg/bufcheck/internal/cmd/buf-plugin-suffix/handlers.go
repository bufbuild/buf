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

package main

import (
	"context"
	"slices"
	"strings"

	"buf.build/go/bufplugin/check"
	"buf.build/go/bufplugin/descriptor"
	"buf.build/go/bufplugin/option"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	serviceBannedSuffixesOptionKey   = "service_banned_suffixes"
	rpcBannedSuffixesOptionKey       = "rpc_banned_suffixes"
	fieldBannedSuffixesOptionKey     = "field_banned_suffixes"
	enumValueBannedSuffixesOptionKey = "enum_value_banned_suffixes"
	serviceNoChangeSuffixesOptionKey = "service_no_change_suffixes"
	messageNoChangeSuffixesOptionKey = "message_no_change_suffixes"
	enumNoChangeSuffixesOptionKey    = "enum_no_change_suffixes"
)

func handleLintServiceBannedSuffixes(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	bannedServiceSuffixes, err := getSuffixes(request, serviceBannedSuffixesOptionKey)
	if err != nil {
		return err
	}
	for _, fileDescriptor := range request.FileDescriptors() {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Services().Len(); i++ {
			service := descriptor.Services().Get(i)
			checkDescriptorBannedSuffixes(responseWriter, service, bannedServiceSuffixes, "Service")
		}
	}
	return nil
}

func handleLintRPCBannedSuffixes(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	bannedRPCSuffixes, err := getSuffixes(request, rpcBannedSuffixesOptionKey)
	if err != nil {
		return err
	}
	for _, fileDescriptor := range request.FileDescriptors() {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Services().Len(); i++ {
			methods := descriptor.Services().Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				method := methods.Get(j)
				checkDescriptorBannedSuffixes(responseWriter, method, bannedRPCSuffixes, "Method")
			}
		}
	}
	return nil
}

func handleLintFieldBannedSuffixes(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	bannedFieldSuffixes, err := getSuffixes(request, fieldBannedSuffixesOptionKey)
	if err != nil {
		return err
	}
	for _, fileDescriptor := range request.FileDescriptors() {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Messages().Len(); i++ {
			message := descriptor.Messages().Get(i)
			checkBannedFieldSuffixesForMessage(responseWriter, message, bannedFieldSuffixes)
		}
	}
	return nil
}

func checkBannedFieldSuffixesForMessage(
	responseWriter check.ResponseWriter,
	messageDescriptor protoreflect.MessageDescriptor,
	bannedFieldSuffixes []string,
) {
	// Check all fields of the message
	for i := 0; i < messageDescriptor.Fields().Len(); i++ {
		field := messageDescriptor.Fields().Get(i)
		checkDescriptorBannedSuffixes(responseWriter, field, bannedFieldSuffixes, "Field")
	}
	// Check each nested messages
	for i := 0; i < messageDescriptor.Messages().Len(); i++ {
		message := messageDescriptor.Messages().Get(i)
		checkBannedFieldSuffixesForMessage(responseWriter, message, bannedFieldSuffixes)
	}
}

func handleLintEnumValueBannedSuffixes(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	bannedEnumValueSuffixes, err := getSuffixes(request, enumValueBannedSuffixesOptionKey)
	if err != nil {
		return err
	}
	for _, fileDescriptor := range request.FileDescriptors() {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Enums().Len(); i++ {
			enum := descriptor.Enums().Get(i)
			for j := 0; j < enum.Values().Len(); j++ {
				enumValue := enum.Values().Get(j)
				checkDescriptorBannedSuffixes(responseWriter, enumValue, bannedEnumValueSuffixes, "Enum value")
			}
		}
		// Check messages for nested enums
		for i := 0; i < descriptor.Messages().Len(); i++ {
			message := descriptor.Messages().Get(i)
			checkBannedEnumValueSuffixesForMessage(responseWriter, message, bannedEnumValueSuffixes)
		}
	}
	return nil
}

func checkBannedEnumValueSuffixesForMessage(
	responseWriter check.ResponseWriter,
	messageDescriptor protoreflect.MessageDescriptor,
	bannedEnumValueSuffixes []string,
) {
	// Check each nested enum
	for i := 0; i < messageDescriptor.Enums().Len(); i++ {
		enum := messageDescriptor.Enums().Get(i)
		for j := 0; j < enum.Values().Len(); j++ {
			enumValue := enum.Values().Get(j)
			checkDescriptorBannedSuffixes(responseWriter, enumValue, bannedEnumValueSuffixes, "Enum value")
		}
	}
	// Check each nested message for nested enums
	for i := 0; i < messageDescriptor.Messages().Len(); i++ {
		message := messageDescriptor.Messages().Get(i)
		checkBannedEnumValueSuffixesForMessage(responseWriter, message, bannedEnumValueSuffixes)
	}
}

func checkDescriptorBannedSuffixes(
	responseWriter check.ResponseWriter,
	descriptor protoreflect.Descriptor,
	bannedSuffixes []string,
	descriptorTypeName string,
) {
	for _, bannedSuffix := range bannedSuffixes {
		if strings.HasSuffix(string(descriptor.FullName()), bannedSuffix) {
			responseWriter.AddAnnotation(
				check.WithDescriptor(descriptor),
				check.WithMessagef(
					"%s name %q has banned suffix %q.",
					descriptorTypeName,
					descriptor.FullName(),
					bannedSuffix,
				),
			)
		}
	}
}

func handleBreakingServiceSuffixesNoChange(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	serviceNoChangeSuffixes, err := getSuffixes(request, serviceNoChangeSuffixesOptionKey)
	if err != nil {
		return err
	}
	previousNoChangeServiceNameToServiceDescriptor := mapServiceNameToServiceDescriptorForFilesAndNoChangeSuffixes(
		request.AgainstFileDescriptors(),
		serviceNoChangeSuffixes,
	)
	currentNoChangeServiceNameToServiceDescriptor := mapServiceNameToServiceDescriptorForFilesAndNoChangeSuffixes(
		request.FileDescriptors(),
		nil,
	)
	for previousServiceName, previousServiceDescriptor := range previousNoChangeServiceNameToServiceDescriptor {
		if currentServiceDescriptor, ok := currentNoChangeServiceNameToServiceDescriptor[previousServiceName]; ok {
			previousMethodNames := getAllMethodNamesSortedForService(previousServiceDescriptor)
			currentMethodNames := getAllMethodNamesSortedForService(currentServiceDescriptor)
			if !slicesext.ElementsEqual(
				previousMethodNames,
				currentMethodNames,
			) {
				responseWriter.AddAnnotation(
					check.WithDescriptor(currentServiceDescriptor),
					check.WithAgainstDescriptor(previousServiceDescriptor),
					check.WithMessagef(
						"Service %q has a suffix configured for no changes has different methods, previously %s, currently %s.",
						previousServiceName,
						previousMethodNames,
						currentMethodNames,
					),
				)
			}
		} else {
			responseWriter.AddAnnotation(
				check.WithAgainstDescriptor(previousServiceDescriptor),
				check.WithMessagef(
					"Service %q has a suffix configured for no changes has been deleted.",
					previousServiceName,
				),
			)
		}
	}
	return nil
}

// mapServiceNameToServiceDescriptorForFilesAndNoChangeSuffixes maps the service name to
// service descriptors for the given check files based on the the no change suffixes.
// If no suffixes are passed, then all services are returned.
func mapServiceNameToServiceDescriptorForFilesAndNoChangeSuffixes(
	fileDescriptors []descriptor.FileDescriptor,
	serviceNoChangeSuffixes []string,
) map[string]protoreflect.ServiceDescriptor {
	result := map[string]protoreflect.ServiceDescriptor{}
	for _, fileDescriptor := range fileDescriptors {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Services().Len(); i++ {
			service := descriptor.Services().Get(i)
			if checkDescriptorHasNoChangeSuffix(service, serviceNoChangeSuffixes) {
				result[string(service.FullName())] = service
			}
		}
	}
	return result
}

func getAllMethodNamesSortedForService(serviceDescriptor protoreflect.ServiceDescriptor) []string {
	var methodNames []string
	for i := 0; i < serviceDescriptor.Methods().Len(); i++ {
		method := serviceDescriptor.Methods().Get(i)
		methodNames = append(methodNames, string(method.FullName()))
	}
	slices.Sort(methodNames)
	return methodNames
}

func handleBreakingMessageSuffixesNoChange(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	messageNoChangeSuffixes, err := getSuffixes(request, messageNoChangeSuffixesOptionKey)
	if err != nil {
		return err
	}
	previousNoChangeMessageNameToMessageDescriptor := mapMessageNameToMessageDescriptorForFilesAndNoChangeSuffixes(
		request.AgainstFileDescriptors(),
		messageNoChangeSuffixes,
	)
	currentNoChangeMessageNameToMessageDescriptor := mapMessageNameToMessageDescriptorForFilesAndNoChangeSuffixes(
		request.FileDescriptors(),
		messageNoChangeSuffixes,
	)
	for previousMessageName, previousMessageDescriptor := range previousNoChangeMessageNameToMessageDescriptor {
		if currentMessageDescriptor, ok := currentNoChangeMessageNameToMessageDescriptor[previousMessageName]; ok {
			previousFieldNames := getFieldNamesSortedForMessage(previousMessageDescriptor)
			currentFieldNames := getFieldNamesSortedForMessage(currentMessageDescriptor)
			if !slicesext.ElementsEqual(
				previousFieldNames,
				currentFieldNames,
			) {
				responseWriter.AddAnnotation(
					check.WithDescriptor(currentMessageDescriptor),
					check.WithAgainstDescriptor(previousMessageDescriptor),
					check.WithMessagef(
						"Message %q has a suffix configured for no changes has different fields, previously %s, currently %s.",
						previousMessageName,
						previousFieldNames,
						currentFieldNames,
					),
				)
			}
		} else {
			responseWriter.AddAnnotation(
				check.WithAgainstDescriptor(previousMessageDescriptor),
				check.WithMessagef(
					"Message %q has a suffix configured for no changes has been deleted.",
					previousMessageName,
				),
			)
		}
	}
	return nil
}

// mapMessageNameToMessageDescriptorForFilesAndNoChangeSuffixes maps the message name to
// message descriptors for the given check files based on the the no change suffixes.
// If no suffixes are passed, then all messages are returned.
func mapMessageNameToMessageDescriptorForFilesAndNoChangeSuffixes(
	fileDescriptors []descriptor.FileDescriptor,
	messageNoChangeSuffixes []string,
) map[string]protoreflect.MessageDescriptor {
	result := map[string]protoreflect.MessageDescriptor{}
	for _, fileDescriptor := range fileDescriptors {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		messages := getNestedMessageDescriptors(descriptor.Messages(), messageNoChangeSuffixes)
		for _, message := range messages {
			result[string(message.FullName())] = message
		}
	}
	return result
}

func getNestedMessageDescriptors(
	messageDescriptors protoreflect.MessageDescriptors,
	messageNoChangeSuffixes []string,
) []protoreflect.MessageDescriptor {
	var messages []protoreflect.MessageDescriptor
	for i := 0; i < messageDescriptors.Len(); i++ {
		message := messageDescriptors.Get(i)
		if checkDescriptorHasNoChangeSuffix(message, messageNoChangeSuffixes) {
			messages = append(messages, message)
		}
		nestedMessages := getNestedMessageDescriptors(message.Messages(), messageNoChangeSuffixes)
		messages = append(messages, nestedMessages...)
	}
	return messages
}

func getFieldNamesSortedForMessage(messageDescriptor protoreflect.MessageDescriptor) []string {
	var fieldNames []string
	for i := 0; i < messageDescriptor.Fields().Len(); i++ {
		field := messageDescriptor.Fields().Get(i)
		fieldNames = append(fieldNames, string(field.FullName()))
	}
	slices.Sort(fieldNames)
	return fieldNames
}

func handleBreakingEnumSuffixesNoChange(
	ctx context.Context,
	responseWriter check.ResponseWriter,
	request check.Request,
) error {
	enumNoChangeSuffixes, err := getSuffixes(request, enumNoChangeSuffixesOptionKey)
	if err != nil {
		return err
	}
	previousNoChangeEnumNameToEnumDescriptor := mapEnumNameToEnumDescriptorForFilesAndNoChangeSuffixes(
		request.AgainstFileDescriptors(),
		enumNoChangeSuffixes,
	)
	currentNoChangeEnumNameToEnumDescriptor := mapEnumNameToEnumDescriptorForFilesAndNoChangeSuffixes(
		request.FileDescriptors(),
		enumNoChangeSuffixes,
	)
	for previousEnumName, previousEnumDescriptor := range previousNoChangeEnumNameToEnumDescriptor {
		if currentEnumDescriptor, ok := currentNoChangeEnumNameToEnumDescriptor[previousEnumName]; ok {
			previousEnumValueNames := getEnumValueNamesSortedForEnum(previousEnumDescriptor)
			currentEnumValueNames := getEnumValueNamesSortedForEnum(currentEnumDescriptor)
			if !slicesext.ElementsEqual(
				previousEnumValueNames,
				currentEnumValueNames,
			) {
				responseWriter.AddAnnotation(
					check.WithDescriptor(currentEnumDescriptor),
					check.WithAgainstDescriptor(previousEnumDescriptor),
					check.WithMessagef(
						"Enum %q has a suffix configured for no changes has different enum values, previously %s, currently %s.",
						previousEnumName,
						previousEnumValueNames,
						currentEnumValueNames,
					),
				)
			}
		} else {
			responseWriter.AddAnnotation(
				check.WithAgainstDescriptor(previousEnumDescriptor),
				check.WithMessagef(
					"Enum %q has a suffix configured for no changes has been deleted.",
					previousEnumName,
				),
			)
		}
	}
	return nil
}

func mapEnumNameToEnumDescriptorForFilesAndNoChangeSuffixes(
	fileDescriptors []descriptor.FileDescriptor,
	enumNoChangeSuffixes []string,
) map[string]protoreflect.EnumDescriptor {
	result := map[string]protoreflect.EnumDescriptor{}
	for _, fileDescriptor := range fileDescriptors {
		descriptor := fileDescriptor.ProtoreflectFileDescriptor()
		for i := 0; i < descriptor.Enums().Len(); i++ {
			enum := descriptor.Enums().Get(i)
			if checkDescriptorHasNoChangeSuffix(enum, enumNoChangeSuffixes) {
				result[string(enum.FullName())] = enum
			}
		}
		enums := getNestedEnumDescriptors(descriptor.Messages(), enumNoChangeSuffixes)
		for _, enum := range enums {
			result[string(enum.FullName())] = enum
		}
	}
	return result
}

func getEnumValueNamesSortedForEnum(enumDescriptor protoreflect.EnumDescriptor) []string {
	var enumValueNames []string
	for i := 0; i < enumDescriptor.Values().Len(); i++ {
		enumValue := enumDescriptor.Values().Get(i)
		enumValueNames = append(enumValueNames, string(enumValue.FullName()))
	}
	slices.Sort(enumValueNames)
	return enumValueNames
}

func getNestedEnumDescriptors(
	messageDescriptors protoreflect.MessageDescriptors,
	enumNoChangeSuffixes []string,
) []protoreflect.EnumDescriptor {
	var enums []protoreflect.EnumDescriptor
	for i := 0; i < messageDescriptors.Len(); i++ {
		message := messageDescriptors.Get(i)
		for j := 0; j < message.Enums().Len(); j++ {
			enum := message.Enums().Get(j)
			if checkDescriptorHasNoChangeSuffix(enum, enumNoChangeSuffixes) {
				enums = append(enums, enum)
			}
		}
		nestedEnums := getNestedEnumDescriptors(message.Messages(), enumNoChangeSuffixes)
		enums = append(enums, nestedEnums...)
	}
	return enums
}

func checkDescriptorHasNoChangeSuffix(
	descriptor protoreflect.Descriptor,
	noChangeSuffixes []string,
) bool {
	if noChangeSuffixes == nil {
		return true
	}
	for _, noChangeSuffix := range noChangeSuffixes {
		if strings.HasSuffix(string(descriptor.FullName()), noChangeSuffix) {
			return true
		}
	}
	return false
}

func getSuffixes(
	request check.Request,
	optionKey string,
) ([]string, error) {
	return option.GetStringSliceValue(request.Options(), optionKey)
}
