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

package bufmoduleapi

import (
	"context"
	"errors"
	"io/fs"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"go.uber.org/zap"
)

// NewModuleDataProvider returns a new ModuleDataProvider for the given API client.
//
// A warning is printed to the logger if a given Module is deprecated.
func NewModuleDataProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) bufmodule.ModuleDataProvider {
	return newModuleDataProvider(logger, clientProvider)
}

// *** PRIVATE ***

type moduleDataProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
}

func newModuleDataProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) *moduleDataProvider {
	return &moduleDataProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *moduleDataProvider) GetOptionalModuleDatasForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalModuleData, error) {
	// TODO: Do the work to coalesce ModuleKeys by registry hostname, make calls out to the CommitService
	// per registry, then get back the resulting data, and order it in the same order as the input ModuleKeys.
	// Make sure to respect 250 max.
	optionalModuleDatas := make([]bufmodule.OptionalModuleData, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		moduleData, err := a.getModuleDataForModuleKey(ctx, moduleKey)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		optionalModuleDatas[i] = bufmodule.NewOptionalModuleData(moduleData)
	}
	return optionalModuleDatas, nil
}

func (a *moduleDataProvider) getModuleDataForModuleKey(
	ctx context.Context,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.ModuleData, error) {
	registryHostname := moduleKey.ModuleFullName().Registry()

	protoResourceRef, err := getProtoResourceRefForModuleKey(moduleKey)
	if err != nil {
		return nil, err
	}

	_ = registryHostname
	_ = protoResourceRef

	return nil, errors.New("TODO")
}

func getProtoResourceRefForModuleKey(moduleKey bufmodule.ModuleKey) (*modulev1beta1.ResourceRef, error) {
	// CommitID is optional.
	if commitID := moduleKey.CommitID(); commitID != "" {
		// Note that we could actually just use the Digest. We don't wnat to have to invoke
		// moduleKey.Digest() unnecessarily, as this could cause unnecessary lazy loading.
		return &modulev1beta1.ResourceRef{
			Value: &modulev1beta1.ResourceRef_Id{
				Id: commitID,
			},
		}, nil
	}
	// Naming differently to make sure we differentiate between this and the
	// retrieved digest below.
	moduleKeyDigest, err := moduleKey.Digest()
	if err != nil {
		return nil, err
	}
	protoModuleKeyDigest, err := DigestToProto(moduleKeyDigest)
	if err != nil {
		return nil, err
	}
	return &modulev1beta1.ResourceRef{
		Value: &modulev1beta1.ResourceRef_Name_{
			Name: &modulev1beta1.ResourceRef_Name{
				Owner:  moduleKey.ModuleFullName().Owner(),
				Module: moduleKey.ModuleFullName().Name(),
				Child: &modulev1beta1.ResourceRef_Name_Digest{
					Digest: protoModuleKeyDigest,
				},
			},
		},
	}, nil
}
