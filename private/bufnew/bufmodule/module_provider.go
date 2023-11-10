// Copyright 2020-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bufmodule

import (
	"context"
	"errors"
	"fmt"
	"sync"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufcas"
	"github.com/bufbuild/buf/private/pkg/syncextended"
)

// ModuleProvider provides Modules.
//
// A Modules returned from a ModuleProvider *must* have ModuleFullName and CommitID.
type ModuleProvider interface {
	// GetModuleForModuleKey gets the Module for the ModuleKey.
	//
	// Modules returned from ModuleProviders will always have IsTargetModule() set to true.
	// The ModuleSetBuilder can choose to edit this during building.
	GetModuleForModuleKey(ctx context.Context, moduleKey ModuleKey) (Module, error)
}

// NewAPIModuleProvider returns a new ModuleProvider for the given API client.
func NewAPIModuleProvider(clientProvider bufapi.ClientProvider) ModuleProvider {
	return newLazyModuleProvider(newAPIModuleProvider(clientProvider), nil)
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

func (a *apiModuleProvider) GetModuleForModuleKey(
	ctx context.Context,
	moduleKey ModuleKey,
) (Module, error) {
	// Note that we could actually just use the Digest. However, we want to force the caller
	// to provide a CommitID, so that we can document that all Modules returned from a
	// ModuleProvider will have a CommitID. We also want to prevent callers from having
	// to invoke moduleKey.Digest() unnecessarily, as this could cause unnecessary lazy loading.
	// If we were to instead have GetModuleForDigest(context.Context, ModuleFullName, bufcas.Digest),
	// we would never have the CommitID, even in cases where we have it via the ModuleKey.
	// If we were to provide both GetModuleForModuleKey and GetModuleForDigest, then why would anyone
	// ever call GetModuleForModuleKey? This forces a single call pattern for now.
	return a.getModuleForResourceRef(
		ctx,
		moduleKey.ModuleFullName().Registry(),
		&modulev1beta1.ResourceRef{
			Value: &modulev1beta1.ResourceRef_Id{
				Id: moduleKey.CommitID(),
			},
		},
	)
}

func (a *apiModuleProvider) getModuleForResourceRef(
	ctx context.Context,
	registryHostname string,
	resourceRef *modulev1beta1.ResourceRef,
) (Module, error) {
	response, err := a.clientProvider.CommitServiceClient(registryHostname).GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					{
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
	// Cache may be nil.
	cache *cache
}

// Cache may be nil.
func newLazyModuleProvider(delegate ModuleProvider, cache *cache) *lazyModuleProvider {
	if lazyModuleProvider, ok := delegate.(*lazyModuleProvider); ok {
		delegate = lazyModuleProvider.delegate
	}
	return &lazyModuleProvider{
		delegate: delegate,
		cache:    cache,
	}
}

func (l *lazyModuleProvider) GetModuleForModuleKey(
	ctx context.Context,
	moduleKey ModuleKey,
) (Module, error) {
	return newLazyModule(
		ctx,
		l.cache,
		moduleKey,
		// Always set to true by default.
		true,
		func() (Module, error) {
			// Using ctx on GetModuleForModuleKey and ignoring the contexts passed to
			// Module functions - arguable both ways for different reasons.
			return l.delegate.GetModuleForModuleKey(ctx, moduleKey)
		},
	), nil
}

// lazyModule

type lazyModule struct {
	ModuleKey

	// Cache may be nil.
	cache *cache

	isTargetModule bool
	moduleSet      ModuleSet

	getModuleAndDigest func() (Module, bufcas.Digest, error)
	getModuleDeps      func() ([]ModuleDep, error)
}

func newLazyModule(
	ctx context.Context,
	// May be nil.
	cache *cache,
	moduleKey ModuleKey,
	isTargetModule bool,
	getModuleFunc func() (Module, error),
) Module {
	lazyModule := &lazyModule{
		ModuleKey:      moduleKey,
		isTargetModule: isTargetModule,
		getModuleAndDigest: syncextended.OnceValues3(
			func() (Module, bufcas.Digest, error) {
				module, err := getModuleFunc()
				if err != nil {
					return nil, nil, err
				}
				expectedDigest, err := moduleKey.Digest()
				if err != nil {
					return nil, nil, err
				}
				actualDigest, err := module.Digest()
				if err != nil {
					return nil, nil, err
				}
				if !bufcas.DigestEqual(expectedDigest, actualDigest) {
					return nil, nil, fmt.Errorf("expected digest %v, got %v", expectedDigest, actualDigest)
				}
				if expectedDigest == nil {
					// This should never happen.
					return nil, nil, fmt.Errorf("digest was nil for ModuleKey %v", moduleKey)
				}
				return module, actualDigest, nil
			},
		),
	}
	lazyModule.getModuleDeps = sync.OnceValues(
		func() ([]ModuleDep, error) {
			module, _, err := lazyModule.getModuleAndDigest()
			if err != nil {
				return nil, err
			}
			if cache != nil {
				// Prefer declared dependencies via the cache if they exist, as these are not read from remote.
				// For example, a Module read may have deps within a Workspace, we want to prefer those deps
				// If we have a cache, we're saying that all expected deps are within the cache, therefore
				// we can use it.
				//
				// Make sure to pass the lazyModule, not the module! The lazyModule is what will be within the cache.
				return getModuleDeps(ctx, cache, lazyModule)
			}
			return module.ModuleDeps()
		},
	)
	return lazyModule
}

func (m *lazyModule) OpaqueID() string {
	return m.ModuleKey.ModuleFullName().String()
}

func (*lazyModule) BucketID() string {
	return ""
}

func (m *lazyModule) GetFile(ctx context.Context, path string) (File, error) {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return nil, err
	}
	return module.GetFile(ctx, path)
}

func (m *lazyModule) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return nil, err
	}
	return module.StatFileInfo(ctx, path)
}

func (m *lazyModule) WalkFileInfos(
	ctx context.Context,
	f func(FileInfo) error,
	options ...WalkFileInfosOption,
) error {
	module, _, err := m.getModuleAndDigest()
	if err != nil {
		return err
	}
	return module.WalkFileInfos(ctx, f, options...)
}

func (m *lazyModule) Digest() (bufcas.Digest, error) {
	// This does not result in a remote call if you are just reading digests.
	return m.ModuleKey.Digest()
	// TODO: Make sure we don't need to check the remote digest here. We probably do not.
	// Checking the remote digest is commented out here.
	//_, digest, err := m.getModuleAndDigest()
	//return digest, err
}

func (m *lazyModule) ModuleDeps() ([]ModuleDep, error) {
	return m.getModuleDeps()
}

func (m *lazyModule) IsTargetModule() bool {
	return m.isTargetModule
}

func (m *lazyModule) ModuleSet() ModuleSet {
	return m.moduleSet
}

func (m *lazyModule) setIsTargetModule(isTargetModule bool) {
	m.isTargetModule = isTargetModule
}

func (m *lazyModule) setModuleSet(moduleSet ModuleSet) {
	m.moduleSet = moduleSet
}

func (*lazyModule) isModuleReadBucket() {}
func (*lazyModule) isModule()           {}
