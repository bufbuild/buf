// Copyright 2020-2022 Buf Technologies, Inc.
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

package appcmd

import (
	"fmt"
	"strings"
)

func ValidateRemoteNotEmpty(remote string) error {
	if remote == "" {
		return NewInvalidArgumentError("you must specify a remote module")
	}
	return nil
}

func ValidateRemoteHasNoPaths(remote string) error {
	parts := strings.Split(remote, "/")
	if len(parts) > 1 {
		paths := "/" + strings.Join(parts[1:], "/")
		// The remote has no paths, but ends in a /, which is valid
		if paths == "/" {
			return nil
		}
		return NewInvalidArgumentError(fmt.Sprintf(`invalid remote address, must not contain any paths. Try removing "%s" from the address.`, paths))
	}
	return nil
}
