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

package bufconfig

import "github.com/bufbuild/buf/private/pkg/syserror"

var (
	// DefaultLintConfigV1 is the default lint config for v1.
	DefaultLintConfigV1 LintConfig = newLintConfigNoValidate(
		defaultCheckConfigV1,
		"",
		false,
		false,
		false,
		"",
		false,
		nil,
	)

	// DefaultLintConfigV2 is the default lint config for v2.
	DefaultLintConfigV2 LintConfig = newLintConfigNoValidate(
		defaultCheckConfigV2,
		"",
		false,
		false,
		false,
		"",
		false,
		nil,
	)
)

// LintConfig is lint configuration for a specific Module.
type LintConfig interface {
	CheckConfig

	EnumZeroValueSuffix() string
	RPCAllowSameRequestResponse() bool
	RPCAllowGoogleProtobufEmptyRequests() bool
	RPCAllowGoogleProtobufEmptyResponses() bool
	ServiceSuffix() string
	AllowCommentIgnores() bool
	// Will always be empty for FileVersionV1Beta1 and FileVersionV1.
	Plugins() []LintPluginConfig

	isLintConfig()
}

// NewLintConfig returns a new LintConfig.
func NewLintConfig(
	checkConfig CheckConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
	plugins []LintPluginConfig,
) (LintConfig, error) {
	return newLintConfig(
		checkConfig,
		enumZeroValueSuffix,
		rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix,
		allowCommentIgnores,
		plugins,
	)
}

// *** PRIVATE ***

type lintConfig struct {
	CheckConfig

	enumZeroValueSuffix                  string
	rpcAllowSameRequestResponse          bool
	rpcAllowGoogleProtobufEmptyRequests  bool
	rpcAllowGoogleProtobufEmptyResponses bool
	serviceSuffix                        string
	allowCommentIgnores                  bool
	plugins                              []LintPluginConfig
}

func newLintConfig(
	checkConfig CheckConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
	plugins []LintPluginConfig,
) (*lintConfig, error) {
	if len(plugins) > 0 {
		switch fileVersion := checkConfig.FileVersion(); fileVersion {
		case FileVersionV1Beta1, FileVersionV1:
			return nil, syserror.Newf("got LintPluginConfigs %v for FileVersion %v", plugins, fileVersion)
		case FileVersionV2:
		default:
			return nil, syserror.Newf("unknown FileVersion: %v", fileVersion)
		}
	}
	return newLintConfigNoValidate(
		checkConfig,
		enumZeroValueSuffix,
		rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix,
		allowCommentIgnores,
		plugins,
	), nil
}

func newLintConfigNoValidate(
	checkConfig CheckConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobufEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
	plugins []LintPluginConfig,
) *lintConfig {
	return &lintConfig{
		CheckConfig:                          checkConfig,
		enumZeroValueSuffix:                  enumZeroValueSuffix,
		rpcAllowSameRequestResponse:          rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobufEmptyRequests:  rpcAllowGoogleProtobufEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses: rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix:                        serviceSuffix,
		allowCommentIgnores:                  allowCommentIgnores,
		plugins:                              plugins,
	}
}

func (l *lintConfig) EnumZeroValueSuffix() string {
	return l.enumZeroValueSuffix
}

func (l *lintConfig) RPCAllowSameRequestResponse() bool {
	return l.rpcAllowSameRequestResponse
}

func (l *lintConfig) RPCAllowGoogleProtobufEmptyRequests() bool {
	return l.rpcAllowGoogleProtobufEmptyRequests
}

func (l *lintConfig) RPCAllowGoogleProtobufEmptyResponses() bool {
	return l.rpcAllowGoogleProtobufEmptyResponses
}

func (l *lintConfig) ServiceSuffix() string {
	return l.serviceSuffix
}

func (l *lintConfig) AllowCommentIgnores() bool {
	return l.allowCommentIgnores
}

func (l *lintConfig) Plugins() []LintPluginConfig {
	return l.plugins
}

func (*lintConfig) isLintConfig() {}
