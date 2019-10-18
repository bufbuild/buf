package bufbreaking

import "github.com/bufbuild/buf/internal/buf/bufcheck/internal"

type checker struct {
	*internal.Checker
}

func newChecker(internalChecker *internal.Checker) *checker {
	return &checker{Checker: internalChecker}
}

func (c *checker) internalBreaking() *internal.Checker {
	return c.Checker
}

func internalCheckersToCheckers(internalCheckers []*internal.Checker) []Checker {
	if internalCheckers == nil {
		return nil
	}
	checkers := make([]Checker, len(internalCheckers))
	for i, internalChecker := range internalCheckers {
		checkers[i] = newChecker(internalChecker)
	}
	return checkers
}

func checkersToInternalCheckers(checkers []Checker) []*internal.Checker {
	if checkers == nil {
		return nil
	}
	internalCheckers := make([]*internal.Checker, len(checkers))
	for i, checker := range checkers {
		internalCheckers[i] = checker.internalBreaking()
	}
	return internalCheckers
}
