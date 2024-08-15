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

package bufcheckserverhandle

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

// HandleBreakingEnumNoDelete is a check function.
var HandleBreakingEnumNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingEnumNoDelete)

func handleBreakingEnumNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToEnum, err := bufprotosource.NestedNameToEnum(previousFile)
	if err != nil {
		return err
	}
	nestedNameToEnum, err := bufprotosource.NestedNameToEnum(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousEnum := range previousNestedNameToEnum {
		if _, ok := nestedNameToEnum[previousNestedName]; !ok {
			// TODO: search for enum in other files and return that the enum was moved?
			_, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousEnum.Location(),
				`Previously present enum %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingExtensionNoDelete is a check function.
var HandleBreakingExtensionNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingExtensionNoDelete)

func handleBreakingExtensionNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToExtension, err := bufprotosource.NestedNameToExtension(previousFile)
	if err != nil {
		return err
	}
	nestedNameToExtension, err := bufprotosource.NestedNameToExtension(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousExtension := range previousNestedNameToExtension {
		if _, ok := nestedNameToExtension[previousNestedName]; !ok {
			_, location, err := getDescriptorAndLocationForDeletedElement(file, previousNestedName)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousExtension.Location(),
				`Previously present extension %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingFileDelete is a check function.
var HandleBreakingFileNoDelete = bufcheckserverutil.NewRuleHandler(handleBreakingFileNoDelete)

func handleBreakingFileNoDelete(
	_ context.Context,
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
) error {
	previousFilePathToFile, err := bufprotosource.FilePathToFile(request.AgainstProtosourceFiles()...)
	if err != nil {
		return err
	}
	filePathToFile, err := bufprotosource.FilePathToFile(request.ProtosourceFiles()...)
	if err != nil {
		return err
	}
	for previousFilePath := range previousFilePathToFile {
		if _, ok := filePathToFile[previousFilePath]; !ok {
			// Add previous descriptor to check for ignores. This will mean that if
			// we have ignore_unstable_packages set, this file will cause the ignore
			// to happen.
			responseWriter.AddProtosourceAnnotation(
				nil,
				nil, // TODO: File does not have a Location, make sure that client handles the ignore checks
				`Previously present file %q was deleted.`,
				previousFilePath,
			)
		}
	}
	return nil
}

// HandleBreakingMessageNoDelete is a check function.
var HandleBreakingMessageNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingMessageNoDelete)

func handleBreakingMessageNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNestedNameToMessage, err := bufprotosource.NestedNameToMessage(previousFile)
	if err != nil {
		return err
	}
	nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
	if err != nil {
		return err
	}
	for previousNestedName, previousMessage := range previousNestedNameToMessage {
		if _, ok := nestedNameToMessage[previousNestedName]; !ok {
			_, location := getDescriptorAndLocationForDeletedMessage(file, nestedNameToMessage, previousNestedName)
			responseWriter.AddProtosourceAnnotation(
				location,
				previousMessage.Location(),
				`Previously present message %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

// HandleBreakingServiceNoDelete is a check function.
var HandleBreakingServiceNoDelete = bufcheckserverutil.NewBreakingFilePairRuleHandler(handleBreakingServiceNoDelete)

func handleBreakingServiceNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousFile bufprotosource.File,
	file bufprotosource.File,
) error {
	previousNameToService, err := bufprotosource.NameToService(previousFile)
	if err != nil {
		return err
	}
	nameToService, err := bufprotosource.NameToService(file)
	if err != nil {
		return err
	}
	for previousName, previousService := range previousNameToService {
		if _, ok := nameToService[previousName]; !ok {
			responseWriter.AddProtosourceAnnotation(
				nil,
				previousService.Location(),
				`Previously present service %q was deleted from file.`,
				previousName,
			)
		}
	}
	return nil
}

// HandleBreakingEnumSameType is a check function.
var HandleBreakingEnumSameType = bufcheckserverutil.NewBreakingEnumPairRuleHandler(handleBreakingEnumSameType)

func handleBreakingEnumSameType(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
) error {
	previousDescriptor, err := previousEnum.AsDescriptor()
	if err != nil {
		return err
	}
	descriptor, err := enum.AsDescriptor()
	if err != nil {
		return err
	}
	if previousDescriptor.IsClosed() != descriptor.IsClosed() {
		previousState, currentState := "closed", "open"
		if descriptor.IsClosed() {
			previousState, currentState = currentState, previousState
		}
		responseWriter.AddProtosourceAnnotation(
			withBackupLocation(enum.Features().EnumTypeLocation(), enum.Location()),
			withBackupLocation(previousEnum.Features().EnumTypeLocation(), previousEnum.Location()),
			`Enum %q changed from %s to %s.`,
			enum.Name(),
			previousState,
			currentState,
		)
	}
	return nil
}

// HandleBreakingEnumValueNoDelete is a check function.
var HandleBreakingEnumValueNoDelete = bufcheckserverutil.NewBreakingEnumPairRuleHandler(handleBreakingEnumValueNoDelete)

func handleBreakingEnumValueNoDelete(
	responseWriter bufcheckserverutil.ResponseWriter,
	request bufcheckserverutil.Request,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
) error {
	return checkEnumValueNoDeleteWithRules(
		responseWriter,
		previousEnum,
		enum,
		false,
		false,
	)
}

func checkEnumValueNoDeleteWithRules(
	responseWriter bufcheckserverutil.ResponseWriter,
	previousEnum bufprotosource.Enum,
	enum bufprotosource.Enum,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
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
		if _, ok := numberToNameToEnumValue[previousNumber]; !ok {
			if !isDeletedEnumValueAllowedWithRules(
				previousNumber,
				previousNameToEnumValue,
				enum,
				allowIfNumberReserved,
				allowIfNameReserved,
			) {
				suffix := ""
				if allowIfNumberReserved && allowIfNameReserved {
					return errors.New("both allowIfNumberReserved and allowIfNameReserved set")
				}
				if allowIfNumberReserved {
					suffix = fmt.Sprintf(` without reserving the number "%d"`, previousNumber)
				}
				if allowIfNameReserved {
					nameSuffix := ""
					if len(previousNameToEnumValue) > 1 {
						nameSuffix = "s"
					}
					suffix = fmt.Sprintf(` without reserving the name%s %s`, nameSuffix, stringutil.JoinSliceQuoted(getSortedEnumValueNames(previousNameToEnumValue), ", "))
				}
				responseWriter.AddProtosourceAnnotation(
					enum.Location(),
					previousEnum.Location(),
					`Previously present enum value "%d" on enum %q was deleted%s.`,
					previousNumber,
					enum.Name(),
					suffix,
				)
			}
		}
	}
	return nil
}

func isDeletedEnumValueAllowedWithRules(
	previousNumber int,
	previousNameToEnumValue map[string]bufprotosource.EnumValue,
	enum bufprotosource.Enum,
	allowIfNumberReserved bool,
	allowIfNameReserved bool,
) bool {
	if allowIfNumberReserved {
		return bufprotosource.NumberInReservedRanges(previousNumber, enum.ReservedTagRanges()...)
	}
	if allowIfNameReserved {
		// if true for all names, then ok
		for previousName := range previousNameToEnumValue {
			if !bufprotosource.NameInReservedNames(previousName, enum.ReservedNames()...) {
				return false
			}
		}
		return true
	}
	return false
}
