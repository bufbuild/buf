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

package bufcheckclient

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
)

// All functions that take a config ignore the FileVersion. The FileVersion should instruct
// what check.Client is passed to NewClient, ie a v1beta1, v1, or v2 default client.
type Client interface {
	Lint(ctx context.Context, config bufconfig.LintConfig, image bufimage.Image) error
	ConfiguredLintRules(ctx context.Context, config bufconfig.LintConfig) ([]check.Rule, error)
	AllLintRules(ctx context.Context) ([]check.Rule, error)

	Breaking(ctx context.Context, config bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image) error
	ConfiguredBreakingRules(ctx context.Context, config bufconfig.BreakingConfig) ([]check.Rule, error)
	AllBreakingRules(ctx context.Context) ([]check.Rule, error)
}

func NewClient(logger *zap.Logger, clients []check.Client) Client {
	return newClient(logger, clients)
}
