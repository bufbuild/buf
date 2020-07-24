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
	"fmt"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufcli"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/buf/bufwire"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const experimentalGitCloneFlagName = "experimental-git-clone"

// NewBufwireEnvReader returns a new EnvReader.
func NewBufwireEnvReader(
	logger *zap.Logger,
	inputFlagName string,
	configOverrideFlagName string,
) bufwire.EnvReader {
	return bufwire.NewEnvReader(
		logger,
		buffetch.NewRefParser(
			logger,
		),
		bufcli.NewFetchReader(logger),
		bufconfig.NewProvider(logger),
		bufmod.NewBucketBuilder(logger),
		bufbuild.NewBuilder(logger),
		inputFlagName,
		configOverrideFlagName,
	)
}

// NewBufwireImageReader returns a new ImageReader.
func NewBufwireImageReader(
	logger *zap.Logger,
	imageFlagName string,
) bufwire.ImageReader {
	return bufwire.NewImageReader(
		logger,
		buffetch.NewImageRefParser(
			logger,
		),
		bufcli.NewFetchReader(logger),
		imageFlagName,
	)
}

// NewBufwireImageWriter returns a new ImageWriter.
func NewBufwireImageWriter(
	logger *zap.Logger,
) bufwire.ImageWriter {
	return bufwire.NewImageWriter(
		logger,
		buffetch.NewImageRefParser(
			logger,
		),
		buffetch.NewWriter(
			logger,
		),
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

// WarnExperimental warns that the command is experimental.
func WarnExperimental(container applog.Container) {
	container.Logger().Warn(`This command has been released for early evaluation only and is experimental. It is not ready for production, and is likely to to have significant changes.`)
}

// BindExperimentalGitClone binds the experimental-git-clone flag
func BindExperimentalGitClone(flagSet *pflag.FlagSet, value *bool) {
	flagSet.BoolVar(
		value,
		experimentalGitCloneFlagName,
		false,
		"Use the git binary to clone instead of the internal git library.",
	)
	_ = flagSet.MarkHidden(experimentalGitCloneFlagName)
	_ = flagSet.MarkDeprecated(
		experimentalGitCloneFlagName,
		fmt.Sprintf(
			"Flag --%s is deprecated. The formerly-experimental git clone functionality is now the only clone functionality used, and this flag has no effect.",
			experimentalGitCloneFlagName,
		),
	)
}
