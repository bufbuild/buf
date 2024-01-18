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

package bufremoteplugin

import (
	"fmt"
	"strings"

	"github.com/bufbuild/buf/private/pkg/app/appcmd"
)

const (
	// PluginsPathName is the path prefix used to signify that
	// a name belongs to a plugin.
	PluginsPathName = "plugins"
)

// ParsePluginPath parses a string in the format <buf.build/owner/plugins/name>
// into remote, owner and name.
func ParsePluginPath(pluginPath string) (remote string, owner string, name string, _ error) {
	if pluginPath == "" {
		return "", "", "", appcmd.NewInvalidArgumentError("you must specify a plugin path")
	}
	components := strings.Split(pluginPath, "/")
	if len(components) != 4 || components[2] != PluginsPathName {
		return "", "", "", appcmd.NewInvalidArgumentErrorf("%s is not a valid plugin path", pluginPath)
	}
	return components[0], components[1], components[3], nil
}

// ParsePluginVersionPath parses a string in the format <buf.build/owner/plugins/name[:version]>
// into remote, owner, name and version. The version is empty if not specified.
func ParsePluginVersionPath(pluginVersionPath string) (remote string, owner string, name string, version string, _ error) {
	remote, owner, name, err := ParsePluginPath(pluginVersionPath)
	if err != nil {
		return "", "", "", "", err
	}
	components := strings.Split(name, ":")
	switch len(components) {
	case 2:
		return remote, owner, components[0], components[1], nil
	case 1:
		return remote, owner, name, "", nil
	default:
		return "", "", "", "", fmt.Errorf("invalid version: %q", name)
	}
}
