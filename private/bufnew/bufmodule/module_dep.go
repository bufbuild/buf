package bufmodule

import (
	"context"
	"errors"

	"github.com/bufbuild/buf/private/bufpkg/bufcas"
)

type ModuleDep interface {
	Module(context.Context) (Module, error)
	Digest(context.Context) (bufcas.Digest, error)
	IsColocated() bool

	isModuleDep()
}

// *** PRIVATE ***

type moduleDep struct{}

func newModuleDep() *moduleDep {
	return &moduleDep{}
}

func (m *moduleDep) Module(ctx context.Context) (Module, error) {
	return nil, errors.New("TODO")
}

func (m *moduleDep) Digest(ctx context.Context) (bufcas.Digest, error) {
	return nil, errors.New("TODO")
}

func (m *moduleDep) IsColocated() bool {
	return false
}

func (*moduleDep) isModuleDep() {}
