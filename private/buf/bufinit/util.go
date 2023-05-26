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

package bufinit

import (
	"errors"
	"fmt"

	"github.com/bufbuild/buf/private/pkg/normalpath"
)

func normalizeAndValidateProtoFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty path")
	}
	if path == "." {
		return "", fmt.Errorf("path cannot be '.'")
	}
	return normalpath.NormalizeAndValidate(path)
}

func reverseComponents(path string) []string {
	components := normalpath.Components(path)
	// https://github.com/golang/go/wiki/SliceTricks#reversing
	for i := len(components)/2 - 1; i >= 0; i-- {
		opp := len(components) - 1 - i
		components[i], components[opp] = components[opp], components[i]
	}
	return components
}
