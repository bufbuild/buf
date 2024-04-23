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

var (
	// DefaultBreakingConfigV1 is the default breaking config for v1.
	DefaultBreakingConfigV1 BreakingConfig = NewBreakingConfig(
		defaultCheckConfigV1,
		false,
	)

	// DefaultBreakingConfigV2 is the default breaking config for v1.
	DefaultBreakingConfigV2 BreakingConfig = NewBreakingConfig(
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

// NewBreakingConfig returns a new BreakingConfig.
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
