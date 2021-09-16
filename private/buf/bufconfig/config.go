// Copyright 2020-2021 Buf Technologies, Inc.
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

import (
	"github.com/bufbuild/buf/private/buf/bufcheck/bufbreaking/bufbreakingconfig"
	"github.com/bufbuild/buf/private/buf/bufcheck/buflint/buflintconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleconfig"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
)

func newConfigV1Beta1(externalConfig ExternalConfigV1Beta1) (*Config, error) {
	buildConfig, err := bufmoduleconfig.NewConfigV1Beta1(externalConfig.Build, externalConfig.Deps...)
	if err != nil {
		return nil, err
	}
	breakingConfig, err := bufbreakingconfig.NewConfigV1Beta1(externalConfig.Breaking)
	if err != nil {
		return nil, err
	}
	lintConfig, err := buflintconfig.NewConfigV1Beta1(externalConfig.Lint)
	if err != nil {
		return nil, err
	}
	var moduleIdentity bufmoduleref.ModuleIdentity
	if externalConfig.Name != "" {
		moduleIdentity, err = bufmoduleref.ModuleIdentityForString(externalConfig.Name)
		if err != nil {
			return nil, err
		}
	}
	return &Config{
		Version:        V1Beta1Version,
		ModuleIdentity: moduleIdentity,
		Build:          buildConfig,
		Breaking:       breakingConfig,
		Lint:           lintConfig,
	}, nil
}

func newConfigV1(externalConfig ExternalConfigV1) (*Config, error) {
	buildConfig, err := bufmoduleconfig.NewConfigV1(externalConfig.Build, externalConfig.Deps...)
	if err != nil {
		return nil, err
	}
	breakingConfig, err := bufbreakingconfig.NewConfigV1(externalConfig.Breaking)
	if err != nil {
		return nil, err
	}
	lintConfig, err := buflintconfig.NewConfigV1(externalConfig.Lint)
	if err != nil {
		return nil, err
	}
	var moduleIdentity bufmoduleref.ModuleIdentity
	if externalConfig.Name != "" {
		moduleIdentity, err = bufmoduleref.ModuleIdentityForString(externalConfig.Name)
		if err != nil {
			return nil, err
		}
	}
	if err := bufmoduleref.ValidateLicenseID(externalConfig.License); err != nil {
		return nil, err
	}
	return &Config{
		Version:        V1Version,
		ModuleIdentity: moduleIdentity,
		LicenseID:      externalConfig.License,
		Build:          buildConfig,
		Breaking:       breakingConfig,
		Lint:           lintConfig,
	}, nil
}
