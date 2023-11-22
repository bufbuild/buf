// Copyright 2020-2023 Buf Technologies, Inc.
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

var (
	DefaultLintConfig LintConfig = defaultLintConfigV1

	defaultLintConfigV1Beta1 = newLintConfig(
		defaultCheckConfigV1Beta1,
		"",
		false,
		false,
		false,
		"",
		false,
	)
	defaultLintConfigV1 = newLintConfig(
		defaultCheckConfigV1,
		"",
		false,
		false,
		false,
		"",
		false,
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

	isLintConfig()
}

// *** PRIVATE ***

type lintConfig struct {
	checkConfig

	enumZeroValueSuffix                  string
	rpcAllowSameRequestResponse          bool
	rpcAllowGoogleProtobuEmptyRequests   bool
	rpcAllowGoogleProtobufEmptyResponses bool
	serviceSuffix                        string
	allowCommentIgnores                  bool
}

func newLintConfig(
	checkConfig checkConfig,
	enumZeroValueSuffix string,
	rpcAllowSameRequestResponse bool,
	rpcAllowGoogleProtobuEmptyRequests bool,
	rpcAllowGoogleProtobufEmptyResponses bool,
	serviceSuffix string,
	allowCommentIgnores bool,
) *lintConfig {
	return &lintConfig{
		checkConfig:                          checkConfig,
		enumZeroValueSuffix:                  enumZeroValueSuffix,
		rpcAllowSameRequestResponse:          rpcAllowSameRequestResponse,
		rpcAllowGoogleProtobuEmptyRequests:   rpcAllowGoogleProtobuEmptyRequests,
		rpcAllowGoogleProtobufEmptyResponses: rpcAllowGoogleProtobufEmptyResponses,
		serviceSuffix:                        serviceSuffix,
		allowCommentIgnores:                  allowCommentIgnores,
	}
}

func (l *lintConfig) EnumZeroValueSuffix() string {
	return l.enumZeroValueSuffix
}

func (l *lintConfig) RPCAllowSameRequestResponse() bool {
	return l.rpcAllowSameRequestResponse
}

func (l *lintConfig) RPCAllowGoogleProtobufEmptyRequests() bool {
	return l.rpcAllowGoogleProtobuEmptyRequests
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

func (*lintConfig) isLintConfig() {}
