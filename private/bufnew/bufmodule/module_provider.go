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

// ModuleProvider provides Modules for ModuleInfos.
//
// TODO: Add plural method? Will make calls below a lot more efficient in the case
// of overlapinfog FileNodes.
type ModuleProvider interface {
	GetModuleForModuleInfo(context.Context, ModuleInfo) (Module, error)
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

func (a *apiModuleProvider) GetModuleForModuleInfo(ctx context.Context, moduleInfo ModuleInfo) (Module, error) {
	response, err := a.client.GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					&modulev1beta1.GetCommitNodesRequest_Value{
						ResourceRef: &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: moduleInfo.CommitID(),
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
	// Can ignore the Commit field, as we already have all this information on ModuleInfo.
	// TODO: deal with Deps field when we have figured out deps on Modules
	return newLazyLoadModule(
		ModuleInfoToModuleInfo(moduleInfo),
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

func (l *lazyLoadModuleProvider) GetModuleForModuleInfo(ctx context.Context, moduleInfo ModuleInfo) (Module, error) {
	return newLazyLoadModule(
		ModuleInfoToModuleInfo(moduleInfo),
		func() (Module, error) {
			// Using ctx on GetModuleForModuleInfo and ignoring the contexts passed to
			// Module functions - arguable both ways for different reasons.
			return l.delegate.GetModuleForModuleInfo(ctx, moduleInfo)
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
