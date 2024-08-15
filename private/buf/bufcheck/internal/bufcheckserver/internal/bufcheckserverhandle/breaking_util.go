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

func getDescriptorAndLocationForDeletedElement(
	file bufprotosource.File,
	previousNestedName string,
) (bufprotosource.Descriptor, bufprotosource.Location, error) {
	if strings.Contains(previousNestedName, ".") {
		nestedNameToMessage, err := bufprotosource.NestedNameToMessage(file)
		if err != nil {
			return nil, nil, err
		}
		split := strings.Split(previousNestedName, ".")
		for i := len(split) - 1; i > 0; i-- {
			if message, ok := nestedNameToMessage[strings.Join(split[0:i], ".")]; ok {
				return message, message.Location(), nil
			}
		}
	}
	return file, nil, nil
}

func getDescriptorAndLocationForDeletedMessage(
	file bufprotosource.File,
	nestedNameToMessage map[string]bufprotosource.Message,
	previousNestedName string,
) (bufprotosource.Descriptor, bufprotosource.Location) {
	if strings.Contains(previousNestedName, ".") {
		split := strings.Split(previousNestedName, ".")
		for i := len(split) - 1; i > 0; i-- {
			if message, ok := nestedNameToMessage[strings.Join(split[0:i], ".")]; ok {
				return message, message.Location()
			}
		}
	}
	return file, nil
}

func withBackupLocation(locs ...bufprotosource.Location) bufprotosource.Location {
	for _, loc := range locs {
		if loc != nil {
			return loc
		}
	}
	return nil
}
