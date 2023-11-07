package bufmodule

import (
	"context"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

type ModuleDep interface {
	Module(context.Context) (Module, error)
	Digest(context.Context) (bufcas.Digest, error)
	IsColocated() bool

	isModuleDep()
}
