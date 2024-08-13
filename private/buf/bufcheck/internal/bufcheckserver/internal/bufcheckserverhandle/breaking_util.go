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
	"strings"

	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
)

func getLocationAndPreviousLocationForDeletedElement(
	file bufprotosource.File,
	previousFile bufprotosource.File,
	previousNestedName string,
) (bufprotosource.Location, bufprotosource.Location, error) {
	var location bufprotosource.Location
	var previousLocation bufprotosource.Location
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
		if err != nil {
			return nil, nil, err
		}
		previousNestedNameToMessage, err := bufprotosource.NestedNameToMessage(previousFile)
		if err != nil {
			return nil, nil, err
		}
		split := strings.Split(previousNestedName, ".")
		for i := len(split) - 1; i > 0; i-- {
			messageName := strings.Join(split[0:i], ".")
			if message, ok := nestedNameToMessage[messageName]; ok {
				location = message.Location()
			}
			if previousMessage, ok := previousNestedNameToMessage[messageName]; ok {
				previousLocation = previousMessage.Location()
			}
		}
	}
	return location, previousLocation, nil
}

func withBackupLocation(locs ...bufprotosource.Location) bufprotosource.Location {
	for _, loc := range locs {
		if loc != nil {
			return loc
		}
	}
	return nil
}
