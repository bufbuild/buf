// Copyright 2020 Buf Technologies Inc.
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
	"github.com/bufbuild/buf/internal/pkg/util/utilstring"
	"github.com/stretchr/testify/assert"
)

// RunTestDefaultConfigBuilder runs the test.
func RunTestDefaultConfigBuilder(
	t *testing.T,
	checkerBuilders []*internal.CheckerBuilder,
	idToCategories map[string][]string,
	defaultCategories []string,
) {
	_, err := internal.ConfigBuilder{}.NewConfig(checkerBuilders, idToCategories, defaultCategories)
	assert.NoError(t, err)
}

// RunTestCheckerBuilders runs the test.
func RunTestCheckerBuilders(
	t *testing.T,
	checkerBuilders []*internal.CheckerBuilder,
	idToCategories map[string][]string,
	allCategories []string,
) {
	idsMap := make(map[string]struct{}, len(checkerBuilders))
	for _, checkerBuilder := range checkerBuilders {
		_, ok := idsMap[checkerBuilder.ID()]
		assert.False(t, ok, "duplicated id %q", checkerBuilder.ID())
		idsMap[checkerBuilder.ID()] = struct{}{}
	}
	allCategoriesMap := utilstring.SliceToMap(allCategories)
	for id := range idsMap {
		expectedID := utilstring.ToUpperSnakeCase(id)
		assert.Equal(t, expectedID, id)
		categories, ok := idToCategories[id]
		assert.True(t, ok, "id %q categories are not configured", id)
		assert.True(t, len(categories) > 0, "id %q must have categories", id)
		for _, category := range categories {
			expectedCategory := utilstring.ToUpperSnakeCase(category)
			assert.Equal(t, expectedCategory, category)
			_, ok := allCategoriesMap[category]
			assert.True(t, ok, "category %q configured for id %q is not a known category", category, id)
		}
	}
	for id := range idToCategories {
		_, ok := idsMap[id]
		assert.True(t, ok, "id %q configured in categories is not added to checkerBuilders", id)
	}
}
