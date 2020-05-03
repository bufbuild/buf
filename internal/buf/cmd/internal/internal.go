// Copyright 2020 Buf Technologies Inc.
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
	"github.com/bufbuild/buf/internal/buf/bufos"
	"go.uber.org/zap"
)

const (
	inputHTTPSUsernameEnvKey      = "BUF_INPUT_HTTPS_USERNAME"
	inputHTTPSPasswordEnvKey      = "BUF_INPUT_HTTPS_PASSWORD"
	inputSSHKeyFileEnvKey         = "BUF_INPUT_SSH_KEY_FILE"
	inputSSHKeyPassphraseEnvKey   = "BUF_INPUT_SSH_KEY_PASSPHRASE"
	inputSSHKnownHostsFilesEnvKey = "BUF_INPUT_SSH_KNOWN_HOSTS_FILES"
)

// Timeout should be set through context for calls to EnvReader, not through http.Client
var defaultHTTPClient = &http.Client{}

// NewBufosEnvReader returns a new bufos.EnvReader.
func NewBufosEnvReader(
	logger *zap.Logger,
	inputFlagName string,
	configOverrideFlagName string,
	experimentalGitClone bool,
) bufos.EnvReader {
	return bufos.NewEnvReader(
		logger,
		defaultHTTPClient,
		bufconfig.NewProvider(logger),
		bufbuild.NewHandler(logger),
		inputFlagName,
		configOverrideFlagName,
		inputHTTPSUsernameEnvKey,
		inputHTTPSPasswordEnvKey,
		inputSSHKeyFileEnvKey,
		inputSSHKeyPassphraseEnvKey,
		inputSSHKnownHostsFilesEnvKey,
		experimentalGitClone,
	)
}

// NewBufosImageWriter returns a new bufos.ImageWriter.
func NewBufosImageWriter(
	logger *zap.Logger,
	outputFlagName string,
) bufos.ImageWriter {
	return bufos.NewImageWriter(
		logger,
		outputFlagName,
	)
}

// NewBuflintHandler returns a new buflint.Handler.
func NewBuflintHandler(
	logger *zap.Logger,
) buflint.Handler {
	return buflint.NewHandler(
		logger,
		buflint.NewRunner(logger),
	)
}

// NewBufbreakingHandler returns a new bufbreaking.Handler.
func NewBufbreakingHandler(
	logger *zap.Logger,
) bufbreaking.Handler {
	return bufbreaking.NewHandler(
		logger,
		bufbreaking.NewRunner(logger),
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
