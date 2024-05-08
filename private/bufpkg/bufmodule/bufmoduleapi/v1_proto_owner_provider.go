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

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/pkg/cache"
	"go.uber.org/zap"
)

// v1ProtoOwnerProvider provides a per-call provider of proto Modules.
//
// We don't want to persist these across calls - this could grow over time and this cache
// isn't an LRU cache, and the information also may change over time.
type v1ProtoOwnerProvider struct {
	logger          *zap.Logger
	clientProvider  bufapi.V1OwnerServiceClientProvider
	protoOwnerCache cache.Cache[string, *ownerv1.Owner]
}

func newV1ProtoOwnerProvider(
	logger *zap.Logger,
	clientProvider bufapi.V1OwnerServiceClientProvider,
) *v1ProtoOwnerProvider {
	return &v1ProtoOwnerProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *v1ProtoOwnerProvider) getV1ProtoOwnerForProtoOwnerID(
	ctx context.Context,
	registry string,
	// Dashless
	protoOwnerID string,
) (*ownerv1.Owner, error) {
	return a.protoOwnerCache.GetOrAdd(
		registry+"/"+protoOwnerID,
		func() (*ownerv1.Owner, error) {
			response, err := a.clientProvider.V1OwnerServiceClient(registry).GetOwners(
				ctx,
				connect.NewRequest(
					&ownerv1.GetOwnersRequest{
						OwnerRefs: []*ownerv1.OwnerRef{
							{
								Value: &ownerv1.OwnerRef_Id{
									Id: protoOwnerID,
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
