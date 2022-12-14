// Copyright 2020-2022 Buf Technologies, Inc.
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

package appfeature_test

import (
	"os"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app/appfeature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFeatureFlag appfeature.FeatureFlag = "BUF_FEATURE_TEST"

func TestFeatureFlags(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {
		if err := os.Unsetenv(string(testFeatureFlag)); err != nil {
			t.Errorf("failed to unset env var: %v", err)
		}
	})
	flags := appfeature.NewContainer(appfeature.EnvironmentFunc(os.Getenv))
	verifyFlag := func(valueToSet string, expected bool) {
		t.Helper()
		require.NoError(t, os.Setenv(string(testFeatureFlag), valueToSet))
		assert.Equalf(t, expected, flags.FeatureEnabled(testFeatureFlag), "expected %q to equal %v", valueToSet, expected)
	}
	verifyFlag(" 1 ", true)
	verifyFlag("true", true)
	verifyFlag(" T\t", true)
	verifyFlag("0", false)
	verifyFlag("false", false)
	// No default specified
	verifyFlag("", false)
	flags.SetFeatureDefault(testFeatureFlag, true)
	verifyFlag("", true)
}
