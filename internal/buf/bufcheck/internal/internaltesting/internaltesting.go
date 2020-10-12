// Copyright 2020 Buf Technologies, Inc.
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

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal"
	"github.com/bufbuild/buf/internal/pkg/stringutil"
	"github.com/stretchr/testify/assert"
)

// RunTestVersionSpec tests the VersionSpec.
func RunTestVersionSpec(t *testing.T, versionSpec *internal.VersionSpec) {
	runTestDefaultConfigBuilder(t, versionSpec)
	runTestCheckerBuilders(t, versionSpec)
}

func runTestDefaultConfigBuilder(t *testing.T, versionSpec *internal.VersionSpec) {
	_, err := internal.ConfigBuilder{}.NewConfig(versionSpec)
	assert.NoError(t, err)
}

func runTestCheckerBuilders(t *testing.T, versionSpec *internal.VersionSpec) {
	idsMap := make(map[string]struct{}, len(versionSpec.CheckerBuilders))
	for _, checkerBuilder := range versionSpec.CheckerBuilders {
		_, ok := idsMap[checkerBuilder.ID()]
		assert.False(t, ok, "duplicated id %q", checkerBuilder.ID())
		idsMap[checkerBuilder.ID()] = struct{}{}
	}
	allCategoriesMap := stringutil.SliceToMap(versionSpec.AllCategories)
	for id := range idsMap {
		expectedID := stringutil.ToUpperSnakeCase(id)
		assert.Equal(t, expectedID, id)
		categories, ok := versionSpec.IDToCategories[id]
		assert.True(t, ok, "id %q categories are not configured", id)
		assert.True(t, len(categories) > 0, "id %q must have categories", id)
		for _, category := range categories {
			expectedCategory := stringutil.ToUpperSnakeCase(category)
			assert.Equal(t, expectedCategory, category)
			_, ok := allCategoriesMap[category]
			assert.True(t, ok, "category %q configured for id %q is not a known category", category, id)
		}
	}
	for id := range versionSpec.IDToCategories {
		_, ok := idsMap[id]
		assert.True(t, ok, "id %q configured in categories is not added to checkerBuilders", id)
	}
}
