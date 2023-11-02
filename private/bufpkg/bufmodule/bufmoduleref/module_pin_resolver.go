package bufmoduleref

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"github.com/bufbuild/buf/private/pkg/connectclient"
)

type modulePinResolver struct {
	clientConfig *connectclient.Config
}

func newModulePinResolver(clientConfig *connectclient.Config) ModulePinResolver {
	return &modulePinResolver{
		clientConfig: clientConfig,
	}
}

func (r *modulePinResolver) ResolveModulePins(
	ctx context.Context,
	moduleRefsToResolve []ModuleReference,
	options ...ResolveModulePinsOption,
) ([]ModulePin, error) {
	var opts resolveModulePinsOpts
	for _, opt := range options {
		opt(&opts)
	}
	if len(moduleRefsToResolve) == 0 {
		return opts.existingModulePins, nil
	}
	// We don't know the module identity for which we are resolving module pins, but we know there's
	// at least one module reference, so we can select a remote based on the references.
	selectedRef := SelectReferenceForRemote(moduleRefsToResolve)
	remote := selectedRef.Remote()
	// TODO: someday we'd like to be able to execute the core algorithm client-side.
	service := connectclient.Make(r.clientConfig, remote, registryv1alpha1connect.NewResolveServiceClient)
	protoDependencyModuleReferences := NewProtoModuleReferencesForModuleReferences(moduleRefsToResolve...)
	currentProtoModulePins := NewProtoModulePinsForModulePins(opts.existingModulePins...)
	resp, err := service.GetModulePins(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetModulePinsRequest{
			ModuleReferences:  protoDependencyModuleReferences,
			CurrentModulePins: currentProtoModulePins,
		}),
	)
	if err != nil {
		if remote != bufconnect.DefaultRemote {
			// TODO: Taken from bufcli.NewInvalidRemoteError, which we can't dep on; we could instead
			// expose this via a sentinel error and let the CLI command do this.
			return nil, fmt.Errorf("%w. Are you sure %q is a Buf Schema Registry?", err, remote)
		}
		return nil, err
	}
	dependencyModulePins, err := NewModulePinsForProtos(resp.Msg.ModulePins...)
	if err != nil {
		return nil, err
	}
	return dependencyModulePins, nil
}

var _ ModulePinResolver = (*modulePinResolver)(nil)
