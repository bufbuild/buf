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

package bufremoteplugin

import (
	"fmt"

	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
)

var (
	stringToDotnetTargetFramework = map[string]registryv1alpha1.DotnetTargetFramework{
		"netstandard1.0": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_0,
		"netstandard1.1": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_1,
		"netstandard1.2": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_2,
		"netstandard1.3": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_3,
		"netstandard1.4": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_4,
		"netstandard1.5": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_5,
		"netstandard1.6": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_1_6,
		"netstandard2.0": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_2_0,
		"netstandard2.1": registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NETSTANDARD_2_1,
		"net5.0":         registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NET_5_0,
		"net6.0":         registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NET_6_0,
		"net7.0":         registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NET_7_0,
		"net8.0":         registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_NET_8_0,
	}
)

// DotnetTargetFrameworkFromString converts the target framework name to the equivalent enum.
// It returns an error if the specified string is unknown.
func DotnetTargetFrameworkFromString(framework string) (registryv1alpha1.DotnetTargetFramework, error) {
	frameworkEnum, ok := stringToDotnetTargetFramework[framework]
	if !ok {
		return 0, fmt.Errorf("unknown target framework %q", framework)
	}
	return frameworkEnum, nil
}

// DotnetTargetFrameworkToString converts the target framework enum to the equivalent string.
// It returns an error if the specified enum is unspecified or unknown.
func DotnetTargetFrameworkToString(framework registryv1alpha1.DotnetTargetFramework) (string, error) {
	// This isn't performance critical code - just scan the existing mapping instead of storing in both directions.
	for frameworkStr, frameworkEnum := range stringToDotnetTargetFramework {
		if frameworkEnum == framework {
			return frameworkStr, nil
		}
	}
	return "", fmt.Errorf("unknown target framework %v", framework)
}
