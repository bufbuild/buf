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
	"context"
	"fmt"
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufremoteplugin/bufremotepluginconfig"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotnetTargetPlatformMapping(t *testing.T) {
	t.Parallel()
	assert.Len(t, stringToDotnetTargetFramework, len(registryv1alpha1.DotnetTargetFramework_name)-1)
	for value := range registryv1alpha1.DotnetTargetFramework_name {
		targetFramework := registryv1alpha1.DotnetTargetFramework(value)
		if targetFramework == registryv1alpha1.DotnetTargetFramework_DOTNET_TARGET_FRAMEWORK_UNSPECIFIED {
			continue
		}
		// Verify round trip
		strTargetFramework, err := DotnetTargetFrameworkToString(targetFramework)
		require.NoErrorf(t, err, "missing mapping for target framework %v", targetFramework)
		targetFrameworkFromStr, err := DotnetTargetFrameworkFromString(strTargetFramework)
		require.NoError(t, err)
		assert.Equal(t, targetFramework, targetFrameworkFromStr)
	}
}

func TestDotnetTargetPlatformExternalConfigMapping(t *testing.T) {
	// We validate the string values for target frameworks in bufremoteconfig.
	// This test will fail if we add a new target framework to the proto and didn't update the validation.
	t.Parallel()
	ctx := context.Background()
	for targetFrameworkStr := range stringToDotnetTargetFramework {
		externalCfg := fmt.Sprintf(
			`version: v1
name: buf.build/grpc/csharp
plugin_version: v1.0.0
registry:
  nuget:
    target_frameworks:
      - %s
`, targetFrameworkStr)
		_, err := bufremotepluginconfig.GetConfigForData(ctx, []byte(externalCfg))
		require.NoErrorf(t, err, "bufremotepluginconfig.validateNugetTargetFramework needs updating for %s", targetFrameworkStr)
	}
}
