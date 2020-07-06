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
	"net/http"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buffetch"
	"github.com/bufbuild/buf/internal/buf/bufmod"
	"github.com/bufbuild/buf/internal/buf/bufwire"
	"github.com/bufbuild/buf/internal/pkg/app/applog"
	"github.com/bufbuild/buf/internal/pkg/git"
	"github.com/bufbuild/buf/internal/pkg/httpauth"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	experimentalGitCloneFlagName  = "experimental-git-clone"
	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"
)

var (
	// Timeout should be set through context for calls to EnvReader, not through http.Client
	defaultHTTPClient        = &http.Client{}
	defaultHTTPAuthenticator = httpauth.NewMultiAuthenticator(
		httpauth.NewNetrcAuthenticator(),
		// must keep this for legacy purposes
		httpauth.NewEnvAuthenticator(
			inputHTTPSPasswordEnvKey,
			inputHTTPSPasswordEnvKey,
		),
	)
	defaultGitClonerOptions = git.ClonerOptions{
		HTTPSUsernameEnvKey:      inputHTTPSUsernameEnvKey,
		HTTPSPasswordEnvKey:      inputHTTPSPasswordEnvKey,
		SSHKeyFileEnvKey:         inputSSHKeyFileEnvKey,
		SSHKnownHostsFilesEnvKey: inputSSHKnownHostsFilesEnvKey,
	}
)

// NewBufwireEnvReader returns a new bufos.EnvReader.
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
		buffetch.NewReader(
			logger,
			defaultHTTPClient,
			defaultHTTPAuthenticator,
			git.NewCloner(logger, defaultGitClonerOptions),
		),
		bufconfig.NewProvider(logger),
		bufmod.NewBucketBuilder(logger),
		bufbuild.NewBuilder(logger),
		inputFlagName,
		configOverrideFlagName,
	)
}

// NewBufwireImageWriter returns a new bufos.ImageWriter.
func NewBufwireImageWriter(
	logger *zap.Logger,
) bufwire.ImageWriter {
	return bufwire.NewImageWriter(
		logger,
		buffetch.NewRefParser(
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

// IsFormatJSON returns true if the format is JSON.
//
// This will probably eventually need to be split between the image/check flags
// and the ls flags as we may have different formats for each.
func IsFormatJSON(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return true, nil
	default:
		return false, fmt.Errorf("--%s: unknown format: %q", flagName, s)
	}
}

// IsLintFormatJSON returns true if the format is JSON for lint.
//
// Also allows config-ignore-yaml.
func IsLintFormatJSON(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return true, nil
	case "config-ignore-yaml":
		return false, nil
	default:
		return false, fmt.Errorf("--%s: unknown format: %q", flagName, s)
	}
}

// IsLintFormatConfigIgnoreYAML returns true if the format is config-ignore-yaml.
func IsLintFormatConfigIgnoreYAML(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return false, nil
	case "config-ignore-yaml":
		return true, nil
	default:
		return false, fmt.Errorf("--%s: unknown format: %q", flagName, s)
	}
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
