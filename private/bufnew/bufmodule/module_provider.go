package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"buf.build/gen/go/bufbuild/registry/connectrpc/go/buf/registry/module/v1beta1/modulev1beta1connect"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
)

// ModuleProvider provides Modules for ModulePins.
//
// TODO: Add plural method? Will make calls below a lot more efficient in the case
// of overlapping FileNodes.
type ModuleProvider interface {
	GetModuleForModulePin(context.Context, ModulePin) (Module, error)
}

// NewAPIModuleProvider returns a new ModuleProvider for the given API client.
//
// The Modules returned will be lazily-loaded: All functions except for the ModuleInfo
// functions will be loaded only when called. This allows us to more widely use the Module
// as a type (such as with dependencies) without incurring the lookup and building cost when
// all we want is ModuleInfo-related properties.
func NewAPIModuleProvider(client modulev1beta1connect.CommitServiceClient) ModuleProvider {
	return newLazyLoadModuleProvider(newAPIModuleProvider(client))
}

// *** PRIVATE ***

// apiModuleProvider

type apiModuleProvider struct {
	client modulev1beta1connect.CommitServiceClient
}

func newAPIModuleProvider(client modulev1beta1connect.CommitServiceClient) *apiModuleProvider {
	return &apiModuleProvider{
		client: client,
	}
}

func (a *apiModuleProvider) GetModuleForModulePin(ctx context.Context, modulePin ModulePin) (Module, error) {
	response, err := a.client.GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					&modulev1beta1.GetCommitNodesRequest_Value{
						ResourceRef: &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: modulePin.CommitID(),
							},
						},
					},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.CommitNodes) != 1 {
		// TODO: should we just triple-document the API such that we guarantee the length of
		// the response is equal to the length of the request, and not do these checks everywhere?
		return nil, fmt.Errorf("expected 1 CommitNode, got %d", len(response.Msg.CommitNodes))
	}
	//commitNode := response.Msg.CommitNodes[0]
	// Can ignore the Commit field, as we already have all this information on ModulePin.
	// TODO: deal with Deps field when we have figured out deps on Modules
	return newLazyLoadModule(
		ModulePinToModuleInfo(modulePin),
		func() (Module, error) {
			// TODO: convert FileNodes to Blobs to a *module
			return nil, errors.New("TODO")
		},
	), nil
}

// lazyLoadModuleProvider

type lazyLoadModuleProvider struct {
	delegate ModuleProvider
}

func newLazyLoadModuleProvider(delegate ModuleProvider) *lazyLoadModuleProvider {
	return &lazyLoadModuleProvider{
		delegate: delegate,
	}
}

func (l *lazyLoadModuleProvider) GetModuleForModulePin(ctx context.Context, modulePin ModulePin) (Module, error) {
	return newLazyLoadModule(
		ModulePinToModuleInfo(modulePin),
		func() (Module, error) {
			// Using ctx on GetModuleForModulePin and ignoring the contexts passed to
			// Module functions - arguable both ways for different reasons.
			return l.delegate.GetModuleForModulePin(ctx, modulePin)
		},
	), nil
}

// lazyLoadModule

type lazyLoadModule struct {
	ModuleInfo

	getModule func() (Module, error)
}

func newLazyLoadModule(
	moduleInfo ModuleInfo,
	getModule func() (Module, error),
) Module {
	return &lazyLoadModule{
		ModuleInfo: moduleInfo,
		getModule:  sync.OnceValues(getModule),
	}
}

func (m *lazyLoadModule) GetFile(ctx context.Context, path string) (File, error) {
	module, err := m.getModule()
	if err != nil {
		return nil, err
	}
	return module.GetFile(ctx, path)
}

func (m *lazyLoadModule) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	module, err := m.getModule()
	if err != nil {
		return nil, err
	}
	return module.StatFileInfo(ctx, path)
}

func (m *lazyLoadModule) WalkFileInfos(ctx context.Context, f func(FileInfo) error) error {
	module, err := m.getModule()
	if err != nil {
		return err
	}
	return module.WalkFileInfos(ctx, f)
}

func (*lazyLoadModule) isModuleReadBucket() {}
func (*lazyLoadModule) isModule()           {}
