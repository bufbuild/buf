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

package bufcheckserverutil

import (
	"context"
	"fmt"

	"buf.build/go/bufplugin/check"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

// NewBreakingFilePairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingFilePairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		file bufprotosource.File,
		previousFile bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			filePathToFile, err := bufprotosource.FilePathToFile(request.ProtosourceFiles()...)
			if err != nil {
				return err
			}
			previousFilePathToFile, err := bufprotosource.FilePathToFile(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			for previousFilePath, previousFile := range previousFilePathToFile {
				if file, ok := filePathToFile[previousFilePath]; ok {
					if err := f(responseWriter, request, file, previousFile); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingEnumPairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingEnumPairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		enum bufprotosource.Enum,
		previousEnum bufprotosource.Enum,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			fullNameToEnum, err := bufprotosource.FullNameToEnum(request.ProtosourceFiles()...)
			if err != nil {
				return err
			}
			previousFullNameToEnum, err := bufprotosource.FullNameToEnum(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			for previousFullName, previousEnum := range previousFullNameToEnum {
				if enum, ok := fullNameToEnum[previousFullName]; ok {
					if err := f(responseWriter, request, enum, previousEnum); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingEnumValuePairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingEnumValuePairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		nameToEnumValue map[string]bufprotosource.EnumValue,
		previousNameToEnumValue map[string]bufprotosource.EnumValue,
	) error,
) check.RuleHandler {
	return NewBreakingEnumPairRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			enum bufprotosource.Enum,
			previousEnum bufprotosource.Enum,
		) error {
			numberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(enum)
			if err != nil {
				return err
			}
			previousNumberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			for previousNumber, previousNameToEnumValue := range previousNumberToNameToEnumValue {
				if nameToEnumValue, ok := numberToNameToEnumValue[previousNumber]; ok {
					if err := f(responseWriter, request, nameToEnumValue, previousNameToEnumValue); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingMessagePairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingMessagePairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		message bufprotosource.Message,
		previousMessage bufprotosource.Message,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			fullNameToMessage, err := bufprotosource.FullNameToMessage(request.ProtosourceFiles()...)
			if err != nil {
				return err
			}
			previousFullNameToMessage, err := bufprotosource.FullNameToMessage(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			for previousFullName, previousMessage := range previousFullNameToMessage {
				if message, ok := fullNameToMessage[previousFullName]; ok {
					if err := f(responseWriter, request, message, previousMessage); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingFieldPairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingFieldPairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		field bufprotosource.Field,
		previousField bufprotosource.Field,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			// Fields on messages.
			fullNameToMessage, err := bufprotosource.FullNameToMessage(request.ProtosourceFiles()...)
			if err != nil {
				return err
			}
			previousFullNameToMessage, err := bufprotosource.FullNameToMessage(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			for previousFullName, previousMessage := range previousFullNameToMessage {
				if message, ok := fullNameToMessage[previousFullName]; ok {
					previousNumberToField, err := bufprotosource.NumberToMessageField(previousMessage)
					if err != nil {
						return err
					}
					numberToField, err := bufprotosource.NumberToMessageField(message)
					if err != nil {
						return err
					}
					for previousNumber, previousField := range previousNumberToField {
						if field, ok := numberToField[previousNumber]; ok {
							if err := f(responseWriter, request, field, previousField); err != nil {
								return err
							}
						}
					}
				}
			}
			// Extensions.
			typeToNumberToField := make(map[string]map[int]bufprotosource.Field)
			for _, file := range request.ProtosourceFiles() {
				if err := addToTypeToNumberToExtension(file, typeToNumberToField); err != nil {
					return err
				}
			}
			previousTypeToNumberToField := make(map[string]map[int]bufprotosource.Field)
			for _, previousFile := range request.AgainstProtosourceFiles() {
				if err := addToTypeToNumberToExtension(previousFile, previousTypeToNumberToField); err != nil {
					return err
				}
			}
			for previousType, previousNumberToField := range previousTypeToNumberToField {
				numberToField := typeToNumberToField[previousType]
				for previousNumber, previousField := range previousNumberToField {
					if field, ok := numberToField[previousNumber]; ok {
						if err := f(responseWriter, request, field, previousField); err != nil {
							return err
						}
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingServicePairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingServicePairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		service bufprotosource.Service,
		previousService bufprotosource.Service,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			fullNameToService, err := bufprotosource.FullNameToService(request.ProtosourceFiles()...)
			if err != nil {
				return err
			}
			previousFullNameToService, err := bufprotosource.FullNameToService(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			for previousFullName, previousService := range previousFullNameToService {
				if service, ok := fullNameToService[previousFullName]; ok {
					if err := f(responseWriter, request, service, previousService); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// NewBreakingMethodPairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingMethodPairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		method bufprotosource.Method,
		previousMethod bufprotosource.Method,
	) error,
) check.RuleHandler {
	return NewBreakingServicePairRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			service bufprotosource.Service,
			previousService bufprotosource.Service,
		) error {
			nameToMethod, err := bufprotosource.NameToMethod(service)
			if err != nil {
				return err
			}
			previousNameToMethod, err := bufprotosource.NameToMethod(previousService)
			if err != nil {
				return err
			}
			for previousName, previousMethod := range previousNameToMethod {
				if method, ok := nameToMethod[previousName]; ok {
					if err := f(responseWriter, request, method, previousMethod); err != nil {
						return err
					}
				}
			}
			return nil
		},
	)
}

// *** PRIVATE ***

func addToTypeToNumberToExtension(container bufprotosource.ContainerDescriptor, typeToNumberToExt map[string]map[int]bufprotosource.Field) error {
	for _, extension := range container.Extensions() {
		numberToExt := typeToNumberToExt[extension.Extendee()]
		if numberToExt == nil {
			numberToExt = make(map[int]bufprotosource.Field)
			typeToNumberToExt[extension.Extendee()] = numberToExt
		}
		if existing, ok := numberToExt[extension.Number()]; ok {
			return fmt.Errorf("duplicate extension %d of %s: %s in %q and %s in %q",
				extension.Number(), extension.Extendee(),
				existing.FullName(), existing.File().Path(),
				extension.FullName(), extension.File().Path())
		}
		numberToExt[extension.Number()] = extension
	}
	for _, message := range container.Messages() {
		if err := addToTypeToNumberToExtension(message, typeToNumberToExt); err != nil {
			return err
		}
	}
	return nil
}
