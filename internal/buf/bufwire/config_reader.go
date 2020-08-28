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

package bufwire

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"go.uber.org/zap"
)

type configReader struct {
	logger                 *zap.Logger
	configProvider         bufconfig.Provider
	configOverrideFlagName string
}

func newConfigReader(
	logger *zap.Logger,
	configProvider bufconfig.Provider,
	configOverrideFlagName string,
) *configReader {
	return &configReader{
		logger:                 logger.Named("bufwire"),
		configProvider:         configProvider,
		configOverrideFlagName: configOverrideFlagName,
	}
}

func (e *configReader) GetConfig(
	ctx context.Context,
	configOverride string,
) (*bufconfig.Config, error) {
	// if there was no file, this just returns default config
	readWriteBucket, err := storageos.NewReadWriteBucket(".")
	if err != nil {
		return nil, err
	}
	return e.getConfig(ctx, readWriteBucket, configOverride)
}

func (e *configReader) getConfig(
	ctx context.Context,
	readBucket storage.ReadBucket,
	configOverride string,
) (*bufconfig.Config, error) {
	if configOverride != "" {
		return e.parseConfigOverride(ctx, configOverride)
	}
	// if there was no file, this just returns default config
	return e.configProvider.GetConfig(ctx, readBucket)
}

func (e *configReader) parseConfigOverride(ctx context.Context, value string) (*bufconfig.Config, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("config override value is empty")
	}
	var data []byte
	var err error
	switch filepath.Ext(value) {
	case ".json", ".yaml":
		data, err = ioutil.ReadFile(value)
		if err != nil {
			return nil, fmt.Errorf("%s: could not read file: %v", e.configOverrideFlagName, err)
		}
	default:
		data = []byte(value)
	}
	config, err := e.configProvider.GetConfigForData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.configOverrideFlagName, err)
	}
	return config, nil
}
