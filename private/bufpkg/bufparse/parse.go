// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufparse

import (
	"errors"
	"strings"
)

func parseFullNameComponents(path string) (registry string, owner string, name string, err error) {
	slashSplit := strings.Split(path, "/")
	if len(slashSplit) != 3 {
		return "", "", "", newInvalidFullNameStringError(path)
	}
	registry = strings.TrimSpace(slashSplit[0])
	if registry == "" {
		return "", "", "", newInvalidFullNameStringError(path)
	}
	owner = strings.TrimSpace(slashSplit[1])
	if owner == "" {
		return "", "", "", newInvalidFullNameStringError(path)
	}
	name = strings.TrimSpace(slashSplit[2])
	if name == "" {
		return "", "", "", newInvalidFullNameStringError(path)
	}
	return registry, owner, name, nil
}

func parseRefComponents(path string) (registry string, owner string, name string, ref string, err error) {
	// split by the first "/" to separate the registry and remaining part
	slashSplit := strings.SplitN(path, "/", 2)
	if len(slashSplit) != 2 {
		return "", "", "", "", newInvalidRefStringError(path)
	}
	registry, rest := slashSplit[0], slashSplit[1]
	// split the remaining part by ":" to separate the reference
	colonSplit := strings.Split(rest, ":")
	switch len(colonSplit) {
	case 1:
		// path excluding registry has no colon, no need to handle its ref
	case 2:
		ref = strings.TrimSpace(colonSplit[1])
		if ref == "" {
			return "", "", "", "", newInvalidRefStringError(path)
		}
	default:
		return "", "", "", "", newInvalidRefStringError(path)
	}
	registry, owner, name, err = parseFullNameComponents(registry + "/" + colonSplit[0])
	if err != nil {
		return "", "", "", "", newInvalidRefStringError(path)
	}
	return registry, owner, name, ref, nil
}

func newInvalidFullNameStringError(s string) error {
	return NewParseError(
		"full name",
		s,
		errors.New("must be in the form registry/owner/name"),
	)
}

func newInvalidRefStringError(s string) error {
	return NewParseError(
		"reference",
		s,
		errors.New("must be in the form registry/owner/name[:ref]"),
	)
}
