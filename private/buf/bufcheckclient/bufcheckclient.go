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
	// Lint lints the given Image with the given LintConfig.
	//
	// The Image should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Lint(ctx context.Context, config bufconfig.LintConfig, image bufimage.Image) error
	ConfiguredLintRules(ctx context.Context, config bufconfig.LintConfig) ([]check.Rule, error)
	AllLintRules(ctx context.Context) ([]check.Rule, error)

	// Breaking checks the given Images for breaking changes with the given BreakingConfig.
	//
	// The Images should have source code info for this to work properly.
	//
	// Images should *not* be filtered with regards to imports before passing to this function.
	// TODO: reconcile with bufbreaking.
	//
	// An error of type bufanalysis.FileAnnotationSet will be returned lint failure.
	Breaking(ctx context.Context, config bufconfig.BreakingConfig, image bufimage.Image, againstImage bufimage.Image) error
	ConfiguredBreakingRules(ctx context.Context, config bufconfig.BreakingConfig) ([]check.Rule, error)
	AllBreakingRules(ctx context.Context) ([]check.Rule, error)
}

// If you want to use the default v1beta1/v1/v2 Client, pass it.
// If you want to also use a plugin Client, merge the Clients with a check.NewMultiClient.
func NewClient(logger *zap.Logger, checkClient check.Client) Client {
	return newClient(logger, checkClient)
}
