// Copyright 2020-2021 Buf Technologies, Inc.
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

package appprotoexec

import (
	"fmt"
	"os/exec"

	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/storage/storageos"
	"go.uber.org/zap"
)

// newHandler returns a new appproto.Handler based on the plugin name and optional path.
//
// pluginPath is optional.
//
// - If the plugin path is set, this returns a new binary handler for that path.
// - If the plugin path is unset, this does exec.LookPath for a binary named protoc-gen-pluginName,
//   and if one is found, a new binary handler is returned for this.
// - Else, if the name is in ProtocProxyPluginNames, this returns a new protoc proxy handler.
// - Else, this returns error.
func newHandler(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	pluginName string,
	pluginPath string,
) (appproto.Handler, error) {
	if pluginPath != "" {
		pluginPath, err := exec.LookPath(pluginPath)
		if err != nil {
			return nil, err
		}
		return newBinaryHandler(logger, pluginPath), nil
	}
	pluginPath, err := exec.LookPath("protoc-gen-" + pluginName)
	if err == nil {
		return newBinaryHandler(logger, pluginPath), nil
	}
	// we always look for protoc-gen-X first, but if not, check the builtins
	if _, ok := ProtocProxyPluginNames[pluginName]; ok {
		protocPath, err := exec.LookPath("protoc")
		if err != nil {
			return nil, err
		}
		return newProtocProxyHandler(logger, storageosProvider, protocPath, pluginName), nil
	}
	return nil, fmt.Errorf("could not find protoc plugin for name %s", pluginName)
}
