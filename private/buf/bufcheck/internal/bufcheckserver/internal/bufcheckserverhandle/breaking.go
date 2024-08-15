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
	"github.com/bufbuild/buf/private/buf/bufcheck/internal/bufcheckserver/internal/bufcheckserverutil"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

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
