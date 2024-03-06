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

package internal

import (
	"context"
	"errors"
	"io/fs"

	"github.com/bufbuild/buf/private/buf/bufcli"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
)

// GetModuleConfigForProtocPlugin gets ModuleConfigs for the protoc plugin implementations.
//
// This is the same in both plugins so we just pulled it out to a common spot.
func GetModuleConfigForProtocPlugin(
	ctx context.Context,
	configOverride string,
) (bufconfig.ModuleConfig, error) {
	bufYAMLFile, err := bufcli.GetBufYAMLFileForDirPathOrOverride(
		ctx,
		".",
		configOverride,
	)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return bufconfig.DefaultModuleConfigV1, nil
		}
		return nil, err
	}
	for _, moduleConfig := range bufYAMLFile.ModuleConfigs() {
		// If we have a v1beta1 or v1 buf.yaml, dirPath will be ".". Using the ModuleConfig from
		// a v1beta1 or v1 buf.yaml file matches the pre-refactor behavior.
		//
		// If we have a v2 buf.yaml, we say that we need to have a module with dirPath of ".", otherwise
		// we can't deduce what ModuleConfig to use.
		if dirPath := moduleConfig.DirPath(); dirPath == "." {
			return moduleConfig, nil
		}
	}
	// TODO: point to a webpage that explains this.
	return nil, errors.New(`could not determine which module to pull configuration from. See the docs for more details.`)
}
