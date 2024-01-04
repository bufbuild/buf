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
	"fmt"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/cache"
	"go.uber.org/zap"
)

// protoModuleProvider provides a per-call provider of proto Modules.
//
// We don't want to persist these across calls - this could grow over time and this cache
// isn't an LRU cache, and the information also may change over time.
type protoModuleProvider struct {
	logger           *zap.Logger
	clientProvider   bufapi.ModuleServiceClientProvider
	protoModuleCache cache.Cache[string, *modulev1beta1.Module]
}

func newProtoModuleProvider(
	logger *zap.Logger,
	clientProvider bufapi.ModuleServiceClientProvider,
) *protoModuleProvider {
	return &protoModuleProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *protoModuleProvider) getProtoModuleForModuleID(
	ctx context.Context,
	registry string,
	moduleID string,
) (*modulev1beta1.Module, error) {
	return a.protoModuleCache.GetOrAdd(
		registry+"/"+moduleID,
		func() (*modulev1beta1.Module, error) {
			response, err := a.clientProvider.ModuleServiceClient(registry).GetModules(
				ctx,
				connect.NewRequest(
					&modulev1beta1.GetModulesRequest{
						ModuleRefs: []*modulev1beta1.ModuleRef{
							{
								Value: &modulev1beta1.ModuleRef_Id{
									Id: moduleID,
								},
							},
						},
					},
				),
			)
			if err != nil {
				return nil, err
			}
			if len(response.Msg.Modules) != 1 {
				return nil, fmt.Errorf("expected 1 Module, got %d", len(response.Msg.Modules))
			}
			return response.Msg.Modules[0], nil
		},
	)
}