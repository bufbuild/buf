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
	"fmt"
	"io/fs"
	"time"

	modulev1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/module/v1beta1"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufapi"
	"github.com/bufbuild/buf/private/bufpkg/bufmodule"
	"go.uber.org/zap"
)

// NewCommitProvider returns a new CommitProvider for the given API client.
//
// A warning is printed to the logger if a given Module is deprecated.
func NewCommitProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) bufmodule.CommitProvider {
	return newCommitProvider(logger, clientProvider)
}

// *** PRIVATE ***

type commitProvider struct {
	logger         *zap.Logger
	clientProvider bufapi.ClientProvider
}

func newCommitProvider(
	logger *zap.Logger,
	clientProvider bufapi.ClientProvider,
) *commitProvider {
	return &commitProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (a *commitProvider) GetOptionalCommitsForModuleKeys(
	ctx context.Context,
	moduleKeys ...bufmodule.ModuleKey,
) ([]bufmodule.OptionalCommit, error) {
	// We don't want to persist these across calls - this could grow over time and this cache
	// isn't an LRU cache, and the information also may change over time.
	protoModuleProvider := newProtoModuleProvider(a.logger, a.clientProvider)
	protoOwnerProvider := newProtoOwnerProvider(a.logger, a.clientProvider)
	// TODO: Do the work to coalesce ModuleKeys by registry hostname, make calls out to the CommitService
	// per registry, then get back the resulting data, and order it in the same order as the input ModuleKeys.
	// Make sure to respect 250 max.
	optionalCommits := make([]bufmodule.OptionalCommit, len(moduleKeys))
	for i, moduleKey := range moduleKeys {
		moduleData, err := a.getCommitForModuleKey(
			ctx,
			protoModuleProvider,
			protoOwnerProvider,
			moduleKey,
		)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		optionalCommits[i] = bufmodule.NewOptionalCommit(moduleData)
	}
	return optionalCommits, nil
}

func (a *commitProvider) getCommitForModuleKey(
	ctx context.Context,
	protoModuleProvider *protoModuleProvider,
	protoOwnerProvider *protoOwnerProvider,
	moduleKey bufmodule.ModuleKey,
) (bufmodule.Commit, error) {
	registryHostname := moduleKey.ModuleFullName().Registry()

	protoCommitID, err := CommitIDToProto(moduleKey.CommitID())
	if err != nil {
		return nil, err
	}
	response, err := a.clientProvider.CommitServiceClient(registryHostname).GetCommits(
		ctx,
		connect.NewRequest(
			&modulev1beta1.GetCommitsRequest{
				ResourceRefs: []*modulev1beta1.ResourceRef{
					{
						Value: &modulev1beta1.ResourceRef_Id{
							Id: protoCommitID,
						},
					},
				},
				DigestType: modulev1beta1.DigestType_DIGEST_TYPE_B5,
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, &fs.PathError{Op: "read", Path: moduleKey.String(), Err: fs.ErrNotExist}
		}
		return nil, err
	}
	if len(response.Msg.Commits) != 1 {
		return nil, fmt.Errorf("expected 1 Commit, got %d", len(response.Msg.Commits))
	}
	protoCommit := response.Msg.Commits[0]
	receivedDigest, err := ProtoToDigest(protoCommit.Digest)
	if err != nil {
		return nil, err
	}
	return bufmodule.NewCommit(
		moduleKey,
		func() (time.Time, error) {
			return protoCommit.CreateTime.AsTime(), nil
		},
		bufmodule.CommitWithReceivedDigest(receivedDigest),
	)
}
