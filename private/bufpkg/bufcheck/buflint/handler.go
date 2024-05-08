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

package buflint

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/buflint/internal/buflintcheck"
	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/bufpkg/bufprotosource"
	"github.com/bufbuild/buf/private/pkg/tracing"
	"go.uber.org/zap"
)

type handler struct {
	logger *zap.Logger
	tracer tracing.Tracer
	runner *internal.Runner
}

func newHandler(logger *zap.Logger, tracer tracing.Tracer) *handler {
	return &handler{
		logger: logger,
		tracer: tracer,
		// linting allows for comment ignores
		// note that comment ignores still need to be enabled within the config
		// for a given check, this just says that comment ignores are allowed
		// in the first place
		runner: internal.NewRunner(
			logger,
			tracer,
			internal.RunnerWithIgnorePrefix(buflintcheck.CommentIgnorePrefix),
		),
	}
}

func (h *handler) Check(
	ctx context.Context,
	config bufconfig.LintConfig,
	image bufimage.Image,
) error {
	if config.Disabled() {
		return nil
	}
	files, err := bufprotosource.NewFiles(ctx, image)
	if err != nil {
		return err
	}
	internalConfig, err := internalConfigForConfig(config, true)
	if err != nil {
		return err
	}
	return h.runner.Check(ctx, internalConfig, nil, files)
}
