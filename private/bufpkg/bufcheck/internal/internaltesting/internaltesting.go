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

package internaltesting

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bufbuild/buf/private/bufpkg/bufcheck/internal"
	"github.com/bufbuild/buf/private/pkg/stringutil"
)

// RunTestVersionSpec tests the VersionSpec.
func RunTestVersionSpec(t *testing.T, versionSpec *internal.VersionSpec) {
	runTestDefaultConfigBuilder(t, versionSpec)
	runTestRuleBuilders(t, versionSpec)
}

func runTestDefaultConfigBuilder(t *testing.T, versionSpec *internal.VersionSpec) {
	_, err := internal.ConfigBuilder{}.NewConfig(versionSpec)
	assert.NoError(t, err)
}

func runTestRuleBuilders(t *testing.T, versionSpec *internal.VersionSpec) {
	idsMap := make(map[string]struct{}, len(versionSpec.RuleBuilders))
	for _, ruleBuilder := range versionSpec.RuleBuilders {
		_, ok := idsMap[ruleBuilder.ID()]
		assert.False(t, ok, "duplicated id %q", ruleBuilder.ID())
		idsMap[ruleBuilder.ID()] = struct{}{}
	}
	for id := range idsMap {
		expectedID := stringutil.ToUpperSnakeCase(id)
		assert.Equal(t, expectedID, id)
		categories, ok := versionSpec.IDToCategories[id]
		assert.True(t, ok, "id %q categories are not configured", id)
		for _, category := range categories {
			expectedCategory := stringutil.ToUpperSnakeCase(category)
			assert.Equal(t, expectedCategory, category)
		}
	}
	for id := range versionSpec.IDToCategories {
		_, ok := idsMap[id]
		assert.True(t, ok, "id %q configured in categories is not added to ruleBuilders", id)
	}
}
