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

package internal

import (
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufimage/bufimagebuild"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule"
	"github.com/bufbuild/buf/internal/buf/bufcore/bufmodule/bufmodulebuild"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufwire"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"go.uber.org/zap"
)

// NewBufwireEnvReader returns a new EnvReader.
func NewBufwireEnvReader(
	logger *zap.Logger,
	configOverrideFlagName string,
	moduleResolver bufmodule.ModuleResolver,
	moduleReader bufmodule.ModuleReader,
) bufwire.EnvReader {
	return bufwire.NewEnvReader(
		logger,
		bufcli.NewFetchReader(logger, moduleResolver, moduleReader),
		bufconfig.NewProvider(logger),
		bufmodulebuild.NewModuleBucketBuilder(logger),
		bufmodulebuild.NewModuleFileSetBuilder(logger, moduleReader),
		bufimagebuild.NewBuilder(logger),
		configOverrideFlagName,
	)
}

// NewBufwireImageReader returns a new ImageReader.
func NewBufwireImageReader(
	logger *zap.Logger,
) bufwire.ImageReader {
	return bufwire.NewImageReader(
		logger,
		bufcli.NewFetchImageReader(logger),
	)
}

// NewBufwireImageWriter returns a new ImageWriter.
func NewBufwireImageWriter(
	logger *zap.Logger,
) bufwire.ImageWriter {
	return bufwire.NewImageWriter(
		logger,
		buffetch.NewWriter(
			logger,
		),
	)
}

// NewBufwireConfigReader returns a new EnvReader.
func NewBufwireConfigReader(
	logger *zap.Logger,
	configOverrideFlagName string,
) bufwire.ConfigReader {
	return bufwire.NewConfigReader(
		logger,
		bufconfig.NewProvider(logger),
		configOverrideFlagName,
	)
}

// NewBuflintHandler returns a new buflint.Handler.
func NewBuflintHandler(
	logger *zap.Logger,
) buflint.Handler {
	return buflint.NewHandler(
		logger,
	)
}

// NewBufbreakingHandler returns a new bufbreaking.Handler.
func NewBufbreakingHandler(
	logger *zap.Logger,
) bufbreaking.Handler {
	return bufbreaking.NewHandler(
		logger,
	)
}

// WarnBeta warns that the command is beta.
func WarnBeta(container applog.Container) {
	container.Logger().Warn(`This command has been released for early evaluation only and is experimental. It is not ready for production, and is likely to to have significant changes.`)
}
