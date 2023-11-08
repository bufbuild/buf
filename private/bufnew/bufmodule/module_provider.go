package bufmodule

import (
	"context"
	"errors"
	"fmt"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
)

// ModuleProvider provides Modules for ModuleInfos.
//
// TODO: Add plural method? Will make calls below a lot more efficient in the case
// of overlapinfog FileNodes.
type ModuleProvider interface {
	// GetModuleForModuleInfo gets the Module for the given ModuleInfo.
	//
	// The ModuleInfo must have a non-nil ModuleFullName.
	GetModuleForModuleInfo(context.Context, ModuleInfo) (Module, error)
}

// NewAPIModuleProvider returns a new ModuleProvider for the given API client.
//
// The Modules returned will be lazily-loaded: All functions except for the ModuleInfo
// functions will be loaded only when called. This allows us to more widely use the Module
// as a type (such as with dependencies) without incurring the lookup and building cost when
// all we want is ModuleInfo-related properties.
func NewAPIModuleProvider(clientProvider bufapi.ClientProvider) ModuleProvider {
	return newLazyModuleProvider(newAPIModuleProvider(clientProvider))
}

// *** PRIVATE ***

// apiModuleProvider

type apiModuleProvider struct {
	clientProvider bufapi.ClientProvider
}

func newAPIModuleProvider(clientProvider bufapi.ClientProvider) *apiModuleProvider {
	return &apiModuleProvider{
		clientProvider: clientProvider,
	}
}

func (a *apiModuleProvider) GetModuleForModuleInfo(ctx context.Context, moduleInfo ModuleInfo) (Module, error) {
	moduleFullName := moduleInfo.ModuleFullName()
	if moduleFullName == nil {
		return nil, fmt.Errorf("ModuleInfo %v did not have ModuleFullName, cannot call GetModuleForGetModuleInfo", moduleInfo)
	}
	var resourceRef *modulev1beta1.ResourceRef
	if commitID := moduleInfo.CommitID(); commitID != "" {
		resourceRef = &modulev1beta1.ResourceRef{
			Value: &modulev1beta1.ResourceRef_Id{
				Id: moduleInfo.CommitID(),
			},
		}
	} else {
		digest, err := moduleInfo.Digest()
		if err != nil {
			return nil, err
		}
		resourceRef = &modulev1beta1.ResourceRef{
			Value: &modulev1beta1.ResourceRef_Name_{
				Name: &modulev1beta1.ResourceRef_Name{
					Owner:  moduleFullName.Owner(),
					Module: moduleFullName.Name(),
					// TODO: change to digest when PR is merged
					Child: &modulev1beta1.ResourceRef_Name_Ref{
						Ref: digest.String(),
					},
				},
			},
		}
	}
	response, err := a.clientProvider.CommitServiceClient(moduleFullName.Registry()).GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					&modulev1beta1.GetCommitNodesRequest_Value{
						ResourceRef: resourceRef,
					},
				},
			},
		),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Msg.CommitNodes) != 1 {
		return nil, fmt.Errorf("expected 1 CommitNode, got %d", len(response.Msg.CommitNodes))
	}
	//commitNode := response.Msg.CommitNodes[0]
	// Can ignore the Commit field, as we already have all this information on ModuleInfo.
	// TODO: deal with Deps field when we have figured out deps on Modules
	return nil, errors.New("TODO")
}

// lazyModuleProvider

type lazyModuleProvider struct {
	delegate ModuleProvider
}

func newLazyModuleProvider(delegate ModuleProvider) *lazyModuleProvider {
	if lazyModuleProvider, ok := delegate.(*lazyModuleProvider); ok {
		return lazyModuleProvider
	}
	return &lazyModuleProvider{
		delegate: delegate,
	}
}

func (l *lazyModuleProvider) GetModuleForModuleInfo(ctx context.Context, moduleInfo ModuleInfo) (Module, error) {
	return newLazyModule(
		ctx,
		moduleInfo,
		func() (Module, error) {
			// Using ctx on GetModuleForModuleInfo and ignoring the contexts passed to
			// Module functions - arguable both ways for different reasons.
			return l.delegate.GetModuleForModuleInfo(ctx, moduleInfo)
		},
	), nil
}
