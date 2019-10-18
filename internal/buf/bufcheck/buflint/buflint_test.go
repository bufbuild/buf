package buflint

import (
	"testing"

	"github.com/bufbuild/buf/internal/buf/bufcheck/internal/internaltesting"
)

func TestDefaultConfigBuilder(t *testing.T) {
	t.Parallel()
	internaltesting.RunTestDefaultConfigBuilder(
		t,
		v1CheckerBuilders,
		v1IDToCategories,
		v1DefaultCategories,
	)
}

func TestCheckerBuilders(t *testing.T) {
	t.Parallel()
	internaltesting.RunTestCheckerBuilders(
		t,
		v1CheckerBuilders,
		v1IDToCategories,
		v1AllCategories,
	)
}
