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

	defaultBreakingConfigV1Beta1 = newBreakingConfig(
		defaultCheckConfigV1Beta1,
		false,
	)
	defaultBreakingConfigV1 = newBreakingConfig(
		defaultCheckConfigV1,
		false,
	)
)

// BreakingConfig is breaking configuration for a specific Module.
type BreakingConfig interface {
	CheckConfig

	IgnoreUnstablePackages() bool

	isBreakingConfig()
}

// *** PRIVATE ***

type breakingConfig struct {
	checkConfig

	ignoreUnstablePackages bool
}

func newBreakingConfig(
	checkConfig checkConfig,
	ignoreUnstablePackages bool,
) *breakingConfig {
	return &breakingConfig{
		checkConfig:            checkConfig,
		ignoreUnstablePackages: ignoreUnstablePackages,
	}
}

func (b *breakingConfig) IgnoreUnstablePackages() bool {
	return b.ignoreUnstablePackages
}

func (*breakingConfig) isBreakingConfig() {}
