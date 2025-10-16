package casing

import (
	"testing"

	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzerstesting"
)

func TestAll(t *testing.T) {
	analyzerstesting.Run(t, New())
}
