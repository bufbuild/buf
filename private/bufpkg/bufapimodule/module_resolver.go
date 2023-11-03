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

package bufapimodule

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufconnect"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule/bufmoduleref"
	registryv1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	"go.uber.org/zap"
)

type moduleResolver struct {
	logger                        *zap.Logger
	repositoryCommitClientFactory RepositoryCommitServiceClientFactory
	resolveServiceClientFactory   ResolveServiceClientFactory
}

func newModuleResolver(
	logger *zap.Logger,
	repositoryCommitServiceClientFactory RepositoryCommitServiceClientFactory,
	resolveServiceClientFactory ResolveServiceClientFactory,
) *moduleResolver {
	return &moduleResolver{
		logger:                        logger,
		repositoryCommitClientFactory: repositoryCommitServiceClientFactory,
		resolveServiceClientFactory:   resolveServiceClientFactory,
	}
}

func (m *moduleResolver) GetModulePin(
	ctx context.Context,
	moduleReference bufmoduleref.ModuleReference,
) (bufmoduleref.ModulePin, error) {
	repositoryCommitService := m.repositoryCommitClientFactory(moduleReference.Remote())
	resp, err := repositoryCommitService.GetRepositoryCommitByReference(
		ctx,
		connect.NewRequest(&registryv1alpha1.GetRepositoryCommitByReferenceRequest{
			RepositoryOwner: moduleReference.Owner(),
			RepositoryName:  moduleReference.Repository(),
			Reference:       moduleReference.Reference(),
		}),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			// Required by ModuleResolver interface spec
			return nil, &fs.PathError{Op: "read", Path: moduleReference.String(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if resp.Msg.RepositoryCommit == nil {
		return nil, errors.New("empty response")
	}
	return bufmoduleref.NewModulePin(
		moduleReference.Remote(),
		moduleReference.Owner(),
		moduleReference.Repository(),
		resp.Msg.RepositoryCommit.Name,
		resp.Msg.RepositoryCommit.ManifestDigest,
	)
}

func (r *moduleResolver) GetModulePins(
	ctx context.Context,
	moduleRefsToResolve []bufmoduleref.ModuleReference,
	existingModulePins []bufmoduleref.ModulePin,
) ([]bufmoduleref.ModulePin, error) {
	if len(moduleRefsToResolve) == 0 {
		return existingModulePins, nil
	}
	// We don't know the module identity for which we are resolving module pins, but we know there's
	// at least one module reference, so we can select a remote based on the references.
	selectedRef := bufmoduleref.SelectReferenceForRemote(moduleRefsToResolve)
	remote := selectedRef.Remote()
	// TODO: someday we'd like to be able to execute the core algorithm client-side.
	service := r.resolveServiceClientFactory(remote)
	protoDependencyModuleReferences := bufmoduleref.NewProtoModuleReferencesForModuleReferences(moduleRefsToResolve...)
	currentProtoModulePins := bufmoduleref.NewProtoModulePinsForModulePins(existingModulePins...)
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
	dependencyModulePins, err := bufmoduleref.NewModulePinsForProtos(resp.Msg.ModulePins...)
	if err != nil {
		return nil, err
	}
	return dependencyModulePins, nil
}
