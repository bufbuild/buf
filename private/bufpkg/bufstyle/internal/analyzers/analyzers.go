package analyzers

import (
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/america"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/casing"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/packagefilename"
	"github.com/bufbuild/buf/private/bufpkg/bufstyle/internal/analyzers/typeban"
	"golang.org/x/tools/go/analysis"
)

// New returns all Analyzers.
//
// We don't store this as a global because we modify these.
func New() []*analysis.Analyzer {
	return append(
		america.New(),
		casing.New(),
		packagefilename.New(),
		typeban.New(),
	)
}
