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

package bufmoduleref

import (
	"fmt"
	"strings"
)

// parseModuleReferenceComponents parses and returns the remote, owner, repository,
// and ref (branch, commit, draft, or tag) from the given path.
func parseModuleReferenceComponents(path string) (remote string, owner string, repository string, ref string, err error) {
	// split by the first "/" to separate the remote and remaining part
	slashSplit := strings.SplitN(path, "/", 2)
	if len(slashSplit) != 2 {
		return "", "", "", "", newInvalidModuleReferenceStringError(path)
	}
	remote, rest := slashSplit[0], slashSplit[1]
	// split the remaining part by ":" to separate the reference
	colonSplit := strings.Split(rest, ":")
	switch len(colonSplit) {
	case 1:
		// path excluding remote has no colon, no need to handle its ref
	case 2:
		ref = strings.TrimSpace(colonSplit[1])
		if ref == "" {
			return "", "", "", "", newInvalidModuleReferenceStringError(path)
		}
	default:
		return "", "", "", "", newInvalidModuleReferenceStringError(path)
	}
	remote, owner, repository, err = parseModuleIdentityComponents(remote + "/" + colonSplit[0])
	if err != nil {
		return "", "", "", "", newInvalidModuleReferenceStringError(path)
	}
	return remote, owner, repository, ref, nil
}

func parseModuleIdentityComponents(path string) (remote string, owner string, repository string, err error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 3 {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	remote = strings.TrimSpace(slashSplit[0])
	if remote == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	owner = strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	repository = strings.TrimSpace(slashSplit[2])
	if repository == "" {
		return "", "", "", newInvalidModuleIdentityStringError(path)
	}
	return remote, owner, repository, nil
}

func modulePinLess(a ModulePin, b ModulePin) bool {
	return modulePinCompareTo(a, b) < 0
}

// return -1 if less
// return 1 if greater
// return 0 if equal
func modulePinCompareTo(a ModulePin, b ModulePin) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil && b != nil {
		return -1
	}
	if a != nil && b == nil {
		return 1
	}
	if a.Remote() < b.Remote() {
		return -1
	}
	if a.Remote() > b.Remote() {
		return 1
	}
	if a.Owner() < b.Owner() {
		return -1
	}
	if a.Owner() > b.Owner() {
		return 1
	}
	if a.Repository() < b.Repository() {
		return -1
	}
	if a.Repository() > b.Repository() {
		return 1
	}
	if a.Commit() < b.Commit() {
		return -1
	}
	if a.Commit() > b.Commit() {
		return 1
	}
	if a.Digest() < b.Digest() {
		return -1
	}
	if a.Digest() > b.Digest() {
		return 1
	}
	return 0
}

func newInvalidModuleOwnerStringError(s string) error {
	return fmt.Errorf("module owner %q is invalid: must be in the form remote/owner", s)
}

func newInvalidModuleIdentityStringError(s string) error {
	return fmt.Errorf("module identity %q is invalid: must be in the form remote/owner/repository", s)
}

func newInvalidModuleReferenceStringError(s string) error {
	return fmt.Errorf("module reference %q is invalid: must be in the form remote/owner/repository:reference", s)
}
