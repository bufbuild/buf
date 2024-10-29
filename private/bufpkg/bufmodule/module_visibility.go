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

package bufmodule

import (
	"fmt"
)

const (
	// ModuleVisibilityPublic says the Module is public on the registry.
	ModuleVisibilityPublic = iota + 1
	// ModuleVisibilityPublic says the Module is private on the registry.
	ModuleVisibilityPrivate
)

// ModuleVisibility is the visibility of a Module on a registry.
//
// Only used for Upload for now.
type ModuleVisibility int

// ParseModuleVisibility parses the ModuleVisibility from the string.
func ParseModuleVisibility(s string) (ModuleVisibility, error) {
	switch s {
	case "public":
		return ModuleVisibilityPublic, nil
	case "private":
		return ModuleVisibilityPrivate, nil
	default:
		return 0, fmt.Errorf("unknown ModuleVisibility: %q", s)
	}
}
