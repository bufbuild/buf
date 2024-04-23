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

package bufctl

import (
	"github.com/bufbuild/buf/private/bufpkg/bufconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufimage"
)

type imageWithConfig struct {
	bufimage.Image

	lintConfig     bufconfig.LintConfig
	breakingConfig bufconfig.BreakingConfig
}

func newImageWithConfig(
	image bufimage.Image,
	lintConfig bufconfig.LintConfig,
	breakingConfig bufconfig.BreakingConfig,
) *imageWithConfig {
	return &imageWithConfig{
		Image:          image,
		lintConfig:     lintConfig,
		breakingConfig: breakingConfig,
	}
}

func (i *imageWithConfig) LintConfig() bufconfig.LintConfig {
	return i.lintConfig
}

func (i *imageWithConfig) BreakingConfig() bufconfig.BreakingConfig {
	return i.breakingConfig
}

func (*imageWithConfig) isImageWithConfig() {}
