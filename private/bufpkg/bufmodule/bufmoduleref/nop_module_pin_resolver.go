package bufmoduleref

import "context"

type nopModulePinResolver struct{}

func newNopModulePinResolver() ModulePinResolver {
	return &nopModulePinResolver{}
}

func (r *nopModulePinResolver) ResolveModulePins(
	ctx context.Context,
	moduleRefsToResolve []ModuleReference,
	opts ...ResolveModulePinsOption,
) ([]ModulePin, error) {
	var options resolveModulePinsOpts
	for _, opt := range opts {
		opt(&options)
	}
	return options.existingModulePins, nil
}

var _ ModulePinResolver = (*nopModulePinResolver)(nil)
