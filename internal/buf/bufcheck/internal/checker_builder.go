package internal

import (
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/protodesc"
)

// CheckerBuilder is a checker builder.
type CheckerBuilder struct {
	id         string
	newPurpose func(ConfigBuilder) (string, error)
	newCheck   func(ConfigBuilder) (CheckFunc, error)
}

// NewCheckerBuilder returns a new CheckerBuilder.
func NewCheckerBuilder(
	id string,
	newPurpose func(ConfigBuilder) (string, error),
	newCheck func(ConfigBuilder) (CheckFunc, error),
) *CheckerBuilder {
	return &CheckerBuilder{
		id:         id,
		newPurpose: newPurpose,
		newCheck:   newCheck,
	}
}

// NewNopCheckerBuilder returns a new CheckerBuilder for the direct
// purpose and CheckFunc.
func NewNopCheckerBuilder(
	id string,
	purpose string,
	checkFunc CheckFunc,
) *CheckerBuilder {
	return NewCheckerBuilder(
		id,
		newNopPurpose(purpose),
		newNopCheckFunc(checkFunc),
	)
}

// NewChecker returns a new Checker.
//
// Categories will be sorted and Purpose will be prepended with "Checks that "
// and appended with ".".
//
// Categories is an actual copy from the checkerBuilder.
func (c *CheckerBuilder) NewChecker(configBuilder ConfigBuilder, categories []string) (*Checker, error) {
	purpose, err := c.newPurpose(configBuilder)
	if err != nil {
		return nil, err
	}
	check, err := c.newCheck(configBuilder)
	if err != nil {
		return nil, err
	}
	return newChecker(
		c.id,
		categories,
		purpose,
		check,
	), nil
}

// ID returns the id.
func (c *CheckerBuilder) ID() string {
	return c.id
}

func newNopPurpose(purpose string) func(ConfigBuilder) (string, error) {
	return func(ConfigBuilder) (string, error) {
		return purpose, nil
	}
}

func newNopCheckFunc(
	f func(string, []protodesc.File, []protodesc.File) ([]*analysis.Annotation, error),
) func(ConfigBuilder) (CheckFunc, error) {
	return func(ConfigBuilder) (CheckFunc, error) {
		return f, nil
	}
}
