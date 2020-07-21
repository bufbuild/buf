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

package tracingapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/buf/internal/pkg/observability"
	"github.com/bufbuild/buf/internal/pkg/observability/observabilityzap"
	"go.uber.org/zap"
)

const (
	zapName     = "zap"
	disableName = "disable"
)

// ExternalConfig describes the supported tracing backends and
// their configuration.
type ExternalConfig struct {
	Use string `json:"use,omitempty" yaml:"use,omitempty"`
}

// Config describes the supported tracing backends and
// their coniguration.
type Config struct {
	Use string
}

// NewConfig creates a new Config from an ExternalConfig, with
// defaults set.
func NewConfig(
	externalConfig ExternalConfig,
) (*Config, error) {
	config := &Config{}
	switch strings.ToLower(externalConfig.Use) {
	case "", zapName:
		config.Use = zapName
	case disableName:
		config.Use = disableName
	default:
		return nil, fmt.Errorf("unknown tracing app: %s", externalConfig.Use)
	}
	return config, nil
}

// NewExporter creates a new tracing exporter.
func NewExporter(logger *zap.Logger, config *Config) (observability.Exporter, error) {
	switch strings.ToLower(config.Use) {
	case zapName:
		return observabilityzap.NewExporter(logger), nil
	case disableName:
		return noopExporter{}, nil
	default:
		return nil, fmt.Errorf("unknown tracing app: %s", config.Use)
	}
}

type noopExporter struct{}

func (noopExporter) Run(_ context.Context) error {
	return nil
}
