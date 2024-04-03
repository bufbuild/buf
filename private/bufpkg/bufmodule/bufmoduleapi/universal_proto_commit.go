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
	"time"

	modulev1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1"
	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"github.com/bufbuild/buf/private/pkg/slicesext"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/gofrs/uuid/v5"
)

type universalProtoCommit struct {
	// Dashless
	ID string
	// Dashless
	OwnerID string
	// Dashless
	ModuleID   string
	CreateTime time.Time
	Digest     bufmodule.Digest
}

func newUniversalProtoCommitForV1(v1ProtoCommit *modulev1.Commit) (*universalProtoCommit, error) {
	digest, err := V1ProtoToDigest(v1ProtoCommit.Digest)
	if err != nil {
		return nil, err
	}
	return &universalProtoCommit{
		ID:         v1ProtoCommit.Id,
		OwnerID:    v1ProtoCommit.OwnerId,
		ModuleID:   v1ProtoCommit.ModuleId,
		CreateTime: v1ProtoCommit.CreateTime.AsTime(),
		Digest:     digest,
	}, nil
}

func newUniversalProtoCommitForV1Beta1(v1beta1ProtoCommit *modulev1beta1.Commit) (*universalProtoCommit, error) {
	digest, err := V1Beta1ProtoToDigest(v1beta1ProtoCommit.Digest)
	if err != nil {
		return nil, err
	}
	return &universalProtoCommit{
		ID:         v1beta1ProtoCommit.Id,
		OwnerID:    v1beta1ProtoCommit.OwnerId,
		ModuleID:   v1beta1ProtoCommit.ModuleId,
		CreateTime: v1beta1ProtoCommit.CreateTime.AsTime(),
		Digest:     digest,
	}, nil
}

func getUniversalProtoCommitForRegistryAndCommitID(
	ctx context.Context,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
	registry string,
	commitID uuid.UUID,
	digestType bufmodule.DigestType,
) (*universalProtoCommit, error) {
	universalProtoCommits, err := getUniversalProtoCommitsForRegistryAndCommitIDs(ctx, clientProvider, registry, []uuid.UUID{commitID}, digestType)
	if err != nil {
		return nil, err
	}
	// We already do length checking in getUniversalProtoCommitsForRegistryAndCommitIDs.
	return universalProtoCommits[0], nil
}

func getUniversalProtoCommitsForRegistryAndCommitIDs(
	ctx context.Context,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
	registry string,
	commitIDs []uuid.UUID,
	digestType bufmodule.DigestType,
) ([]*universalProtoCommit, error) {
	switch digestType {
	case bufmodule.DigestTypeB4:
		v1beta1ProtoResourceRefs := commitIDsToV1Beta1ProtoResourceRefs(commitIDs)
		v1beta1ProtoCommits, err := getV1Beta1ProtoCommitsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1beta1ProtoResourceRefs, digestType)
		if err != nil {
			return nil, err
		}
		return slicesext.MapError(v1beta1ProtoCommits, newUniversalProtoCommitForV1Beta1)
	case bufmodule.DigestTypeB5:
		v1ProtoResourceRefs := commitIDsToV1ProtoResourceRefs(commitIDs)
		v1ProtoCommits, err := getV1ProtoCommitsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1ProtoResourceRefs)
		if err != nil {
			return nil, err
		}
		return slicesext.MapError(v1ProtoCommits, newUniversalProtoCommitForV1)
	default:
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

func getUniversalProtoCommitsForRegistryAndModuleRefs(
	ctx context.Context,
	clientProvider interface {
		bufapi.V1CommitServiceClientProvider
		bufapi.V1Beta1CommitServiceClientProvider
	},
	registry string,
	moduleRefs []bufmodule.ModuleRef,
	digestType bufmodule.DigestType,
) ([]*universalProtoCommit, error) {
	switch digestType {
	case bufmodule.DigestTypeB4:
		v1beta1ProtoResourceRefs := moduleRefsToV1Beta1ProtoResourceRefs(moduleRefs)
		v1beta1ProtoCommits, err := getV1Beta1ProtoCommitsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1beta1ProtoResourceRefs, digestType)
		if err != nil {
			return nil, err
		}
		return slicesext.MapError(v1beta1ProtoCommits, newUniversalProtoCommitForV1Beta1)
	case bufmodule.DigestTypeB5:
		v1ProtoResourceRefs := moduleRefsToV1ProtoResourceRefs(moduleRefs)
		v1ProtoCommits, err := getV1ProtoCommitsForRegistryAndResourceRefs(ctx, clientProvider, registry, v1ProtoResourceRefs)
		if err != nil {
			return nil, err
		}
		return slicesext.MapError(v1ProtoCommits, newUniversalProtoCommitForV1)
	default:
		return nil, syserror.Newf("unknown DigestType: %v", digestType)
	}
}

func getV1ProtoCommitsForRegistryAndResourceRefs(
	ctx context.Context,
	clientProvider bufapi.V1CommitServiceClientProvider,
	registry string,
	v1ProtoResourceRefs []*modulev1.ResourceRef,
) ([]*modulev1.Commit, error) {
	response, err := clientProvider.V1CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1.GetCommitsRequest{
				// TODO FUTURE: chunking
				ResourceRefs: v1ProtoResourceRefs,
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}
	if len(response.Msg.Commits) != len(v1ProtoResourceRefs) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(v1ProtoResourceRefs), len(response.Msg.Commits))
	}
	return response.Msg.Commits, nil
}

func getV1Beta1ProtoCommitsForRegistryAndResourceRefs(
	ctx context.Context,
	clientProvider bufapi.V1Beta1CommitServiceClientProvider,
	registry string,
	v1beta1ProtoResourceRefs []*modulev1beta1.ResourceRef,
	digestType bufmodule.DigestType,
) ([]*modulev1beta1.Commit, error) {
	v1beta1ProtoDigestType, err := digestTypeToV1Beta1Proto(digestType)
	if err != nil {
		return nil, err
	}
	response, err := clientProvider.V1Beta1CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitsRequest{
				// TODO FUTURE: chunking
				ResourceRefs: v1beta1ProtoResourceRefs,
				DigestType:   v1beta1ProtoDigestType,
			},
		),
	)
	if err != nil {
		return nil, maybeNewNotFoundError(err)
	}
	if len(response.Msg.Commits) != len(v1beta1ProtoResourceRefs) {
		return nil, fmt.Errorf("expected %d Commits, got %d", len(v1beta1ProtoResourceRefs), len(response.Msg.Commits))
	}
	return response.Msg.Commits, nil
}
