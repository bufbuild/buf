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

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	storagev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/storage/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufnew/bufapi"
	"github.com/bufbuild/buf/private/pkg/storage"
)

// ModuleDataProvider provides ModulesDatas.
type ModuleDataProvider interface {
	// GetModuleDataForModuleKey gets the ModuleData for the ModuleKey.
	GetModuleDataForModuleKey(ctx context.Context, moduleKey ModuleKey) (ModuleData, error)
}

// NewAPIModuleDataProvider returns a new ModuleDataProvider for the given API client.
func NewAPIModuleDataProvider(clientProvider bufapi.ClientProvider) ModuleDataProvider {
	return newAPIModuleDataProvider(clientProvider)
}

// *** PRIVATE ***

// apiModuleDataProvider

type apiModuleDataProvider struct {
	clientProvider bufapi.ClientProvider
}

func newAPIModuleDataProvider(clientProvider bufapi.ClientProvider) *apiModuleDataProvider {
	return &apiModuleDataProvider{
		clientProvider: clientProvider,
	}
}

func (a *apiModuleDataProvider) GetModuleDataForModuleKey(
	ctx context.Context,
	moduleKey ModuleKey,
) (ModuleData, error) {
	// Note that we could actually just use the Digest. However, we want to force the caller
	// to provide a CommitID, so that we can document that all Modules returned from a
	// ModuleDataProvider will have a CommitID. We also want to prevent callers from having
	// to invoke moduleKey.Digest() unnecessarily, as this could cause unnecessary lazy loading.
	// If we were to instead have GetModuleDataForDigest(context.Context, ModuleFullName, bufcas.Digest),
	// we would never have the CommitID, even in cases where we have it via the ModuleKey.
	// If we were to provide both GetModuleDataForModuleKey and GetModuleForDigest, then why would anyone
	// ever call GetModuleDataForModuleKey? This forces a single call pattern for now.
	response, err := a.clientProvider.CommitServiceClient(moduleKey.ModuleFullName().Registry()).GetCommitNodes(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitNodesRequest{
				Values: []*modulev1beta1.GetCommitNodesRequest_Value{
					{
						ResourceRef: &modulev1beta1.ResourceRef{
							Value: &modulev1beta1.ResourceRef_Id{
								Id: moduleKey.CommitID(),
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
		return nil, fmt.Errorf("expected 1 CommitNode, got %d", len(response.Msg.CommitNodes))
	}
	// TODO: tamper-proof protoCommitNode.Commit vs moduleKey.Digest()? Implications on lazy loading?
	protoCommitNode := response.Msg.CommitNodes[0]
	return NewModuleData(
		moduleKey,
		func() (storage.ReadBucket, error) {
			return a.getBucketForProtoFileNodes(ctx, protoCommitNode.FileNodes)
		},
		func() ([]ModuleKey, error) {
			return a.getModuleKeysForProtoCommits(ctx, protoCommitNode.Deps)
		},
	)
}

func (a *apiModuleDataProvider) getBucketForProtoFileNodes(ctx context.Context, protoFileNodes []*storagev1beta1.FileNode) (storage.ReadBucket, error) {
	// TODO: tamper-proofing here?
	return nil, errors.New("TODO")
}

func (a *apiModuleDataProvider) getModuleKeysForProtoCommits(ctx context.Context, protoCommits []*modulev1beta1.Commit) ([]ModuleKey, error) {
	return nil, errors.New("TODO")
}

//// lazyModuleDataProvider

//type lazyModuleDataProvider struct {
//delegate ModuleDataProvider
//// Cache may be nil.
//cache *cache
//}

//// Cache may be nil.
//func newLazyModuleDataProvider(delegate ModuleDataProvider, cache *cache) *lazyModuleDataProvider {
//if lazyModuleDataProvider, ok := delegate.(*lazyModuleDataProvider); ok {
//delegate = lazyModuleDataProvider.delegate
//}
//return &lazyModuleDataProvider{
//delegate: delegate,
//cache:    cache,
//}
//}

//func (l *lazyModuleDataProvider) GetModuleDataForModuleKey(
//ctx context.Context,
//moduleKey ModuleKey,
//) (ModuleData, error) {
//return newModuleData(
//moduleKey,
//)

//return newLazyModule(
//ctx,
//l.cache,
//moduleKey,
//// Always set to true by default.
//true,
//func() (Module, error) {
//// Using ctx on GetModuleDataForModuleKey and ignoring the contexts passed to
//// Module functions - arguable both ways for different reasons.
//return l.delegate.GetModuleDataForModuleKey(ctx, moduleKey)
//},
//), nil
//}

//// lazyModule

//type lazyModule struct {
//ModuleKey

//// Cache may be nil.
//cache *cache

//isTargetModule bool
//moduleSet      ModuleSet

//getModuleAndDigest func() (Module, bufcas.Digest, error)
//getModuleDeps      func() ([]ModuleDep, error)
//}

//func newLazyModule(
//ctx context.Context,
//// May be nil.
//cache *cache,
//moduleKey ModuleKey,
//isTargetModule bool,
//getModuleFunc func() (Module, error),
//) Module {
//lazyModule := &lazyModule{
//ModuleKey:      moduleKey,
//isTargetModule: isTargetModule,
//getModuleAndDigest: syncextended.OnceValues3(
//func() (Module, bufcas.Digest, error) {
//module, err := getModuleFunc()
//if err != nil {
//return nil, nil, err
//}
//expectedDigest, err := moduleKey.Digest()
//if err != nil {
//return nil, nil, err
//}
//actualDigest, err := module.Digest()
//if err != nil {
//return nil, nil, err
//}
//if !bufcas.DigestEqual(expectedDigest, actualDigest) {
//return nil, nil, fmt.Errorf("expected digest %v, got %v", expectedDigest, actualDigest)
//}
//if expectedDigest == nil {
//// This should never happen.
//return nil, nil, fmt.Errorf("digest was nil for ModuleKey %v", moduleKey)
//}
//return module, actualDigest, nil
//},
//),
//}
//lazyModule.getModuleDeps = sync.OnceValues(
//func() ([]ModuleDep, error) {
//module, _, err := lazyModule.getModuleAndDigest()
//if err != nil {
//return nil, err
//}
//if cache != nil {
//// Prefer declared dependencies via the cache if they exist, as these are not read from remote.
//// For example, a Module read may have deps within a Workspace, we want to prefer those deps
//// If we have a cache, we're saying that all expected deps are within the cache, therefore
//// we can use it.
////
//// Make sure to pass the lazyModule, not the module! The lazyModule is what will be within the cache.
//return getModuleDeps(ctx, cache, lazyModule)
//}
//return module.ModuleDeps()
//},
//)
//return lazyModule
//}

//func (m *lazyModule) OpaqueID() string {
//return m.ModuleKey.ModuleFullName().String()
//}

//func (*lazyModule) BucketID() string {
//return ""
//}

//func (m *lazyModule) GetFile(ctx context.Context, path string) (File, error) {
//module, _, err := m.getModuleAndDigest()
//if err != nil {
//return nil, err
//}
//return module.GetFile(ctx, path)
//}

//func (m *lazyModule) StatFileInfo(ctx context.Context, path string) (FileInfo, error) {
//module, _, err := m.getModuleAndDigest()
//if err != nil {
//return nil, err
//}
//return module.StatFileInfo(ctx, path)
//}

//func (m *lazyModule) WalkFileInfos(
//ctx context.Context,
//f func(FileInfo) error,
//options ...WalkFileInfosOption,
//) error {
//module, _, err := m.getModuleAndDigest()
//if err != nil {
//return err
//}
//return module.WalkFileInfos(ctx, f, options...)
//}

//func (m *lazyModule) Digest() (bufcas.Digest, error) {
//// This does not result in a remote call if you are just reading digests.
//return m.ModuleKey.Digest()
//// TODO: Make sure we don't need to check the remote digest here. We probably do not.
//// Checking the remote digest is commented out here.
////_, digest, err := m.getModuleAndDigest()
////return digest, err
//}

//func (m *lazyModule) ModuleDeps() ([]ModuleDep, error) {
//return m.getModuleDeps()
//}

//func (m *lazyModule) IsTargetModule() bool {
//return m.isTargetModule
//}

//func (m *lazyModule) ModuleSet() ModuleSet {
//return m.moduleSet
//}

//func (m *lazyModule) setIsTargetModule(isTargetModule bool) {
//m.isTargetModule = isTargetModule
//}

//func (m *lazyModule) setModuleSet(moduleSet ModuleSet) {
//m.moduleSet = moduleSet
//}

//func (*lazyModule) isModuleReadBucket() {}
//func (*lazyModule) isModule()           {}
