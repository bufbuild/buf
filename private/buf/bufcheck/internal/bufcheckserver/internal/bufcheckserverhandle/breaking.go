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

	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

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
	for previousNestedName := range previousNestedNameToEnum {
		if _, ok := nestedNameToEnum[previousNestedName]; !ok {
			location, previousLocation, err := getLocationAndPreviousLocationForDeletedElement(
				file,
				previousFile,
				previousNestedName,
			)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousLocation,
				`Previously present enum %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

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
	for previousNestedName := range previousNestedNameToExtension {
		if _, ok := nestedNameToExtension[previousNestedName]; !ok {
			location, previousLocation, err := getLocationAndPreviousLocationForDeletedElement(
				file,
				previousFile,
				previousNestedName,
			)
			if err != nil {
				return err
			}
			responseWriter.AddProtosourceAnnotation(
				location,
				previousLocation,
				`Previously present extension %q was deleted from file.`,
				previousNestedName,
			)
		}
	}
	return nil
}

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
	// TODO(doria): we could probably simplify the bufprotosource functions since we don't
	// need the file here anymore.
	for previousFilePath, _ := range previousFilePathToFile {
		if _, ok := filePathToFile[previousFilePath]; !ok {
			// Add previous descriptor to check for ignores. This will mean that if
			// we have ignore_unstable_packages set, this file will cause the ignore
			// to happen.
			responseWriter.AddProtosourceAnnotation(
				nil,
				nil,
				`Previously present file %q was deleted.`,
				previousFilePath,
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
