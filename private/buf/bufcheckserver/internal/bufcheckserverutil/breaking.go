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

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/bufplugin-go/check"
)

// NewBreakingFilePairRuleHandler returns a new check.RuleHandler for the given function.
func NewBreakingFilePairRuleHandler(
	f func(
		responseWriter ResponseWriter,
		request Request,
		previousFile bufprotosource.File,
		file bufprotosource.File,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			previousFilePathToFile, err := bufprotosource.FilePathToFile(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			filePathToFile, err := bufprotosource.FilePathToFile(request.ProtosourceFiles()...)
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
		previousEnum bufprotosource.Enum,
		enum bufprotosource.Enum,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			previousFullNameToEnum, err := bufprotosource.FullNameToEnum(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			fullNameToEnum, err := bufprotosource.FullNameToEnum(request.ProtosourceFiles()...)
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
		previousNameToEnumValue map[string]bufprotosource.EnumValue,
		nameToEnumValue map[string]bufprotosource.EnumValue,
	) error,
) check.RuleHandler {
	return NewBreakingEnumPairRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			previousEnum bufprotosource.Enum,
			enum bufprotosource.Enum,
		) error {
			previousNumberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(previousEnum)
			if err != nil {
				return err
			}
			numberToNameToEnumValue, err := bufprotosource.NumberToNameToEnumValue(enum)
			if err != nil {
				return err
			}
			for previousNumber, previousNameToEnumValue := range previousNumberToNameToEnumValue {
				if nameToEnumValue, ok := numberToNameToEnumValue[previousNumber]; ok {
					if err := f(responseWriter, request, previousNameToEnumValue, nameToEnumValue); err != nil {
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
		previousMessage bufprotosource.Message,
		message bufprotosource.Message,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			previousFullNameToMessage, err := bufprotosource.FullNameToMessage(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := bufprotosource.FullNameToMessage(request.ProtosourceFiles()...)
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
		previousField bufprotosource.Field,
		field bufprotosource.Field,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			// Fields on messages.
			previousFullNameToMessage, err := bufprotosource.FullNameToMessage(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			fullNameToMessage, err := bufprotosource.FullNameToMessage(request.ProtosourceFiles()...)
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
							if err := f(responseWriter, request, previousField, field); err != nil {
								return err
							}
						}
					}
				}
			}
			// Extensions.
			previousTypeToNumberToField := make(map[string]map[int]bufprotosource.Field)
			for _, previousFile := range request.AgainstProtosourceFiles() {
				if err := addToTypeToNumberToExtension(previousFile, previousTypeToNumberToField); err != nil {
					return err
				}
			}
			typeToNumberToField := make(map[string]map[int]bufprotosource.Field)
			for _, file := range request.ProtosourceFiles() {
				if err := addToTypeToNumberToExtension(file, typeToNumberToField); err != nil {
					return err
				}
			}
			for previousType, previousNumberToField := range previousTypeToNumberToField {
				numberToField := typeToNumberToField[previousType]
				for previousNumber, previousField := range previousNumberToField {
					if field, ok := numberToField[previousNumber]; ok {
						if err := f(responseWriter, request, previousField, field); err != nil {
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
		previousService bufprotosource.Service,
		service bufprotosource.Service,
	) error,
) check.RuleHandler {
	return NewRuleHandler(
		func(
			_ context.Context,
			responseWriter ResponseWriter,
			request Request,
		) error {
			previousFullNameToService, err := bufprotosource.FullNameToService(request.AgainstProtosourceFiles()...)
			if err != nil {
				return err
			}
			fullNameToService, err := bufprotosource.FullNameToService(request.ProtosourceFiles()...)
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
		previousMethod bufprotosource.Method,
		method bufprotosource.Method,
	) error,
) check.RuleHandler {
	return NewBreakingServicePairRuleHandler(
		func(
			responseWriter ResponseWriter,
			request Request,
			previousService bufprotosource.Service,
			service bufprotosource.Service,
		) error {
			previousNameToMethod, err := bufprotosource.NameToMethod(previousService)
			if err != nil {
				return err
			}
			nameToMethod, err := bufprotosource.NameToMethod(service)
			if err != nil {
				return err
			}
			for previousName, previousMethod := range previousNameToMethod {
				if method, ok := nameToMethod[previousName]; ok {
					if err := f(responseWriter, request, previousMethod, method); err != nil {
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
