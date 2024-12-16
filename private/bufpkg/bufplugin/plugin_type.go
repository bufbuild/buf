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

package bufplugin

import (
	"fmt"
)

const (
	// PluginTypeCheck says the Plugin is a check plugin.
	PluginTypeCheck = iota + 1
)

var (
	// AllPluginTypeStrings is all format strings without aliases.
	//
	// Sorted in the order we want to display them.
	AllPluginTypeStrings = []string{
		"check",
	}
)

// PluginType is the type of a Plugin.
type PluginType int

// ParsePluginType parses the PluginType from the string.
func ParsePluginType(s string) (PluginType, error) {
	switch s {
	case "check":
		return PluginVisibilityPublic, nil
	default:
		return 0, fmt.Errorf("unknown PluginType: %q", s)
	}
}