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
	"testing"

	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestFeatureContainer(t *testing.T) {
	t.Parallel()
	verifyFlag := func(flag app.FeatureFlag, valueToSet string, expected bool) {
		t.Helper()
		env := map[string]string{
			flag.Name: valueToSet,
		}
		flags := app.NewFeatureContainer(app.NewEnvContainer(env))
		assert.Equalf(t, expected, flags.FeatureEnabled(flag), "expected %q to equal %v", valueToSet, expected)
	}
	var flag = app.FeatureFlag{Name: "BUF_FEATURE_TEST", Default: false}
	verifyFlag(flag, " 1 ", true)
	verifyFlag(flag, "true", true)
	verifyFlag(flag, " T\t", true)
	verifyFlag(flag, "0", false)
	verifyFlag(flag, "false", false)
	verifyFlag(flag, "", false)

	var flagDefaultTrue = app.FeatureFlag{Name: "BUF_FEATURE_DEFAULT_ENABLED", Default: true}
	verifyFlag(flagDefaultTrue, "", true)
	verifyFlag(flagDefaultTrue, "false", false)
}
