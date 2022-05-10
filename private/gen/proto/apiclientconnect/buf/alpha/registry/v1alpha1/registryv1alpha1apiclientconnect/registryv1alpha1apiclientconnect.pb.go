// Copyright 2020-2022 Buf Technologies, Inc.
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

// Code generated by protoc-gen-go-apiclientconnect. DO NOT EDIT.

package registryv1alpha1apiclientconnect

import (
	context "context"
	registryv1alpha1api "github.com/bufbuild/buf/private/gen/proto/api/buf/alpha/registry/v1alpha1/registryv1alpha1api"
	registryv1alpha1apiclient "github.com/bufbuild/buf/private/gen/proto/apiclient/buf/alpha/registry/v1alpha1/registryv1alpha1apiclient"
	registryv1alpha1connect "github.com/bufbuild/buf/private/gen/proto/connect/buf/alpha/registry/v1alpha1/registryv1alpha1connect"
	connect_go "github.com/bufbuild/connect-go"
	zap "go.uber.org/zap"
)

// NewProvider returns a new Provider.
func NewProvider(
	logger *zap.Logger,
	httpClient connect_go.HTTPClient,
	options ...ProviderOption,
) registryv1alpha1apiclient.Provider {
	provider := &provider{
		logger:     logger,
		httpClient: httpClient,
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

type provider struct {
	logger                  *zap.Logger
	httpClient              connect_go.HTTPClient
	addressMapper           func(string) string
	contextModifierProvider func(string) (func(context.Context) context.Context, error)
	scheme                  string
}

// ProviderOption is an option for a new Provider.
type ProviderOption func(*provider)

// WithAddressMapper maps the address with the given function.
func WithAddressMapper(addressMapper func(string) string) ProviderOption {
	return func(provider *provider) {
		provider.addressMapper = addressMapper
	}
}

// WithContextModifierProvider provides a function that  modifies the context before every RPC invocation.
// Applied before the address mapper.
func WithContextModifierProvider(contextModifierProvider func(address string) (func(context.Context) context.Context, error)) ProviderOption {
	return func(provider *provider) {
		provider.contextModifierProvider = contextModifierProvider
	}
}

// WithScheme prepends the given scheme to the underlying transport address
func WithScheme(scheme string) ProviderOption {
	return func(provider *provider) {
		provider.scheme = scheme
	}
}

// buildAddress modifies the given address with any additional options for transport such as the scheme and any subdomains
func (p *provider) buildAddress(address string) string {
	if p.addressMapper != nil {
		address = p.addressMapper(address)
	}
	if p.scheme != "" {
		address = p.scheme + "://" + address
	}
	return address
}

func (p *provider) NewAdminService(ctx context.Context, baseURL string) (registryv1alpha1api.AdminService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &adminService{
		logger: p.logger,
		client: registryv1alpha1connect.NewAdminServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewAuthnService(ctx context.Context, baseURL string) (registryv1alpha1api.AuthnService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &authnService{
		logger: p.logger,
		client: registryv1alpha1connect.NewAuthnServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewAuthzService(ctx context.Context, baseURL string) (registryv1alpha1api.AuthzService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &authzService{
		logger: p.logger,
		client: registryv1alpha1connect.NewAuthzServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewConvertService(ctx context.Context, baseURL string) (registryv1alpha1api.ConvertService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &convertService{
		logger: p.logger,
		client: registryv1alpha1connect.NewConvertServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDisplayService(ctx context.Context, baseURL string) (registryv1alpha1api.DisplayService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &displayService{
		logger: p.logger,
		client: registryv1alpha1connect.NewDisplayServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDocService(ctx context.Context, baseURL string) (registryv1alpha1api.DocService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &docService{
		logger: p.logger,
		client: registryv1alpha1connect.NewDocServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewDownloadService(ctx context.Context, baseURL string) (registryv1alpha1api.DownloadService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &downloadService{
		logger: p.logger,
		client: registryv1alpha1connect.NewDownloadServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewGenerateService(ctx context.Context, baseURL string) (registryv1alpha1api.GenerateService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &generateService{
		logger: p.logger,
		client: registryv1alpha1connect.NewGenerateServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewGithubService(ctx context.Context, baseURL string) (registryv1alpha1api.GithubService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &githubService{
		logger: p.logger,
		client: registryv1alpha1connect.NewGithubServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewImageService(ctx context.Context, baseURL string) (registryv1alpha1api.ImageService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &imageService{
		logger: p.logger,
		client: registryv1alpha1connect.NewImageServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewJSONSchemaService(ctx context.Context, baseURL string) (registryv1alpha1api.JSONSchemaService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &jSONSchemaService{
		logger: p.logger,
		client: registryv1alpha1connect.NewJSONSchemaServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewLocalResolveService(ctx context.Context, baseURL string) (registryv1alpha1api.LocalResolveService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &localResolveService{
		logger: p.logger,
		client: registryv1alpha1connect.NewLocalResolveServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewOrganizationService(ctx context.Context, baseURL string) (registryv1alpha1api.OrganizationService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &organizationService{
		logger: p.logger,
		client: registryv1alpha1connect.NewOrganizationServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewOwnerService(ctx context.Context, baseURL string) (registryv1alpha1api.OwnerService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &ownerService{
		logger: p.logger,
		client: registryv1alpha1connect.NewOwnerServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewPluginService(ctx context.Context, baseURL string) (registryv1alpha1api.PluginService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &pluginService{
		logger: p.logger,
		client: registryv1alpha1connect.NewPluginServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewPushService(ctx context.Context, baseURL string) (registryv1alpha1api.PushService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &pushService{
		logger: p.logger,
		client: registryv1alpha1connect.NewPushServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRecommendationService(ctx context.Context, baseURL string) (registryv1alpha1api.RecommendationService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &recommendationService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRecommendationServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewReferenceService(ctx context.Context, baseURL string) (registryv1alpha1api.ReferenceService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &referenceService{
		logger: p.logger,
		client: registryv1alpha1connect.NewReferenceServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryBranchService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryBranchService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryBranchService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryBranchServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryCommitService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryCommitService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryCommitService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryCommitServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryTagService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryTagService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryTagService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTagServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryTrackCommitService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryTrackCommitService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryTrackCommitService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTrackCommitServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewRepositoryTrackService(ctx context.Context, baseURL string) (registryv1alpha1api.RepositoryTrackService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &repositoryTrackService{
		logger: p.logger,
		client: registryv1alpha1connect.NewRepositoryTrackServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewResolveService(ctx context.Context, baseURL string) (registryv1alpha1api.ResolveService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &resolveService{
		logger: p.logger,
		client: registryv1alpha1connect.NewResolveServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewSearchService(ctx context.Context, baseURL string) (registryv1alpha1api.SearchService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &searchService{
		logger: p.logger,
		client: registryv1alpha1connect.NewSearchServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewTokenService(ctx context.Context, baseURL string) (registryv1alpha1api.TokenService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &tokenService{
		logger: p.logger,
		client: registryv1alpha1connect.NewTokenServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}

func (p *provider) NewUserService(ctx context.Context, baseURL string) (registryv1alpha1api.UserService, error) {
	var contextModifier func(context.Context) context.Context
	var err error
	if p.contextModifierProvider != nil {
		contextModifier, err = p.contextModifierProvider(baseURL)
		if err != nil {
			return nil, err
		}
	}
	return &userService{
		logger: p.logger,
		client: registryv1alpha1connect.NewUserServiceClient(
			p.httpClient,
			p.buildAddress(baseURL),
			connect_go.WithGRPC(),
		),
		contextModifier: contextModifier,
	}, nil
}
