// Copyright 2020-2023 Buf Technologies, Inc.
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
	"fmt"

	"github.com/bufbuild/buf/private/pkg/storage"
	"go.uber.org/zap"
)

func newReadConfigOptions() *readConfigOptions {
	return &readConfigOptions{}
}

type readConfigOptions struct {
	override string
}

func readConfigVersion(
	ctx context.Context,
	logger *zap.Logger,
	readBucket storage.ReadBucket,
	options ...ReadConfigOption,
) (string, error) {
	provider := NewConfigDataProvider(logger)
	data, id, unmarshalNonStrict, _, err := ReadDataFromConfig(
		ctx,
		logger,
		provider,
		readBucket,
		options...,
	)
	if err != nil {
		return "", err
	}
	var externalConfigVersion ExternalConfigVersion
	if err := unmarshalNonStrict(data, &externalConfigVersion); err != nil {
		return "", err
	}
	switch version := externalConfigVersion.Version; version {
	case V1Version, V1Beta1Version, V2Version:
		return version, nil
	default:
		return "", fmt.Errorf(`%s has no version set. Please add "version: %s"`, id, V2Version)
	}
}
