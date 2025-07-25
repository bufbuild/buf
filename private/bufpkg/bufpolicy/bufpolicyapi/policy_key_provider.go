// Copyright 2020-2025 Buf Technologies, Inc.
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

package bufpolicyapi

import (
	"context"
	"log/slog"

	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// NewPolicyKeyProvider returns a new PolicyKeyProvider for the given API clients.
func NewPolicyKeyProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapipolicy.V1Beta1CommitServiceClientProvider
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
	},
) bufpolicy.PolicyKeyProvider {
	return newPolicyKeyProvider(logger, clientProvider)
}

// *** PRIVATE ***

type policyKeyProvider struct {
	logger         *slog.Logger
	clientProvider interface {
		bufregistryapipolicy.V1Beta1CommitServiceClientProvider
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
	}
}

func newPolicyKeyProvider(
	logger *slog.Logger,
	clientProvider interface {
		bufregistryapipolicy.V1Beta1CommitServiceClientProvider
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
	},
) *policyKeyProvider {
	return &policyKeyProvider{
		logger:         logger,
		clientProvider: clientProvider,
	}
}

func (p *policyKeyProvider) GetPolicyKeysForPolicyRefs(
	ctx context.Context,
	policyRefs []bufparse.Ref,
	digestType bufpolicy.DigestType,
) ([]bufpolicy.PolicyKey, error) {
	if len(policyRefs) == 0 {
		return nil, nil
	}
	// Check unique policyRefs.
	if _, err := xslices.ToUniqueValuesMapError(
		policyRefs,
		func(policyRef bufparse.Ref) (string, error) {
			return policyRef.String(), nil
		},
	); err != nil {
		return nil, err
	}
	registryToIndexedPolicyRefs := xslices.ToIndexedValuesMap(
		policyRefs,
		func(policyRef bufparse.Ref) string {
			return policyRef.FullName().Registry()
		},
	)
	indexedPolicyKeys := make([]xslices.Indexed[bufpolicy.PolicyKey], 0, len(policyRefs))
	for registry, indexedPolicyRefs := range registryToIndexedPolicyRefs {
		indexedRegistryPolicyKeys, err := p.getIndexedPolicyKeysForRegistryAndIndexedPolicyRefs(
			ctx,
			registry,
			indexedPolicyRefs,
			digestType,
		)
		if err != nil {
			return nil, err
		}
		indexedPolicyKeys = append(indexedPolicyKeys, indexedRegistryPolicyKeys...)
	}
	return xslices.IndexedToSortedValues(indexedPolicyKeys), nil
}

func (p *policyKeyProvider) getIndexedPolicyKeysForRegistryAndIndexedPolicyRefs(
	ctx context.Context,
	registry string,
	indexedPolicyRefs []xslices.Indexed[bufparse.Ref],
	digestType bufpolicy.DigestType,
) ([]xslices.Indexed[bufpolicy.PolicyKey], error) {
	resourceRefs := xslices.Map(indexedPolicyRefs, func(indexedPolicyRef xslices.Indexed[bufparse.Ref]) *policyv1beta1.ResourceRef {
		resourceRefName := &policyv1beta1.ResourceRef_Name{
			Owner:  indexedPolicyRef.Value.FullName().Owner(),
			Policy: indexedPolicyRef.Value.FullName().Name(),
		}
		if ref := indexedPolicyRef.Value.Ref(); ref != "" {
			resourceRefName.Child = &policyv1beta1.ResourceRef_Name_Ref{
				Ref: ref,
			}
		}
		return &policyv1beta1.ResourceRef{
			Value: &policyv1beta1.ResourceRef_Name_{
				Name: resourceRefName,
			},
		}
	})
	policyResponse, err := p.clientProvider.V1Beta1CommitServiceClient(registry).GetCommits(
		ctx,
		connect.NewRequest(&policyv1beta1.GetCommitsRequest{
			ResourceRefs: resourceRefs,
		}),
	)
	if err != nil {
		return nil, err
	}
	commits := policyResponse.Msg.Commits
	if len(commits) != len(indexedPolicyRefs) {
		return nil, syserror.New("did not get the expected number of policy datas")
	}

	indexedPolicyKeys := make([]xslices.Indexed[bufpolicy.PolicyKey], len(commits))
	for i, commit := range commits {
		commitID, err := uuidutil.FromDashless(commit.Id)
		if err != nil {
			return nil, err
		}
		digest, err := V1Beta1ProtoToDigest(commit.Digest)
		if err != nil {
			return nil, err
		}
		policyKey, err := bufpolicy.NewPolicyKey(
			// Note we don't have to resolve owner_name and policy_name since we already have them.
			indexedPolicyRefs[i].Value.FullName(),
			commitID,
			func() (bufpolicy.Digest, error) {
				return digest, nil
			},
		)
		if err != nil {
			return nil, err
		}
		indexedPolicyKeys[i] = xslices.Indexed[bufpolicy.PolicyKey]{
			Value: policyKey,
			Index: indexedPolicyRefs[i].Index,
		}
	}
	return indexedPolicyKeys, nil
}
