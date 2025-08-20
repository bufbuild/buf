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
	"fmt"
	"log/slog"
	"time"

	ownerv1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/owner/v1"
	policyv1beta1 "buf.build/gen/go/bufbuild/registry/protocolbuffers/go/buf/registry/policy/v1beta1"
	"buf.build/go/standard/xslices"
	"connectrpc.com/connect"
	"github.com/bufbuild/buf/private/bufpkg/bufpolicy"
	"github.com/bufbuild/buf/private/bufpkg/bufregistryapi/bufregistryapipolicy"
	"github.com/bufbuild/buf/private/pkg/syserror"
	"github.com/bufbuild/buf/private/pkg/uuidutil"
)

// NewUploader returns a new Uploader for the given API client.
func NewUploader(
	logger *slog.Logger,
	policyClientProvider interface {
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
		bufregistryapipolicy.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) bufpolicy.Uploader {
	return newUploader(logger, policyClientProvider, options...)
}

// UploaderOption is an option for a new Uploader.
type UploaderOption func(*uploader)

// *** PRIVATE ***

type uploader struct {
	logger               *slog.Logger
	policyClientProvider interface {
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
		bufregistryapipolicy.V1Beta1UploadServiceClientProvider
	}
}

func newUploader(
	logger *slog.Logger,
	policyClientProvider interface {
		bufregistryapipolicy.V1Beta1PolicyServiceClientProvider
		bufregistryapipolicy.V1Beta1UploadServiceClientProvider
	},
	options ...UploaderOption,
) *uploader {
	uploader := &uploader{
		logger:               logger,
		policyClientProvider: policyClientProvider,
	}
	for _, option := range options {
		option(uploader)
	}
	return uploader
}

func (u *uploader) Upload(
	ctx context.Context,
	policies []bufpolicy.Policy,
	options ...bufpolicy.UploadOption,
) ([]bufpolicy.Commit, error) {
	uploadOptions, err := bufpolicy.NewUploadOptions(options)
	if err != nil {
		return nil, err
	}
	registryToIndexedPolicyKeys := xslices.ToIndexedValuesMap(
		policies,
		func(policy bufpolicy.Policy) string {
			return policy.FullName().Registry()
		},
	)
	indexedCommits := make([]xslices.Indexed[bufpolicy.Commit], 0, len(policies))
	for registry, indexedPolicyKeys := range registryToIndexedPolicyKeys {
		indexedRegistryPolicyDatas, err := u.uploadIndexedPoliciesForRegistry(
			ctx,
			registry,
			indexedPolicyKeys,
			uploadOptions,
		)
		if err != nil {
			return nil, err
		}
		indexedCommits = append(indexedCommits, indexedRegistryPolicyDatas...)
	}
	return xslices.IndexedToSortedValues(indexedCommits), nil
}

func (u *uploader) uploadIndexedPoliciesForRegistry(
	ctx context.Context,
	registry string,
	indexedPolicies []xslices.Indexed[bufpolicy.Policy],
	uploadOptions bufpolicy.UploadOptions,
) ([]xslices.Indexed[bufpolicy.Commit], error) {
	if uploadOptions.CreateIfNotExist() {
		// We must attempt to create each Policy one at a time, since CreatePolicies will return
		// an `AlreadyExists` if any of the Policies we are attempting to create already exists,
		// and no new Policies will be created.
		for _, indexedPolicy := range indexedPolicies {
			policy := indexedPolicy.Value
			if _, err := u.createPolicyIfNotExist(
				ctx,
				registry,
				policy,
				uploadOptions.CreatePolicyVisibility(),
			); err != nil {
				return nil, err
			}
		}
	}
	contents, err := xslices.MapError(indexedPolicies, func(indexedPolicy xslices.Indexed[bufpolicy.Policy]) (*policyv1beta1.UploadRequest_Content, error) {
		policy := indexedPolicy.Value
		if !policy.IsLocal() {
			return nil, syserror.New("expected local Policy in uploadIndexedPoliciesForRegistry")
		}
		if policy.FullName() == nil {
			return nil, syserror.Newf("expected Policy name for local Policy: %s", policy.Description())
		}
		config, err := policy.Config()
		if err != nil {
			return nil, err
		}
		lintConfig := config.LintConfig()
		breakingConfig := config.BreakingConfig()
		pluginConfigs := config.PluginConfigs()
		pluginConfigsProto, err := xslices.MapError(pluginConfigs, func(pluginConfig bufpolicy.PluginConfig) (*policyv1beta1.PolicyConfig_CheckPluginConfig, error) {
			optionsProto, err := pluginConfig.Options().ToProto()
			if err != nil {
				return nil, err
			}
			ref := pluginConfig.Ref()
			if ref == nil {
				return nil, fmt.Errorf("expected remote PluginConfig to have a Ref")
			}
			return &policyv1beta1.PolicyConfig_CheckPluginConfig{
				Name: &policyv1beta1.PolicyConfig_CheckPluginConfig_Name{
					Owner:  ref.FullName().Owner(),
					Plugin: ref.FullName().Name(),
					Ref:    ref.Ref(),
				},
				Options: optionsProto,
				Args:    pluginConfig.Args(),
			}, nil
		})
		if err != nil {
			return nil, err
		}
		return &policyv1beta1.UploadRequest_Content{
			PolicyRef: &policyv1beta1.PolicyRef{
				Value: &policyv1beta1.PolicyRef_Name_{
					Name: &policyv1beta1.PolicyRef_Name{
						Owner:  policy.FullName().Owner(),
						Policy: policy.FullName().Name(),
					},
				},
			},
			Config: &policyv1beta1.PolicyConfig{
				Lint: &policyv1beta1.PolicyConfig_LintConfig{
					Use:                                  lintConfig.UseIDsAndCategories(),
					Except:                               lintConfig.ExceptIDsAndCategories(),
					EnumZeroValueSuffix:                  lintConfig.EnumZeroValueSuffix(),
					RpcAllowSameRequestResponse:          lintConfig.RPCAllowSameRequestResponse(),
					RpcAllowGoogleProtobufEmptyRequests:  lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
					RpcAllowGoogleProtobufEmptyResponses: lintConfig.RPCAllowGoogleProtobufEmptyRequests(),
					ServiceSuffix:                        lintConfig.ServiceSuffix(),
				},
				Breaking: &policyv1beta1.PolicyConfig_BreakingConfig{
					Use:                    breakingConfig.UseIDsAndCategories(),
					Except:                 breakingConfig.ExceptIDsAndCategories(),
					IgnoreUnstablePackages: breakingConfig.IgnoreUnstablePackages(),
				},
				Plugins: pluginConfigsProto,
			},
			ScopedLabelRefs: xslices.Map(uploadOptions.Labels(), func(label string) *policyv1beta1.ScopedLabelRef {
				return &policyv1beta1.ScopedLabelRef{
					Value: &policyv1beta1.ScopedLabelRef_Name{
						Name: label,
					},
				}
			}),
		}, nil
	})
	if err != nil {
		return nil, err
	}

	uploadResponse, err := u.policyClientProvider.V1Beta1UploadServiceClient(registry).Upload(
		ctx,
		connect.NewRequest(&policyv1beta1.UploadRequest{
			Contents: contents,
		}))
	if err != nil {
		return nil, err
	}
	policyCommits := uploadResponse.Msg.Commits
	if len(policyCommits) != len(indexedPolicies) {
		return nil, syserror.Newf("expected %d Commits, found %d", len(indexedPolicies), len(policyCommits))
	}

	indexedCommits := make([]xslices.Indexed[bufpolicy.Commit], 0, len(indexedPolicies))
	for i, policyCommit := range policyCommits {
		policyFullName := indexedPolicies[i].Value.FullName()
		commitID, err := uuidutil.FromDashless(policyCommit.Id)
		if err != nil {
			return nil, err
		}
		policyKey, err := bufpolicy.NewPolicyKey(
			policyFullName,
			commitID,
			func() (bufpolicy.Digest, error) {
				return V1Beta1ProtoToDigest(policyCommit.Digest)
			},
		)
		if err != nil {
			return nil, err
		}
		commit := bufpolicy.NewCommit(
			policyKey,
			func() (time.Time, error) {
				return policyCommit.CreateTime.AsTime(), nil
			},
		)
		indexedCommits = append(
			indexedCommits,
			xslices.Indexed[bufpolicy.Commit]{
				Value: commit,
				Index: i,
			},
		)
	}
	return indexedCommits, nil
}

func (u *uploader) createPolicyIfNotExist(
	ctx context.Context,
	primaryRegistry string,
	policy bufpolicy.Policy,
	createPolicyVisibility bufpolicy.PolicyVisibility,
) (*policyv1beta1.Policy, error) {
	v1Beta1ProtoCreatePolicyVisibility, err := policyVisibilityToV1Beta1Proto(createPolicyVisibility)
	if err != nil {
		return nil, err
	}
	response, err := u.policyClientProvider.V1Beta1PolicyServiceClient(primaryRegistry).CreatePolicies(
		ctx,
		connect.NewRequest(
			&policyv1beta1.CreatePoliciesRequest{
				Values: []*policyv1beta1.CreatePoliciesRequest_Value{
					{
						OwnerRef: &ownerv1.OwnerRef{
							Value: &ownerv1.OwnerRef_Name{
								Name: policy.FullName().Owner(),
							},
						},
						Name:       policy.FullName().Name(),
						Visibility: v1Beta1ProtoCreatePolicyVisibility,
					},
				},
			},
		),
	)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			// If a policy already existed, then we check validate its contents.
			policies, err := u.validatePoliciesExist(ctx, primaryRegistry, []bufpolicy.Policy{policy})
			if err != nil {
				return nil, err
			}
			if len(policies) != 1 {
				return nil, syserror.Newf("expected 1 Policy, found %d", len(policies))
			}
			return policies[0], nil
		}
		return nil, err
	}
	if len(response.Msg.Policies) != 1 {
		return nil, syserror.Newf("expected 1 Policy, found %d", len(response.Msg.Policies))
	}
	// Otherwise we return the policy we created.
	return response.Msg.Policies[0], nil
}

func (u *uploader) validatePoliciesExist(
	ctx context.Context,
	primaryRegistry string,
	policies []bufpolicy.Policy,
) ([]*policyv1beta1.Policy, error) {
	response, err := u.policyClientProvider.V1Beta1PolicyServiceClient(primaryRegistry).GetPolicies(
		ctx,
		connect.NewRequest(
			&policyv1beta1.GetPoliciesRequest{
				PolicyRefs: xslices.Map(
					policies,
					func(policy bufpolicy.Policy) *policyv1beta1.PolicyRef {
						return &policyv1beta1.PolicyRef{
							Value: &policyv1beta1.PolicyRef_Name_{
								Name: &policyv1beta1.PolicyRef_Name{
									Owner:  policy.FullName().Owner(),
									Policy: policy.FullName().Name(),
								},
							},
						}
					},
				),
			},
		),
	)
	if err != nil {
		return nil, err
	}
	return response.Msg.Policies, nil
}
