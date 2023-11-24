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
	DefaultBreakingConfig BreakingConfig = defaultBreakingConfigV1

	defaultBreakingConfigV1Beta1 = NewBreakingConfig(
		defaultCheckConfigV1Beta1,
		false,
	)
	defaultBreakingConfigV1 = NewBreakingConfig(
		defaultCheckConfigV1,
		false,
	)
	defaultBreakingConfigV2 = NewBreakingConfig(
		defaultCheckConfigV2,
		false,
	)
)

// BreakingConfig is breaking configuration for a specific Module.
type BreakingConfig interface {
	CheckConfig

	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

func NewBreakingConfig(
	checkConfig CheckConfig,
	ignoreUnstablePackages bool,
) BreakingConfig {
	return newBreakingConfig(
		checkConfig,
		ignoreUnstablePackages,
	)
}

// *** PRIVATE ***

type breakingConfig struct {
	CheckConfig

	ignoreUnstablePackages bool
}

func newBreakingConfig(
	checkConfig CheckConfig,
	ignoreUnstablePackages bool,
) *breakingConfig {
	return &breakingConfig{
		CheckConfig:            checkConfig,
		ignoreUnstablePackages: ignoreUnstablePackages,
	}
}

func (b *breakingConfig) IgnoreUnstablePackages() bool {
	return b.ignoreUnstablePackages
}

func (*breakingConfig) isBreakingConfig() {}
