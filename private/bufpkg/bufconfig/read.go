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

package bufconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bufbuild/buf/private/pkg/storage"
)

func readConfigOS(
	ctx context.Context,
	readBucket storage.ReadBucket,
	options ...ReadConfigOSOption,
) (*Config, error) {
	readConfigOSOptions := newReadConfigOSOptions()
	for _, option := range options {
		option(readConfigOSOptions)
	}
	if readConfigOSOptions.override != "" {
		var data []byte
		var err error
		switch filepath.Ext(readConfigOSOptions.override) {
		case ".json", ".yaml", ".yml":
			data, err = os.ReadFile(readConfigOSOptions.override)
			if err != nil {
				return nil, fmt.Errorf("could not read file: %v", err)
			}
		default:
			data = []byte(readConfigOSOptions.override)
		}
		return GetConfigForData(ctx, data)
	}
	return GetConfigForBucket(ctx, readBucket)
}

type readConfigOSOptions struct {
	override string
}

func newReadConfigOSOptions() *readConfigOSOptions {
	return &readConfigOSOptions{}
}
