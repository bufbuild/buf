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
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/bufplugin-go/check"
	"go.uber.org/zap"
)

type client struct {
	logger       *zap.Logger
	checkClients []check.Client
}

func newClient(
	logger *zap.Logger,
	checkClients []check.Client,
) *client {
	return &client{
		logger:       logger,
		checkClients: checkClients,
	}
}

func (c *client) Lint(ctx context.Context, config bufconfig.LintConfig, image bufimage.Image) error {
	return errors.New("TODO")
}

func (c *client) ConfiguredLintRules(ctx context.Context, config bufconfig.LintConfig) ([]check.Rule, error) {
	return nil, errors.New("TODO")
}

func (c *client) AllLintRules(ctx context.Context) ([]check.Rule, error) {
	return nil, errors.New("TODO")
}

func (c *client) Breaking(ctx context.Context, config bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image) error {
	return errors.New("TODO")
}

func (c *client) ConfiguredBreakingRules(ctx context.Context, config bufconfig.BreakingConfig) ([]check.Rule, error) {
	return nil, errors.New("TODO")
}

func (c *client) AllBreakingRules(ctx context.Context) ([]check.Rule, error) {
	return nil, errors.New("TODO")
}
