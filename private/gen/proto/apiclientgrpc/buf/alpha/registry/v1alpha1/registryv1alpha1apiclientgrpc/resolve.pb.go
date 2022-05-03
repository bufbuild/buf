// Code generated by protoc-gen-go-apiclientgrpc. DO NOT EDIT.

package registryv1alpha1apiclientgrpc

import (
	context "context"
	v1alpha11 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/module/v1alpha1"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	zap "go.uber.org/zap"
)

type resolveService struct {
	logger          *zap.Logger
	client          v1alpha1.ResolveServiceClient
	contextModifier func(context.Context) context.Context
}

// GetModulePins finds all the latest digests and respective dependencies of
// the provided module references and picks a set of distinct modules pins.
//
// Note that module references with commits should still be passed to this function
// to make sure this function can do dependency resolution.
//
// This function also deals with tiebreaking what ModulePin wins for the same repository.
func (s *resolveService) GetModulePins(
	ctx context.Context,
	moduleReferences []*v1alpha11.ModuleReference,
	currentModulePins []*v1alpha11.ModulePin,
) (modulePins []*v1alpha11.ModulePin, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetModulePins(
		ctx,
		&v1alpha1.GetModulePinsRequest{
			ModuleReferences:  moduleReferences,
			CurrentModulePins: currentModulePins,
		},
	)
	if err != nil {
		return nil, err
	}
	return response.ModulePins, nil
}

type localResolveService struct {
	logger          *zap.Logger
	client          v1alpha1.LocalResolveServiceClient
	contextModifier func(context.Context) context.Context
}

// GetLocalModulePins gets the latest pins for the specified local module references.
// It also includes all of the modules transitive dependencies for the specified references.
//
// We want this for two reasons:
//
// 1. It makes it easy to say "we know we're looking for owner/repo on this specific remote".
//    While we could just do this in GetModulePins by being aware of what our remote is
//    (something we probably still need to know, DNS problems aside, which are more
//    theoretical), this helps.
// 2. Having a separate method makes us able to say "do not make decisions about what
//    wins between competing pins for the same repo". This should only be done in
//    GetModulePins, not in this function, i.e. only done at the top level.
func (s *localResolveService) GetLocalModulePins(
	ctx context.Context,
	localModuleReferences []*v1alpha1.LocalModuleReference,
) (localModuleResolveResults []*v1alpha1.LocalModuleResolveResult, dependencies []*v1alpha11.ModulePin, _ error) {
	if s.contextModifier != nil {
		ctx = s.contextModifier(ctx)
	}
	response, err := s.client.GetLocalModulePins(
		ctx,
		&v1alpha1.GetLocalModulePinsRequest{
			LocalModuleReferences: localModuleReferences,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return response.LocalModuleResolveResults, response.Dependencies, nil
}
