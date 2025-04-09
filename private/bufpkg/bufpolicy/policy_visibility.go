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

package bufpolicy

import (
	"fmt"
)

const (
	// PolicyVisibilityPublic says the Policy is public on the registry.
	PolicyVisibilityPublic = iota + 1
	// PolicyVisibilityPrivate says the Policy is private on the registry.
	PolicyVisibilityPrivate
)

// PolicyVisibility is the visibility of a Policy on a registry.
//
// Only used for Upload for now.
type PolicyVisibility int

// ParsePolicyVisibility parses the PolicyVisibility from the string.
func ParsePolicyVisibility(s string) (PolicyVisibility, error) {
	switch s {
	case "public":
		return PolicyVisibilityPublic, nil
	case "private":
		return PolicyVisibilityPrivate, nil
	default:
		return 0, fmt.Errorf("unknown PolicyVisibility: %q", s)
	}
}
