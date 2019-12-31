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
