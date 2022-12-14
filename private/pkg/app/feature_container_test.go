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

package app_test

import (
	"os"
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFeatureFlag app.FeatureFlag = "BUF_FEATURE_TEST"

func TestFeatureContainer(t *testing.T) {
	t.Parallel()
	verifyFlag := func(valueToSet string, expected bool, configureFunc func(features app.FeatureContainer)) {
		t.Helper()
		env := map[string]string{
			string(testFeatureFlag): valueToSet,
		}
		flags := app.NewFeatureContainer(app.NewEnvContainer(env))
		if configureFunc != nil {
			configureFunc(flags)
		}
		require.NoError(t, os.Setenv(string(testFeatureFlag), valueToSet))
		assert.Equalf(t, expected, flags.FeatureEnabled(testFeatureFlag), "expected %q to equal %v", valueToSet, expected)
	}
	verifyFlag(" 1 ", true, nil)
	verifyFlag("true", true, nil)
	verifyFlag(" T\t", true, nil)
	verifyFlag("0", false, nil)
	verifyFlag("false", false, nil)
	// No default specified
	verifyFlag("", false, nil)
	verifyFlag("", true, func(features app.FeatureContainer) {
		features.SetFeatureDefault(testFeatureFlag, true)
	})
}
