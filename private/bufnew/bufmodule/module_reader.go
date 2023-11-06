package bufmodule

import "context"

type ModuleReader interface {
	GetModule(ctx context.Context, moduleRef ModuleRef) (Module, error)
}
