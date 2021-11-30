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

package bufbreakingconfig

import (
	breakingv1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/breaking/v1"
)

const (
	// These versions match the versions in bufconfig. We cannot take an explicit dependency
	// on bufconfig without creating a circular dependency.
	v1Beta1Version = "v1beta1"
	v1Version      = "v1"
)

// Config is the breaking check config.
type Config struct {
	// Use is a list of the rule and/or category IDs that are included in the breaking change check.
	Use []string
	// Except is a list of the rule and/or category IDs that are excluded from the breaking change check.
	Except []string
	// IgnoreRootPaths is a list of the paths of directories and/or files that should be ignored by the breaking change check.
	// All paths are relative to the root of the module.
	IgnoreRootPaths []string
	// IgnoreIDOrCategoryToRootPaths is a map of rule and/or category IDs to directory and/or file paths to exclude from the
	// breaking change check.
	IgnoreIDOrCategoryToRootPaths map[string][]string
	// IgnoreUnstablePackages ignores packages with a last component that is one of the unstable forms recognised
	// by the PACKAGE_VERSION_SUFFIX:
	//   v\d+test.*
	//   v\d+(alpha|beta)\d+
	//   v\d+p\d+(alpha|beta)\d+
	IgnoreUnstablePackages bool
	// Version represents the version of the breaking change rule and category IDs that should be used with this config.
	Version string
}

// NewConfigV1Beta1 returns a new Config.
func NewConfigV1Beta1(externalConfig ExternalConfigV1Beta1) *Config {
	return &Config{
		Use:                           externalConfig.Use,
		Except:                        externalConfig.Except,
		IgnoreRootPaths:               externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths: externalConfig.IgnoreOnly,
		IgnoreUnstablePackages:        externalConfig.IgnoreUnstablePackages,
		Version:                       v1Beta1Version,
	}
}

// NewConfigV1 returns a new Config.
func NewConfigV1(externalConfig ExternalConfigV1) *Config {
	return &Config{
		Use:                           externalConfig.Use,
		Except:                        externalConfig.Except,
		IgnoreRootPaths:               externalConfig.Ignore,
		IgnoreIDOrCategoryToRootPaths: externalConfig.IgnoreOnly,
		IgnoreUnstablePackages:        externalConfig.IgnoreUnstablePackages,
		Version:                       v1Version,
	}
}

// ConfigForProto returns the Config given the proto.
func ConfigForProto(protoConfig *breakingv1.Config) *Config {
	return &Config{
		Use:                           protoConfig.GetUseIds(),
		Except:                        protoConfig.GetExceptIds(),
		IgnoreRootPaths:               protoConfig.GetIgnorePaths(),
		IgnoreIDOrCategoryToRootPaths: ignoreIDOrCategoryToRootPathsForProto(protoConfig.GetIgnoreIdPaths()),
		IgnoreUnstablePackages:        protoConfig.GetIgnoreUnstablePackages(),
		Version:                       protoConfig.GetVersion(),
	}
}

// ExternalConfigV1Beta1 is an external config.
type ExternalConfigV1Beta1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

// ExternalConfigV1 is an external config.
type ExternalConfigV1 struct {
	Use    []string `json:"use,omitempty" yaml:"use,omitempty"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
	// IgnoreRootPaths
	Ignore []string `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	// IgnoreIDOrCategoryToRootPaths
	IgnoreOnly             map[string][]string `json:"ignore_only,omitempty" yaml:"ignore_only,omitempty"`
	IgnoreUnstablePackages bool                `json:"ignore_unstable_packages,omitempty" yaml:"ignore_unstable_packages,omitempty"`
}

func ignoreIDOrCategoryToRootPathsForProto(protoIgnoreIDPaths []*breakingv1.IDPaths) map[string][]string {
	if protoIgnoreIDPaths == nil {
		return nil
	}
	ignoreIDOrCategoryToRootPaths := make(map[string][]string)
	for _, protoIgnoreIDPath := range protoIgnoreIDPaths {
		ignoreIDOrCategoryToRootPaths[protoIgnoreIDPath.GetId()] = protoIgnoreIDPath.GetPaths()
	}
	return ignoreIDOrCategoryToRootPaths
}
