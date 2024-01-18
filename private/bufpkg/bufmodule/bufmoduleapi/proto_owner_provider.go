// Copyright 2020-2024 Buf Technologies, Inc.
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

	ownerv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/cache"
	"go.uber.org/zap"
)

// protoOwnerProvider provides a per-call provider of proto Modules.
//
// We don't want to persist these across calls - this could grow over time and this cache
// isn't an LRU cache, and the information also may change over time.
type protoOwnerProvider struct {
	logger          *zap.Logger
	clientProvider  bufapi.OwnerServiceClientProvider
	protoOwnerCache cache.Cache[string, *ownerv1beta1.Owner]
}

func newProtoOwnerProvider(
	logger *zap.Logger,
	clientProvider bufapi.OwnerServiceClientProvider,
) *protoOwnerProvider {
	return &protoOwnerProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *protoOwnerProvider) getProtoOwnerForOwnerID(
	ctx context.Context,
	registry string,
	ownerID string,
) (*ownerv1beta1.Owner, error) {
	return a.protoOwnerCache.GetOrAdd(
		registry+"/"+ownerID,
		func() (*ownerv1beta1.Owner, error) {
			response, err := a.clientProvider.OwnerServiceClient(registry).GetOwners(
				ctx,
				connect.NewRequest(
					&ownerv1beta1.GetOwnersRequest{
						OwnerRefs: []*ownerv1beta1.OwnerRef{
							{
								Value: &ownerv1beta1.OwnerRef_Id{
									Id: ownerID,
								},
							},
						},
					},
				),
			)
			if err != nil {
				return nil, err
			}
			if len(response.Msg.Owners) != 1 {
				return nil, fmt.Errorf("expected 1 Owner, got %d", len(response.Msg.Owners))
			}
			return response.Msg.Owners[0], nil
		},
	)
}
