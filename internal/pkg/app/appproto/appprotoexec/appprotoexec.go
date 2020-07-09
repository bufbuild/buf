// Copyright 2020 Buf Technologies, Inc.
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

// Package appprotoexec provides appproto.Handlers for binary plugins.
package appprotoexec

import (
	"fmt"
	"os/exec"

	"github.com/bufbuild/buf/internal/pkg/app/appproto"
	"go.uber.org/zap"
)

const (
	// DefaultMajorVersion is the default major version.
	DefaultMajorVersion = 3
	// DefaultMinorVersion is the default minor version.
	DefaultMinorVersion = 12
	// DefaultPatchVersion is the default patch version.
	DefaultPatchVersion = 3
	// DefaultSuffixVersion is the default suffix version.
	DefaultSuffixVersion = ""
)

var (
	// ProtocProxyPluginNames are the names of the plugins that should be proxied through protoc
	// in the absense of a binary.
	ProtocProxyPluginNames = map[string]struct{}{
		"cpp":    {},
		"csharp": {},
		"java":   {},
		"js":     {},
		"objc":   {},
		"php":    {},
		"python": {},
		"ruby":   {},
	}
)

// NewHandler returns a new Handler based on the plugin name and optional path.
//
// protocPath and pluginPath are optional.
//
// - If the plugin path is set, this returns a new binary handler for that path.
// - If the plugin path is unset, this does exec.LookPath for a binary named protoc-gen-pluginName,
//   and if one is found, a new binary handler is returned for this.
// - Else, if the name is in ProtocProxyPluginNames, this returns a new protoc proxy handler.
// - Else, this returns error.
func NewHandler(
	logger *zap.Logger,
	pluginName string,
	protocPath string,
	pluginPath string,
) (appproto.Handler, error) {
	if pluginPath != "" {
		return NewBinaryHandler(logger, pluginPath)
	}
	pluginPath, err := exec.LookPath("protoc-gen-" + pluginName)
	if err == nil {
		return newBinaryHandler(logger, pluginPath), nil
	}
	if _, ok := ProtocProxyPluginNames[pluginName]; ok {
		if protocPath == "" {
			protocPath = "protoc"
		}
		return NewProtocProxyHandler(logger, protocPath, pluginName)
	}
	return nil, fmt.Errorf("could not find protoc plugin for name %s", pluginName)
}

// NewBinaryHandler returns a new Handler for the given plugin path.
//
// exec.LookPath is called on the pluginPath, and error is returned if exec.LookPath returns an error.
func NewBinaryHandler(
	logger *zap.Logger,
	pluginPath string,
) (appproto.Handler, error) {
	pluginPath, err := exec.LookPath(pluginPath)
	if err != nil {
		return nil, err
	}
	return newBinaryHandler(logger, pluginPath), nil
}

// NewProtocProxyHandler returns a new Handler that proxies through protoc.
//
// This can be used for the builtin plugins.
//
// exec.LookPath is called on the protocPath, and error is returned if exec.LookPath returns an error.
func NewProtocProxyHandler(
	logger *zap.Logger,
	protocPath string,
	pluginName string,
) (appproto.Handler, error) {
	protocPath, err := exec.LookPath(protocPath)
	if err != nil {
		return nil, err
	}
	return newProtocProxyHandler(logger, protocPath, pluginName), nil
}
