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

package bufplugin

import (
	"fmt"
)

const (
	// PluginVisibilityPublic says the Plugin is public on the registry.
	PluginVisibilityPublic = iota + 1
	// PluginVisibilityPrivate says the Plugin is private on the registry.
	PluginVisibilityPrivate
)

// PluginVisibility is the visibility of a Plugin on a registry.
//
// Only used for Upload for now.
type PluginVisibility int

// ParsePluginVisibility parses the PluginVisibility from the string.
func ParsePluginVisibility(s string) (PluginVisibility, error) {
	switch s {
	case "public":
		return PluginVisibilityPublic, nil
	case "private":
		return PluginVisibilityPrivate, nil
	default:
		return 0, fmt.Errorf("unknown PluginVisibility: %q", s)
	}
}
